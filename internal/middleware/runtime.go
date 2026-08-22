package middleware

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"net/http"
	"regexp"
	"runtime/debug"
	"strings"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"

	"github.com/wyw14/cry-082/internal/platform/metrics"
)

const RequestIDKey = "request_id"

var validRequestID = regexp.MustCompile(`^[A-Za-z0-9._:-]{8,96}$`)

type clientBudget struct {
	remaining float64
	updatedAt time.Time
}

type RequestRuntime struct {
	logger         *zap.Logger
	metrics        *metrics.Registry
	allowedOrigins map[string]struct{}
	ratePerSecond  float64
	burst          float64
	now            func() time.Time
	mu             sync.Mutex
	clients        map[string]clientBudget
}

func NewRequestRuntime(logger *zap.Logger, registry *metrics.Registry, allowedOrigins []string, ratePerSecond float64, burst int) *RequestRuntime {
	origins := make(map[string]struct{}, len(allowedOrigins))
	for _, origin := range allowedOrigins {
		trimmed := strings.TrimSpace(origin)
		if trimmed != "" {
			origins[trimmed] = struct{}{}
		}
	}
	return &RequestRuntime{logger: logger, metrics: registry, allowedOrigins: origins, ratePerSecond: ratePerSecond, burst: float64(burst), now: time.Now, clients: make(map[string]clientBudget)}
}

func (r *RequestRuntime) Handle() gin.HandlerFunc {
	return func(c *gin.Context) {
		started := r.now()
		finishMetrics := func(string, string, int, time.Duration) {}
		if r.metrics != nil {
			finishMetrics = r.metrics.Begin()
		}
		r.assignRequestID(c)
		r.setSecurityHeaders(c)
		defer func() {
			if recovered := recover(); recovered != nil {
				r.logger.Error("panic recovered", zap.String("request_id", GetRequestID(c)), zap.String("panic", fmt.Sprint(recovered)), zap.ByteString("stack", debug.Stack()))
				c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"code": "INTERNAL_ERROR", "message": "服务暂时不可用", "field_errors": []any{}, "request_id": GetRequestID(c)})
			}
			path := c.FullPath()
			if path == "" {
				path = "unmatched"
			}
			elapsed := r.now().Sub(started)
			finishMetrics(c.Request.Method, path, c.Writer.Status(), elapsed)
			r.logger.Info("http request completed", zap.String("request_id", GetRequestID(c)), zap.String("method", c.Request.Method), zap.String("route", path), zap.Int("status", c.Writer.Status()), zap.Int("response_bytes", c.Writer.Size()), zap.Duration("elapsed", elapsed), zap.String("remote_ip", c.ClientIP()))
		}()
		if r.applyCORS(c) {
			return
		}
		if !r.take(c.ClientIP()) {
			c.AbortWithStatusJSON(http.StatusTooManyRequests, gin.H{"code": "RATE_LIMITED", "message": "请求过于频繁", "field_errors": []any{}, "request_id": GetRequestID(c)})
			return
		}
		c.Next()
	}
}

func (r *RequestRuntime) assignRequestID(c *gin.Context) {
	requestID := strings.TrimSpace(c.GetHeader("X-Request-ID"))
	if !validRequestID.MatchString(requestID) {
		buffer := make([]byte, 16)
		if _, err := rand.Read(buffer); err == nil {
			requestID = hex.EncodeToString(buffer)
		} else {
			requestID = fmt.Sprintf("local-%d", r.now().UnixNano())
		}
	}
	c.Set(RequestIDKey, requestID)
	c.Header("X-Request-ID", requestID)
}

func (r *RequestRuntime) setSecurityHeaders(c *gin.Context) {
	c.Header("X-Content-Type-Options", "nosniff")
	c.Header("X-Frame-Options", "DENY")
	c.Header("Referrer-Policy", "no-referrer")
	c.Header("Content-Security-Policy", "default-src 'self'")
	c.Header("Cache-Control", "no-store")
}

func (r *RequestRuntime) applyCORS(c *gin.Context) bool {
	origin := strings.TrimSpace(c.GetHeader("Origin"))
	if _, allowed := r.allowedOrigins[origin]; origin != "" && allowed {
		c.Header("Access-Control-Allow-Origin", origin)
		c.Header("Vary", "Origin")
		c.Header("Access-Control-Allow-Headers", "Authorization, Content-Type, X-Request-ID, Idempotency-Key")
		c.Header("Access-Control-Allow-Methods", "GET, POST, PATCH, DELETE, OPTIONS")
	}
	if c.Request.Method == http.MethodOptions {
		c.AbortWithStatus(http.StatusNoContent)
		return true
	}
	return false
}

func (r *RequestRuntime) take(client string) bool {
	r.mu.Lock()
	defer r.mu.Unlock()
	now := r.now()
	budget, exists := r.clients[client]
	if !exists {
		budget = clientBudget{remaining: r.burst, updatedAt: now}
	}
	budget.remaining += now.Sub(budget.updatedAt).Seconds() * r.ratePerSecond
	if budget.remaining > r.burst {
		budget.remaining = r.burst
	}
	budget.updatedAt = now
	if budget.remaining < 1 {
		r.clients[client] = budget
		return false
	}
	budget.remaining--
	r.clients[client] = budget
	return true
}

func GetRequestID(c *gin.Context) string {
	value, _ := c.Get(RequestIDKey)
	requestID, _ := value.(string)
	return requestID
}
