package mirror

import (
	"context"
	"fmt"
	"log/slog"

	"code.gitea.io/sdk/gitea"
	configPkg "github.com/USA-RedDragon/gitea-mirror/internal/config"
	"github.com/google/go-github/v58/github"
)

func getPATUserRepos(client *github.Client, data chan *github.Repository, filter configPkg.FilterConfig) error {
	opt := &github.RepositoryListByAuthenticatedUserOptions{
		Affiliation: "owner",
		ListOptions: github.ListOptions{PerPage: 100},
	}
	for {
		repos, resp, err := client.Repositories.ListByAuthenticatedUser(context.Background(), opt)

		if err != nil {
			return err
		}
		for _, repo := range repos {
			if filter.MatchInclusion(*repo.Name) &&
				!filter.MatchExclusion(*repo.Name) &&
				(!filter.OnlyArchived || (filter.OnlyArchived && *repo.Archived)) {
				data <- repo
			}
		}
		if resp.NextPage == 0 {
			return nil
		}
		opt.Page = resp.NextPage
	}
}

func getAppUserRepos(client *github.Client, user string, data chan *github.Repository, filter configPkg.FilterConfig) error {
	opt := &github.RepositoryListByUserOptions{
		ListOptions: github.ListOptions{PerPage: 100},
	}
	for {
		repos, resp, err := client.Repositories.ListByUser(context.Background(), user, opt)

		if err != nil {
			return err
		}
		for _, repo := range repos {
			if filter.MatchInclusion(*repo.Name) &&
				!filter.MatchExclusion(*repo.Name) &&
				(!filter.OnlyArchived || (filter.OnlyArchived && *repo.Archived)) {
				data <- repo
			}
		}
		if resp.NextPage == 0 {
			return nil
		}
		opt.Page = resp.NextPage
	}
}

func getOrgRepos(client *github.Client, entity string, data chan *github.Repository, filter configPkg.FilterConfig) error {
	opt := &github.RepositoryListByOrgOptions{
		ListOptions: github.ListOptions{PerPage: 100},
	}
	for {
		repos, resp, err := client.Repositories.ListByOrg(context.Background(), entity, opt)

		if err != nil {
			return err
		}
		for _, repo := range repos {
			if filter.MatchInclusion(*repo.Name) &&
				!filter.MatchExclusion(*repo.Name) &&
				(filter.OnlyArchived == false || *repo.Archived == filter.OnlyArchived) {
				data <- repo
			}
		}
		if resp.NextPage == 0 {
			return nil
		}
		opt.Page = resp.NextPage
	}
}

func Run(config *configPkg.Config) error {
	if len(config.Mirrors) == 0 {
		slog.Error("No mirrors defined")
		return nil
	}

	githubClient, giteaClient, err := authenticate(config)
	if err != nil {
		slog.Error("Error authenticating", "error", err)
		return err
	}

	for _, mirror := range config.Mirrors {
		reposChannel := make(chan *github.Repository)
		from := mirror.From
		switch from.Type {
		case configPkg.User:
			slog.Info("Mirroring user", "user", from.Name)
			go func() {
				if config.GitHubAuth.Token != "" {
					err := getPATUserRepos(githubClient, reposChannel, from.Filter)
					defer close(reposChannel)
					if err != nil {
						slog.Error("Error getting repos", "error", err)
					}
				} else {
					err := getAppUserRepos(githubClient, from.Name, reposChannel, from.Filter)
					defer close(reposChannel)
					if err != nil {
						slog.Error("Error getting repos", "error", err)
					}
				}
			}()
		case configPkg.Organization:
			slog.Info("Mirroring org", "org", from.Name)
			go func() {
				err := getOrgRepos(githubClient, from.Name, reposChannel, from.Filter)
				defer close(reposChannel)
				if err != nil {
					slog.Error("Error getting repos", "error", err)
				}
			}()
		default:
			slog.Error("Unknown source type", "type", from.Type)
			defer close(reposChannel)
			continue
		}

		for githubRepo := range reposChannel {
			if githubRepo.Description == nil {
				githubRepo.Description = new(string)
			}
			slog.Info("Mirroring", "repository", *githubRepo.Name)
			foundRepo, _, err := giteaClient.GetRepo(mirror.To.Name, *githubRepo.Name)
			if err != nil || foundRepo == nil {
				_, _, err = giteaClient.MigrateRepo(gitea.MigrateRepoOption{
					RepoName:       fmt.Sprintf("%s%s%s", mirror.Prefix, *githubRepo.Name, mirror.Suffix),
					RepoOwner:      mirror.To.Name,
					Service:        gitea.GitServiceGithub,
					CloneAddr:      *githubRepo.CloneURL,
					AuthToken:      config.GitHubAuth.MirroringToken,
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
					slog.Error("Error mirroring", "repo", *githubRepo.Name, "error", err)
				}
				slog.Info("Mirror complete")
			} else {
				slog.Info("Repo already exists, skipping")
			}
		}
	}

	return nil
}
