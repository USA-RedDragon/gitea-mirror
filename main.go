package main

import (
	"context"
	"os"

	"code.gitea.io/sdk/gitea"
	"github.com/fatih/color"
	"github.com/google/go-github/v58/github"
)

func getUserRepos(client *github.Client, data chan *github.Repository) error {
	opt := &github.RepositoryListByAuthenticatedUserOptions{
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

func getOrgRepos(client *github.Client, entity string, data chan *github.Repository) error {
	opt := &github.RepositoryListByOrgOptions{
		ListOptions: github.ListOptions{PerPage: 100},
	}
	for {
		repos, resp, err := client.Repositories.ListByOrg(context.Background(), entity, opt)

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

func getRepos(client *github.Client, entity string, data chan *github.Repository, org bool) error {
	if org {
		return getOrgRepos(client, entity, data)
	} else {
		return getUserRepos(client, data)
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
	giteaUser := os.Getenv("GITEA_USER")
	if giteaOrg == "" && giteaUser == "" {
		color.Red("GITEA_ORG or GITEA_USER is not set")
		os.Exit(1)
	} else if giteaOrg != "" && giteaUser != "" {
		color.Red("GITEA_ORG and GITEA_USER are both set")
		os.Exit(1)
	}
	githubOrg := os.Getenv("GITHUB_ORG")
	githubUser := os.Getenv("GITHUB_USER")
	if giteaOrg == "" && giteaUser == "" {
		color.Red("GITHUB_ORG or GITHUB_USER is not set")
		os.Exit(1)
	} else if giteaOrg != "" && giteaUser != "" {
		color.Red("GITHUB_ORG and GITHUB_USER are both set")
		os.Exit(1)
	}

	var gitHubEntity string
	var org bool
	if githubOrg != "" {
		gitHubEntity = githubOrg
		org = true
	} else {
		gitHubEntity = githubUser
		org = false
	}

	var giteaEntity string
	if giteaOrg != "" {
		giteaEntity = giteaOrg
	} else {
		giteaEntity = giteaUser
	}

	githubClient := github.NewClient(nil).WithAuthToken(githubToken)
	giteaClient, err := gitea.NewClient(giteaURL, gitea.SetToken(giteaToken))
	if err != nil {
		color.Red("Error creating Gitea client: %s", err)
		os.Exit(1)
	}

	reposChannel := make(chan *github.Repository)
	go func() {
		err := getRepos(githubClient, gitHubEntity, reposChannel, org)
		close(reposChannel)
		if err != nil {
			color.Red("Error getting repos: %s", err)
		}
	}()

	for githubRepo := range reposChannel {
		if githubRepo.Description == nil {
			githubRepo.Description = new(string)
		}
		color.Cyan("Mirroring %s", *githubRepo.Name)
		foundRepo, _, err := giteaClient.GetRepo(giteaEntity, *githubRepo.Name)
		if err != nil || foundRepo == nil {
			_, _, err = giteaClient.MigrateRepo(gitea.MigrateRepoOption{
				RepoName:       *githubRepo.Name,
				RepoOwner:      giteaEntity,
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
				LFS:            true,
			})
			if err != nil {
				color.Red("Error mirroring %s: %s", *githubRepo.Name, err)
			}
			color.Green("Mirror complete")
		} else {
			color.Yellow("Repo already exists, skipping")
		}
	}
}
