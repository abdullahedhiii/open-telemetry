package main

import (
	"encoding/json"
	"fmt"
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
func verifyPassword(hashedPassword, password string) bool { return true } // Add password verification

func loginUser(w http.ResponseWriter, r *http.Request) {
	// Check DB initialization first and return early if nil
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
		),
	)
	defer span.End()

	w.Header().Set("Content-Type", "application/json")

	// Add nil check for metrics
	if loginAttempts != nil {
		loginAttempts.Add(ctx, 1, metric.WithAttributes(attribute.String("endpoint", "login")))
	}

	var loginReq LoginRequest
	if err := json.NewDecoder(r.Body).Decode(&loginReq); err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "Invalid JSON")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(ErrorResponse{
			Success: false,
			Message: "Invalid JSON payload",
			Error:   err.Error(),
		})
		return
	}

	// Validate input
	if loginReq.Email == "" || loginReq.Password == "" {
		span.SetStatus(codes.Error, "Missing required fields")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(ErrorResponse{
			Success: false,
			Message: "Email and password are required",
		})
		return
	}

	var user User
	dbCtx, dbSpan := tracer.Start(ctx, "db.lookup_user")
	err := DB.WithContext(dbCtx).Where("email = ?", loginReq.Email).First(&user).Error
	dbSpan.End()

	if err != nil {
		span.RecordError(err)
		if err == gorm.ErrRecordNotFound {
			w.WriteHeader(http.StatusUnauthorized)
			json.NewEncoder(w).Encode(ErrorResponse{
				Success: false,
				Message: "Invalid credentials",
			})
		} else {
			w.WriteHeader(http.StatusInternalServerError)
			json.NewEncoder(w).Encode(ErrorResponse{
				Success: false,
				Message: "Database error",
				Error:   err.Error(),
			})
		}
		return
	}

	// Add password verification (you'll need to implement this properly)
	if !verifyPassword(user.Password, loginReq.Password) {
		span.SetStatus(codes.Error, "Invalid password")
		w.WriteHeader(http.StatusUnauthorized)
		json.NewEncoder(w).Encode(ErrorResponse{
			Success: false,
			Message: "Invalid credentials",
		})
		return
	}

	token, err := generateJWT(&user)
	if err != nil {
		span.RecordError(err)
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(ErrorResponse{
			Success: false,
			Message: "Token generation failed",
			Error:   err.Error(),
		})
		return
	}

	// Record metrics with nil check
	if authDuration != nil {
		authDuration.Record(ctx, time.Since(startTime).Seconds(),
			metric.WithAttributes(attribute.String("endpoint", "login")),
		)
	}
	span.SetStatus(codes.Ok, "Login successful")

	json.NewEncoder(w).Encode(AuthResponse{
		Success: true,
		Message: "Login successful",
		Token:   token,
		User:    &user,
	})
}

func registerUser(w http.ResponseWriter, r *http.Request) {
	// Check DB initialization first
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
		),
	)
	defer span.End()

	w.Header().Set("Content-Type", "application/json")

	// Add nil check for metrics
	if registerAttempts != nil {
		registerAttempts.Add(ctx, 1, metric.WithAttributes(attribute.String("endpoint", "register")))
	}

	var req RegisterRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		span.RecordError(err)
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(ErrorResponse{
			Success: false,
			Message: "Invalid JSON payload",
			Error:   err.Error(),
		})
		return
	}

	// Validate input
	if req.Email == "" || req.Password == "" || req.Username == "" {
		span.SetStatus(codes.Error, "Missing required fields")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(ErrorResponse{
			Success: false,
			Message: "Username, email, and password are required",
		})
		return
	}

	var existing User

	// Check email uniqueness
	if err := DB.Where("email = ?", req.Email).First(&existing).Error; err == nil {
		w.WriteHeader(http.StatusConflict)
		json.NewEncoder(w).Encode(ErrorResponse{
			Success: false,
			Message: "Email already in use",
		})
		return
	} else if err != gorm.ErrRecordNotFound {
		span.RecordError(err)
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(ErrorResponse{
			Success: false,
			Message: "Database error",
			Error:   err.Error(),
		})
		return
	}

	// Check username uniqueness
	if err := DB.Where("username = ?", req.Username).First(&existing).Error; err == nil {
		w.WriteHeader(http.StatusConflict)
		json.NewEncoder(w).Encode(ErrorResponse{
			Success: false,
			Message: "Username already taken",
		})
		return
	} else if err != gorm.ErrRecordNotFound {
		span.RecordError(err)
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(ErrorResponse{
			Success: false,
			Message: "Database error",
			Error:   err.Error(),
		})
		return
	}

	hashedPwd, err := hashPassword(req.Password)
	if err != nil {
		span.RecordError(err)
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(ErrorResponse{
			Success: false,
			Message: "Password hashing failed",
			Error:   err.Error(),
		})
		return
	}

	user := &User{
		Username: req.Username,
		Email:    req.Email,
		Password: hashedPwd,
	}

	if err := DB.Create(user).Error; err != nil {
		span.RecordError(err)
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(ErrorResponse{
			Success: false,
			Message: "User creation failed",
			Error:   err.Error(),
		})
		return
	}

	// Record metrics with nil check
	if authDuration != nil {
		authDuration.Record(ctx, time.Since(startTime).Seconds(),
			metric.WithAttributes(attribute.String("endpoint", "register")),
		)
	}
	span.SetStatus(codes.Ok, "Registration successful")

	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(AuthResponse{
		Success: true,
		Message: "Registration successful",
		User:    user,
	})
}
