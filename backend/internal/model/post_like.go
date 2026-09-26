package model

import "time"

// PostLike records one like from a user on a community post.
// The unique index guarantees each user can like each post at most once.
type PostLike struct {
	ID        uint      `gorm:"primaryKey" json:"id"`
	UserID    uint      `gorm:"uniqueIndex:idx_post_like_user_post;not null" json:"user_id"`
	PostID    uint      `gorm:"uniqueIndex:idx_post_like_user_post;not null" json:"post_id"`
	CreatedAt time.Time `json:"created_at"`
}
