// Copyright © 2019 Prometheus Team
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package github

import (
	"context"
	"os"

	"github.com/google/go-github/v25/github"
	"github.com/pkg/errors"
	"golang.org/x/oauth2"
)

// ReadAll iterates over GitHub pages.
func ReadAll(list func(*github.ListOptions) (*github.Response, error)) error {
	opt := github.ListOptions{PerPage: 10}
	for {
		resp, err := list(&opt)
		if err != nil {
			return err
		}
		if resp == nil || resp.NextPage == 0 {
			return nil
		}
		opt.Page = resp.NextPage
	}
}

// TokenVarName is the name of the environment variable containing the token.
const TokenVarName = "GITHUB_TOKEN"

// NewClient returns a new GitHub client with an authentication token read from TokenVarName.
func NewClient(ctx context.Context) (*github.Client, error) {
	token := os.Getenv(TokenVarName)
	if len(token) == 0 {
		return nil, errors.Errorf("%s not defined", TokenVarName)
	}

	c := github.NewClient(
		oauth2.NewClient(
			ctx,
			oauth2.StaticTokenSource(
				&oauth2.Token{
					AccessToken: token,
				},
			),
		),
	)
	_, _, err := c.Zen(ctx)
	if err != nil {
		return nil, err
	}
	return c, nil
}
