package main

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"time"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/metric"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/trace"

	"gorm.io/gorm"
)

type LoginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

type RegisterRequest struct {
	Username string `json:"username"`
	Email    string `json:"email"`
	Password string `json:"password"`
}

type AuthResponse struct {
	Success bool   `json:"success"`
	Message string `json:"message"`
	Token   string `json:"token,omitempty"`
	User    *User  `json:"user,omitempty"`
}

type ErrorResponse struct {
	Success bool   `json:"success"`
	Message string `json:"message"`
	Error   string `json:"error,omitempty"`
}

func generateJWT(user *User) (string, error)              { return "dummy-jwt", nil }
func hashPassword(password string) (string, error)        { return "hashed-password", nil }
func verifyPassword(hashedPassword, password string) bool { return true }

func loginUser(w http.ResponseWriter, r *http.Request) {
	fmt.Println("Login request received")
	if DB == nil {
		fmt.Println("DB is not initialized")
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(ErrorResponse{
			Success: false,
			Message: "Database not available",
			Error:   "DB_NOT_INITIALIZED",
		})
		return
	}

	ctx := otel.GetTextMapPropagator().Extract(r.Context(), propagation.HeaderCarrier(r.Header))
	tracer := otel.Tracer("stock-tracker-app-tracer")
	startTime := time.Now()

	ctx, span := tracer.Start(ctx, "auth.login",
		trace.WithAttributes(
			attribute.String("http.method", r.Method),
			attribute.String("http.route", "/login"),
			attribute.String("component", "auth_service"),
			attribute.String("operation", "user_login"),
			attribute.String("user_agent", r.UserAgent()),
			attribute.String("remote_addr", r.RemoteAddr),
		),
	)
	defer span.End()

	traceID := span.SpanContext().TraceID().String()
	spanID := span.SpanContext().SpanID().String()

	baseAttrs := []attribute.KeyValue{
		attribute.String("endpoint", "login"),
		attribute.String("method", r.Method),
		attribute.String("trace_id", traceID),
		attribute.String("span_id", spanID),
		attribute.String("component", "auth_service"),
	}

	w.Header().Set("Content-Type", "application/json")

	if httpRequestCount != nil {
		httpRequestCount.Add(ctx, 1, metric.WithAttributes(baseAttrs...))
	}

	if loginAttempts != nil {
		loginAttempts.Add(ctx, 1, metric.WithAttributes(baseAttrs...))
	}

	Logger.InfoContext(ctx, "Login attempt started",
		slog.String("remote_addr", r.RemoteAddr),
		slog.String("user_agent", r.UserAgent()),
	)

	ctx, parseSpan := tracer.Start(ctx, "auth.login.parse_request",
		trace.WithAttributes(
			attribute.String("operation", "parse_json"),
		),
	)

	var loginReq LoginRequest
	if err := json.NewDecoder(r.Body).Decode(&loginReq); err != nil {
		parseSpan.RecordError(err)
		parseSpan.SetStatus(codes.Error, "Invalid JSON")
		parseSpan.End()
		span.RecordError(err)
		span.SetStatus(codes.Error, "Invalid JSON")

		Logger.ErrorContext(ctx, "Failed to parse login request",
			slog.String("error", err.Error()),
		)

		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(ErrorResponse{
			Success: false,
			Message: "Invalid JSON payload",
			Error:   err.Error(),
		})
		return
	}
	parseSpan.SetStatus(codes.Ok, "JSON parsed successfully")
	parseSpan.End()

	ctx, validateSpan := tracer.Start(ctx, "auth.login.validate_request",
		trace.WithAttributes(
			attribute.String("operation", "validate_fields"),
			attribute.String("email", loginReq.Email),
		),
	)

	if loginReq.Email == "" || loginReq.Password == "" {
		validateSpan.SetStatus(codes.Error, "Missing required fields")
		validateSpan.End()
		span.SetStatus(codes.Error, "Missing required fields")

		Logger.ErrorContext(ctx, "Login validation failed - missing fields",
			slog.String("email", loginReq.Email),
			slog.Bool("email_empty", loginReq.Email == ""),
			slog.Bool("password_empty", loginReq.Password == ""),
		)

		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(ErrorResponse{
			Success: false,
			Message: "Email and password are required",
		})
		return
	}
	validateSpan.SetStatus(codes.Ok, "Request validated successfully")
	validateSpan.End()

	var user User
	dbStartTime := time.Now()
	dbCtx, dbSpan := tracer.Start(ctx, "auth.login.db_lookup",
		trace.WithAttributes(
			attribute.String("operation", "find_user_by_email"),
			attribute.String("email", loginReq.Email),
			attribute.String("table", "users"),
		),
	)

	err := DB.WithContext(dbCtx).Where("email = ?", loginReq.Email).First(&user).Error
	dbDuration := time.Since(dbStartTime)

	if dbQueryCount != nil {
		dbQueryCount.Add(ctx, 1, metric.WithAttributes(append(baseAttrs,
			attribute.String("query_type", "select"),
			attribute.String("table", "users"),
		)...))
	}

	if dbQueryDuration != nil {
		dbQueryDuration.Record(ctx, dbDuration.Seconds(), metric.WithAttributes(append(baseAttrs,
			attribute.String("query_type", "select"),
			attribute.String("table", "users"),
		)...))
	}

	if err != nil {
		dbSpan.RecordError(err)
		dbSpan.SetStatus(codes.Error, "Database lookup failed")
		dbSpan.End()
		span.RecordError(err)

		if err == gorm.ErrRecordNotFound {
			Logger.InfoContext(ctx, "Login failed - user not found",
				slog.String("email", loginReq.Email),
				slog.Duration("db_duration", dbDuration),
			)

			w.WriteHeader(http.StatusUnauthorized)
			json.NewEncoder(w).Encode(ErrorResponse{
				Success: false,
				Message: "Invalid credentials",
			})
		} else {
			Logger.ErrorContext(ctx, "Database error during login",
				slog.String("email", loginReq.Email),
				slog.String("error", err.Error()),
				slog.Duration("db_duration", dbDuration),
			)

			w.WriteHeader(http.StatusInternalServerError)
			json.NewEncoder(w).Encode(ErrorResponse{
				Success: false,
				Message: "Database error",
				Error:   err.Error(),
			})
		}
		return
	}

	dbSpan.SetStatus(codes.Ok, "User found successfully")
	dbSpan.SetAttributes(
		attribute.Int("user_id", int(user.ID)),
		attribute.String("username", user.Username),
	)
	dbSpan.End()

	ctx, verifySpan := tracer.Start(ctx, "auth.login.verify_password",
		trace.WithAttributes(
			attribute.String("operation", "password_verification"),
			attribute.Int("user_id", int(user.ID)),
		),
	)

	if !verifyPassword(user.Password, loginReq.Password) {
		verifySpan.SetStatus(codes.Error, "Invalid password")
		verifySpan.End()
		span.SetStatus(codes.Error, "Invalid password")

		Logger.InfoContext(ctx, "Login failed - invalid password",
			slog.String("email", loginReq.Email),
			slog.Int("user_id", int(user.ID)),
		)

		w.WriteHeader(http.StatusUnauthorized)
		json.NewEncoder(w).Encode(ErrorResponse{
			Success: false,
			Message: "Invalid credentials",
		})
		return
	}
	verifySpan.SetStatus(codes.Ok, "Password verified successfully")
	verifySpan.End()

	ctx, tokenSpan := tracer.Start(ctx, "auth.login.generate_token",
		trace.WithAttributes(
			attribute.String("operation", "jwt_generation"),
			attribute.Int("user_id", int(user.ID)),
		),
	)

	token, err := generateJWT(&user)
	if err != nil {
		tokenSpan.RecordError(err)
		tokenSpan.SetStatus(codes.Error, "Token generation failed")
		tokenSpan.End()
		span.RecordError(err)

		Logger.ErrorContext(ctx, "JWT generation failed",
			slog.String("email", loginReq.Email),
			slog.Int("user_id", int(user.ID)),
			slog.String("error", err.Error()),
		)

		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(ErrorResponse{
			Success: false,
			Message: "Token generation failed",
			Error:   err.Error(),
		})
		return
	}
	tokenSpan.SetStatus(codes.Ok, "Token generated successfully")
	tokenSpan.End()

	if authDuration != nil {
		authDuration.Record(ctx, time.Since(startTime).Seconds(),
			metric.WithAttributes(append(baseAttrs,
				attribute.String("status", "success"),
				attribute.Int("user_id", int(user.ID)),
			)...),
		)
	}

	span.SetStatus(codes.Ok, "Login successful")
	span.SetAttributes(
		attribute.Int("user_id", int(user.ID)),
		attribute.String("username", user.Username),
		attribute.String("status", "success"),
	)

	Logger.InfoContext(ctx, "Login successful",
		slog.String("email", loginReq.Email),
		slog.Int("user_id", int(user.ID)),
		slog.String("username", user.Username),
		slog.Duration("total_duration", time.Since(startTime)),
	)

	json.NewEncoder(w).Encode(AuthResponse{
		Success: true,
		Message: "Login successful",
		Token:   token,
		User:    &user,
	})
}

func registerUser(w http.ResponseWriter, r *http.Request) {
	if DB == nil {
		fmt.Println("DB is not initialized")
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(ErrorResponse{
			Success: false,
			Message: "Database not available",
			Error:   "DB_NOT_INITIALIZED",
		})
		return
	}

	ctx := otel.GetTextMapPropagator().Extract(r.Context(), propagation.HeaderCarrier(r.Header))
	tracer := otel.Tracer("stock-tracker-app-tracer")
	startTime := time.Now()

	ctx, span := tracer.Start(ctx, "auth.register",
		trace.WithAttributes(
			attribute.String("http.method", r.Method),
			attribute.String("http.route", "/register"),
			attribute.String("component", "auth_service"),
			attribute.String("operation", "user_registration"),
			attribute.String("user_agent", r.UserAgent()),
			attribute.String("remote_addr", r.RemoteAddr),
		),
	)
	defer span.End()

	traceID := span.SpanContext().TraceID().String()
	spanID := span.SpanContext().SpanID().String()

	baseAttrs := []attribute.KeyValue{
		attribute.String("endpoint", "register"),
		attribute.String("method", r.Method),
		attribute.String("trace_id", traceID),
		attribute.String("span_id", spanID),
		attribute.String("component", "auth_service"),
	}

	w.Header().Set("Content-Type", "application/json")

	if httpRequestCount != nil {
		httpRequestCount.Add(ctx, 1, metric.WithAttributes(baseAttrs...))
	}

	if registerAttempts != nil {
		registerAttempts.Add(ctx, 1, metric.WithAttributes(baseAttrs...))
	}

	Logger.InfoContext(ctx, "Registration attempt started",
		slog.String("remote_addr", r.RemoteAddr),
		slog.String("user_agent", r.UserAgent()),
	)

	ctx, parseSpan := tracer.Start(ctx, "auth.register.parse_request",
		trace.WithAttributes(
			attribute.String("operation", "parse_json"),
		),
	)

	var req RegisterRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		parseSpan.RecordError(err)
		parseSpan.SetStatus(codes.Error, "Invalid JSON")
		parseSpan.End()
		span.RecordError(err)

		Logger.ErrorContext(ctx, "Failed to parse registration request",
			slog.String("error", err.Error()),
		)

		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(ErrorResponse{
			Success: false,
			Message: "Invalid JSON payload",
			Error:   err.Error(),
		})
		return
	}
	parseSpan.SetStatus(codes.Ok, "JSON parsed successfully")
	parseSpan.End()

	ctx, validateSpan := tracer.Start(ctx, "auth.register.validate_request",
		trace.WithAttributes(
			attribute.String("operation", "validate_fields"),
			attribute.String("email", req.Email),
			attribute.String("username", req.Username),
		),
	)

	if req.Email == "" || req.Password == "" || req.Username == "" {
		validateSpan.SetStatus(codes.Error, "Missing required fields")
		validateSpan.End()
		span.SetStatus(codes.Error, "Missing required fields")

		Logger.ErrorContext(ctx, "Registration validation failed - missing fields",
			slog.String("email", req.Email),
			slog.String("username", req.Username),
			slog.Bool("email_empty", req.Email == ""),
			slog.Bool("username_empty", req.Username == ""),
			slog.Bool("password_empty", req.Password == ""),
		)

		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(ErrorResponse{
			Success: false,
			Message: "Username, email, and password are required",
		})
		return
	}
	validateSpan.SetStatus(codes.Ok, "Request validated successfully")
	validateSpan.End()

	var existing User
	dbStartTime := time.Now()
	dbCtx, emailCheckSpan := tracer.Start(ctx, "auth.register.check_email",
		trace.WithAttributes(
			attribute.String("operation", "check_email_exists"),
			attribute.String("email", req.Email),
			attribute.String("table", "users"),
		),
	)

	err := DB.WithContext(dbCtx).Where("email = ?", req.Email).First(&existing).Error
	emailCheckDuration := time.Since(dbStartTime)

	if dbQueryCount != nil {
		dbQueryCount.Add(ctx, 1, metric.WithAttributes(append(baseAttrs,
			attribute.String("query_type", "select"),
			attribute.String("table", "users"),
			attribute.String("check_type", "email"),
		)...))
	}

	if dbQueryDuration != nil {
		dbQueryDuration.Record(ctx, emailCheckDuration.Seconds(), metric.WithAttributes(append(baseAttrs,
			attribute.String("query_type", "select"),
			attribute.String("table", "users"),
			attribute.String("check_type", "email"),
		)...))
	}

	if err == nil {
		emailCheckSpan.SetStatus(codes.Error, "Email already exists")
		emailCheckSpan.End()

		Logger.InfoContext(ctx, "Registration failed - email already exists",
			slog.String("email", req.Email),
			slog.Duration("db_duration", emailCheckDuration),
		)

		w.WriteHeader(http.StatusConflict)
		json.NewEncoder(w).Encode(ErrorResponse{
			Success: false,
			Message: "Email already in use",
		})
		return
	} else if err != gorm.ErrRecordNotFound {
		emailCheckSpan.RecordError(err)
		emailCheckSpan.SetStatus(codes.Error, "Database error")
		emailCheckSpan.End()
		span.RecordError(err)

		Logger.ErrorContext(ctx, "Database error during email check",
			slog.String("email", req.Email),
			slog.String("error", err.Error()),
			slog.Duration("db_duration", emailCheckDuration),
		)

		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(ErrorResponse{
			Success: false,
			Message: "Database error",
			Error:   err.Error(),
		})
		return
	}
	emailCheckSpan.SetStatus(codes.Ok, "Email available")
	emailCheckSpan.End()

	dbStartTime = time.Now()
	dbCtx, usernameCheckSpan := tracer.Start(ctx, "auth.register.check_username",
		trace.WithAttributes(
			attribute.String("operation", "check_username_exists"),
			attribute.String("username", req.Username),
			attribute.String("table", "users"),
		),
	)

	err = DB.WithContext(dbCtx).Where("username = ?", req.Username).First(&existing).Error
	usernameCheckDuration := time.Since(dbStartTime)

	if dbQueryCount != nil {
		dbQueryCount.Add(ctx, 1, metric.WithAttributes(append(baseAttrs,
			attribute.String("query_type", "select"),
			attribute.String("table", "users"),
			attribute.String("check_type", "username"),
		)...))
	}

	if dbQueryDuration != nil {
		dbQueryDuration.Record(ctx, usernameCheckDuration.Seconds(), metric.WithAttributes(append(baseAttrs,
			attribute.String("query_type", "select"),
			attribute.String("table", "users"),
			attribute.String("check_type", "username"),
		)...))
	}

	if err == nil {
		usernameCheckSpan.SetStatus(codes.Error, "Username already exists")
		usernameCheckSpan.End()

		Logger.InfoContext(ctx, "Registration failed - username already taken",
			slog.String("username", req.Username),
			slog.Duration("db_duration", usernameCheckDuration),
		)

		w.WriteHeader(http.StatusConflict)
		json.NewEncoder(w).Encode(ErrorResponse{
			Success: false,
			Message: "Username already taken",
		})
		return
	} else if err != gorm.ErrRecordNotFound {
		usernameCheckSpan.RecordError(err)
		usernameCheckSpan.SetStatus(codes.Error, "Database error")
		usernameCheckSpan.End()
		span.RecordError(err)

		Logger.ErrorContext(ctx, "Database error during username check",
			slog.String("username", req.Username),
			slog.String("error", err.Error()),
			slog.Duration("db_duration", usernameCheckDuration),
		)

		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(ErrorResponse{
			Success: false,
			Message: "Database error",
			Error:   err.Error(),
		})
		return
	}
	usernameCheckSpan.SetStatus(codes.Ok, "Username available")
	usernameCheckSpan.End()

	ctx, hashSpan := tracer.Start(ctx, "auth.register.hash_password",
		trace.WithAttributes(
			attribute.String("operation", "password_hashing"),
		),
	)

	hashedPwd, err := hashPassword(req.Password)
	if err != nil {
		hashSpan.RecordError(err)
		hashSpan.SetStatus(codes.Error, "Password hashing failed")
		hashSpan.End()
		span.RecordError(err)

		Logger.ErrorContext(ctx, "Password hashing failed",
			slog.String("email", req.Email),
			slog.String("username", req.Username),
			slog.String("error", err.Error()),
		)

		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(ErrorResponse{
			Success: false,
			Message: "Password hashing failed",
			Error:   err.Error(),
		})
		return
	}
	hashSpan.SetStatus(codes.Ok, "Password hashed successfully")
	hashSpan.End()

	user := &User{
		Username: req.Username,
		Email:    req.Email,
		Password: hashedPwd,
	}

	dbStartTime = time.Now()
	dbCtx, createSpan := tracer.Start(ctx, "auth.register.create_user",
		trace.WithAttributes(
			attribute.String("operation", "create_user"),
			attribute.String("email", req.Email),
			attribute.String("username", req.Username),
			attribute.String("table", "users"),
		),
	)

	err = DB.WithContext(dbCtx).Create(user).Error
	createDuration := time.Since(dbStartTime)

	if dbQueryCount != nil {
		dbQueryCount.Add(ctx, 1, metric.WithAttributes(append(baseAttrs,
			attribute.String("query_type", "insert"),
			attribute.String("table", "users"),
		)...))
	}

	if dbQueryDuration != nil {
		dbQueryDuration.Record(ctx, createDuration.Seconds(), metric.WithAttributes(append(baseAttrs,
			attribute.String("query_type", "insert"),
			attribute.String("table", "users"),
		)...))
	}

	if err != nil {
		createSpan.RecordError(err)
		createSpan.SetStatus(codes.Error, "User creation failed")
		createSpan.End()
		span.RecordError(err)

		Logger.ErrorContext(ctx, "User creation failed",
			slog.String("email", req.Email),
			slog.String("username", req.Username),
			slog.String("error", err.Error()),
			slog.Duration("db_duration", createDuration),
		)

		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(ErrorResponse{
			Success: false,
			Message: "User creation failed",
			Error:   err.Error(),
		})
		return
	}

	createSpan.SetStatus(codes.Ok, "User created successfully")
	createSpan.SetAttributes(
		attribute.Int("user_id", int(user.ID)),
	)
	createSpan.End()

	if authDuration != nil {
		authDuration.Record(ctx, time.Since(startTime).Seconds(),
			metric.WithAttributes(append(baseAttrs,
				attribute.String("status", "success"),
				attribute.Int("user_id", int(user.ID)),
			)...),
		)
	}

	span.SetStatus(codes.Ok, "Registration successful")
	span.SetAttributes(
		attribute.Int("user_id", int(user.ID)),
		attribute.String("username", user.Username),
		attribute.String("email", user.Email),
		attribute.String("status", "success"),
	)

	Logger.InfoContext(ctx, "Registration successful",
		slog.String("email", req.Email),
		slog.String("username", req.Username),
		slog.Int("user_id", int(user.ID)),
		slog.Duration("total_duration", time.Since(startTime)),
	)

	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(AuthResponse{
		Success: true,
		Message: "Registration successful",
		User:    user,
	})
}
