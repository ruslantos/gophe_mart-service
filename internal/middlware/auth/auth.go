package auth

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/ruslantos/gophemart-service/internal/service"
)

var (
	secretKey = []byte("secret-key")
)

type contextKey string

const (
	UserIDKey contextKey = "userID"
)

func AuthMiddleware(userService *service.Service) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// при регистрации и авторизации не проверяем
			if r.URL.Path == "/api/user/login" || r.URL.Path == "/api/user/register" {
				next.ServeHTTP(w, r)
				return
			}

			cookie, err := r.Cookie("user")
			if err != nil {
				http.Error(w, "Authorization required", http.StatusUnauthorized)
				return
			}

			cookieUserID, ok := verifyCookie(cookie)
			if !ok {
				http.Error(w, "Invalid cookie", http.StatusUnauthorized)
				return
			}

			ctx := context.WithValue(r.Context(), UserIDKey, cookieUserID)
			r = r.WithContext(ctx)

			next.ServeHTTP(w, r)
		})
	}
}

func CreateSignedCookie(userID string) http.Cookie {
	cookieValue := createToken(userID)

	cookie := http.Cookie{
		Name:     "user",
		Value:    cookieValue,
		Path:     "/",
		Expires:  time.Now().Add(24 * time.Hour),
		HttpOnly: true,
	}

	return cookie
}
func GetUserIDFromContext(ctx context.Context) string {
	userID, ok := ctx.Value(UserIDKey).(string)
	if !ok {
		return ""
	}
	return userID
}
func createToken(userID string) string {
	h := hmac.New(sha256.New, secretKey)
	h.Write([]byte(userID))
	signature := base64.URLEncoding.EncodeToString(h.Sum(nil))

	return fmt.Sprintf("%s|%s", userID, signature)
}
func verifyCookie(cookie *http.Cookie) (string, bool) {
	if cookie == nil {
		return "", false
	}

	parts := strings.SplitN(cookie.Value, "|", 2)
	if len(parts) != 2 {
		return "", false
	}

	userID := parts[0]
	signature := parts[1]

	h := hmac.New(sha256.New, secretKey)
	h.Write([]byte(userID))
	expectedSignature := base64.URLEncoding.EncodeToString(h.Sum(nil))

	if !hmac.Equal([]byte(signature), []byte(expectedSignature)) {
		return "", false
	}

	return userID, true
}
