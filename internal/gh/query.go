package gh

import "fmt"

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

const projectBoardQuery = `
query ProjectBoard($projectId: ID!, $itemsAfter: String) {
  node(id: $projectId) {
    ... on ProjectV2 {
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
      items(first: 100, after: $itemsAfter) {
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

func (c *Client) FetchProjectBoard(projectID string) (*Project, error) {
	type fieldNode struct {
		Typename string               `json:"__typename"`
		ID       string               `json:"id"`
		Name     string               `json:"name"`
		Options  []SingleSelectOption `json:"options"`
	}
	type assigneesConn struct {
		Nodes []struct {
			Login string `json:"login"`
		} `json:"nodes"`
	}
	type labelsConn struct {
		Nodes []struct {
			Name string `json:"name"`
		} `json:"nodes"`
	}
	type contentNode struct {
		Typename  string        `json:"__typename"`
		ID        string        `json:"id"`
		Number    int           `json:"number"`
		Title     string        `json:"title"`
		Body      string        `json:"body"`
		URL       string        `json:"url"`
		Assignees assigneesConn `json:"assignees"`
		Labels    labelsConn    `json:"labels"`
	}
	type fieldValueNode struct {
		Typename string `json:"__typename"`
		OptionID string `json:"optionId"`
		Field    struct {
			ID   string `json:"id"`
			Name string `json:"name"`
		} `json:"field"`
	}
	type itemNode struct {
		ID          string `json:"id"`
		FieldValues struct {
			Nodes []fieldValueNode `json:"nodes"`
		} `json:"fieldValues"`
		Content *contentNode `json:"content"`
	}
	type itemsConn struct {
		PageInfo struct {
			HasNextPage bool   `json:"hasNextPage"`
			EndCursor   string `json:"endCursor"`
		} `json:"pageInfo"`
		Nodes []itemNode `json:"nodes"`
	}
	type projectNode struct {
		ID     string `json:"id"`
		Title  string `json:"title"`
		Number int    `json:"number"`
		URL    string `json:"url"`
		Fields struct {
			Nodes []fieldNode `json:"nodes"`
		} `json:"fields"`
		Items itemsConn `json:"items"`
	}
	type response struct {
		Node projectNode `json:"node"`
	}

	project := &Project{ID: projectID}

	var after *string
	for {
		variables := map[string]any{
			"projectId":  projectID,
			"itemsAfter": after,
		}

		var resp response
		if err := c.gql.Do(projectBoardQuery, variables, &resp); err != nil {
			return nil, fmt.Errorf("fetch project board: %w", err)
		}

		if after == nil {
			project.Title = resp.Node.Title
			project.Number = resp.Node.Number
			project.URL = resp.Node.URL
			for _, f := range resp.Node.Fields.Nodes {
				if f.Typename == "ProjectV2SingleSelectField" && f.Name == "Status" {
					project.Status = SingleSelectField{
						ID:      f.ID,
						Name:    f.Name,
						Options: f.Options,
					}
					break
				}
			}
		}

		for _, n := range resp.Node.Items.Nodes {
			item := Item{ID: n.ID}

			for _, fv := range n.FieldValues.Nodes {
				if fv.Typename == "ProjectV2ItemFieldSingleSelectValue" &&
					fv.Field.ID == project.Status.ID {
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

			project.Items = append(project.Items, item)
		}

		if !resp.Node.Items.PageInfo.HasNextPage {
			break
		}
		cursor := resp.Node.Items.PageInfo.EndCursor
		after = &cursor
	}

	return project, nil
}
