package server

import (
	"fmt"
	"net/http"
	"os"
	"strings"

	"github.com/golang-jwt/jwt/v5"
)

// adminClaims mirrors backend_server's UserClaims shape
// (KNIRV_CORP/packages/server/backend_server/internal/web/middleware/auth.go)
// closely enough to decode the same tokens it issues — just the Role field,
// since that's all requireAdminJWT needs.
type adminClaims struct {
	Role string `json:"role"`
	jwt.RegisteredClaims
}

// adminJWTSecret resolves the shared JWT secret backend_server signs tokens
// with. KNIRVGATEWAY has no YAML config loader (see internal/config —
// env-vars-only via godotenv), so unlike KNIRVSERVER's equivalent
// jwtSigningSecret() (packages/KNIRVSERVER/main.go), this can only read the
// env var — there is no security.jwt_secret/auth.jwt_secret YAML fallback
// here. KNIRV_JWT_SECRET MUST be exported into KNIRVGATEWAY's environment
// (it inherits from whatever process spawns it, see
// pkg/knirvgateway/manager.go's env := os.Environ()) and MUST match
// whatever backend_server actually signs with, or every admin request will
// be rejected even for real admins.
func adminJWTSecret() string {
	return strings.TrimSpace(os.Getenv("KNIRV_JWT_SECRET"))
}

// requireAdminJWT validates the request's bearer token against the shared
// backend JWT secret and requires its role claim to be "admin". This is a
// minimal, KNIRVGATEWAY-local equivalent of backend_server's
// middleware.RequireRole("admin")
// (KNIRV_CORP/packages/server/backend_server/internal/web/middleware/rbac.go)
// — duplicated rather than imported because backend_server is a separate Go
// module in a separate repo and this codebase's convention is no
// cross-package Go imports between services (see CLAUDE.md). Unlike
// backend_server's RequireRole, which treats "admin" as an implicit member
// of every allowed-role list, this has no broader role list to bypass — it
// only ever accepts "admin".
func requireAdminJWT(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		secret := adminJWTSecret()
		if secret == "" {
			http.Error(w, `{"error":"admin auth not configured"}`, http.StatusServiceUnavailable)
			return
		}

		authHeader := strings.TrimSpace(r.Header.Get("Authorization"))
		parts := strings.SplitN(authHeader, " ", 2)
		if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") || strings.TrimSpace(parts[1]) == "" {
			http.Error(w, `{"error":"missing bearer token"}`, http.StatusUnauthorized)
			return
		}
		tokenString := strings.TrimSpace(parts[1])

		claims := &adminClaims{}
		token, err := jwt.ParseWithClaims(tokenString, claims, func(t *jwt.Token) (interface{}, error) {
			if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
				return nil, fmt.Errorf("unexpected signing method: %v", t.Header["alg"])
			}
			return []byte(secret), nil
		})
		if err != nil || !token.Valid {
			http.Error(w, `{"error":"invalid or expired token"}`, http.StatusUnauthorized)
			return
		}

		if !strings.EqualFold(claims.Role, "admin") {
			http.Error(w, `{"error":"admin role required"}`, http.StatusForbidden)
			return
		}

		next.ServeHTTP(w, r)
	}
}
