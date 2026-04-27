package gh

import (
	"fmt"

	"github.com/cli/go-gh/v2/pkg/api"
	"github.com/cli/go-gh/v2/pkg/repository"
)

func DetectOwnerFromCurrentRepo() (InitParams, error) {
	repo, err := repository.Current()
	if err != nil {
		return InitParams{}, fmt.Errorf("detect current repository: %w", err)
	}

	gql, err := api.DefaultGraphQLClient()
	if err != nil {
		return InitParams{}, fmt.Errorf("graphql client: %w", err)
	}

	const q = `query($login: String!) {
	  repositoryOwner(login: $login) { __typename }
	}`
	var resp struct {
		RepositoryOwner *struct {
			Typename string `json:"__typename"`
		} `json:"repositoryOwner"`
	}
	if err := gql.Do(q, map[string]any{"login": repo.Owner}, &resp); err != nil {
		return InitParams{}, fmt.Errorf("resolve owner type for %q: %w", repo.Owner, err)
	}
	if resp.RepositoryOwner == nil {
		return InitParams{}, fmt.Errorf("owner %q not found", repo.Owner)
	}

	switch resp.RepositoryOwner.Typename {
	case "Organization":
		return InitParams{OrgLogin: repo.Owner}, nil
	case "User":
		return InitParams{UserLogin: repo.Owner}, nil
	default:
		return InitParams{}, fmt.Errorf("unsupported owner type %q for %q", resp.RepositoryOwner.Typename, repo.Owner)
	}
}
