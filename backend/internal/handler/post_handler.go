package handler

import (
	"log/slog"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"

	"github.com/gbadopt/gbadopt/internal/constants"
	"github.com/gbadopt/gbadopt/internal/dto"
	"github.com/gbadopt/gbadopt/internal/middleware"
	"github.com/gbadopt/gbadopt/internal/model"
	"github.com/gbadopt/gbadopt/internal/service"
	"github.com/gbadopt/gbadopt/internal/util"
)

// PostHandler exposes community post endpoints.
type PostHandler struct {
	svc    *service.PostService
	logger *slog.Logger
}

// NewPostHandler creates a PostHandler.
func NewPostHandler(svc *service.PostService, logger *slog.Logger) *PostHandler {
	return &PostHandler{svc: svc, logger: logger}
}

// List handles GET /posts.
func (h *PostHandler) List(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "10"))
	postType := c.Query("post_type")
	keyword := c.Query("keyword")
	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 100 {
		pageSize = 10
	}
	items, total, err := h.svc.List(postType, keyword, page, pageSize, middleware.GetUserID(c))
	if err != nil {
		c.Error(err)
		return
	}
	c.JSON(http.StatusOK, dto.OK(dto.PageData{List: items, Total: total, Page: page, Size: pageSize}))
}

// Get handles GET /posts/:id.
func (h *PostHandler) Get(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		c.Error(util.NewAppError(http.StatusBadRequest, constants.CodeBadRequest, "invalid post id"))
		return
	}
	p, err := h.svc.Get(uint(id), middleware.GetUserID(c))
	if err != nil {
		c.Error(err)
		return
	}
	c.JSON(http.StatusOK, dto.OK(p))
}

// Create handles POST /posts.
func (h *PostHandler) Create(c *gin.Context) {
	var req dto.PostCreateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.Error(util.NewAppError(http.StatusBadRequest, constants.CodeBadRequest, constants.MsgInvalidParam+": "+err.Error()))
		return
	}
	p := &model.CommunityPost{Title: req.Title, Content: req.Content, Images: req.Images, PostType: req.PostType}
	created, err := h.svc.Create(middleware.GetUserID(c), p)
	if err != nil {
		c.Error(err)
		return
	}
	c.JSON(http.StatusCreated, dto.OK(created))
}

// Like handles PUT /posts/:id/like: idempotently records the caller's like.
func (h *PostHandler) Like(c *gin.Context) {
	h.apply(c, true)
}

// Unlike handles PUT /posts/:id/unlike: idempotently removes the caller's like.
func (h *PostHandler) Unlike(c *gin.Context) {
	h.apply(c, false)
}

func (h *PostHandler) apply(c *gin.Context, liked bool) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		c.Error(util.NewAppError(http.StatusBadRequest, constants.CodeBadRequest, "invalid post id"))
		return
	}
	var p *model.CommunityPost
	if liked {
		p, err = h.svc.Like(middleware.GetUserID(c), uint(id))
	} else {
		p, err = h.svc.Unlike(middleware.GetUserID(c), uint(id))
	}
	if err != nil {
		c.Error(err)
		return
	}
	c.JSON(http.StatusOK, dto.OK(gin.H{"id": p.ID, "like_count": p.LikeCount, "liked": liked}))
}

// MyLikedIDs handles GET /users/me/liked-posts.
func (h *PostHandler) MyLikedIDs(c *gin.Context) {
	ids, err := h.svc.LikedPostIDs(middleware.GetUserID(c))
	if err != nil {
		c.Error(err)
		return
	}
	c.JSON(http.StatusOK, dto.OK(gin.H{"post_ids": ids}))
}
