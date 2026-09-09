package entity

import "errors"

var (
	ErrCommentsDisabled = errors.New("comments are disabled")
	ErrPostNotFound     = errors.New("post not found")
	ErrCommentTooLong   = errors.New("comment is too long")
	ErrInvalidParent    = errors.New("invalid parent comment")
	ErrCommentNotFound  = errors.New("comment not found")
)
