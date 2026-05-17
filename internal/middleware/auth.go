package middleware

import (
	"net/http"

	"github.com/qutaq/short_url/internal/auth"
)

const cookieName = "auth_token"

// AuthMiddleware гарантирует наличие подписанной cookie с идентификатором
// пользователя и сохраняет этот идентификатор в контексте запроса.
func AuthMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var userID string

		cookie, err := r.Cookie(cookieName)
		if err == nil {
			userID, err = auth.VerifyToken(cookie.Value)
		}

		if err != nil || userID == "" {
			userID, err = auth.GenerateUserID()
			if err != nil {
				http.Error(w, "Internal Server Error", http.StatusInternalServerError)
				return
			}
			http.SetCookie(w, &http.Cookie{
				Name:     cookieName,
				Value:    auth.SignUserID(userID),
				Path:     "/",
				HttpOnly: true,
			})
		}

		ctx := auth.ContextWithUserID(r.Context(), userID)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}
