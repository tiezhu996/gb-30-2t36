package service

import (
	"errors"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"gorm.io/gorm"

	"github.com/gbadopt/gbadopt/internal/repository"
)

const (
	// SELECT * FROM "community_posts" WHERE "community_posts"."id" = $1 ORDER BY "community_posts"."id" LIMIT $2 FOR UPDATE
	lockPostSQL = `SELECT \* FROM "community_posts" WHERE "community_posts"\."id" = \$1 ORDER BY "community_posts"\."id" LIMIT \$2 FOR UPDATE`
	// SELECT count(*) FROM "post_likes" WHERE user_id = $1 AND post_id = $2
	countLikeSQL = `SELECT count\(\*\) FROM "post_likes" WHERE user_id = \$1 AND post_id = \$2`
	// INSERT INTO "post_likes" (...) VALUES (...) RETURNING "id"
	insertLikeSQL = `INSERT INTO "post_likes" \("user_id","post_id","created_at"\) VALUES \(\$1,\$2,\$3\) RETURNING "id"`
	// DELETE FROM "post_likes" WHERE user_id = $1 AND post_id = $2
	deleteLikeSQL = `DELETE FROM "post_likes" WHERE user_id = \$1 AND post_id = \$2`
	// UPDATE "community_posts" SET "like_count"=GREATEST(like_count + $1, 0) WHERE id = $2
	updateCountSQL = `UPDATE "community_posts" SET "like_count"=GREATEST\(like_count \+ \$1, 0\) WHERE id = \$2`
)

// errDuplicateKey mimics the PostgreSQL driver error string that the
// repository layer maps to repository.ErrDuplicate.
var errDuplicateKey = errors.New("ERROR: duplicate key value violates unique constraint \"idx_post_like_user_post\" (SQLSTATE 23505)")

func postRow(likeCount int) *sqlmock.Rows {
	return sqlmock.NewRows([]string{"id", "user_id", "org_id", "title", "content", "images", "post_type", "like_count", "comment_count", "status", "created_at"}).
		AddRow(1, 2, nil, "t", "c", "[]", "story", likeCount, 0, "published", nil)
}

func newPostSvc(gormDB *gorm.DB) *PostService {
	return NewPostService(gormDB,
		repository.NewCommunityPostRepository(gormDB),
		repository.NewPostLikeRepository(gormDB),
		repository.NewOrganizationRepository(gormDB), nil, newTestLogger())
}

func TestLikeAdds(t *testing.T) {
	gormDB, mock := newServiceDB(t)
	svc := newPostSvc(gormDB)

	mock.ExpectBegin()
	mock.ExpectQuery(lockPostSQL).
		WithArgs(uint(1), 1).
		WillReturnRows(postRow(5))
	mock.ExpectQuery(countLikeSQL).
		WithArgs(uint(7), uint(1)).
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(0))
	mock.ExpectQuery(insertLikeSQL).
		WithArgs(uint(7), uint(1), sqlmock.AnyArg()).
		WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(1))
	mock.ExpectExec(updateCountSQL).
		WithArgs(1, uint(1)).
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectQuery(lockPostSQL).
		WithArgs(uint(1), 1).
		WillReturnRows(postRow(6))
	mock.ExpectCommit()

	p, err := svc.Like(7, 1)
	if err != nil {
		t.Fatalf("Like: %v", err)
	}
	if p.LikeCount != 6 || !p.Liked {
		t.Fatalf("Like = count %d liked %v, want 6/true", p.LikeCount, p.Liked)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("expectations: %v", err)
	}
}

func TestRepeatedLikeIsIdempotent(t *testing.T) {
	gormDB, mock := newServiceDB(t)
	svc := newPostSvc(gormDB)

	// Like row already exists: no insert, no counter change.
	mock.ExpectBegin()
	mock.ExpectQuery(lockPostSQL).
		WithArgs(uint(1), 1).
		WillReturnRows(postRow(6))
	mock.ExpectQuery(countLikeSQL).
		WithArgs(uint(7), uint(1)).
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(1))
	mock.ExpectQuery(lockPostSQL).
		WithArgs(uint(1), 1).
		WillReturnRows(postRow(6))
	mock.ExpectCommit()

	p, err := svc.Like(7, 1)
	if err != nil {
		t.Fatalf("Like: %v", err)
	}
	if p.LikeCount != 6 || !p.Liked {
		t.Fatalf("repeat Like = count %d liked %v, want 6/true", p.LikeCount, p.Liked)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("expectations: %v", err)
	}
}

func TestUnlikeRemovesAndClampsAtZero(t *testing.T) {
	gormDB, mock := newServiceDB(t)
	svc := newPostSvc(gormDB)

	mock.ExpectBegin()
	mock.ExpectQuery(lockPostSQL).
		WithArgs(uint(1), 1).
		WillReturnRows(postRow(1))
	mock.ExpectQuery(countLikeSQL).
		WithArgs(uint(7), uint(1)).
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(1))
	mock.ExpectExec(deleteLikeSQL).
		WithArgs(uint(7), uint(1)).
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectExec(updateCountSQL).
		WithArgs(-1, uint(1)).
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectQuery(lockPostSQL).
		WithArgs(uint(1), 1).
		WillReturnRows(postRow(0))
	mock.ExpectCommit()

	p, err := svc.Unlike(7, 1)
	if err != nil {
		t.Fatalf("Unlike: %v", err)
	}
	if p.LikeCount < 0 {
		t.Errorf("like_count = %d, must never be negative", p.LikeCount)
	}
	if p.Liked {
		t.Errorf("liked = true, want false")
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("expectations: %v", err)
	}
}

func TestRepeatedUnlikeIsIdempotent(t *testing.T) {
	gormDB, mock := newServiceDB(t)
	svc := newPostSvc(gormDB)

	// No like row: no delete, no counter change.
	mock.ExpectBegin()
	mock.ExpectQuery(lockPostSQL).
		WithArgs(uint(1), 1).
		WillReturnRows(postRow(0))
	mock.ExpectQuery(countLikeSQL).
		WithArgs(uint(7), uint(1)).
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(0))
	mock.ExpectQuery(lockPostSQL).
		WithArgs(uint(1), 1).
		WillReturnRows(postRow(0))
	mock.ExpectCommit()

	p, err := svc.Unlike(7, 1)
	if err != nil {
		t.Fatalf("Unlike: %v", err)
	}
	if p.LikeCount != 0 || p.Liked {
		t.Fatalf("repeat Unlike = count %d liked %v, want 0/false", p.LikeCount, p.Liked)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("expectations: %v", err)
	}
}

func TestLikeDuplicateRaceDoesNotDoubleCount(t *testing.T) {
	gormDB, mock := newServiceDB(t)
	svc := newPostSvc(gormDB)

	// Absent at check time, but insert hits the unique index because a
	// concurrent identical request won: no counter increment may happen.
	mock.ExpectBegin()
	mock.ExpectQuery(lockPostSQL).
		WithArgs(uint(1), 1).
		WillReturnRows(postRow(5))
	mock.ExpectQuery(countLikeSQL).
		WithArgs(uint(7), uint(1)).
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(0))
	mock.ExpectQuery(insertLikeSQL).
		WithArgs(uint(7), uint(1), sqlmock.AnyArg()).
		WillReturnError(errDuplicateKey)
	mock.ExpectQuery(lockPostSQL).
		WithArgs(uint(1), 1).
		WillReturnRows(postRow(6))
	mock.ExpectCommit()

	p, err := svc.Like(7, 1)
	if err != nil {
		t.Fatalf("Like: %v", err)
	}
	if p.LikeCount != 6 {
		t.Fatalf("like_count = %d, want 6 (no double increment)", p.LikeCount)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("expectations: %v", err)
	}
}
