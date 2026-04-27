package gh

import (
	"errors"
	"fmt"

	"github.com/cli/go-gh/v2/pkg/api"
)

type ClientType string

const (
	ClientTypeUser         ClientType = "user"
	ClientTypeOrganization ClientType = "organization"
)

var ErrInvalidClientType = errors.New("specify exactly one of --user or --org")

type Client struct {
	gql        *api.GraphQLClient
	ClientType ClientType
	Login      string
}

type InitParams struct {
	UserLogin string
	OrgLogin  string
}

func NewClient(p InitParams) (*Client, error) {
	if p.UserLogin != "" && p.OrgLogin != "" {
		return nil, ErrInvalidClientType
	}
	if p.UserLogin == "" && p.OrgLogin == "" {
		return nil, ErrInvalidClientType
	}

	gql, err := api.DefaultGraphQLClient()
	if err != nil {
		return nil, fmt.Errorf("graphql client: %w", err)
	}

	c := &Client{gql: gql}
	if p.UserLogin != "" {
		c.ClientType = ClientTypeUser
		c.Login = p.UserLogin
	} else {
		c.ClientType = ClientTypeOrganization
		c.Login = p.OrgLogin
	}
	return c, nil
}

func (c *Client) ownerSelector() string {
	if c.ClientType == ClientTypeOrganization {
		return "organization"
	}
	return "user"
}
