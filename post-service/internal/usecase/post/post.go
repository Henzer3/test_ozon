package post

import (
	"context"
	"errors"
	"log/slog"

	"github.com/Henzer3/test-ozon/post-service/internal/entity"
	"github.com/Henzer3/test-ozon/post-service/internal/port"
)

type postRepository interface {
	CreatePost(ctx context.Context, input port.CreatePostInput) (entity.Post, error)
	GetPosts(ctx context.Context) ([]entity.Post, error)
	GetPost(ctx context.Context, id int64) (entity.Post, error)
}

type commentRepository interface {
	CreateComment(ctx context.Context, input port.CreateCommentInput) (entity.Comment, error)
	Comments(ctx context.Context, input port.CommentPageInput) (entity.CommentPage, error)
	GetComment(ctx context.Context, id int64) (entity.Comment, error)
}

type Service struct {
	log               *slog.Logger
	postRepository    postRepository
	commentRepository commentRepository
}

func New(log *slog.Logger, postRepository postRepository, commentRepository commentRepository) *Service {
	return &Service{log: log, postRepository: postRepository, commentRepository: commentRepository}
}

func (s *Service) CreatePost(ctx context.Context, input port.CreatePostInput) (entity.Post, error) {
	post, err := s.postRepository.CreatePost(ctx, input)
	if err != nil {
		s.log.Error("failed to create post", "error", err)
		return entity.Post{}, err
	}

	return post, nil
}

func (s *Service) Posts(ctx context.Context) ([]entity.Post, error) {
	posts, err := s.postRepository.GetPosts(ctx)
	if err != nil {
		s.log.Error("failed to get posts", "error", err)
		return []entity.Post{}, err
	}

	return posts, nil
}

func (s *Service) Post(ctx context.Context, id int64) (entity.Post, error) {
	post, err := s.postRepository.GetPost(ctx, id)
	if err != nil {
		if errors.Is(err, entity.ErrPostNotFound) {
			return entity.Post{}, err
		}
		s.log.Error("failed to get post", "error", err)
		return entity.Post{}, err
	}

	return post, nil
}

func (s *Service) CreateComment(ctx context.Context, input port.CreateCommentInput) (entity.Comment, error) {
	post, err := s.postRepository.GetPost(ctx, input.PostID)
	if err != nil {
		if errors.Is(err, entity.ErrPostNotFound) {
			return entity.Comment{}, err
		}
		s.log.Error("failed to get post", "error", err)
		return entity.Comment{}, err
	}

	if !post.CommentsEnabled {
		return entity.Comment{}, entity.ErrCommentsDisabled
	}

	if input.ParentID != nil {
		parent, err := s.commentRepository.GetComment(ctx, *input.ParentID)
		if err != nil {
			if errors.Is(err, entity.ErrCommentNotFound) {
				return entity.Comment{}, err
			}
			s.log.Error("failed to get comment", "error", err)
			return entity.Comment{}, err
		}

		if parent.PostID != post.ID {
			return entity.Comment{}, entity.ErrInvalidParent
		}

	}

	comment, err := s.commentRepository.CreateComment(ctx, input)

	if err != nil {
		s.log.Error("failed to create comment", "error", err)
		return entity.Comment{}, err
	}

	return comment, nil
}

func (s *Service) Comments(ctx context.Context, input port.CommentPageInput) (entity.CommentPage, error) {
	_, err := s.postRepository.GetPost(ctx, input.PostID)
	if err != nil {
		if errors.Is(err, entity.ErrPostNotFound) {
			return entity.CommentPage{}, err
		}
		s.log.Error("failed to get post", "error", err)
		return entity.CommentPage{}, err
	}

	commentPage, err := s.commentRepository.Comments(ctx, input)
	if err != nil {
		s.log.Error("failed to get comments", "error", err)
		return entity.CommentPage{}, err
	}

	return commentPage, nil
}
