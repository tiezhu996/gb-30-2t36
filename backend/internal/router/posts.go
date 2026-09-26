package router

import (
	"github.com/gin-gonic/gin"

	"github.com/gbadopt/gbadopt/internal/config"
	"github.com/gbadopt/gbadopt/internal/handler"
	"github.com/gbadopt/gbadopt/internal/middleware"
)

func registerPostRoutes(v1 *gin.RouterGroup, cfg *config.Config, ph *handler.PostHandler, ch *handler.CommentHandler, limiter *middleware.RateLimiter) {
	posts := v1.Group("/posts")
	// Public reads; a valid token personalizes the liked flag.
	posts.GET("", middleware.AuthOptional(cfg), ph.List)
	posts.GET("/:id", middleware.AuthOptional(cfg), ph.Get)
	posts.GET("/:id/comments", ch.List)
	auth := posts.Group("", middleware.AuthRequired(cfg))
	auth.POST("", limiter.Limit(), ph.Create)
	auth.POST("/:id/comments", limiter.Limit(), ch.Create)
	auth.PUT("/:id/like", ph.Like)
	auth.PUT("/:id/unlike", ph.Unlike)
	v1.DELETE("/comments/:id", middleware.AuthRequired(cfg), ch.Delete)
}

// registerLikedRoutes exposes the signed-in user's liked post ids. It lives
// outside /posts because gin cannot mix a static segment (/posts/liked) with
// the /posts/:id parameter on the same level.
func registerLikedRoutes(v1 *gin.RouterGroup, cfg *config.Config, ph *handler.PostHandler) {
	v1.GET("/users/me/liked-posts", middleware.AuthRequired(cfg), ph.MyLikedIDs)
}
