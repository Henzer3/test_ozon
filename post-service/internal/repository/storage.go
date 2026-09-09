package repository

import (
	"context"
	"database/sql"
	"errors"
	"log/slog"

	"github.com/Henzer3/test-ozon/post-service/internal/entity"
	"github.com/Henzer3/test-ozon/post-service/internal/port"
	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/jmoiron/sqlx"
)

type DB struct {
	log  *slog.Logger
	conn *sqlx.DB
}

func New(log *slog.Logger, address string) (*DB, error) {
	db, err := sqlx.Connect("pgx", address)
	if err != nil {
		log.Error("connection problem", "address", address, "error", err)
		return nil, err
	}

	return &DB{
		log:  log,
		conn: db,
	}, nil
}

func (db *DB) Close() error {
	if err := db.conn.Close(); err != nil {
		db.log.Error("cant close db conn", "err", err)
		return err
	}
	return nil
}

func (db *DB) CreatePost(ctx context.Context, input port.CreatePostInput) (entity.Post, error) {
	const query = `
		INSERT INTO post_service.posts (
			author_id,
			title,
			content,
			comments_enabled
		)
		VALUES ($1, $2, $3, $4)
		RETURNING id, author_id, title, content, comments_enabled
	`

	var post postRow
	err := db.conn.GetContext(
		ctx,
		&post,
		query,
		input.AuthorID,
		input.Title,
		input.Content,
		input.CommentsEnabled,
	)
	if err != nil {
		db.log.Error("create post error in storage", "err", err)
		return entity.Post{}, err
	}

	return post.entity(), nil
}

func (db *DB) GetPosts(ctx context.Context) ([]entity.Post, error) {
	const query = `
		SELECT id, author_id, title, content, comments_enabled
		FROM post_service.posts
		ORDER BY id DESC
	`

	var rows []postRow
	if err := db.conn.SelectContext(ctx, &rows, query); err != nil {
		db.log.Error("get posts error in storage", "err", err)
		return nil, err
	}

	posts := make([]entity.Post, len(rows))
	for i := range rows {
		posts[i] = rows[i].entity()
	}

	return posts, nil
}

func (db *DB) GetPost(ctx context.Context, id int64) (entity.Post, error) {
	const query = `
		SELECT id, author_id, title, content, comments_enabled
		FROM post_service.posts
		WHERE id = $1
	`

	var post postRow
	if err := db.conn.GetContext(ctx, &post, query, id); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return entity.Post{}, entity.ErrPostNotFound
		}
		db.log.Error("get post error in storage", "err", err)
		return entity.Post{}, err
	}

	return post.entity(), nil
}

func (db *DB) CreateComment(ctx context.Context, input port.CreateCommentInput) (entity.Comment, error) {
	const query = `
		INSERT INTO post_service.comments (
			post_id,
			parent_id,
			author_id,
			content
		)
		VALUES ($1, $2, $3, $4)
		RETURNING id, post_id, parent_id, author_id, content
	`

	var comment commentRow
	err := db.conn.GetContext(
		ctx,
		&comment,
		query,
		input.PostID,
		input.ParentID,
		input.AuthorID,
		input.Content,
	)
	if err != nil {
		db.log.Error("create comment error in storage", "err", err)
		return entity.Comment{}, err
	}

	return comment.entity(), nil
}

func (db *DB) GetComment(ctx context.Context, id int64) (entity.Comment, error) {
	const query = `
		SELECT id, post_id, parent_id, author_id, content
		FROM post_service.comments
		WHERE id = $1
	`

	var comment commentRow
	if err := db.conn.GetContext(ctx, &comment, query, id); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return entity.Comment{}, entity.ErrCommentNotFound
		}
		db.log.Error("get comment error in storage", "err", err)
		return entity.Comment{}, err
	}

	return comment.entity(), nil
}

func (db *DB) Comments(ctx context.Context, input port.CommentPageInput) (entity.CommentPage, error) {
	const query = `
		SELECT id, post_id, parent_id, author_id, content
		FROM post_service.comments
		WHERE post_id = $1
		  AND parent_id IS NULL
		  AND ($2::BIGINT IS NULL OR id < $2)
		ORDER BY id DESC
		LIMIT $3
	`

	var roots []commentRow
	if err := db.conn.SelectContext(
		ctx,
		&roots,
		query,
		input.PostID,
		input.AfterID,
		input.Limit+1,
	); err != nil {
		db.log.Error("cant get comments in storage", "err", err)
		return entity.CommentPage{}, err
	}

	hasNextPage := len(roots) > input.Limit
	if hasNextPage {
		roots = roots[:input.Limit]
	}

	if len(roots) == 0 {
		return entity.CommentPage{
			Comments:    []entity.Comment{},
			HasNextPage: false,
		}, nil
	}

	rootIDs := make([]int64, len(roots))
	for i := range roots {
		rootIDs[i] = roots[i].ID
	}

	descendants, err := db.getCommentDescendants(ctx, rootIDs)
	if err != nil {
		db.log.Error("cant get descendants comment in storage, cant ", "err", err)
		return entity.CommentPage{}, err
	}

	childrenByParent := make(map[int64][]commentRow)
	for i := range descendants {
		if descendants[i].ParentID == nil {
			continue
		}
		parentID := *descendants[i].ParentID
		childrenByParent[parentID] = append(childrenByParent[parentID], descendants[i])
	}

	comments := make([]entity.Comment, len(roots))
	for i := range roots {
		comments[i] = buildCommentTree(roots[i], childrenByParent)
	}

	endCursor := roots[len(roots)-1].ID
	return entity.CommentPage{
		Comments:    comments,
		EndCursor:   &endCursor,
		HasNextPage: hasNextPage,
	}, nil
}

func (db *DB) getCommentDescendants(ctx context.Context, rootIDs []int64) ([]commentRow, error) {
	query, args, err := sqlx.In(`
		WITH RECURSIVE comment_tree AS (
			SELECT id, post_id, parent_id, author_id, content
			FROM post_service.comments
			WHERE parent_id IN (?)

			UNION ALL

			SELECT child.id,
			       child.post_id,
			       child.parent_id,
			       child.author_id,
			       child.content
			FROM post_service.comments AS child
			JOIN comment_tree AS parent ON child.parent_id = parent.id
		)
		SELECT id, post_id, parent_id, author_id, content
		FROM comment_tree
		ORDER BY id ASC
	`, rootIDs)
	if err != nil {
		return nil, err
	}

	query = db.conn.Rebind(query)
	var descendants []commentRow
	if err := db.conn.SelectContext(ctx, &descendants, query, args...); err != nil {
		return nil, err
	}

	return descendants, nil
}

func buildCommentTree(row commentRow, childrenByParent map[int64][]commentRow) entity.Comment {
	children := childrenByParent[row.ID]
	replies := make([]entity.Comment, len(children))
	for i := range children {
		replies[i] = buildCommentTree(children[i], childrenByParent)
	}

	comment := row.entity()
	comment.Replies = replies
	return comment
}

type postRow struct {
	ID              int64  `db:"id"`
	AuthorID        int64  `db:"author_id"`
	Title           string `db:"title"`
	Content         string `db:"content"`
	CommentsEnabled bool   `db:"comments_enabled"`
}

func (row postRow) entity() entity.Post {
	return entity.Post{
		ID:              row.ID,
		AuthorID:        row.AuthorID,
		Title:           row.Title,
		Content:         row.Content,
		CommentsEnabled: row.CommentsEnabled,
	}
}

type commentRow struct {
	ID       int64  `db:"id"`
	PostID   int64  `db:"post_id"`
	ParentID *int64 `db:"parent_id"`
	AuthorID int64  `db:"author_id"`
	Content  string `db:"content"`
}

func (row commentRow) entity() entity.Comment {
	return entity.Comment{
		ID:       row.ID,
		PostID:   row.PostID,
		ParentID: row.ParentID,
		AuthorID: row.AuthorID,
		Content:  row.Content,
	}
}
