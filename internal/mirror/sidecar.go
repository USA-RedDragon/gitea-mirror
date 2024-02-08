package mirror

import (
	"log/slog"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"

	git "github.com/go-git/go-git/v5"
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
				slog.Error("No remote URL found")
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
				// Check if the PAT is valid
				// If it's not, refresh it and update the git config
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
