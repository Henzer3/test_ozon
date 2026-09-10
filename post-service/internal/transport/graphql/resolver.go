package graphql

import (
	"context"
	"encoding/base64"
	"fmt"
	"strconv"

	"github.com/Henzer3/test-ozon/post-service/internal/entity"
	"github.com/Henzer3/test-ozon/post-service/internal/port"
	"github.com/Henzer3/test-ozon/post-service/internal/transport/graphql/convertor"
	"github.com/Henzer3/test-ozon/post-service/internal/transport/graphql/model"
	"github.com/vektah/gqlparser/v2/gqlerror"
)

const maxCommentPageSize = 100
const maxLengthComment = 2000

type PostService interface {
	CreatePost(ctx context.Context, input port.CreatePostInput) (entity.Post, error)
	Posts(ctx context.Context) ([]entity.Post, error)
	Post(ctx context.Context, id int64) (entity.Post, error)
}

type CommentService interface {
	CreateComment(ctx context.Context, input port.CreateCommentInput) (entity.Comment, error)
	Comments(ctx context.Context, input port.CommentPageInput) (entity.CommentPage, error)
}

type SubscriptionService interface {
	SubscribeComments(ctx context.Context, postID int64) (<-chan entity.Comment, error)
}

type Resolver struct {
	PostService         PostService
	CommentService      CommentService
	SubscriptionService SubscriptionService
}

func NewResolver(
	postService PostService,
	commentService CommentService,
	subscriptionService SubscriptionService,
) *Resolver {
	return &Resolver{
		PostService:         postService,
		CommentService:      commentService,
		SubscriptionService: subscriptionService,
	}
}

func (r *Resolver) CreatePost(ctx context.Context, input model.CreatePostInput) (*model.Post, error) {
	user, ok := port.UserFromContext(ctx)
	if !ok {
		return nil, unauthenticatedError()
	}

	post, err := r.PostService.CreatePost(ctx, port.CreatePostInput{
		AuthorID:        user.ID,
		Title:           input.Title,
		Content:         input.Content,
		CommentsEnabled: input.CommentsEnabled,
	})
	if err != nil {
		return nil, convertor.ConvertError(err)
	}

	return postToModel(post), nil
}

func (r *Resolver) CreateComment(ctx context.Context, input model.CreateCommentInput) (*model.Comment, error) {
	user, ok := port.UserFromContext(ctx)
	if !ok {
		return nil, unauthenticatedError()
	}

	postID, err := parseID("postID", input.PostID)
	if err != nil {
		return nil, err
	}

	var parentID *int64
	if input.ParentID != nil {
		parsedParentID, parseErr := parseID("parentID", *input.ParentID)
		err = parseErr
		if err != nil {
			return nil, err
		}
		parentID = &parsedParentID
	}

	if len([]rune(input.Content)) > maxLengthComment {
		return nil, convertor.ConvertError(entity.ErrCommentTooLong)
	}

	comment, err := r.CommentService.CreateComment(ctx, port.CreateCommentInput{
		AuthorID: user.ID,
		PostID:   postID,
		ParentID: parentID,
		Content:  input.Content,
	})
	if err != nil {
		return nil, convertor.ConvertError(err)
	}

	return commentToModel(comment), nil
}

func (r *Resolver) Posts(ctx context.Context) ([]*model.Post, error) {
	posts, err := r.PostService.Posts(ctx)
	if err != nil {
		return nil, convertor.ConvertError(err)
	}

	result := make([]*model.Post, len(posts))
	for i := range posts {
		result[i] = postToModel(posts[i])
	}

	return result, nil
}

func (r *Resolver) GetPost(ctx context.Context, id int64) (*model.Post, error) {
	post, err := r.PostService.Post(ctx, id)
	if err != nil {
		return nil, convertor.ConvertError(err)
	}

	return postToModel(post), nil
}

func (r *Resolver) Comments(ctx context.Context, postID int64, first int, after *string) (*model.CommentConnection, error) {
	if first < 1 || first > maxCommentPageSize {
		return nil, badUserInputError("first must be between 1 and 100")
	}

	afterID, err := decodeCursor(after)
	if err != nil {
		return nil, err
	}

	page, err := r.CommentService.Comments(ctx, port.CommentPageInput{
		PostID:  postID,
		Limit:   first,
		AfterID: afterID,
	})
	if err != nil {
		return nil, convertor.ConvertError(err)
	}

	nodes := make([]*model.Comment, len(page.Comments))
	for i := range page.Comments {
		nodes[i] = commentToModel(page.Comments[i])
	}

	return &model.CommentConnection{
		Nodes: nodes,
		PageInfo: &model.PageInfo{
			EndCursor:   encodeCursor(page.EndCursor),
			HasNextPage: page.HasNextPage,
		},
	}, nil
}

func (r *Resolver) SubscribeComments(ctx context.Context, postID int64) (<-chan *model.Comment, error) {
	events, err := r.SubscriptionService.SubscribeComments(ctx, postID)
	if err != nil {
		return nil, convertor.ConvertError(err)
	}

	result := make(chan *model.Comment, 1)
	go func() {
		defer close(result)

		for {
			select {
			case <-ctx.Done():
				return
			case comment, ok := <-events:
				if !ok {
					return
				}

				select {
				case result <- commentToModel(comment):
				case <-ctx.Done():
					return
				}
			}
		}
	}()

	return result, nil
}

func postToModel(post entity.Post) *model.Post {
	return &model.Post{
		ID:              strconv.FormatInt(post.ID, 10),
		AuthorID:        strconv.FormatInt(post.AuthorID, 10),
		Title:           post.Title,
		Content:         post.Content,
		CommentsEnabled: post.CommentsEnabled,
	}
}

func commentToModel(comment entity.Comment) *model.Comment {
	replies := make([]*model.Comment, len(comment.Replies))
	for i := range comment.Replies {
		replies[i] = commentToModel(comment.Replies[i])
	}

	result := &model.Comment{
		ID:       strconv.FormatInt(comment.ID, 10),
		PostID:   strconv.FormatInt(comment.PostID, 10),
		AuthorID: strconv.FormatInt(comment.AuthorID, 10),
		Content:  comment.Content,
		Replies:  replies,
	}

	if comment.ParentID != nil {
		result.ParentID = new(strconv.FormatInt(*comment.ParentID, 10))
	}

	return result
}

func parseID(field string, value string) (int64, error) {
	id, err := strconv.ParseInt(value, 10, 64)
	if err != nil || id <= 0 {
		return 0, badUserInputError(fmt.Sprintf("%s must be a positive integer", field))
	}

	return id, nil
}

func decodeCursor(cursor *string) (*int64, error) {
	if cursor == nil {
		return nil, nil
	}

	decoded, err := base64.RawURLEncoding.DecodeString(*cursor)
	if err != nil {
		return nil, badUserInputError("after contains an invalid cursor")
	}

	id, err := parseID("after cursor", string(decoded))
	if err != nil {
		return nil, badUserInputError("after contains an invalid cursor")
	}

	return &id, nil
}

func encodeCursor(id *int64) *string {
	if id == nil {
		return nil
	}

	cursor := base64.RawURLEncoding.EncodeToString(
		[]byte(strconv.FormatInt(*id, 10)),
	)

	return &cursor
}

func badUserInputError(message string) error {
	return &gqlerror.Error{
		Message: message,
		Extensions: map[string]any{
			"code": "BAD_USER_INPUT",
		},
	}
}

func unauthenticatedError() error {
	return &gqlerror.Error{
		Message: "authentication required",
		Extensions: map[string]any{
			"code": "UNAUTHENTICATED",
		},
	}
}
