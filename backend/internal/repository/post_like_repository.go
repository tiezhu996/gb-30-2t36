package repository

import (
	"gorm.io/gorm"

	"github.com/gbadopt/gbadopt/internal/model"
)

// PostLikeRepository handles post-like persistence.
type PostLikeRepository struct{ db *gorm.DB }

// NewPostLikeRepository creates the repository.
func NewPostLikeRepository(db *gorm.DB) *PostLikeRepository { return &PostLikeRepository{db: db} }

// ExistsTx reports whether a user has liked a post within an outer transaction.
func (r *PostLikeRepository) ExistsTx(tx *gorm.DB, userID, postID uint) (bool, error) {
	var count int64
	if err := tx.Model(&model.PostLike{}).
		Where("user_id = ? AND post_id = ?", userID, postID).
		Count(&count).Error; err != nil {
		return false, err
	}
	return count > 0, nil
}

// CreateTx inserts a like within an outer transaction.
func (r *PostLikeRepository) CreateTx(tx *gorm.DB, userID, postID uint) error {
	return translate(tx.Create(&model.PostLike{UserID: userID, PostID: postID}).Error)
}

// DeleteTx removes a like within an outer transaction.
func (r *PostLikeRepository) DeleteTx(tx *gorm.DB, userID, postID uint) error {
	return tx.Where("user_id = ? AND post_id = ?", userID, postID).
		Delete(&model.PostLike{}).Error
}

// LikedPostIDs returns the ids of the posts a user has liked.
func (r *PostLikeRepository) LikedPostIDs(userID uint) ([]uint, error) {
	var ids []uint
	if err := r.db.Model(&model.PostLike{}).
		Where("user_id = ?", userID).
		Pluck("post_id", &ids).Error; err != nil {
		return nil, err
	}
	return ids, nil
}

// LikedIDSet returns the liked post ids of a user as a lookup set,
// scoped to the given post ids when non-empty.
func (r *PostLikeRepository) LikedIDSet(userID uint, postIDs []uint) (map[uint]bool, error) {
	set := make(map[uint]bool)
	if userID == 0 || len(postIDs) == 0 {
		return set, nil
	}
	ids, err := r.LikedPostIDsByPosts(userID, postIDs)
	if err != nil {
		return nil, err
	}
	for _, id := range ids {
		set[id] = true
	}
	return set, nil
}

// LikedPostIDsByPosts returns liked post ids restricted to the given posts.
func (r *PostLikeRepository) LikedPostIDsByPosts(userID uint, postIDs []uint) ([]uint, error) {
	var ids []uint
	if err := r.db.Model(&model.PostLike{}).
		Where("user_id = ? AND post_id IN ?", userID, postIDs).
		Pluck("post_id", &ids).Error; err != nil {
		return nil, err
	}
	return ids, nil
}
