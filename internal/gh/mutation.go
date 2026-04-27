package gh

import "fmt"

const updateSingleSelectMutation = `
mutation UpdateStatus($projectId: ID!, $itemId: ID!, $fieldId: ID!, $optionId: String!) {
  updateProjectV2ItemFieldValue(input: {
    projectId: $projectId
    itemId: $itemId
    fieldId: $fieldId
    value: { singleSelectOptionId: $optionId }
  }) {
    projectV2Item { id }
  }
}
`

func (c *Client) UpdateItemStatus(projectID, itemID, fieldID, optionID string) error {
	variables := map[string]any{
		"projectId": projectID,
		"itemId":    itemID,
		"fieldId":   fieldID,
		"optionId":  optionID,
	}
	var resp struct {
		UpdateProjectV2ItemFieldValue struct {
			ProjectV2Item struct {
				ID string `json:"id"`
			} `json:"projectV2Item"`
		} `json:"updateProjectV2ItemFieldValue"`
	}
	if err := c.gql.Do(updateSingleSelectMutation, variables, &resp); err != nil {
		return fmt.Errorf("update item status: %w", err)
	}
	return nil
}
