package mirror

import (
	"net/http"
	"os"

	"code.gitea.io/sdk/gitea"
	"github.com/USA-RedDragon/gitea-mirror/internal/config"
	"github.com/bradleyfalzon/ghinstallation/v2"
	"github.com/gofri/go-github-ratelimit/github_ratelimit"
	"github.com/google/go-github/v62/github"
)

func authenticate(config *config.Config) (*github.Client, *github.Client, *gitea.Client, error) {
	var githubClient *github.Client
	var githubAppClient *github.Client
	var giteaClient *gitea.Client

	rateLimiter, err := github_ratelimit.NewRateLimitWaiterClient(nil)
	if err != nil {
		return nil, nil, nil, err
	}

	if config.GitHubAuth.Token != "" {
		githubClient = github.NewClient(rateLimiter).WithAuthToken(config.GitHubAuth.Token)
	} else {
		itr, err := ghinstallation.NewKeyFromFile(rateLimiter.Transport, int64(config.GitHubAuth.AppID), int64(config.GitHubAuth.InstallationID), config.GitHubAuth.PrivateKeyPath)
		if err != nil {
			return nil, nil, nil, err
		}
		githubClient = github.NewClient(&http.Client{Transport: itr})

		privatePem, err := os.ReadFile(config.GitHubAuth.PrivateKeyPath)
		if err != nil {
			return nil, nil, nil, err
		}

		appItr, err := ghinstallation.NewAppsTransport(rateLimiter.Transport, int64(config.GitHubAuth.AppID), privatePem)
		if err != nil {
			return nil, nil, nil, err
		}
		githubAppClient = github.NewClient(&http.Client{Transport: appItr})
	}

	if githubAppClient != nil && config.GitHubAuth.EnterpriseURL != "" {
		var err error
		githubAppClient, err = githubAppClient.WithEnterpriseURLs(config.GitHubAuth.EnterpriseURL, config.GitHubAuth.EnterpriseURL)
		if err != nil {
			return nil, nil, nil, err
		}
	}

	if githubClient != nil && config.GitHubAuth.EnterpriseURL != "" {
		var err error
		githubClient, err = githubClient.WithEnterpriseURLs(config.GitHubAuth.EnterpriseURL, config.GitHubAuth.EnterpriseURL)
		if err != nil {
			return nil, nil, nil, err
		}
	}

	giteaClient, err = gitea.NewClient(config.GiteaAuth.URL, gitea.SetToken(config.GiteaAuth.Token))
	if err != nil {
		return nil, nil, nil, err
	}

	return githubClient, githubAppClient, giteaClient, nil
}
