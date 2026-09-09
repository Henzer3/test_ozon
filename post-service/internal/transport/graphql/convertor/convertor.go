package convertor

import (
	"errors"

	"github.com/Henzer3/test-ozon/post-service/internal/entity"
	"github.com/vektah/gqlparser/v2/gqlerror"
)

func ConvertError(err error) error {
	if err == nil {
		return nil
	}

	switch {
	case errors.Is(err, entity.ErrCommentsDisabled):
		return &gqlerror.Error{
			Message: "comments are disabled for this post",
			Extensions: map[string]any{
				"code": "COMMENTS_DISABLED",
			},
		}

	case errors.Is(err, entity.ErrPostNotFound):
		return &gqlerror.Error{
			Message: "post not found",
			Extensions: map[string]any{
				"code": "POST_NOT_FOUND",
			},
		}

	case errors.Is(err, entity.ErrCommentTooLong):
		return &gqlerror.Error{
			Message: "comment is too long",
			Extensions: map[string]any{
				"code": "COMMENT_TOO_LONG",
			},
		}

	case errors.Is(err, entity.ErrInvalidParent):
		return &gqlerror.Error{
			Message: "invalid parent comment",
			Extensions: map[string]any{
				"code": "INVALID_PARENT",
			},
		}

	case errors.Is(err, entity.ErrCommentNotFound):
		return &gqlerror.Error{
			Message: "comment not found",
			Extensions: map[string]any{
				"code": "COMMENT_NOT_FOUND",
			},
		}

	default:
		return &gqlerror.Error{
			Message: "internal server error",
			Extensions: map[string]any{
				"code": "INTERNAL",
			},
		}
	}
}
