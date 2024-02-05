# gitea-mirror

A simple Go program to mirror repositories from GitHub to Gitea.

## Environment Variables

Note: Only set one of `GITEA_USER` or `GITEA_ORG` and `GITHUB_USER` or `GITHUB_ORG` depending on if the repositories are owned by a user or an organization.

|      Name      |         Description          |
| -------------- | ---------------------------- |
| `GITHUB_TOKEN` | GitHub Personal Access Token |
| `GITEA_URL`    | Gitea instance URL           |
| `GITEA_TOKEN`  | Gitea access token           |
| `GITEA_ORG`    | Gitea organization           |
| `GITHUB_ORG`   | GitHub organization          |
| `GITEA_USER`   | Gitea user                   |
| `GITHUB_USER`  | GitHub user                  |
