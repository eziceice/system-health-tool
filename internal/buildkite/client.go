package buildkite

import (
	"github.com/buildkite/go-buildkite/v3/buildkite"
	"go.uber.org/zap"
	"time"
)

type Client struct {
	bkClient       *buildkite.Client
	minDeployments int
}

func New(bkToken string, minDeployments int) Client {
	config, err := buildkite.NewTokenConfig(bkToken, false)
	if err != nil {
		zap.S().Errorf("cannot init config for BuildKite client %v", err)
	}

	client := Client{bkClient: buildkite.NewClient(config.Client()), minDeployments: minDeployments}
	return client
}

func (client *Client) GetRecentDeployments(pipeline string, bkOrg string) []buildkite.Build {
	startPage := 1
	successBuilds := make([]buildkite.Build, 0)
	threeMonthsAgo := time.Now().Local().AddDate(0, -3, 0)
	notFoundEnough := true
	for notFoundEnough {
		buildsListOptions := &buildkite.BuildsListOptions{State: []string{"passed"}, ListOptions: buildkite.ListOptions{PerPage: 100, Page: startPage}, Branch: "master"}
		builds, _, err := client.bkClient.Builds.ListByPipeline(bkOrg, pipeline, buildsListOptions)
		if err != nil {
			zap.S().Errorf("unexpected error happened while getting buildkite builds %v", err)
		}
		for _, build := range builds {
			if build.FinishedAt.Local().Before(threeMonthsAgo) || len(successBuilds) == client.minDeployments {
				notFoundEnough = false
				break
			}

			if *build.State == "passed" && *build.Blocked == false && *build.Message != "Audit" {
				successBuilds = append(successBuilds, build)
			}
		}
		startPage = startPage + 1
	}
	return successBuilds
}
