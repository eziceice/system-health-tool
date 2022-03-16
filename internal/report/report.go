package report

import (
	"context"
	"fmt"
	"github.com/buildkite/go-buildkite/v3/buildkite"
	"github.com/google/go-github/v41/github"
	"strings"
	bkInternal "system-health-tool/internal/buildkite"
	ghInternal "system-health-tool/internal/github"
	"system-health-tool/internal/model"
	"time"
)

const (
	repoOwner     = "smartline"
	pagerDocLower = "1pager.md"
	pagerDocUpper = "1Pager.md"
)

type Report struct {
	Language          string
	RecentCommits     []*github.Commit
	RecentDeployments []buildkite.Build
	Repo              *github.Repository
	Doc               *string
}

func GetReportDetails(ctx context.Context, environments model.Environments, targetRepo string) (string, string) {
	ghClient := ghInternal.NewEnterpriseClient(ctx, environments.BaseUrl, environments.GitHubToken, 2)
	bkClient := bkInternal.New(environments.BuildKiteToken, 3)

	repo := ghClient.GetTargetRepo(ctx, repoOwner, targetRepo)
	pipeline := fmt.Sprintf("%v-%v", repoOwner, targetRepo)
	doc, _ := ghClient.GetContentURL(ctx, repoOwner, targetRepo, pagerDocLower)
	if doc == nil {
		doc, _ = ghClient.GetContentURL(ctx, repoOwner, targetRepo, pagerDocUpper)
	}
	recentCommits := ghClient.GetRecentCommits(ctx, repoOwner, targetRepo)
	deployments := bkClient.GetRecentDeployments(pipeline, environments.Org)

	return generateTitle(repo), generateReport(*repo.Language, recentCommits, deployments, repo, doc)
}

func generateTitle(repo *github.Repository) string {
	preTextString := &strings.Builder{}
	preTextString.WriteString(fmt.Sprintf("System Health Report for <%v|%v>\n", *repo.HTMLURL, *repo.Name))
	preTextString.WriteString("System Health URLs: <https://g3.reainternal.net/|REA GreenGreenGreen>, <https://g3.reainternal.net/|REA Tech Radar>\n")
	return preTextString.String()
}

func generateReport(language string, recentCommits []*github.Commit, recentDeployments []buildkite.Build, repo *github.Repository, doc *string) string {
	layout := "2006-01-02"
	report := Report{language, recentCommits, recentDeployments, repo, doc}

	reportString := &strings.Builder{}
	reportString.WriteString("==================================================================\n")
	reportString.WriteString("\n")

	reportString.WriteString(report.primaryLanguage())
	reportString.WriteString(report.recentCommits(layout))
	reportString.WriteString(report.recentDeployments(layout))
	reportString.WriteString(report.doc())
	reportString.WriteString("==================================================================\n")

	return reportString.String()
}

func (report Report) primaryLanguage() string {
	stringBuilder := &strings.Builder{}
	stringBuilder.WriteString("_General purpose programming language *NOT* marked as “Retire” on Tech Radar?_\n")
	stringBuilder.WriteString("_General purpose programming language marked as *\"Adopt\"* or *\"Consult\"* on Tech Radar?_\n")
	stringBuilder.WriteString(fmt.Sprintf("The primary language of *%v* is: `%v`\n", *report.Repo.Name, report.Language))
	stringBuilder.WriteString("\n")
	stringBuilder.WriteString("\n")
	return stringBuilder.String()
}

func (report Report) recentCommits(layout string) string {
	halfYearAgo := time.Now().Local().AddDate(0, -6, 0).Format(layout)
	now := time.Now().Local().Format(layout)
	repo := *report.Repo.Name
	recentCommits := report.RecentCommits
	stringBuilder := &strings.Builder{}

	stringBuilder.WriteString("_Two or more people have contributed to this codebase, in the last six months?_\n")
	stringBuilder.WriteString(fmt.Sprintf("The recent commits for *%v* in the last *6* months (between *%v* and *%v*) is: \n", repo, halfYearAgo, now))
	if len(recentCommits) == 0 {
		stringBuilder.WriteString(fmt.Sprintf("Sorry there is no commit in the last *6* months for *%v!*\n", repo))
	} else {
		for i, commit := range recentCommits {
			stringBuilder.WriteString(fmt.Sprintf("%v. Name: %v, Commit Date: %v, Commit Message: %v \n", i+1, *commit.Author.Name, commit.Committer.Date.Local().Format(layout), *commit.Message))
		}
	}
	stringBuilder.WriteString("\n")
	stringBuilder.WriteString("\n")
	return stringBuilder.String()
}

func (report Report) recentDeployments(layout string) string {
	threeMonthsAgo := time.Now().Local().AddDate(0, -3, 0).Format(layout)
	now := time.Now().Local().Format(layout)
	repo := *report.Repo.Name
	recentDeployments := report.RecentDeployments
	stringBuilder := &strings.Builder{}

	stringBuilder.WriteString("_Three or more production deployments, this quarter?_\n")
	stringBuilder.WriteString(fmt.Sprintf("The recent prod deployments for *%v* in the last *3* months (between *%v* and *%v*) is: \n", repo, threeMonthsAgo, now))
	if len(recentDeployments) == 0 {
		stringBuilder.WriteString(fmt.Sprintf("Sorry there is no deployments in the last *3* months for this *%v*!\n", repo))
	} else {
		for i, deployment := range recentDeployments {
			stringBuilder.WriteString(fmt.Sprintf("%v. Name: %v, Deployment Date: %v, Build URL: <%v|here> \n", i+1, deployment.Author.Name, deployment.FinishedAt.Local().Format(layout), *deployment.WebURL))
		}
	}
	stringBuilder.WriteString("\n")
	stringBuilder.WriteString("\n")
	return stringBuilder.String()
}

func (report Report) doc() string {
	doc := report.Doc
	repo := *report.Repo.Name
	stringBuilder := &strings.Builder{}

	stringBuilder.WriteString("_1Pager documentation is stored in Git?_\n")
	if doc == nil {
		stringBuilder.WriteString(fmt.Sprintf("cannot find 1Pager documentation for *%v*\n", repo))
	} else {
		stringBuilder.WriteString(fmt.Sprintf("1Pager documentation for *%v* is stored in <%v|git>\n", repo, *doc))
	}
	return stringBuilder.String()
}
