package response

import "github.com/gin-gonic/gin"

func Success(ctx *gin.Context, data any) {
	ctx.JSON(200, gin.H{"code": 0, "message": "ok", "data": data})
}

func BadRequest(ctx *gin.Context, message string) {
	ctx.JSON(400, gin.H{"code": 400, "message": message})
}

func NotFound(ctx *gin.Context, message string) {
	ctx.JSON(404, gin.H{"code": 404, "message": message})
}

func Conflict(ctx *gin.Context, message string) {
	ctx.JSON(409, gin.H{"code": 409, "message": message})
}

func InternalError(ctx *gin.Context, message string) {
	ctx.JSON(500, gin.H{"code": 500, "message": message})
}
