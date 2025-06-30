package main

import (
	"encoding/json"
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

func isValidEmail(email string) bool               { return true }
func isValidUsername(username string) bool         { return true }
func isValidPassword(password string) bool         { return true }
func generateJWT(user *User) (string, error)       { return "dummy-jwt", nil }
func hashPassword(password string) (string, error) { return "hashed-password", nil }
func checkPassword(password, hash string) bool     { return true }

func loginUser(w http.ResponseWriter, r *http.Request) {
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
	loginAttempts.Add(ctx, 1, metric.WithAttributes(attribute.String("endpoint", "login")))

	var loginReq LoginRequest
	if err := json.NewDecoder(r.Body).Decode(&loginReq); err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "Invalid JSON")
		http.Error(w, "Invalid JSON payload", http.StatusBadRequest)
		return
	}

	if loginReq.Email == "" || loginReq.Password == "" {
		span.SetStatus(codes.Error, "Missing fields")
		http.Error(w, "Email and password required", http.StatusBadRequest)
		return
	}

	if !isValidEmail(loginReq.Email) {
		span.SetStatus(codes.Error, "Invalid email format")
		http.Error(w, "Invalid email format", http.StatusBadRequest)
		return
	}

	var user User
	dbCtx, dbSpan := tracer.Start(ctx, "db.lookup_user")
	err := DB.WithContext(dbCtx).Where("email = ?", loginReq.Email).First(&user).Error
	dbSpan.End()

	if err != nil {
		span.RecordError(err)
		if err == gorm.ErrRecordNotFound {
			http.Error(w, "Invalid credentials", http.StatusUnauthorized)
		} else {
			http.Error(w, "Database error", http.StatusInternalServerError)
		}
		return
	}

	if !checkPassword(loginReq.Password, user.Password) {
		span.SetStatus(codes.Error, "Invalid password")
		http.Error(w, "Invalid credentials", http.StatusUnauthorized)
		return
	}

	token, err := generateJWT(&user)
	if err != nil {
		span.RecordError(err)
		http.Error(w, "Token generation failed", http.StatusInternalServerError)
		return
	}

	authDuration.Record(ctx, time.Since(startTime).Seconds(),
		metric.WithAttributes(attribute.String("endpoint", "login")),
	)
	span.SetStatus(codes.Ok, "Login successful")

	json.NewEncoder(w).Encode(AuthResponse{
		Success: true,
		Message: "Login successful",
		Token:   token,
		User:    &user,
	})
}

func registerUser(w http.ResponseWriter, r *http.Request) {
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
	registerAttempts.Add(ctx, 1, metric.WithAttributes(attribute.String("endpoint", "register")))

	var req RegisterRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		span.RecordError(err)
		http.Error(w, "Invalid JSON payload", http.StatusBadRequest)
		return
	}

	if req.Username == "" || req.Email == "" || req.Password == "" {
		http.Error(w, "All fields are required", http.StatusBadRequest)
		return
	}
	if !isValidEmail(req.Email) || !isValidUsername(req.Username) || !isValidPassword(req.Password) {
		http.Error(w, "Invalid input format", http.StatusBadRequest)
		return
	}

	var existing User

	if err := DB.Where("email = ?", req.Email).First(&existing).Error; err == nil {
		http.Error(w, "Email already in use", http.StatusConflict)
		return
	} else if err != gorm.ErrRecordNotFound {
		span.RecordError(err)
		http.Error(w, "Database error", http.StatusInternalServerError)
		return
	}

	if err := DB.Where("username = ?", req.Username).First(&existing).Error; err == nil {
		http.Error(w, "Username already taken", http.StatusConflict)
		return
	} else if err != gorm.ErrRecordNotFound {
		span.RecordError(err)
		http.Error(w, "Database error", http.StatusInternalServerError)
		return
	}

	hashedPwd, err := hashPassword(req.Password)
	if err != nil {
		span.RecordError(err)
		http.Error(w, "Password hashing failed", http.StatusInternalServerError)
		return
	}

	user := &User{
		Username: req.Username,
		Email:    req.Email,
		Password: hashedPwd,
	}

	if err := DB.Create(user).Error; err != nil {
		span.RecordError(err)
		http.Error(w, "User creation failed", http.StatusInternalServerError)
		return
	}

	authDuration.Record(ctx, time.Since(startTime).Seconds(),
		metric.WithAttributes(attribute.String("endpoint", "register")),
	)
	span.SetStatus(codes.Ok, "Registration successful")

	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(AuthResponse{
		Success: true,
		Message: "Registration successful",
		User:    user,
	})
}
