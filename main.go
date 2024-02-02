package main

import (
	"context"
	"os"

	"code.gitea.io/sdk/gitea"
	"github.com/fatih/color"
	"github.com/google/go-github/v58/github"
)

func getRepos(client *github.Client, data chan *github.Repository) error {
	opt := &github.RepositoryListByAuthenticatedUserOptions{
		Affiliation: "organization_member",
		ListOptions: github.ListOptions{PerPage: 100},
	}
	for {
		repos, resp, err := client.Repositories.ListByAuthenticatedUser(context.Background(), opt)
		if err != nil {
			return err
		}
		for _, repo := range repos {
			data <- repo
		}
		if resp.NextPage == 0 {
			return nil
		}
		opt.Page = resp.NextPage
	}
}

func main() {
	githubToken := os.Getenv("GITHUB_TOKEN")
	if githubToken == "" {
		color.Red("GITHUB_TOKEN is not set")
		os.Exit(1)
	}
	giteaToken := os.Getenv("GITEA_TOKEN")
	if giteaToken == "" {
		color.Red("GITEA_TOKEN is not set")
		os.Exit(1)
	}
	giteaURL := os.Getenv("GITEA_URL")
	if giteaURL == "" {
		color.Red("GITEA_URL is not set")
		os.Exit(1)
	}
	giteaOrg := os.Getenv("GITEA_ORG")
	if giteaOrg == "" {
		color.Red("GITEA_ORG is not set")
		os.Exit(1)
	}
	githubClient := github.NewClient(nil).WithAuthToken(githubToken)
	giteaClient, err := gitea.NewClient(giteaURL, gitea.SetToken(giteaToken))
	if err != nil {
		panic(err)
	}

	reposChannel := make(chan *github.Repository)
	go func() {
		err := getRepos(githubClient, reposChannel)
		close(reposChannel)
		if err != nil {
			panic(err)
		}
	}()

	for githubRepo := range reposChannel {
		if githubRepo.Description == nil {
			githubRepo.Description = new(string)
		}
		color.Cyan("Mirroring %s", *githubRepo.Name)
		foundRepo, _, err := giteaClient.GetRepo(giteaOrg, *githubRepo.Name)
		if err != nil || foundRepo == nil {
			_, _, err = giteaClient.MigrateRepo(gitea.MigrateRepoOption{
				RepoName:       *githubRepo.Name,
				CloneAddr:      *githubRepo.CloneURL,
				AuthToken:      githubToken,
				Private:        *githubRepo.Private,
				Description:    *githubRepo.Description,
				Wiki:           true,
				Milestones:     true,
				Labels:         true,
				Issues:         true,
				PullRequests:   true,
				Releases:       true,
				Mirror:         true,
				MirrorInterval: "10m",
			})
			if err != nil {
				panic(err)
			}
		} else {
			color.Yellow("Repo already exists, skipping")
		}
		color.Green("Mirror complete")
	}
}
