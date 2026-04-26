package gh

import (
	"fmt"
	"time"
)

type Comment struct {
	Author    string
	Body      string
	CreatedAt time.Time
}

type ItemContext struct {
	ContentType    ItemContentType
	RepoNameOwner  string
	Number         int
	Title          string
	Body           string
	URL            string
	State          string
	Author         string
	CreatedAt      time.Time
	Assignees      []string
	Labels         []string
	Comments       []Comment
	CommentsCapped bool
}

const itemContextQuery = `
query ItemContext($itemId: ID!) {
  node(id: $itemId) {
    __typename
    ... on ProjectV2Item {
      content {
        __typename
        ... on Issue {
          number
          title
          body
          url
          state
          createdAt
          author { login }
          repository { nameWithOwner }
          assignees(first: 20) { nodes { login } }
          labels(first: 20) { nodes { name } }
          comments(first: 100) {
            totalCount
            pageInfo { hasNextPage }
            nodes {
              author { login }
              body
              createdAt
            }
          }
        }
        ... on PullRequest {
          number
          title
          body
          url
          state
          createdAt
          author { login }
          repository { nameWithOwner }
          assignees(first: 20) { nodes { login } }
          labels(first: 20) { nodes { name } }
          comments(first: 100) {
            totalCount
            pageInfo { hasNextPage }
            nodes {
              author { login }
              body
              createdAt
            }
          }
        }
        ... on DraftIssue {
          title
          body
          createdAt
          creator { login }
          assignees(first: 20) { nodes { login } }
        }
      }
    }
  }
}
`

func (c *Client) FetchItemContext(itemID string) (*ItemContext, error) {
	type loginNode struct {
		Login string `json:"login"`
	}
	type loginConn struct {
		Nodes []loginNode `json:"nodes"`
	}
	type nameNode struct {
		Name string `json:"name"`
	}
	type nameConn struct {
		Nodes []nameNode `json:"nodes"`
	}
	type commentNode struct {
		Author    *loginNode `json:"author"`
		Body      string     `json:"body"`
		CreatedAt time.Time  `json:"createdAt"`
	}
	type commentsConn struct {
		TotalCount int `json:"totalCount"`
		PageInfo   struct {
			HasNextPage bool `json:"hasNextPage"`
		} `json:"pageInfo"`
		Nodes []commentNode `json:"nodes"`
	}
	type repoNode struct {
		NameWithOwner string `json:"nameWithOwner"`
	}
	type contentNode struct {
		Typename   string       `json:"__typename"`
		Number     int          `json:"number"`
		Title      string       `json:"title"`
		Body       string       `json:"body"`
		URL        string       `json:"url"`
		State      string       `json:"state"`
		CreatedAt  time.Time    `json:"createdAt"`
		Author     *loginNode   `json:"author"`
		Creator    *loginNode   `json:"creator"`
		Repository *repoNode    `json:"repository"`
		Assignees  loginConn    `json:"assignees"`
		Labels     nameConn     `json:"labels"`
		Comments   commentsConn `json:"comments"`
	}
	type itemNode struct {
		Typename string       `json:"__typename"`
		Content  *contentNode `json:"content"`
	}
	type response struct {
		Node *itemNode `json:"node"`
	}

	variables := map[string]any{"itemId": itemID}

	var resp response
	if err := c.gql.Do(itemContextQuery, variables, &resp); err != nil {
		return nil, fmt.Errorf("fetch item context: %w", err)
	}
	if resp.Node == nil || resp.Node.Content == nil {
		return nil, fmt.Errorf("item %q has no content", itemID)
	}

	ctn := resp.Node.Content
	ctx := &ItemContext{
		ContentType: ItemContentType(ctn.Typename),
		Number:      ctn.Number,
		Title:       ctn.Title,
		Body:        ctn.Body,
		URL:         ctn.URL,
		State:       ctn.State,
		CreatedAt:   ctn.CreatedAt,
	}
	switch {
	case ctn.Author != nil:
		ctx.Author = ctn.Author.Login
	case ctn.Creator != nil:
		ctx.Author = ctn.Creator.Login
	}
	if ctn.Repository != nil {
		ctx.RepoNameOwner = ctn.Repository.NameWithOwner
	}
	for _, a := range ctn.Assignees.Nodes {
		ctx.Assignees = append(ctx.Assignees, a.Login)
	}
	for _, l := range ctn.Labels.Nodes {
		ctx.Labels = append(ctx.Labels, l.Name)
	}
	for _, c := range ctn.Comments.Nodes {
		login := ""
		if c.Author != nil {
			login = c.Author.Login
		}
		ctx.Comments = append(ctx.Comments, Comment{
			Author:    login,
			Body:      c.Body,
			CreatedAt: c.CreatedAt,
		})
	}
	ctx.CommentsCapped = ctn.Comments.PageInfo.HasNextPage
	return ctx, nil
}
