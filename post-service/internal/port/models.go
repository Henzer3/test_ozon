package port

import "context"

type LoginRequest struct {
	Email    string
	Password string
	AppId    int32
}

type UserPermissions struct {
	ID      int64
	Email   string
	AppID   int32
	IsAdmin bool
}

type userContextKey struct{}

func WithUser(ctx context.Context, user UserPermissions) context.Context {
	return context.WithValue(ctx, userContextKey{}, user)
}

func UserFromContext(ctx context.Context) (UserPermissions, bool) {
	user, ok := ctx.Value(userContextKey{}).(UserPermissions)
	return user, ok
}

type CreatePostInput struct {
	AuthorID        int64
	Title           string
	Content         string
	CommentsEnabled bool
}

type CreateCommentInput struct {
	AuthorID int64
	PostID   int64
	ParentID *int64
	Content  string
}

type CommentPageInput struct {
	PostID  int64
	Limit   int
	AfterID *int64
}
