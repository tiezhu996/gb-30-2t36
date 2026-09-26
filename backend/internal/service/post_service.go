package service

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"gorm.io/gorm"

	"github.com/gbadopt/gbadopt/internal/constants"
	"github.com/gbadopt/gbadopt/internal/model"
	"github.com/gbadopt/gbadopt/internal/repository"
	"github.com/gbadopt/gbadopt/internal/util"
)

// homeOverviewCacheKey mirrors the home handler cache key; likes invalidate it.
const homeOverviewCacheKey = "gbadopt:home:overview"

// PostService handles community posts and likes.
type PostService struct {
	db       *gorm.DB
	repo     *repository.CommunityPostRepository
	likeRepo *repository.PostLikeRepository
	orgRepo  *repository.OrganizationRepository
	redis    *util.RedisClient
	logger   *slog.Logger
}

// NewPostService creates a PostService.
func NewPostService(db *gorm.DB, repo *repository.CommunityPostRepository, likeRepo *repository.PostLikeRepository, orgRepo *repository.OrganizationRepository, redis *util.RedisClient, logger *slog.Logger) *PostService {
	return &PostService{db: db, repo: repo, likeRepo: likeRepo, orgRepo: orgRepo, redis: redis, logger: logger}
}

// Create publishes a post (user or org).
func (s *PostService) Create(userID uint, p *model.CommunityPost) (*model.CommunityPost, error) {
	p.UserID = userID
	if p.Images == "" {
		p.Images = "[]"
	}
	if p.Status == "" {
		p.Status = "published"
	}
	if p.PostType == "" {
		p.PostType = "story"
	}
	if err := s.repo.Create(p); err != nil {
		s.logger.Error(fmt.Sprintf(constants.LogPostCreateFailed, p.Title), "error", err)
		return nil, fmt.Errorf("post create: %w", err)
	}
	s.logger.Info(fmt.Sprintf(constants.LogPostCreateSuccess, p.Title), "id", p.ID)
	return p, nil
}

// Get returns a post by id, marking whether viewerID has liked it.
func (s *PostService) Get(id, viewerID uint) (*model.CommunityPost, error) {
	p, err := s.repo.FindByID(id)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return nil, util.NewAppError(404, constants.CodeNotFound, fmt.Sprintf("CommunityPost[id=%d] not found", id))
		}
		return nil, fmt.Errorf("post get: %w", err)
	}
	if viewerID != 0 {
		liked, err := s.likeRepo.LikedIDSet(viewerID, []uint{p.ID})
		if err != nil {
			return nil, fmt.Errorf("post get liked: %w", err)
		}
		p.Liked = liked[p.ID]
	}
	return p, nil
}

// Like adds the viewer's like. It is idempotent: a repeated or concurrent
// call by the same user never adds a second like nor inflates the counter.
// The post row is locked inside the transaction so that simultaneous clicks
// are serialized against the same (user, post) state.
func (s *PostService) Like(userID, postID uint) (*model.CommunityPost, error) {
	return s.applyLike(userID, postID, true)
}

// Unlike removes the viewer's like. It is idempotent: removing a like that
// does not exist never drives the counter negative.
func (s *PostService) Unlike(userID, postID uint) (*model.CommunityPost, error) {
	return s.applyLike(userID, postID, false)
}

func (s *PostService) applyLike(userID, postID uint, wantLiked bool) (*model.CommunityPost, error) {
	var result *model.CommunityPost
	err := s.db.Transaction(func(tx *gorm.DB) error {
		if _, err := s.repo.GetForUpdateTx(tx, postID); err != nil {
			if errors.Is(err, repository.ErrNotFound) {
				return util.NewAppError(404, constants.CodeNotFound, fmt.Sprintf("CommunityPost[id=%d] not found", postID))
			}
			return fmt.Errorf("post like find: %w", err)
		}
		exists, err := s.likeRepo.ExistsTx(tx, userID, postID)
		if err != nil {
			return fmt.Errorf("post like check: %w", err)
		}
		switch {
		case wantLiked && !exists:
			if err := s.likeRepo.CreateTx(tx, userID, postID); err != nil {
				// Defense in depth: the row lock normally serializes toggles,
				// so a duplicate means a concurrent identical request won.
				if !errors.Is(err, repository.ErrDuplicate) {
					return fmt.Errorf("post like add: %w", err)
				}
			} else if err := s.repo.AdjustLikeCountTx(tx, postID, +1); err != nil {
				return fmt.Errorf("post like count up: %w", err)
			}
		case !wantLiked && exists:
			if err := s.likeRepo.DeleteTx(tx, userID, postID); err != nil {
				return fmt.Errorf("post like remove: %w", err)
			}
			if err := s.repo.AdjustLikeCountTx(tx, postID, -1); err != nil {
				return fmt.Errorf("post like count down: %w", err)
			}
		}
		final, err := s.repo.GetForUpdateTx(tx, postID)
		if err != nil {
			return fmt.Errorf("post like reload: %w", err)
		}
		result = final
		result.Liked = wantLiked
		return nil
	})
	if err != nil {
		return nil, err
	}
	s.logger.Info(fmt.Sprintf(constants.LogPostLikeToggled, postID, userID, wantLiked), "user_id", userID)
	if s.redis != nil {
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()
		_ = s.redis.Delete(ctx, homeOverviewCacheKey)
	}
	return result, nil
}

// List filters posts, marking the ones liked by the viewer.
func (s *PostService) List(postType, keyword string, page, pageSize int, viewerID uint) ([]model.CommunityPost, int64, error) {
	items, total, err := s.repo.List(postType, keyword, page, pageSize)
	if err != nil {
		return nil, 0, fmt.Errorf("post list: %w", err)
	}
	if err := s.attachLiked(items, viewerID); err != nil {
		return nil, 0, err
	}
	return items, total, nil
}

// ListLatest returns recent posts for the home page.
// The home overview cache is shared across viewers, so liked state is not
// attached here; callers merge LikedPostIDs on the client.
func (s *PostService) ListLatest(limit int) ([]model.CommunityPost, error) {
	items, err := s.repo.ListLatest(limit)
	if err != nil {
		return nil, fmt.Errorf("post latest: %w", err)
	}
	return items, nil
}

// LikedPostIDs returns the ids of the posts the user has liked.
func (s *PostService) LikedPostIDs(userID uint) ([]uint, error) {
	ids, err := s.likeRepo.LikedPostIDs(userID)
	if err != nil {
		return nil, fmt.Errorf("post liked ids: %w", err)
	}
	return ids, nil
}

// attachLiked fills the request-scoped Liked flag for a list of posts.
func (s *PostService) attachLiked(items []model.CommunityPost, viewerID uint) error {
	if viewerID == 0 || len(items) == 0 {
		return nil
	}
	ids := make([]uint, 0, len(items))
	for i := range items {
		ids = append(ids, items[i].ID)
	}
	likedSet, err := s.likeRepo.LikedIDSet(viewerID, ids)
	if err != nil {
		return fmt.Errorf("post list liked: %w", err)
	}
	for i := range items {
		items[i].Liked = likedSet[items[i].ID]
	}
	return nil
}
