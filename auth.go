package main

import (
	"database/sql"
	"encoding/json"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/pquerna/otp/totp"
	"golang.org/x/crypto/bcrypt"
)

type Claims struct {
	UserID string `json:"user_id"`
	Email  string `json:"email"`
	jwt.RegisteredClaims
}

type LoginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

type LoginResponse struct {
	Token       string `json:"token,omitempty"`
	User        *User  `json:"user,omitempty"`
	MFARequired bool   `json:"mfa_required"`
	UserID      string `json:"user_id,omitempty"`
}

func getJWTSecret() []byte {
	secret := os.Getenv("JWT_SECRET")
	if secret == "" {
		return []byte("default-secret-do-not-use-in-production")
	}
	return []byte(secret)
}

func generateToken(user *User) (string, error) {
	expirationTime := time.Now().Add(24 * time.Hour)
	claims := &Claims{
		UserID: user.ID,
		Email:  user.Email,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(expirationTime),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(getJWTSecret())
}

func handleLogin(w http.ResponseWriter, r *http.Request, db *sql.DB) {
	var req LoginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	user, err := GetUserByEmail(db, req.Email)
	if err != nil {
		if err == sql.ErrNoRows {
			http.Error(w, "Invalid credentials", http.StatusUnauthorized)
		} else {
			http.Error(w, "Database error", http.StatusInternalServerError)
		}
		return
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(req.Password)); err != nil {
		http.Error(w, "Invalid credentials", http.StatusUnauthorized)
		return
	}

	if user.MFAEnabled {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(LoginResponse{
			MFARequired: true,
			UserID:      user.ID,
		})
		return
	}

	token, err := generateToken(user)
	if err != nil {
		http.Error(w, "Failed to generate token", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(LoginResponse{
		Token: token,
		User:  user,
	})
}

func handleMe(w http.ResponseWriter, r *http.Request, db *sql.DB) {
	// The user ID is added to the context by the AuthMiddleware
	// But since I'm implementing AuthMiddleware inside main.go or here, let's keep it simple for now
	// This function assumes it's behind AuthMiddleware

	// For now, let's just return success if the token is valid, parsing it again to be safe
	authHeader := r.Header.Get("Authorization")
	if authHeader == "" {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	tokenString := strings.Replace(authHeader, "Bearer ", "", 1)
	claims := &Claims{}

	token, err := jwt.ParseWithClaims(tokenString, claims, func(token *jwt.Token) (interface{}, error) {
		return getJWTSecret(), nil
	})

	if err != nil || !token.Valid {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	user, err := GetUserByID(db, claims.UserID)
	if err != nil {
		http.Error(w, "User not found", http.StatusUnauthorized)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(user)
}

func AuthMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Allow OPTIONS requests
		if r.Method == "OPTIONS" {
			next.ServeHTTP(w, r)
			return
		}

		// Allow Access to specific paths without auth (API key auth handled by handlers)
		path := r.URL.Path
		if strings.HasPrefix(path, "/api/auth/login") ||
			strings.HasPrefix(path, "/api/auth/mfa/verify") ||
			strings.Contains(path, "/store/") || // Allow error ingestion endpoints
			strings.HasSuffix(path, "/store") ||
			strings.Contains(path, "/envelope") || // Allow envelope endpoints
			strings.Contains(path, "/coverage") || // Allow coverage endpoints (API key auth)
			strings.HasPrefix(path, "/status/") || // Allow public status pages (frontend route)
			strings.HasPrefix(path, "/api/status/") || // Allow public status page API endpoint
			(strings.HasPrefix(path, "/api/") && strings.HasSuffix(path, "/")) { // Allow project discovery endpoint
			next.ServeHTTP(w, r)
			return
		}

		authHeader := r.Header.Get("Authorization")
		if authHeader == "" {
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}

		tokenString := strings.Replace(authHeader, "Bearer ", "", 1)
		claims := &Claims{}

		token, err := jwt.ParseWithClaims(tokenString, claims, func(token *jwt.Token) (interface{}, error) {
			return getJWTSecret(), nil
		})

		if err != nil || !token.Valid {
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}

		next.ServeHTTP(w, r)
	})
}

func handleMFAVerify(w http.ResponseWriter, r *http.Request, db *sql.DB) {
	var req struct {
		UserID string `json:"user_id"`
		Code   string `json:"code"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	user, err := GetUserByID(db, req.UserID)
	if err != nil {
		http.Error(w, "User not found", http.StatusUnauthorized)
		return
	}

	if !user.MFAEnabled {
		http.Error(w, "MFA not enabled for this user", http.StatusBadRequest)
		return
	}

	valid := totp.Validate(req.Code, user.MFASecret)
	if !valid {
		http.Error(w, "Invalid MFA code", http.StatusUnauthorized)
		return
	}

	token, err := generateToken(user)
	if err != nil {
		http.Error(w, "Failed to generate token", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(LoginResponse{
		Token: token,
		User:  user,
	})
}
