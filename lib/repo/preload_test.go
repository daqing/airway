package repo

import (
	"context"
	"testing"
)

// The preloader derives SQL column names from Go field names, so ID-suffixed
// fields ("ID", "OwnerID", "ArticleID") must map to "id", "owner_id",
// "article_id" — the naive per-uppercase-letter conversion turned them into
// "i_d" / "owner_i_d" and broke every preload query.

type PreloadArticle struct {
	ID       int64  `db:"id"`
	Title    string `db:"title"`
	EditorID int64  `db:"editor_id"`

	Reviews []*PreloadReview
}

func (PreloadArticle) TableName() string { return "preload_articles" }

type PreloadReview struct {
	ID               int64  `db:"id"`
	PostID           int64  `db:"post_id"`
	PreloadArticleID int64  `db:"preload_article_id"`
	Body             string `db:"body"`

	Post *PreloadArticle
}

func (PreloadReview) TableName() string { return "preload_reviews" }

// Relations covers the explicit-configuration path; the article/review
// models rely on struct-field inference instead.
func (r PreloadReview) Relations() map[string]Relation {
	return map[string]Relation{
		"Post": NewBelongsTo(PreloadArticle{}, "PostID"),
	}
}

func setupPreloadFixture(t *testing.T) *DB {
	t.Helper()

	db, err := SetupDBWithDriver("sqlite", ":memory:")
	if err != nil {
		t.Fatalf("setup sqlite: %v", err)
	}

	stmts := []string{
		`CREATE TABLE preload_articles (id INTEGER PRIMARY KEY, title TEXT NOT NULL, editor_id INTEGER NOT NULL)`,
		`CREATE TABLE preload_reviews (id INTEGER PRIMARY KEY, post_id INTEGER NOT NULL, preload_article_id INTEGER NOT NULL, body TEXT NOT NULL)`,
		`INSERT INTO preload_articles (id, title, editor_id) VALUES (1, 'a1', 1), (2, 'a2', 1), (3, 'a3', 2)`,
		`INSERT INTO preload_reviews (id, post_id, preload_article_id, body) VALUES (1, 1, 1, 'r1'), (2, 1, 1, 'r2'), (3, 3, 3, 'r3')`,
	}
	for _, stmt := range stmts {
		if _, err := db.conn.ExecContext(context.Background(), stmt); err != nil {
			t.Fatalf("exec %q: %v", stmt, err)
		}
	}

	return db
}

func TestPreloadInfersHasManyFromTypeName(t *testing.T) {
	db := setupPreloadFixture(t)

	articles, err := FindAll[PreloadArticle]()
	if err != nil {
		t.Fatalf("find articles: %v", err)
	}

	// HasMany through inference: the foreign key is derived from the Go type
	// name (PreloadArticleID -> preload_article_id), which the naive
	// converter mangled into "preload_article_i_d".
	if err := db.Preload("Reviews").Exec(&articles); err != nil {
		t.Fatalf("preload reviews: %v", err)
	}

	wantReviews := map[int64]int{1: 2, 2: 0, 3: 1}
	for _, article := range articles {
		if got := len(article.Reviews); got != wantReviews[article.ID] {
			t.Fatalf("article %d: expected %d reviews, got %d", article.ID, wantReviews[article.ID], got)
		}
	}
}

func TestPreloadUsesRelationsConfig(t *testing.T) {
	db := setupPreloadFixture(t)

	reviews, err := FindAll[PreloadReview]()
	if err != nil {
		t.Fatalf("find reviews: %v", err)
	}

	// Explicit Relations(): fk PostID -> post_id on the main table, pk ID ->
	// id on the related table.
	if err := db.Preload("Post").Exec(&reviews); err != nil {
		t.Fatalf("preload posts: %v", err)
	}

	if len(reviews) != 3 {
		t.Fatalf("expected 3 reviews, got %d", len(reviews))
	}
	for _, review := range reviews {
		if review.Post == nil {
			t.Fatalf("review %d: expected preloaded post", review.ID)
		}
		if review.Post.ID != review.PostID {
			t.Fatalf("review %d: expected post %d, got %d", review.ID, review.PostID, review.Post.ID)
		}
	}
}

func TestSnakeCaseFieldNameCollapsesInitialisms(t *testing.T) {
	cases := map[string]string{
		"ID":        "id",
		"OwnerID":   "owner_id",
		"PostID":    "post_id",
		"CreatedAt": "created_at",
		"UpdatedAt": "updated_at",
		"Title":     "title",
	}
	for in, want := range cases {
		if got := snakeCaseFieldName(in); got != want {
			t.Fatalf("snakeCaseFieldName(%q) = %q, want %q", in, got, want)
		}
	}
}
