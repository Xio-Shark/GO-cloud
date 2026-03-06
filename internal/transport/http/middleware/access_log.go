package middleware

import (
	"log/slog"
	"time"

	"github.com/gin-gonic/gin"

	"go-cloud/pkg/traceutil"
)

func AccessLog() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		start := time.Now()
		ctx.Next()
		slog.Default().InfoContext(
			ctx.Request.Context(),
			"http request",
			"trace_id", traceutil.FromContext(ctx.Request.Context()),
			"method", ctx.Request.Method,
			"path", ctx.Request.URL.Path,
			"status", ctx.Writer.Status(),
			"duration_ms", time.Since(start).Milliseconds(),
			"client_ip", ctx.ClientIP(),
		)
	}
}
