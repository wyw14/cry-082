package middleware

import (
	"context"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
)

const ActorContextKey = "authenticated_actor_id"

type Claims struct{ Subject string }
type AccessVerifier interface {
	VerifyAccessToken(context.Context, string) (Claims, error)
}

type VerifyFunc func(context.Context, string) (Claims, error)

func (f VerifyFunc) VerifyAccessToken(ctx context.Context, raw string) (Claims, error) {
	return f(ctx, raw)
}

func RequireIdentity(verifier AccessVerifier, allowDemo bool) gin.HandlerFunc {
	return func(c *gin.Context) {
		header := strings.TrimSpace(c.GetHeader("Authorization"))
		if strings.HasPrefix(header, "Bearer ") && verifier != nil {
			claims, err := verifier.VerifyAccessToken(c.Request.Context(), strings.TrimSpace(strings.TrimPrefix(header, "Bearer ")))
			if err == nil && claims.Subject != "" {
				c.Set(ActorContextKey, claims.Subject)
				c.Next()
				return
			}
		}
		if allowDemo {
			actor := strings.TrimSpace(c.GetHeader("X-Demo-Actor"))
			if actor == "" {
				actor = "demo-supervisor"
			}
			c.Set(ActorContextKey, actor)
			c.Next()
			return
		}
		c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"code": "AUTHENTICATION_REQUIRED", "message": "需要有效的访问令牌", "field_errors": []any{}, "request_id": GetRequestID(c)})
	}
}
