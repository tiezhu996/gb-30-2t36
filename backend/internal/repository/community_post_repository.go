package repository

import (
	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"github.com/gbadopt/gbadopt/internal/model"
)

// CommunityPostRepository handles post persistence.
type CommunityPostRepository struct{ db *gorm.DB }

// NewCommunityPostRepository creates the repository.
func NewCommunityPostRepository(db *gorm.DB) *CommunityPostRepository {
	return &CommunityPostRepository{db: db}
}

// Create inserts a post.
func (r *CommunityPostRepository) Create(p *model.CommunityPost) error {
	return translate(r.db.Create(p).Error)
}

// FindByID locates a post by id.
func (r *CommunityPostRepository) FindByID(id uint) (*model.CommunityPost, error) {
	var p model.CommunityPost
	if err := translate(r.db.First(&p, id).Error); err != nil {
		return nil, err
	}
	return &p, nil
}

// Update persists a post.
func (r *CommunityPostRepository) Update(p *model.CommunityPost) error {
	return translate(r.db.Save(p).Error)
}

// GetForUpdateTx locks a post row for the duration of an outer transaction.
func (r *CommunityPostRepository) GetForUpdateTx(tx *gorm.DB, id uint) (*model.CommunityPost, error) {
	var p model.CommunityPost
	if err := translate(tx.Clauses(clause.Locking{Strength: "UPDATE"}).First(&p, id).Error); err != nil {
		return nil, err
	}
	return &p, nil
}

// AdjustLikeCountTx changes the like count by delta and clamps it at zero.
func (r *CommunityPostRepository) AdjustLikeCountTx(tx *gorm.DB, id uint, delta int) error {
	expr := gorm.Expr("GREATEST(like_count + ?, 0)", delta)
	return tx.Model(&model.CommunityPost{}).Where("id = ?", id).
		UpdateColumn("like_count", expr).Error
}

// IncrementComment bumps the comment count.
func (r *CommunityPostRepository) IncrementComment(id uint) error {
	return r.db.Model(&model.CommunityPost{}).Where("id = ?", id).
		UpdateColumn("comment_count", gorm.Expr("comment_count + 1")).Error
}

// IncrementCommentTx bumps the comment count within an outer transaction.
func (r *CommunityPostRepository) IncrementCommentTx(tx *gorm.DB, id uint) error {
	return tx.Model(&model.CommunityPost{}).Where("id = ?", id).
		UpdateColumn("comment_count", gorm.Expr("comment_count + 1")).Error
}

// List filters posts by type and keyword with pagination.
func (r *CommunityPostRepository) List(postType, keyword string, page, pageSize int) ([]model.CommunityPost, int64, error) {
	var items []model.CommunityPost
	var total int64
	q := r.db.Model(&model.CommunityPost{}).Where("status = ?", "published")
	if postType != "" {
		q = q.Where("post_type = ?", postType)
	}
	if keyword != "" {
		like := "%" + keyword + "%"
		q = q.Where("title LIKE ? OR content LIKE ?", like, like)
	}
	if err := q.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	if err := q.Order("id DESC").Offset((page - 1) * pageSize).Limit(pageSize).Find(&items).Error; err != nil {
		return nil, 0, err
	}
	return items, total, nil
}

// ListLatest returns latest posts for the home page.
func (r *CommunityPostRepository) ListLatest(limit int) ([]model.CommunityPost, error) {
	var items []model.CommunityPost
	if err := r.db.Where("status = ?", "published").Order("id DESC").Limit(limit).Find(&items).Error; err != nil {
		return nil, err
	}
	return items, nil
}
