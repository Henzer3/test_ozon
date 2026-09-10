package graphql

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/Henzer3/test-ozon/post-service/internal/entity"
	"github.com/Henzer3/test-ozon/post-service/internal/port"
	graphqlmock "github.com/Henzer3/test-ozon/post-service/internal/transport/graphql/mock"
	"github.com/Henzer3/test-ozon/post-service/internal/transport/graphql/model"
	"github.com/stretchr/testify/require"
	"github.com/vektah/gqlparser/v2/gqlerror"
	"go.uber.org/mock/gomock"
)

func TestCreatePost(t *testing.T) {
	input := model.CreatePostInput{Title: "title", Content: "content", CommentsEnabled: true}

	t.Run("requires authentication", func(t *testing.T) {
		resolver, _, _ := newResolverWithMocks(t)
		post, err := resolver.CreatePost(context.Background(), input)
		require.Nil(t, post)
		requireGraphQLErrorCode(t, err, "UNAUTHENTICATED")
	})

	t.Run("success", func(t *testing.T) {
		resolver, posts, _ := newResolverWithMocks(t)
		ctx := port.WithUser(context.Background(), port.UserPermissions{ID: 5})
		posts.EXPECT().CreatePost(ctx, port.CreatePostInput{
			AuthorID: 5, Title: input.Title, Content: input.Content, CommentsEnabled: true,
		}).Return(entity.Post{
			ID: 10, AuthorID: 5, Title: input.Title, Content: input.Content, CommentsEnabled: true,
		}, nil)

		post, err := resolver.CreatePost(ctx, input)
		require.NoError(t, err)
		require.Equal(t, &model.Post{
			ID: "10", AuthorID: "5", Title: input.Title, Content: input.Content, CommentsEnabled: true,
		}, post)
	})

	t.Run("service error", func(t *testing.T) {
		resolver, posts, _ := newResolverWithMocks(t)
		ctx := port.WithUser(context.Background(), port.UserPermissions{ID: 5})
		posts.EXPECT().CreatePost(ctx, gomock.Any()).Return(entity.Post{}, entity.ErrPostNotFound)
		post, err := resolver.CreatePost(ctx, input)
		require.Nil(t, post)
		requireGraphQLErrorCode(t, err, "POST_NOT_FOUND")
	})
}

func TestCreateComment(t *testing.T) {
	t.Run("requires authentication", func(t *testing.T) {
		resolver, _, _ := newResolverWithMocks(t)
		comment, err := resolver.CreateComment(context.Background(), model.CreateCommentInput{PostID: "1", Content: "text"})
		require.Nil(t, comment)
		requireGraphQLErrorCode(t, err, "UNAUTHENTICATED")
	})

	t.Run("invalid post ID", func(t *testing.T) {
		resolver, _, _ := newResolverWithMocks(t)
		ctx := port.WithUser(context.Background(), port.UserPermissions{ID: 5})
		comment, err := resolver.CreateComment(ctx, model.CreateCommentInput{PostID: "bad", Content: "text"})
		require.Nil(t, comment)
		requireGraphQLErrorCode(t, err, "BAD_USER_INPUT")
	})

	t.Run("invalid parent ID", func(t *testing.T) {
		resolver, _, _ := newResolverWithMocks(t)
		ctx := port.WithUser(context.Background(), port.UserPermissions{ID: 5})
		parentID := "0"
		comment, err := resolver.CreateComment(ctx, model.CreateCommentInput{PostID: "1", ParentID: &parentID, Content: "text"})
		require.Nil(t, comment)
		requireGraphQLErrorCode(t, err, "BAD_USER_INPUT")
	})

	t.Run("comment is too long", func(t *testing.T) {
		resolver, _, _ := newResolverWithMocks(t)
		ctx := port.WithUser(context.Background(), port.UserPermissions{ID: 5})
		comment, err := resolver.CreateComment(ctx, model.CreateCommentInput{
			PostID: "1", Content: strings.Repeat("я", maxLengthComment+1),
		})
		require.Nil(t, comment)
		requireGraphQLErrorCode(t, err, "COMMENT_TOO_LONG")
	})

	t.Run("success with parent", func(t *testing.T) {
		resolver, _, comments := newResolverWithMocks(t)
		ctx := port.WithUser(context.Background(), port.UserPermissions{ID: 5})
		parentID := "7"
		parsedParentID := int64(7)
		input := model.CreateCommentInput{PostID: "10", ParentID: &parentID, Content: "reply"}
		comments.EXPECT().CreateComment(ctx, port.CreateCommentInput{
			AuthorID: 5, PostID: 10, ParentID: &parsedParentID, Content: "reply",
		}).Return(entity.Comment{
			ID: 20, PostID: 10, ParentID: &parsedParentID, AuthorID: 5, Content: "reply",
		}, nil)

		comment, err := resolver.CreateComment(ctx, input)
		require.NoError(t, err)
		require.Equal(t, "20", comment.ID)
		require.Equal(t, "10", comment.PostID)
		require.Equal(t, "7", *comment.ParentID)
		require.Equal(t, "5", comment.AuthorID)
		require.Empty(t, comment.Replies)
	})

	t.Run("service error", func(t *testing.T) {
		resolver, _, comments := newResolverWithMocks(t)
		ctx := port.WithUser(context.Background(), port.UserPermissions{ID: 5})
		comments.EXPECT().CreateComment(ctx, gomock.Any()).Return(entity.Comment{}, entity.ErrCommentsDisabled)
		comment, err := resolver.CreateComment(ctx, model.CreateCommentInput{PostID: "10", Content: "text"})
		require.Nil(t, comment)
		requireGraphQLErrorCode(t, err, "COMMENTS_DISABLED")
	})
}

func TestPostsAndPost(t *testing.T) {
	t.Run("posts success", func(t *testing.T) {
		resolver, posts, _ := newResolverWithMocks(t)
		posts.EXPECT().Posts(gomock.Any()).Return([]entity.Post{{ID: 2, AuthorID: 3}, {ID: 1, AuthorID: 4}}, nil)
		result, err := resolver.Posts(context.Background())
		require.NoError(t, err)
		require.Len(t, result, 2)
		require.Equal(t, "2", result[0].ID)
		require.Equal(t, "1", result[1].ID)
	})

	t.Run("posts error", func(t *testing.T) {
		resolver, posts, _ := newResolverWithMocks(t)
		posts.EXPECT().Posts(gomock.Any()).Return(nil, errors.New("database error"))
		result, err := resolver.Posts(context.Background())
		require.Nil(t, result)
		requireGraphQLErrorCode(t, err, "INTERNAL")
	})

	t.Run("post success", func(t *testing.T) {
		resolver, posts, _ := newResolverWithMocks(t)
		posts.EXPECT().Post(gomock.Any(), int64(8)).Return(entity.Post{ID: 8, AuthorID: 2}, nil)
		result, err := resolver.GetPost(context.Background(), 8)
		require.NoError(t, err)
		require.Equal(t, "8", result.ID)
		require.Equal(t, "2", result.AuthorID)
	})

	t.Run("post not found", func(t *testing.T) {
		resolver, posts, _ := newResolverWithMocks(t)
		posts.EXPECT().Post(gomock.Any(), int64(8)).Return(entity.Post{}, entity.ErrPostNotFound)
		result, err := resolver.GetPost(context.Background(), 8)
		require.Nil(t, result)
		requireGraphQLErrorCode(t, err, "POST_NOT_FOUND")
	})
}

func TestComments(t *testing.T) {
	t.Run("invalid page size", func(t *testing.T) {
		resolver, _, _ := newResolverWithMocks(t)
		for _, first := range []int{0, maxCommentPageSize + 1} {
			connection, err := resolver.Comments(context.Background(), 1, first, nil)
			require.Nil(t, connection)
			requireGraphQLErrorCode(t, err, "BAD_USER_INPUT")
		}
	})

	t.Run("invalid cursor", func(t *testing.T) {
		resolver, _, _ := newResolverWithMocks(t)
		cursor := "not-base64!"
		connection, err := resolver.Comments(context.Background(), 1, 20, &cursor)
		require.Nil(t, connection)
		requireGraphQLErrorCode(t, err, "BAD_USER_INPUT")
	})

	t.Run("success", func(t *testing.T) {
		resolver, _, comments := newResolverWithMocks(t)
		afterID := int64(30)
		endCursor := int64(20)
		cursor := encodeCursor(&afterID)
		parentID := int64(20)
		comments.EXPECT().Comments(gomock.Any(), port.CommentPageInput{
			PostID: 10, Limit: 2, AfterID: &afterID,
		}).Return(entity.CommentPage{
			Comments: []entity.Comment{{
				ID: 20, PostID: 10, AuthorID: 3, Content: "root",
				Replies: []entity.Comment{{ID: 21, PostID: 10, ParentID: &parentID, AuthorID: 4, Content: "reply"}},
			}},
			EndCursor: &endCursor, HasNextPage: true,
		}, nil)

		connection, err := resolver.Comments(context.Background(), 10, 2, cursor)
		require.NoError(t, err)
		require.Len(t, connection.Nodes, 1)
		require.Equal(t, "20", connection.Nodes[0].ID)
		require.Len(t, connection.Nodes[0].Replies, 1)
		require.Equal(t, "21", connection.Nodes[0].Replies[0].ID)
		require.Equal(t, "20", *connection.Nodes[0].Replies[0].ParentID)
		require.True(t, connection.PageInfo.HasNextPage)
		require.Equal(t, encodeCursor(&endCursor), connection.PageInfo.EndCursor)
	})

	t.Run("service error", func(t *testing.T) {
		resolver, _, comments := newResolverWithMocks(t)
		comments.EXPECT().Comments(gomock.Any(), gomock.Any()).Return(entity.CommentPage{}, entity.ErrPostNotFound)
		connection, err := resolver.Comments(context.Background(), 10, 2, nil)
		require.Nil(t, connection)
		requireGraphQLErrorCode(t, err, "POST_NOT_FOUND")
	})
}

func TestSubscribeComments(t *testing.T) {
	t.Run("invalid post ID", func(t *testing.T) {
		resolver, _, _, _ := newResolverWithSubscriptionMock(t)

		events, err := resolver.Subscription().CommentAdded(context.Background(), "invalid")

		require.Nil(t, events)
		requireGraphQLErrorCode(t, err, "BAD_USER_INPUT")
	})

	t.Run("service error", func(t *testing.T) {
		resolver, _, _, subscriptions := newResolverWithSubscriptionMock(t)
		subscriptions.EXPECT().SubscribeComments(gomock.Any(), int64(10)).Return(nil, entity.ErrPostNotFound)

		events, err := resolver.Subscription().CommentAdded(context.Background(), "10")

		require.Nil(t, events)
		requireGraphQLErrorCode(t, err, "POST_NOT_FOUND")
	})

	t.Run("forwards comments and stops with context", func(t *testing.T) {
		resolver, _, _, subscriptions := newResolverWithSubscriptionMock(t)
		ctx, cancel := context.WithCancel(context.Background())
		source := make(chan entity.Comment, 1)
		parentID := int64(7)
		want := entity.Comment{ID: 8, PostID: 10, ParentID: &parentID, AuthorID: 3, Content: "new comment"}
		subscriptions.EXPECT().SubscribeComments(ctx, int64(10)).Return((<-chan entity.Comment)(source), nil)

		events, err := resolver.Subscription().CommentAdded(ctx, "10")
		require.NoError(t, err)

		source <- want
		received := <-events
		require.Equal(t, "8", received.ID)
		require.Equal(t, "10", received.PostID)
		require.Equal(t, "7", *received.ParentID)
		require.Equal(t, "3", received.AuthorID)
		require.Equal(t, want.Content, received.Content)

		cancel()
		_, open := <-events
		require.False(t, open)
	})

	t.Run("stops when broker channel closes", func(t *testing.T) {
		resolver, _, _, subscriptions := newResolverWithSubscriptionMock(t)
		source := make(chan entity.Comment)
		close(source)
		subscriptions.EXPECT().SubscribeComments(gomock.Any(), int64(10)).Return((<-chan entity.Comment)(source), nil)

		events, err := resolver.SubscribeComments(context.Background(), 10)
		require.NoError(t, err)
		_, open := <-events
		require.False(t, open)
	})
}

func TestCursorHelpers(t *testing.T) {
	id := int64(42)
	cursor := encodeCursor(&id)
	require.NotNil(t, cursor)
	decoded, err := decodeCursor(cursor)
	require.NoError(t, err)
	require.Equal(t, &id, decoded)
	require.Nil(t, encodeCursor(nil))

	decoded, err = decodeCursor(nil)
	require.NoError(t, err)
	require.Nil(t, decoded)

	decoded, err = decodeCursor(new("MA"))
	require.Nil(t, decoded)
	requireGraphQLErrorCode(t, err, "BAD_USER_INPUT")
}

func newResolverWithMocks(t *testing.T) (*Resolver, *graphqlmock.MockPostService, *graphqlmock.MockCommentService) {
	t.Helper()
	resolver, posts, comments, _ := newResolverWithSubscriptionMock(t)
	return resolver, posts, comments
}

func newResolverWithSubscriptionMock(
	t *testing.T,
) (*Resolver, *graphqlmock.MockPostService, *graphqlmock.MockCommentService, *graphqlmock.MockSubscriptionService) {
	t.Helper()
	controller := gomock.NewController(t)
	posts := graphqlmock.NewMockPostService(controller)
	comments := graphqlmock.NewMockCommentService(controller)
	subscriptions := graphqlmock.NewMockSubscriptionService(controller)
	return NewResolver(posts, comments, subscriptions), posts, comments, subscriptions
}

func requireGraphQLErrorCode(t *testing.T, err error, code string) {
	t.Helper()
	var graphqlError *gqlerror.Error
	require.ErrorAs(t, err, &graphqlError)
	require.Equal(t, code, graphqlError.Extensions["code"])
}
