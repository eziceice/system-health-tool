package github

import (
	"context"
	"fmt"
	"github.com/google/go-github/v41/github"
	"go.uber.org/zap"
	"golang.org/x/oauth2"
	"log"
	"strings"
	"time"
)

type EnterpriseClient struct {
	ghClient   *github.Client
	minCommits int
}

func NewEnterpriseClient(ctx context.Context, baseURL string, githubToken string, minCommits int) EnterpriseClient {
	token := oauth2.StaticTokenSource(&oauth2.Token{AccessToken: githubToken})
	httpClient := oauth2.NewClient(ctx, token)
	client, err := github.NewEnterpriseClient(baseURL, baseURL, httpClient)
	if err != nil {
		zap.S().Errorf("cannot create GitHub client %v", err)
	}
	return EnterpriseClient{ghClient: client, minCommits: minCommits}
}

func (client *EnterpriseClient) GetTargetRepo(ctx context.Context, repoOwner string, targetRepo string) *github.Repository {
	repo, _, err := client.ghClient.Repositories.Get(ctx, repoOwner, targetRepo)
	if err != nil {
		log.Printf("unexpected error happened while getting target repo %v", err)
	}
	return repo
}

func (client *EnterpriseClient) GetRecentCommits(ctx context.Context, repoOwner string, targetRepo string) []*github.Commit {
	opts := &github.CommitsListOptions{ListOptions: github.ListOptions{}}
	recentCommits := make([]*github.Commit, 0)
	halfYearAgo := time.Now().Local().AddDate(0, -6, 0)
	notFoundEnough := true
	for notFoundEnough {
		commits, resp, err := client.ghClient.Repositories.ListCommits(ctx, repoOwner, targetRepo, opts)
		if err != nil {
			zap.S().Errorf("unexpected error happened while getting commits %v", err)
		}

		for _, commit := range commits {
			if commit.Commit.Committer.Date.Local().Before(halfYearAgo) || len(recentCommits) == client.minCommits {
				notFoundEnough = false
				break
			}

			if strings.Contains(*commit.Commit.Message, "Merge pull request") {
				continue
			}

			recentCommits = append(recentCommits, commit.Commit)
		}

		if resp.NextPage == 0 {
			break
		}
		opts.Page = resp.NextPage
	}
	return recentCommits
}

func (client *EnterpriseClient) GetContentURL(ctx context.Context, repoOwner string, targetRepo string, path string) (*string, error) {
	contents, _, resp, err := client.ghClient.Repositories.GetContents(ctx, repoOwner, targetRepo, path, &github.RepositoryContentGetOptions{})
	if err != nil {
		if resp.StatusCode == 404 {
			return nil, fmt.Errorf("cannot found %v in repo %v", path, targetRepo)
		}
		zap.S().Errorf("unexpected error happened while getting content for path %v %v", path, err)
	}
	return contents.HTMLURL, nil
}
