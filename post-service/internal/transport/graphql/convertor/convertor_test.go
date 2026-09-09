package convertor

import (
	"errors"
	"fmt"
	"testing"

	"github.com/Henzer3/test-ozon/post-service/internal/entity"
	"github.com/stretchr/testify/require"
	"github.com/vektah/gqlparser/v2/gqlerror"
)

func TestConvertError(t *testing.T) {
	tests := []struct {
		name    string
		err     error
		message string
		code    string
	}{
		{name: "comments disabled", err: entity.ErrCommentsDisabled, message: "comments are disabled for this post", code: "COMMENTS_DISABLED"},
		{name: "post not found", err: entity.ErrPostNotFound, message: "post not found", code: "POST_NOT_FOUND"},
		{name: "comment too long", err: entity.ErrCommentTooLong, message: "comment is too long", code: "COMMENT_TOO_LONG"},
		{name: "invalid parent", err: entity.ErrInvalidParent, message: "invalid parent comment", code: "INVALID_PARENT"},
		{name: "comment not found", err: entity.ErrCommentNotFound, message: "comment not found", code: "COMMENT_NOT_FOUND"},
		{name: "wrapped domain error", err: fmt.Errorf("wrapped: %w", entity.ErrPostNotFound), message: "post not found", code: "POST_NOT_FOUND"},
		{name: "unknown error", err: errors.New("database failed"), message: "internal server error", code: "INTERNAL"},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			converted := ConvertError(test.err)

			var graphqlError *gqlerror.Error
			require.ErrorAs(t, converted, &graphqlError)
			require.Equal(t, test.message, graphqlError.Message)
			require.Equal(t, test.code, graphqlError.Extensions["code"])
		})
	}
}

func TestConvertErrorNil(t *testing.T) {
	require.NoError(t, ConvertError(nil))
}
