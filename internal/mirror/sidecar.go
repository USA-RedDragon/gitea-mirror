package mirror

import (
	"context"
	"log/slog"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/bradleyfalzon/ghinstallation/v2"
	git "github.com/go-git/go-git/v5"
	"github.com/gofri/go-github-ratelimit/github_ratelimit"
	"github.com/google/go-github/v58/github"
)

func (m *Mirror) runSidecar() {
	reposChan := make(chan string)
	go findRepos(m.config.GiteaAuth.ReposPath, reposChan)
	for {
		select {
		case <-m.stopChan:
			slog.Info("Sidecar stopped")
			m.didSidecarStopChan <- struct{}{}
			return
		case repo := <-reposChan:
			slog.Info("Repo found", "repo", repo)
			gitRepo, err := git.PlainOpen(repo)
			if err != nil {
				slog.Error("Error opening repo", "error", err)
				continue
			}

			// Look at the git config, find the PAT and check if it's valid
			gitConfig, err := gitRepo.Config()
			if err != nil {
				slog.Error("Error getting git config", "error", err)
				continue
			}

			// Grab the remote URL
			remoteURL := gitConfig.Raw.Section("remote").Subsection("origin").Option("url")
			if remoteURL == "" {
				// This is a valid situation where the repo is not mirrored
				continue
			}

			properURL, err := url.Parse(remoteURL)
			if err != nil {
				slog.Error("Error parsing remote URL", "error", err)
				continue
			}

			if properURL.User == nil {
				slog.Error("No user in remote URL")
				continue
			}

			// Check if the PAT is valid
			pat, ok := properURL.User.Password()
			if !ok {
				slog.Error("No password found in remote URL")
				continue
			}

			if properURL.User.Username() == "oauth2" && pat != "" {
				slog.Info("PAT found")

				rateLimiter, err := github_ratelimit.NewRateLimitWaiterClient(nil)
				if err != nil {
					slog.Error("Error creating rate limiter", "error", err)
					continue
				}

				privatePem, err := os.ReadFile(m.config.GitHubAuth.PrivateKeyPath)
				if err != nil {
					slog.Error("Error reading private key", "error", err)
					continue
				}

				appItr, err := ghinstallation.NewAppsTransport(rateLimiter.Transport, int64(m.config.GitHubAuth.AppID), privatePem)
				if err != nil {
					slog.Error("Error creating app transport", "error", err)
					continue
				}
				githubAppClient := github.NewClient(&http.Client{Transport: appItr})

				githubClient := github.NewClient(rateLimiter).WithAuthToken(pat)

				repoParts := strings.Split(strings.TrimPrefix(properURL.Path, "/"), "/")
				if len(repoParts) != 2 {
					slog.Error("Invalid repo path", "path", properURL.Path)
					continue
				}
				orgOrUser := repoParts[0]
				repoWoGit := strings.TrimSuffix(repoParts[1], ".git")

				// Check if the PAT is valid
				_, _, err = githubClient.PullRequests.List(context.Background(), orgOrUser, repoWoGit, &github.PullRequestListOptions{})
				if err != nil {
					// PAT is invalid, refresh it
					slog.Info("PAT is invalid, refreshing", "repo", repo, "error", err)
					installToken, _, err := githubAppClient.Apps.CreateInstallationToken(context.Background(), int64(m.config.GitHubAuth.InstallationID), &github.InstallationTokenOptions{})
					if err != nil {
						slog.Error("Error creating installation token", "error", err)
						continue
					}
					token := installToken.GetToken()
					properURL.User = url.UserPassword("oauth2", token)
					remoteURL = properURL.String()
					gitConfig.Raw.Section("remote").Subsection("origin").SetOption("url", remoteURL)
					slog.Info("Updated remote URL")
				}
			}

		case <-time.After(45 * time.Minute):
			go findRepos(m.config.GiteaAuth.ReposPath, reposChan)
		}
	}
}

func findRepos(basePath string, reposChan chan string) {
	slog.Info("Finding repos")
	// Iterate through the directories in basePath, these are the usernames or orgs
	// For each username or org, iterate through the directories, these are the repos
	// Send the repo name to the reposChan

	err := filepath.Walk(basePath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() {
			// Walk the directory to get the repos
			err := filepath.Walk(path, func(path string, info os.FileInfo, err error) error {
				if err != nil {
					return err
				}
				if info.IsDir() && strings.HasSuffix(path, ".git") {
					reposChan <- path
				}
				return nil
			})
			if err != nil {
				slog.Error("Error walking path", "error", err)
			}
		}
		return nil
	})
	if err != nil {
		slog.Error("Error walking path", "error", err)
	}

}
