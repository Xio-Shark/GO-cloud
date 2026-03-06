package middleware

import (
	"github.com/gin-gonic/gin"

	"go-cloud/pkg/traceutil"
)

func TraceID() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		traceID := ctx.GetHeader("X-Trace-Id")
		if traceID == "" {
			traceID = traceutil.NewTraceID()
		}
		requestCtx := traceutil.WithTraceID(ctx.Request.Context(), traceID)
		ctx.Request = ctx.Request.WithContext(requestCtx)
		ctx.Header("X-Trace-Id", traceID)
		ctx.Next()
	}
}
