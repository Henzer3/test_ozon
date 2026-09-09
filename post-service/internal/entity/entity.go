package entity

type Post struct {
	ID              int64
	AuthorID        int64
	Title           string
	Content         string
	CommentsEnabled bool
	Comments        []Comment
}

type Comment struct {
	ID       int64
	PostID   int64
	ParentID *int64
	AuthorID int64
	Content  string
	Replies  []Comment
}

type CommentPage struct {
	Comments    []Comment
	EndCursor   *int64
	HasNextPage bool
}
