package repository

import (
	"gorm.io/gorm"

	"github.com/gbadopt/gbadopt/internal/model"
)

// PostLikeRepository handles post like persistence.
type PostLikeRepository struct{ db *gorm.DB }

// NewPostLikeRepository creates the repository.
func NewPostLikeRepository(db *gorm.DB) *PostLikeRepository {
	return &PostLikeRepository{db: db}
}

// CreateTx inserts a like within an outer transaction.
// Returns ErrDuplicate when the user has already liked the post.
func (r *PostLikeRepository) CreateTx(tx *gorm.DB, like *model.PostLike) error {
	return translate(tx.Create(like).Error)
}

// DeleteTx removes a like within an outer transaction and reports
// how many rows were removed (0 or 1 thanks to the unique index).
func (r *PostLikeRepository) DeleteTx(tx *gorm.DB, userID, postID uint) (int64, error) {
	res := tx.Where("user_id = ? AND post_id = ?", userID, postID).Delete(&model.PostLike{})
	return res.RowsAffected, res.Error
}

// Exists reports whether the user has liked the post.
func (r *PostLikeRepository) Exists(userID, postID uint) (bool, error) {
	var count int64
	if err := r.db.Model(&model.PostLike{}).Where("user_id = ? AND post_id = ?", userID, postID).Count(&count).Error; err != nil {
		return false, err
	}
	return count > 0, nil
}
