package service

import (
	"errors"
	"fmt"
	"log/slog"

	"gorm.io/gorm"

	"github.com/gbadopt/gbadopt/internal/constants"
	"github.com/gbadopt/gbadopt/internal/model"
	"github.com/gbadopt/gbadopt/internal/repository"
	"github.com/gbadopt/gbadopt/internal/util"
)

// PostService handles community posts and likes.
type PostService struct {
	db      *gorm.DB
	repo    *repository.CommunityPostRepository
	likeRepo *repository.PostLikeRepository
	orgRepo *repository.OrganizationRepository
	logger  *slog.Logger
}

// NewPostService creates a PostService.
func NewPostService(db *gorm.DB, repo *repository.CommunityPostRepository, likeRepo *repository.PostLikeRepository, orgRepo *repository.OrganizationRepository, logger *slog.Logger) *PostService {
	return &PostService{db: db, repo: repo, likeRepo: likeRepo, orgRepo: orgRepo, logger: logger}
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

// Get returns a post by id, marking whether the given user (0 = anonymous)
// has liked it.
func (s *PostService) Get(id, userID uint) (*model.CommunityPost, error) {
	p, err := s.repo.FindByID(id)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return nil, util.NewAppError(404, constants.CodeNotFound, fmt.Sprintf("CommunityPost[id=%d] not found", id))
		}
		return nil, fmt.Errorf("post get: %w", err)
	}
	if userID > 0 {
		liked, err := s.likeRepo.Exists(userID, id)
		if err != nil {
			return nil, fmt.Errorf("post get like state: %w", err)
		}
		p.Liked = liked
	}
	return p, nil
}

// ToggleLike flips the user's like on a post: the first call likes, the next
// unlikes. The like row and the denormalized counter move in one transaction,
// and the unique (user_id, post_id) index makes concurrent clicks from the
// same user settle as a single toggle.
func (s *PostService) ToggleLike(userID, postID uint) (*model.CommunityPost, error) {
	if _, err := s.Get(postID, 0); err != nil {
		return nil, err
	}
	liked := false
	err := s.db.Transaction(func(tx *gorm.DB) error {
		removed, err := s.likeRepo.DeleteTx(tx, userID, postID)
		if err != nil {
			return err
		}
		if removed > 0 {
			liked = false
			return s.repo.DecrementLikeTx(tx, postID)
		}
		if err := s.likeRepo.CreateTx(tx, &model.PostLike{UserID: userID, PostID: postID}); err != nil {
			if errors.Is(err, repository.ErrDuplicate) {
				// A concurrent toggle from the same user already recorded the
				// like; count it once and report the post as liked.
				liked = true
				return nil
			}
			return err
		}
		liked = true
		return s.repo.IncrementLikeTx(tx, postID)
	})
	if err != nil {
		return nil, fmt.Errorf("post like toggle: %w", err)
	}
	if liked {
		s.logger.Info(fmt.Sprintf(constants.LogPostLikeSuccess, postID, userID), "id", postID)
	} else {
		s.logger.Info(fmt.Sprintf(constants.LogPostUnlikeSuccess, postID, userID), "id", postID)
	}
	p, err := s.repo.FindByID(postID)
	if err != nil {
		return nil, fmt.Errorf("post like reload: %w", err)
	}
	p.Liked = liked
	return p, nil
}

// List filters posts.
func (s *PostService) List(postType, keyword string, page, pageSize int) ([]model.CommunityPost, int64, error) {
	items, total, err := s.repo.List(postType, keyword, page, pageSize)
	if err != nil {
		return nil, 0, fmt.Errorf("post list: %w", err)
	}
	return items, total, nil
}

// ListLatest returns recent posts.
func (s *PostService) ListLatest(limit int) ([]model.CommunityPost, error) {
	items, err := s.repo.ListLatest(limit)
	if err != nil {
		return nil, fmt.Errorf("post latest: %w", err)
	}
	return items, nil
}
