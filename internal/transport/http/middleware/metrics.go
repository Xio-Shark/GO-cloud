package middleware

import (
	"strconv"
	"time"

	"github.com/gin-gonic/gin"

	"go-cloud/internal/metrics"
)

func Metrics() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		start := time.Now()
		ctx.Next()
		path := ctx.FullPath()
		if path == "" {
			path = ctx.Request.URL.Path
		}
		status := strconv.Itoa(ctx.Writer.Status())
		metrics.HTTPRequestsTotal.WithLabelValues(ctx.Request.Method, path, status).Inc()
		metrics.HTTPRequestDurationSeconds.WithLabelValues(ctx.Request.Method, path).Observe(time.Since(start).Seconds())
	}
}
