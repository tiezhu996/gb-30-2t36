package model

import "time"

// PostLike records one user's like on a community post.
type PostLike struct {
	ID        uint      `gorm:"primaryKey" json:"id"`
	UserID    uint      `gorm:"index:idx_post_like_user_post,unique;not null" json:"user_id"`
	PostID    uint      `gorm:"index:idx_post_like_user_post,unique;not null" json:"post_id"`
	CreatedAt time.Time `json:"created_at"`
}
