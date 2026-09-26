package service

import (
	"errors"
	"regexp"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"

	"github.com/gbadopt/gbadopt/internal/repository"
)

func newPostService(t *testing.T) (*PostService, sqlmock.Sqlmock) {
	t.Helper()
	db, mock := newServiceDB(t)
	svc := NewPostService(db,
		repository.NewCommunityPostRepository(db),
		repository.NewPostLikeRepository(db),
		repository.NewOrganizationRepository(db),
		newTestLogger())
	return svc, mock
}

func postRows(id uint, likeCount int) *sqlmock.Rows {
	return sqlmock.NewRows([]string{"id", "user_id", "org_id", "title", "content", "images", "post_type", "like_count", "comment_count", "status", "created_at"}).
		AddRow(id, 1, 0, "title", "content", "[]", "story", likeCount, 0, "published", time.Now())
}

func expectFindPost(mock sqlmock.Sqlmock, id uint, likeCount int) {
	mock.ExpectQuery(regexp.QuoteMeta(`SELECT * FROM "community_posts" WHERE "community_posts"."id" = $1 ORDER BY "community_posts"."id" LIMIT $2`)).
		WithArgs(id, 1).
		WillReturnRows(postRows(id, likeCount))
}

func TestPostServiceToggleLikeFirstLike(t *testing.T) {
	svc, mock := newPostService(t)
	expectFindPost(mock, 7, 5)
	mock.ExpectBegin()
	mock.ExpectExec(regexp.QuoteMeta(`DELETE FROM "post_likes" WHERE user_id = $1 AND post_id = $2`)).
		WithArgs(2, 7).
		WillReturnResult(sqlmock.NewResult(0, 0))
	mock.ExpectQuery(regexp.QuoteMeta(`INSERT INTO "post_likes" ("user_id","post_id","created_at") VALUES ($1,$2,$3) RETURNING "id"`)).
		WithArgs(2, 7, sqlmock.AnyArg()).
		WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(1))
	mock.ExpectExec(regexp.QuoteMeta(`UPDATE "community_posts" SET "like_count"=like_count + 1 WHERE id = $1`)).
		WithArgs(7).
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectCommit()
	expectFindPost(mock, 7, 6)

	p, err := svc.ToggleLike(2, 7)
	if err != nil {
		t.Fatalf("ToggleLike: %v", err)
	}
	if !p.Liked {
		t.Errorf("expected liked=true after first toggle")
	}
	if p.LikeCount != 6 {
		t.Errorf("expected like_count=6, got %d", p.LikeCount)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("unmet expectations: %v", err)
	}
}

func TestPostServiceToggleLikeUnlike(t *testing.T) {
	svc, mock := newPostService(t)
	expectFindPost(mock, 7, 5)
	mock.ExpectBegin()
	mock.ExpectExec(regexp.QuoteMeta(`DELETE FROM "post_likes" WHERE user_id = $1 AND post_id = $2`)).
		WithArgs(2, 7).
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectExec(regexp.QuoteMeta(`UPDATE "community_posts" SET "like_count"=like_count - 1 WHERE id = $1 AND like_count > 0`)).
		WithArgs(7).
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectCommit()
	expectFindPost(mock, 7, 4)

	p, err := svc.ToggleLike(2, 7)
	if err != nil {
		t.Fatalf("ToggleLike: %v", err)
	}
	if p.Liked {
		t.Errorf("expected liked=false after second toggle")
	}
	if p.LikeCount != 4 {
		t.Errorf("expected like_count=4, got %d", p.LikeCount)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("unmet expectations: %v", err)
	}
}

// A concurrent toggle from the same user loses the insert race: the unique
// index rejects it and the like is counted only once.
func TestPostServiceToggleLikeConcurrentDuplicate(t *testing.T) {
	svc, mock := newPostService(t)
	expectFindPost(mock, 7, 5)
	mock.ExpectBegin()
	mock.ExpectExec(regexp.QuoteMeta(`DELETE FROM "post_likes" WHERE user_id = $1 AND post_id = $2`)).
		WithArgs(2, 7).
		WillReturnResult(sqlmock.NewResult(0, 0))
	mock.ExpectQuery(regexp.QuoteMeta(`INSERT INTO "post_likes" ("user_id","post_id","created_at") VALUES ($1,$2,$3) RETURNING "id"`)).
		WithArgs(2, 7, sqlmock.AnyArg()).
		WillReturnError(errors.New(`duplicate key value violates unique constraint "idx_post_like_user_post"`))
	mock.ExpectCommit()
	expectFindPost(mock, 7, 5)

	p, err := svc.ToggleLike(2, 7)
	if err != nil {
		t.Fatalf("ToggleLike: %v", err)
	}
	if !p.Liked {
		t.Errorf("expected liked=true when a concurrent like already exists")
	}
	if p.LikeCount != 5 {
		t.Errorf("expected like_count to stay 5, got %d", p.LikeCount)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("unmet expectations: %v", err)
	}
}

func TestPostServiceToggleLikePostMissing(t *testing.T) {
	svc, mock := newPostService(t)
	mock.ExpectQuery(regexp.QuoteMeta(`SELECT * FROM "community_posts" WHERE "community_posts"."id" = $1 ORDER BY "community_posts"."id" LIMIT $2`)).
		WithArgs(99, 1).
		WillReturnRows(sqlmock.NewRows([]string{"id"}))

	if _, err := svc.ToggleLike(2, 99); err == nil {
		t.Fatal("expected error for missing post")
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("unmet expectations: %v", err)
	}
}

func TestPostServiceGetMarksLikedForUser(t *testing.T) {
	svc, mock := newPostService(t)
	expectFindPost(mock, 7, 5)
	mock.ExpectQuery(regexp.QuoteMeta(`SELECT count(*) FROM "post_likes" WHERE user_id = $1 AND post_id = $2`)).
		WithArgs(2, 7).
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(1))

	p, err := svc.Get(7, 2)
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if !p.Liked {
		t.Errorf("expected liked=true for a user who liked the post")
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("unmet expectations: %v", err)
	}
}

func TestPostServiceGetAnonymousNotLiked(t *testing.T) {
	svc, mock := newPostService(t)
	expectFindPost(mock, 7, 5)

	p, err := svc.Get(7, 0)
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if p.Liked {
		t.Errorf("expected liked=false for anonymous request")
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("unmet expectations: %v", err)
	}
}
