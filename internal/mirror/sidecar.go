package mirror

import (
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"time"
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
			// TODO: do the thing
			// Look at the git config, find the PAT and check if it's valid
			// If it's not, refresh it and update the git config
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
