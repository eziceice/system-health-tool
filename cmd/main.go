package main

import (
	"context"
	"github.com/joho/godotenv"
	"go.uber.org/zap"
	"log"
	"system-health-tool/internal/model"
	"system-health-tool/internal/slack"
	"system-health-tool/internal/util"
)

func main() {
	err := godotenv.Load()
	if err != nil {
		log.Fatal("Error loading .env file")
	}

	environments := model.Environments{
		BaseUrl:        util.GetEnv("BASE_URL", ""),
		GitHubToken:    util.GetEnv("GITHUB_TOKEN", ""),
		Org:            util.GetEnv("Org", ""),
		BuildKiteToken: util.GetEnv("BUILDKITE_TOKEN", ""),
		SlackAppToken:  util.GetEnv("SLACKAPP_TOKEN", ""),
		SlackAuthToken: util.GetEnv("SLACKAUTH_TOKEN", ""),
	}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	logger, _ := zap.NewProduction()
	zap.ReplaceGlobals(logger)
	defer logger.Sync()

	zap.S().Info("starting system health bot...")
	slack.Run(ctx, environments)
}
