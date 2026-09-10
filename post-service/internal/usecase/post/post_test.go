package post

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"testing"

	"github.com/Henzer3/test-ozon/post-service/internal/entity"
	"github.com/Henzer3/test-ozon/post-service/internal/port"
	postmock "github.com/Henzer3/test-ozon/post-service/internal/usecase/post/mock"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

func TestCreatePost(t *testing.T) {
	ctx := context.Background()
	input := port.CreatePostInput{AuthorID: 1, Title: "title", Content: "content", CommentsEnabled: true}
	want := entity.Post{ID: 10, AuthorID: 1, Title: "title", Content: "content", CommentsEnabled: true}

	t.Run("success", func(t *testing.T) {
		service, posts, _ := newServiceWithMocks(t)
		posts.EXPECT().CreatePost(ctx, input).Return(want, nil)

		got, err := service.CreatePost(ctx, input)

		require.NoError(t, err)
		require.Equal(t, want, got)
	})

	t.Run("repository error", func(t *testing.T) {
		service, posts, _ := newServiceWithMocks(t)
		wantErr := errors.New("create post failed")
		posts.EXPECT().CreatePost(ctx, input).Return(entity.Post{}, wantErr)

		got, err := service.CreatePost(ctx, input)

		require.ErrorIs(t, err, wantErr)
		require.Equal(t, entity.Post{}, got)
	})
}

func TestPosts(t *testing.T) {
	ctx := context.Background()
	want := []entity.Post{{ID: 2}, {ID: 1}}

	t.Run("success", func(t *testing.T) {
		service, posts, _ := newServiceWithMocks(t)
		posts.EXPECT().GetPosts(ctx).Return(want, nil)

		got, err := service.Posts(ctx)

		require.NoError(t, err)
		require.Equal(t, want, got)
	})

	t.Run("repository error", func(t *testing.T) {
		service, posts, _ := newServiceWithMocks(t)
		wantErr := errors.New("get posts failed")
		posts.EXPECT().GetPosts(ctx).Return(nil, wantErr)

		got, err := service.Posts(ctx)

		require.ErrorIs(t, err, wantErr)
		require.Empty(t, got)
	})
}

func TestPost(t *testing.T) {
	ctx := context.Background()
	want := entity.Post{ID: 7, Title: "post"}

	t.Run("success", func(t *testing.T) {
		service, posts, _ := newServiceWithMocks(t)
		posts.EXPECT().GetPost(ctx, int64(7)).Return(want, nil)

		got, err := service.Post(ctx, 7)

		require.NoError(t, err)
		require.Equal(t, want, got)
	})

	for _, test := range []struct {
		name string
		err  error
	}{
		{name: "post not found", err: entity.ErrPostNotFound},
		{name: "repository error", err: errors.New("get post failed")},
	} {
		t.Run(test.name, func(t *testing.T) {
			service, posts, _ := newServiceWithMocks(t)
			posts.EXPECT().GetPost(ctx, int64(7)).Return(entity.Post{}, test.err)

			got, err := service.Post(ctx, 7)

			require.ErrorIs(t, err, test.err)
			require.Equal(t, entity.Post{}, got)
		})
	}
}

func TestCreateComment(t *testing.T) {
	ctx := context.Background()
	input := port.CreateCommentInput{AuthorID: 2, PostID: 10, Content: "comment"}
	post := entity.Post{ID: 10, CommentsEnabled: true}
	want := entity.Comment{ID: 20, PostID: 10, AuthorID: 2, Content: "comment"}

	t.Run("success without parent", func(t *testing.T) {
		service, posts, comments := newServiceWithMocks(t)
		posts.EXPECT().GetPost(ctx, input.PostID).Return(post, nil)
		comments.EXPECT().CreateComment(ctx, input).Return(want, nil)

		got, err := service.CreateComment(ctx, input)

		require.NoError(t, err)
		require.Equal(t, want, got)
	})

	t.Run("success with parent", func(t *testing.T) {
		service, posts, comments := newServiceWithMocks(t)
		parentID := int64(15)
		inputWithParent := input
		inputWithParent.ParentID = &parentID
		wantWithParent := want
		wantWithParent.ParentID = &parentID

		posts.EXPECT().GetPost(ctx, input.PostID).Return(post, nil)
		comments.EXPECT().GetComment(ctx, parentID).Return(entity.Comment{ID: parentID, PostID: post.ID}, nil)
		comments.EXPECT().CreateComment(ctx, inputWithParent).Return(wantWithParent, nil)

		got, err := service.CreateComment(ctx, inputWithParent)

		require.NoError(t, err)
		require.Equal(t, wantWithParent, got)
	})

	t.Run("post not found", func(t *testing.T) {
		service, posts, _ := newServiceWithMocks(t)
		posts.EXPECT().GetPost(ctx, input.PostID).Return(entity.Post{}, entity.ErrPostNotFound)

		got, err := service.CreateComment(ctx, input)

		require.ErrorIs(t, err, entity.ErrPostNotFound)
		require.Equal(t, entity.Comment{}, got)
	})

	t.Run("get post error", func(t *testing.T) {
		service, posts, _ := newServiceWithMocks(t)
		wantErr := errors.New("get post failed")
		posts.EXPECT().GetPost(ctx, input.PostID).Return(entity.Post{}, wantErr)

		_, err := service.CreateComment(ctx, input)

		require.ErrorIs(t, err, wantErr)
	})

	t.Run("comments disabled", func(t *testing.T) {
		service, posts, _ := newServiceWithMocks(t)
		posts.EXPECT().GetPost(ctx, input.PostID).Return(entity.Post{ID: input.PostID, CommentsEnabled: false}, nil)

		_, err := service.CreateComment(ctx, input)

		require.ErrorIs(t, err, entity.ErrCommentsDisabled)
	})

	t.Run("parent not found", func(t *testing.T) {
		service, posts, comments := newServiceWithMocks(t)
		parentID := int64(15)
		inputWithParent := input
		inputWithParent.ParentID = &parentID
		posts.EXPECT().GetPost(ctx, input.PostID).Return(post, nil)
		comments.EXPECT().GetComment(ctx, parentID).Return(entity.Comment{}, entity.ErrCommentNotFound)

		_, err := service.CreateComment(ctx, inputWithParent)

		require.ErrorIs(t, err, entity.ErrCommentNotFound)
	})

	t.Run("get parent error", func(t *testing.T) {
		service, posts, comments := newServiceWithMocks(t)
		parentID := int64(15)
		inputWithParent := input
		inputWithParent.ParentID = &parentID
		wantErr := errors.New("get parent failed")
		posts.EXPECT().GetPost(ctx, input.PostID).Return(post, nil)
		comments.EXPECT().GetComment(ctx, parentID).Return(entity.Comment{}, wantErr)

		_, err := service.CreateComment(ctx, inputWithParent)

		require.ErrorIs(t, err, wantErr)
	})

	t.Run("parent belongs to another post", func(t *testing.T) {
		service, posts, comments := newServiceWithMocks(t)
		parentID := int64(15)
		inputWithParent := input
		inputWithParent.ParentID = &parentID
		posts.EXPECT().GetPost(ctx, input.PostID).Return(post, nil)
		comments.EXPECT().GetComment(ctx, parentID).Return(entity.Comment{ID: parentID, PostID: 999}, nil)

		_, err := service.CreateComment(ctx, inputWithParent)

		require.ErrorIs(t, err, entity.ErrInvalidParent)
	})

	t.Run("create comment error", func(t *testing.T) {
		service, posts, comments := newServiceWithMocks(t)
		wantErr := errors.New("create comment failed")
		posts.EXPECT().GetPost(ctx, input.PostID).Return(post, nil)
		comments.EXPECT().CreateComment(ctx, input).Return(entity.Comment{}, wantErr)

		_, err := service.CreateComment(ctx, input)

		require.ErrorIs(t, err, wantErr)
	})
}

func TestCreateCommentPublishesAfterSave(t *testing.T) {
	ctx := context.Background()
	input := port.CreateCommentInput{AuthorID: 2, PostID: 10, Content: "comment"}
	post := entity.Post{ID: 10, CommentsEnabled: true}
	comment := entity.Comment{ID: 20, PostID: 10, AuthorID: 2, Content: "comment"}
	service, posts, comments, broker := newServiceWithBrokerMock(t)

	gomock.InOrder(
		posts.EXPECT().GetPost(ctx, input.PostID).Return(post, nil),
		comments.EXPECT().CreateComment(ctx, input).Return(comment, nil),
		broker.EXPECT().Publish(comment),
	)

	result, err := service.CreateComment(ctx, input)

	require.NoError(t, err)
	require.Equal(t, comment, result)
}

func TestSubscribeComments(t *testing.T) {
	ctx := context.Background()
	const postID int64 = 10

	t.Run("success", func(t *testing.T) {
		service, posts, _, broker := newServiceWithBrokerMock(t)
		events := make(chan entity.Comment)
		posts.EXPECT().GetPost(ctx, postID).Return(entity.Post{ID: postID, CommentsEnabled: true}, nil)
		broker.EXPECT().Subscribe(ctx, postID).Return((<-chan entity.Comment)(events))

		result, err := service.SubscribeComments(ctx, postID)

		require.NoError(t, err)
		require.Equal(t, (<-chan entity.Comment)(events), result)
	})

	t.Run("post not found", func(t *testing.T) {
		service, posts, _, _ := newServiceWithBrokerMock(t)
		posts.EXPECT().GetPost(ctx, postID).Return(entity.Post{}, entity.ErrPostNotFound)

		result, err := service.SubscribeComments(ctx, postID)

		require.ErrorIs(t, err, entity.ErrPostNotFound)
		require.Nil(t, result)
	})

	t.Run("comments disabled", func(t *testing.T) {
		service, posts, _, _ := newServiceWithBrokerMock(t)
		posts.EXPECT().GetPost(ctx, postID).Return(entity.Post{ID: postID, CommentsEnabled: false}, nil)

		result, err := service.SubscribeComments(ctx, postID)

		require.ErrorIs(t, err, entity.ErrCommentsDisabled)
		require.Nil(t, result)
	})

	t.Run("repository error", func(t *testing.T) {
		service, posts, _, _ := newServiceWithBrokerMock(t)
		wantErr := errors.New("get post failed")
		posts.EXPECT().GetPost(ctx, postID).Return(entity.Post{}, wantErr)

		result, err := service.SubscribeComments(ctx, postID)

		require.ErrorIs(t, err, wantErr)
		require.Nil(t, result)
	})
}

func TestComments(t *testing.T) {
	ctx := context.Background()
	afterID := int64(30)
	input := port.CommentPageInput{PostID: 10, Limit: 2, AfterID: &afterID}
	var num int64 = 28
	want := entity.CommentPage{
		Comments:    []entity.Comment{{ID: 29}, {ID: 28}},
		EndCursor:   &num,
		HasNextPage: true,
	}

	t.Run("success", func(t *testing.T) {
		service, posts, comments := newServiceWithMocks(t)
		posts.EXPECT().GetPost(ctx, input.PostID).Return(entity.Post{ID: input.PostID}, nil)
		comments.EXPECT().Comments(ctx, input).Return(want, nil)

		got, err := service.Comments(ctx, input)

		require.NoError(t, err)
		require.Equal(t, want, got)
	})

	t.Run("post not found", func(t *testing.T) {
		service, posts, _ := newServiceWithMocks(t)
		posts.EXPECT().GetPost(ctx, input.PostID).Return(entity.Post{}, entity.ErrPostNotFound)

		got, err := service.Comments(ctx, input)

		require.ErrorIs(t, err, entity.ErrPostNotFound)
		require.Equal(t, entity.CommentPage{}, got)
	})

	t.Run("get post error", func(t *testing.T) {
		service, posts, _ := newServiceWithMocks(t)
		wantErr := errors.New("get post failed")
		posts.EXPECT().GetPost(ctx, input.PostID).Return(entity.Post{}, wantErr)

		_, err := service.Comments(ctx, input)

		require.ErrorIs(t, err, wantErr)
	})

	t.Run("get comments error", func(t *testing.T) {
		service, posts, comments := newServiceWithMocks(t)
		wantErr := errors.New("get comments failed")
		posts.EXPECT().GetPost(ctx, input.PostID).Return(entity.Post{ID: input.PostID}, nil)
		comments.EXPECT().Comments(ctx, input).Return(entity.CommentPage{}, wantErr)

		_, err := service.Comments(ctx, input)

		require.ErrorIs(t, err, wantErr)
	})
}

func newServiceWithMocks(t *testing.T) (*Service, *postmock.MockpostRepository, *postmock.MockcommentRepository) {
	t.Helper()
	service, posts, comments, broker := newServiceWithBrokerMock(t)
	broker.EXPECT().Publish(gomock.Any()).AnyTimes()

	return service, posts, comments
}

func newServiceWithBrokerMock(
	t *testing.T,
) (*Service, *postmock.MockpostRepository, *postmock.MockcommentRepository, *postmock.MockcommentBroker) {
	t.Helper()

	controller := gomock.NewController(t)
	posts := postmock.NewMockpostRepository(controller)
	comments := postmock.NewMockcommentRepository(controller)
	broker := postmock.NewMockcommentBroker(controller)
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))

	return New(logger, posts, comments, broker), posts, comments, broker
}
