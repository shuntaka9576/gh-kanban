package gh

import (
	"errors"
	"fmt"
)

// ErrProjectNotFound is returned when a search yields no project whose title
// exactly matches the requested name.
var ErrProjectNotFound = errors.New("project not found")

const listProjectsQuery = `
query ListProjects($login: String!, $first: Int!, $after: String) {
  %s(login: $login) {
    projectsV2(first: $first, after: $after) {
      pageInfo {
        hasNextPage
        endCursor
      }
      nodes {
        id
        number
        title
        url
      }
    }
  }
}
`

const bootstrapByTitleQuery = `
query BootstrapByTitle($login: String!, $query: String!) {
  %s(login: $login) {
    projectsV2(query: $query, first: 5) {
      nodes {
        id
        number
        title
        url
        fields(first: 50) {
          nodes {
            __typename
            ... on ProjectV2SingleSelectField {
              id
              name
              options {
                id
                name
              }
            }
          }
        }
        items(first: 100) {
          totalCount
          pageInfo {
            hasNextPage
            endCursor
          }
          nodes {
            id
            fieldValues(first: 30) {
              nodes {
                __typename
                ... on ProjectV2ItemFieldSingleSelectValue {
                  optionId
                  field {
                    ... on ProjectV2SingleSelectField {
                      id
                      name
                    }
                  }
                }
              }
            }
            content {
              __typename
              ... on Issue {
                id
                number
                title
                body
                url
                assignees(first: 10) {
                  nodes { login }
                }
                labels(first: 10) {
                  nodes { name }
                }
              }
              ... on PullRequest {
                id
                number
                title
                body
                url
                assignees(first: 10) {
                  nodes { login }
                }
                labels(first: 10) {
                  nodes { name }
                }
              }
              ... on DraftIssue {
                id
                title
                body
                assignees(first: 10) {
                  nodes { login }
                }
              }
            }
          }
        }
      }
    }
  }
}
`

const projectByNumberQuery = `
query ProjectByNumber($login: String!, $number: Int!) {
  %s(login: $login) {
    projectV2(number: $number) {
      id
      title
      number
      url
      fields(first: 50) {
        nodes {
          __typename
          ... on ProjectV2SingleSelectField {
            id
            name
            options {
              id
              name
            }
          }
        }
      }
      items(first: 100) {
        pageInfo {
          hasNextPage
          endCursor
        }
        nodes {
          id
          fieldValues(first: 30) {
            nodes {
              __typename
              ... on ProjectV2ItemFieldSingleSelectValue {
                optionId
                field {
                  ... on ProjectV2SingleSelectField {
                    id
                    name
                  }
                }
              }
            }
          }
          content {
            __typename
            ... on Issue {
              id
              number
              title
              body
              url
              assignees(first: 10) {
                nodes { login }
              }
              labels(first: 10) {
                nodes { name }
              }
            }
            ... on PullRequest {
              id
              number
              title
              body
              url
              assignees(first: 10) {
                nodes { login }
              }
              labels(first: 10) {
                nodes { name }
              }
            }
            ... on DraftIssue {
              id
              title
              body
              assignees(first: 10) {
                nodes { login }
              }
            }
          }
        }
      }
    }
  }
}
`

const itemsPageQuery = `
query ItemsPage($projectId: ID!, $cursor: String) {
  node(id: $projectId) {
    ... on ProjectV2 {
      items(first: 100, after: $cursor) {
        totalCount
        pageInfo {
          hasNextPage
          endCursor
        }
        nodes {
          id
          fieldValues(first: 30) {
            nodes {
              __typename
              ... on ProjectV2ItemFieldSingleSelectValue {
                optionId
                field {
                  ... on ProjectV2SingleSelectField {
                    id
                    name
                  }
                }
              }
            }
          }
          content {
            __typename
            ... on Issue {
              id
              number
              title
              body
              url
              assignees(first: 10) {
                nodes { login }
              }
              labels(first: 10) {
                nodes { name }
              }
            }
            ... on PullRequest {
              id
              number
              title
              body
              url
              assignees(first: 10) {
                nodes { login }
              }
              labels(first: 10) {
                nodes { name }
              }
            }
            ... on DraftIssue {
              id
              title
              body
              assignees(first: 10) {
                nodes { login }
              }
            }
          }
        }
      }
    }
  }
}
`

// Internal raw decode types (shared across queries). Names match the GraphQL
// schema closely so the JSON tags line up.

type rawAssigneesConn struct {
	Nodes []struct {
		Login string `json:"login"`
	} `json:"nodes"`
}

type rawLabelsConn struct {
	Nodes []struct {
		Name string `json:"name"`
	} `json:"nodes"`
}

type rawContent struct {
	Typename  string           `json:"__typename"`
	ID        string           `json:"id"`
	Number    int              `json:"number"`
	Title     string           `json:"title"`
	Body      string           `json:"body"`
	URL       string           `json:"url"`
	Assignees rawAssigneesConn `json:"assignees"`
	Labels    rawLabelsConn    `json:"labels"`
}

type rawFieldValue struct {
	Typename string `json:"__typename"`
	OptionID string `json:"optionId"`
	Field    struct {
		ID   string `json:"id"`
		Name string `json:"name"`
	} `json:"field"`
}

type rawItemNode struct {
	ID          string `json:"id"`
	FieldValues struct {
		Nodes []rawFieldValue `json:"nodes"`
	} `json:"fieldValues"`
	Content *rawContent `json:"content"`
}

type rawItemsConn struct {
	TotalCount int `json:"totalCount"`
	PageInfo   struct {
		HasNextPage bool   `json:"hasNextPage"`
		EndCursor   string `json:"endCursor"`
	} `json:"pageInfo"`
	Nodes []rawItemNode `json:"nodes"`
}

type rawFieldNode struct {
	Typename string               `json:"__typename"`
	ID       string               `json:"id"`
	Name     string               `json:"name"`
	Options  []SingleSelectOption `json:"options"`
}

type rawProjectNode struct {
	ID     string `json:"id"`
	Title  string `json:"title"`
	Number int    `json:"number"`
	URL    string `json:"url"`
	Fields struct {
		Nodes []rawFieldNode `json:"nodes"`
	} `json:"fields"`
	Items rawItemsConn `json:"items"`
}

func extractStatus(fields []rawFieldNode) SingleSelectField {
	for _, f := range fields {
		if f.Typename == "ProjectV2SingleSelectField" && f.Name == "Status" {
			return SingleSelectField{
				ID:      f.ID,
				Name:    f.Name,
				Options: f.Options,
			}
		}
	}
	return SingleSelectField{}
}

func decodeItem(n rawItemNode, statusFieldID string) Item {
	item := Item{ID: n.ID}
	for _, fv := range n.FieldValues.Nodes {
		if fv.Typename == "ProjectV2ItemFieldSingleSelectValue" &&
			fv.Field.ID == statusFieldID {
			item.StatusOptionID = fv.OptionID
		}
	}
	if n.Content != nil {
		item.ContentType = ItemContentType(n.Content.Typename)
		item.Title = n.Content.Title
		item.Body = n.Content.Body
		item.URL = n.Content.URL
		item.Number = n.Content.Number
		for _, a := range n.Content.Assignees.Nodes {
			item.Assignees = append(item.Assignees, a.Login)
		}
		for _, l := range n.Content.Labels.Nodes {
			item.Labels = append(item.Labels, l.Name)
		}
	}
	return item
}

// projectFromRaw turns one rawProjectNode into a Project + the cursor for the
// next items page (empty when there are no more pages) + the total item count
// reported by the connection.
func projectFromRaw(raw rawProjectNode) (*Project, string, int) {
	project := &Project{
		ID:     raw.ID,
		Title:  raw.Title,
		Number: raw.Number,
		URL:    raw.URL,
		Status: extractStatus(raw.Fields.Nodes),
	}
	for _, n := range raw.Items.Nodes {
		project.Items = append(project.Items, decodeItem(n, project.Status.ID))
	}
	next := ""
	if raw.Items.PageInfo.HasNextPage {
		next = raw.Items.PageInfo.EndCursor
	}
	return project, next, raw.Items.TotalCount
}

// ListProjects walks every projectsV2 page for the configured owner. Used as a
// fallback to suggest alternatives when an exact-title search misses.
func (c *Client) ListProjects() ([]ProjectSummary, error) {
	type projectNode struct {
		ID     string `json:"id"`
		Number int    `json:"number"`
		Title  string `json:"title"`
		URL    string `json:"url"`
	}
	type projectsV2 struct {
		PageInfo struct {
			HasNextPage bool   `json:"hasNextPage"`
			EndCursor   string `json:"endCursor"`
		} `json:"pageInfo"`
		Nodes []projectNode `json:"nodes"`
	}
	type ownerWrap struct {
		User *struct {
			ProjectsV2 projectsV2 `json:"projectsV2"`
		} `json:"user,omitempty"`
		Organization *struct {
			ProjectsV2 projectsV2 `json:"projectsV2"`
		} `json:"organization,omitempty"`
	}

	query := fmt.Sprintf(listProjectsQuery, c.ownerSelector())

	var (
		projects []ProjectSummary
		after    *string
	)
	for {
		variables := map[string]any{
			"login": c.Login,
			"first": 100,
			"after": after,
		}

		var resp ownerWrap
		if err := c.gql.Do(query, variables, &resp); err != nil {
			return nil, fmt.Errorf("list projectsV2: %w", err)
		}

		var page projectsV2
		switch c.ClientType {
		case ClientTypeUser:
			if resp.User == nil {
				return nil, fmt.Errorf("user %q not found", c.Login)
			}
			page = resp.User.ProjectsV2
		case ClientTypeOrganization:
			if resp.Organization == nil {
				return nil, fmt.Errorf("organization %q not found", c.Login)
			}
			page = resp.Organization.ProjectsV2
		}

		for _, n := range page.Nodes {
			projects = append(projects, ProjectSummary{
				ID:     n.ID,
				Number: n.Number,
				Title:  n.Title,
				URL:    n.URL,
			})
		}

		if !page.PageInfo.HasNextPage {
			break
		}
		cursor := page.PageInfo.EndCursor
		after = &cursor
	}

	return projects, nil
}

// bootstrapByTitle resolves a project by exact-title match AND returns its
// fields + first 100 items in the same single GraphQL request, halving the
// startup latency vs. doing search and board fetch separately.
func (c *Client) bootstrapByTitle(title string) (*BootstrapResult, error) {
	type projectsV2 struct {
		Nodes []rawProjectNode `json:"nodes"`
	}
	type ownerWrap struct {
		User *struct {
			ProjectsV2 projectsV2 `json:"projectsV2"`
		} `json:"user,omitempty"`
		Organization *struct {
			ProjectsV2 projectsV2 `json:"projectsV2"`
		} `json:"organization,omitempty"`
	}

	query := fmt.Sprintf(bootstrapByTitleQuery, c.ownerSelector())
	variables := map[string]any{
		"login": c.Login,
		"query": title,
	}

	var resp ownerWrap
	if err := c.gql.Do(query, variables, &resp); err != nil {
		return nil, fmt.Errorf("bootstrap by title: %w", err)
	}

	var nodes []rawProjectNode
	switch c.ClientType {
	case ClientTypeUser:
		if resp.User == nil {
			return nil, fmt.Errorf("user %q not found", c.Login)
		}
		nodes = resp.User.ProjectsV2.Nodes
	case ClientTypeOrganization:
		if resp.Organization == nil {
			return nil, fmt.Errorf("organization %q not found", c.Login)
		}
		nodes = resp.Organization.ProjectsV2.Nodes
	}

	for _, n := range nodes {
		if n.Title == title {
			project, next, total := projectFromRaw(n)
			return &BootstrapResult{Project: project, NextCursor: next, TotalItems: total}, nil
		}
	}

	return nil, fmt.Errorf("%w: no project with exact title %q (server-side search returned %d candidate(s))", ErrProjectNotFound, title, len(nodes))
}

// BootstrapBySpec resolves a project (by Number when set, otherwise by exact
// Title) and returns the first page of items in the same async path.
// Designed to be called from a tea.Cmd after the TUI is already alive.
func (c *Client) BootstrapBySpec(spec ProjectSpec) (*BootstrapResult, error) {
	if spec.Number > 0 {
		return c.bootstrapByNumber(spec.Number)
	}
	if spec.Title == "" {
		return nil, errors.New("ProjectSpec requires Title or Number")
	}
	return c.bootstrapByTitle(spec.Title)
}

func (c *Client) bootstrapByNumber(number int) (*BootstrapResult, error) {
	type ownerWrap struct {
		User *struct {
			ProjectV2 *rawProjectNode `json:"projectV2"`
		} `json:"user,omitempty"`
		Organization *struct {
			ProjectV2 *rawProjectNode `json:"projectV2"`
		} `json:"organization,omitempty"`
	}

	query := fmt.Sprintf(projectByNumberQuery, c.ownerSelector())
	variables := map[string]any{
		"login":  c.Login,
		"number": number,
	}

	var resp ownerWrap
	if err := c.gql.Do(query, variables, &resp); err != nil {
		return nil, fmt.Errorf("bootstrap by number: %w", err)
	}

	var raw *rawProjectNode
	switch c.ClientType {
	case ClientTypeUser:
		if resp.User == nil {
			return nil, fmt.Errorf("user %q not found", c.Login)
		}
		raw = resp.User.ProjectV2
	case ClientTypeOrganization:
		if resp.Organization == nil {
			return nil, fmt.Errorf("organization %q not found", c.Login)
		}
		raw = resp.Organization.ProjectV2
	}
	if raw == nil {
		return nil, fmt.Errorf("%w: no project #%d under %q", ErrProjectNotFound, number, c.Login)
	}

	project, next, total := projectFromRaw(*raw)
	return &BootstrapResult{Project: project, NextCursor: next, TotalItems: total}, nil
}

// FetchItemsPage gets one additional page of items for an already-known
// project, decoding via the previously-discovered Status field id so that
// per-item Status is populated correctly.
func (c *Client) FetchItemsPage(projectID, statusFieldID, cursor string) (*ItemsPage, error) {
	variables := map[string]any{
		"projectId": projectID,
		"cursor":    cursor,
	}
	var resp struct {
		Node struct {
			Items rawItemsConn `json:"items"`
		} `json:"node"`
	}
	if err := c.gql.Do(itemsPageQuery, variables, &resp); err != nil {
		return nil, fmt.Errorf("fetch items page: %w", err)
	}

	page := &ItemsPage{TotalItems: resp.Node.Items.TotalCount}
	for _, n := range resp.Node.Items.Nodes {
		page.Items = append(page.Items, decodeItem(n, statusFieldID))
	}
	if resp.Node.Items.PageInfo.HasNextPage {
		page.NextCursor = resp.Node.Items.PageInfo.EndCursor
	}
	return page, nil
}
