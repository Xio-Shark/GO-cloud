package middleware

import (
	"log/slog"
	"net/http"

	"github.com/gin-gonic/gin"

	"go-cloud/pkg/traceutil"
)

func Recovery() gin.HandlerFunc {
	return gin.CustomRecovery(func(ctx *gin.Context, recovered any) {
		slog.Default().ErrorContext(
			ctx.Request.Context(),
			"http panic recovered",
			"trace_id", traceutil.FromContext(ctx.Request.Context()),
			"panic", recovered,
		)
		ctx.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{
			"code":    http.StatusInternalServerError,
			"message": "internal server error",
		})
	})
}
