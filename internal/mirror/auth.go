package mirror

import (
	"net/http"

	"code.gitea.io/sdk/gitea"
	"github.com/USA-RedDragon/gitea-mirror/internal/config"
	"github.com/bradleyfalzon/ghinstallation/v2"
	"github.com/gofri/go-github-ratelimit/github_ratelimit"
	"github.com/google/go-github/v58/github"
)

func authenticate(config *config.Config) (*github.Client, *gitea.Client, error) {
	var githubClient *github.Client
	var giteaClient *gitea.Client

	rateLimiter, err := github_ratelimit.NewRateLimitWaiterClient(nil)
	if err != nil {
		return nil, nil, err
	}

	if config.GitHubAuth.Token != "" {
		githubClient = github.NewClient(rateLimiter).WithAuthToken(config.GitHubAuth.Token)
		if config.GitHubAuth.EnterpriseURL != "" {
			var err error
			githubClient, err = githubClient.WithEnterpriseURLs(config.GitHubAuth.EnterpriseURL, config.GitHubAuth.EnterpriseURL)
			if err != nil {
				return nil, nil, err
			}
		}
	} else {
		itr, err := ghinstallation.NewKeyFromFile(rateLimiter.Transport, int64(config.GitHubAuth.AppID), int64(config.GitHubAuth.InstallationID), config.GitHubAuth.PrivateKeyPath)
		if err != nil {
			return nil, nil, err
		}
		githubClient = github.NewClient(&http.Client{Transport: itr})
	}

	giteaClient, err = gitea.NewClient(config.GiteaAuth.URL, gitea.SetToken(config.GiteaAuth.Token))
	if err != nil {
		return nil, nil, err
	}

	return githubClient, giteaClient, nil
}
