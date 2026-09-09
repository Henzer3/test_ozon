package api_test

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"
)

type postTestGraphQLError struct {
	Message    string         `json:"message"`
	Extensions map[string]any `json:"extensions"`
}

type postTestGraphQLResponse struct {
	Data   json.RawMessage        `json:"data"`
	Errors []postTestGraphQLError `json:"errors"`
}

type postTestPost struct {
	ID              string `json:"id"`
	AuthorID        string `json:"authorID"`
	Title           string `json:"title"`
	Content         string `json:"content"`
	CommentsEnabled bool   `json:"commentsEnabled"`
}

type postTestComment struct {
	ID       string            `json:"id"`
	PostID   string            `json:"postID"`
	ParentID *string           `json:"parentID"`
	AuthorID string            `json:"authorID"`
	Content  string            `json:"content"`
	Replies  []postTestComment `json:"replies"`
}

type postTestPageInfo struct {
	EndCursor   *string `json:"endCursor"`
	HasNextPage bool    `json:"hasNextPage"`
}

type postTestCommentConnection struct {
	Nodes    []postTestComment `json:"nodes"`
	PageInfo postTestPageInfo  `json:"pageInfo"`
}

func TestPostService(t *testing.T) {
	userA := "post-user-a-@example.com"
	userB := "post-user-b-@example.com"
	password := "password123"

	tokenA := postTestRegisterAndLogin(t, userA, password)
	tokenB := postTestRegisterAndLogin(t, userB, password)

	postA := postTestCreatePost(t, tokenA, "First user post", "Content from first user")
	postB := postTestCreatePost(t, tokenB, "Second user post", "Content from second user")

	require.NotEmpty(t, postA.ID)
	require.NotEmpty(t, postB.ID)
	require.NotEqual(t, postA.ID, postB.ID)
	require.NotEqual(t, postA.AuthorID, postB.AuthorID)
	require.True(t, postA.CommentsEnabled)
	require.True(t, postB.CommentsEnabled)

	posts := postTestGetPosts(t, tokenA)
	postsByID := make(map[string]postTestPost, len(posts))
	for _, post := range posts {
		postsByID[post.ID] = post
	}

	require.Contains(t, postsByID, postA.ID)
	require.Contains(t, postsByID, postB.ID)
	require.Equal(t, postA.AuthorID, postsByID[postA.ID].AuthorID)
	require.Equal(t, postB.AuthorID, postsByID[postB.ID].AuthorID)

	rootOne := postTestCreateComment(t, tokenB, postA.ID, nil, "Root comment from second user")
	rootTwo := postTestCreateComment(t, tokenA, postA.ID, nil, "Root comment from first user")
	rootThree := postTestCreateComment(t, tokenB, postA.ID, nil, "Another root comment from second user")
	reply := postTestCreateComment(t, tokenA, postA.ID, &rootOne.ID, "Reply from first user")

	require.Equal(t, postA.ID, rootOne.PostID)
	require.Equal(t, postB.AuthorID, rootOne.AuthorID)
	require.Equal(t, postA.AuthorID, rootTwo.AuthorID)
	require.Equal(t, postB.AuthorID, rootThree.AuthorID)
	require.NotNil(t, reply.ParentID)
	require.Equal(t, rootOne.ID, *reply.ParentID)
	require.Equal(t, postA.AuthorID, reply.AuthorID)

	firstPage := postTestGetCommentsPage(t, tokenA, postA.ID, 2, nil)
	require.Len(t, firstPage.Nodes, 2)
	require.Equal(t, rootThree.ID, firstPage.Nodes[0].ID)
	require.Equal(t, rootTwo.ID, firstPage.Nodes[1].ID)
	require.True(t, firstPage.PageInfo.HasNextPage)
	require.NotNil(t, firstPage.PageInfo.EndCursor)

	secondPage := postTestGetCommentsPage(t, tokenA, postA.ID, 2, firstPage.PageInfo.EndCursor)
	require.Len(t, secondPage.Nodes, 1)
	require.Equal(t, rootOne.ID, secondPage.Nodes[0].ID)
	require.False(t, secondPage.PageInfo.HasNextPage)
	require.NotNil(t, secondPage.PageInfo.EndCursor)
	require.Len(t, secondPage.Nodes[0].Replies, 1)
	require.Equal(t, reply.ID, secondPage.Nodes[0].Replies[0].ID)
	require.NotNil(t, secondPage.Nodes[0].Replies[0].ParentID)
	require.Equal(t, rootOne.ID, *secondPage.Nodes[0].Replies[0].ParentID)

	emptyPage := postTestGetCommentsPage(t, tokenA, postA.ID, 2, secondPage.PageInfo.EndCursor)
	require.Empty(t, emptyPage.Nodes)
	require.False(t, emptyPage.PageInfo.HasNextPage)
	require.Nil(t, emptyPage.PageInfo.EndCursor)
}

func postTestRegisterAndLogin(t *testing.T, name, password string) string {
	t.Helper()

	payload, err := json.Marshal(map[string]string{
		"name":     name,
		"password": password,
	})
	require.NoError(t, err)

	registerRequest, err := http.NewRequest(
		http.MethodPost,
		address+"/api/register",
		bytes.NewReader(payload),
	)
	require.NoError(t, err)
	registerRequest.Header.Set("Content-Type", "application/json")

	registerResponse, err := client.Do(registerRequest)
	require.NoError(t, err)
	_, readErr := io.Copy(io.Discard, registerResponse.Body)
	closeErr := registerResponse.Body.Close()
	require.NoError(t, readErr)
	require.NoError(t, closeErr)
	require.Equal(t, http.StatusOK, registerResponse.StatusCode)

	loginRequest, err := http.NewRequest(
		http.MethodPost,
		address+"/api/login",
		bytes.NewReader(payload),
	)
	require.NoError(t, err)
	loginRequest.Header.Set("Content-Type", "application/json")

	loginResponse, err := client.Do(loginRequest)
	require.NoError(t, err)
	defer loginResponse.Body.Close()
	require.Equal(t, http.StatusOK, loginResponse.StatusCode)

	token, err := io.ReadAll(loginResponse.Body)
	require.NoError(t, err)
	require.NotEmpty(t, token)

	return string(token)
}

func postTestCreatePost(t *testing.T, token, title, content string) postTestPost {
	t.Helper()

	const query = `
		mutation CreatePost($input: CreatePostInput!) {
			createPost(input: $input) {
				id
				authorID
				title
				content
				commentsEnabled
			}
		}
	`

	var data struct {
		CreatePost postTestPost `json:"createPost"`
	}
	postTestDoGraphQL(t, token, query, map[string]any{
		"input": map[string]any{
			"title":           title,
			"content":         content,
			"commentsEnabled": true,
		},
	}, &data)

	return data.CreatePost
}

func postTestGetPosts(t *testing.T, token string) []postTestPost {
	t.Helper()

	const query = `
		query Posts {
			posts {
				id
				authorID
				title
				content
				commentsEnabled
			}
		}
	`

	var data struct {
		Posts []postTestPost `json:"posts"`
	}
	postTestDoGraphQL(t, token, query, nil, &data)

	return data.Posts
}

func postTestCreateComment(
	t *testing.T,
	token string,
	postID string,
	parentID *string,
	content string,
) postTestComment {
	t.Helper()

	const query = `
		mutation CreateComment($input: CreateCommentInput!) {
			createComment(input: $input) {
				id
				postID
				parentID
				authorID
				content
			}
		}
	`

	input := map[string]any{
		"postID":  postID,
		"content": content,
	}
	if parentID != nil {
		input["parentID"] = *parentID
	}

	var data struct {
		CreateComment postTestComment `json:"createComment"`
	}
	postTestDoGraphQL(t, token, query, map[string]any{"input": input}, &data)

	return data.CreateComment
}

func postTestGetCommentsPage(
	t *testing.T,
	token string,
	postID string,
	first int,
	after *string,
) postTestCommentConnection {
	t.Helper()

	const query = `
		query Comments($postID: ID!, $first: Int!, $after: String) {
			post(id: $postID) {
				comments(first: $first, after: $after) {
					nodes {
						id
						postID
						parentID
						authorID
						content
						replies {
							id
							postID
							parentID
							authorID
							content
						}
					}
					pageInfo {
						endCursor
						hasNextPage
					}
				}
			}
		}
	`

	var data struct {
		Post struct {
			Comments postTestCommentConnection `json:"comments"`
		} `json:"post"`
	}
	postTestDoGraphQL(t, token, query, map[string]any{
		"postID": postID,
		"first":  first,
		"after":  after,
	}, &data)

	return data.Post.Comments
}

func postTestDoGraphQL(
	t *testing.T,
	token string,
	query string,
	variables map[string]any,
	data any,
) {
	t.Helper()

	payload, err := json.Marshal(map[string]any{
		"query":     query,
		"variables": variables,
	})
	require.NoError(t, err)

	request, err := http.NewRequest(
		http.MethodPost,
		address+"/query",
		bytes.NewReader(payload),
	)
	require.NoError(t, err)
	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("Authorization", "Token "+token)

	response, err := client.Do(request)
	require.NoError(t, err)
	defer response.Body.Close()

	body, err := io.ReadAll(response.Body)
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, response.StatusCode, "response body: %s", body)

	var result postTestGraphQLResponse
	require.NoError(t, json.Unmarshal(body, &result), "response body: %s", body)
	require.Empty(t, result.Errors, "response body: %s", body)
	require.NoError(t, json.Unmarshal(result.Data, data), "response body: %s", body)
}
