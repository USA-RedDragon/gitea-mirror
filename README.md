# gitea-mirror

A simple Go program to mirror repositories from GitHub to Gitea.

## Configuration

## Sidecar Mode

In order to allow mirroring without utilizing a PAT, the program can be run as a sidecar to a Gitea instance. This allows the program to inject app-generated tokens into the Gitea instance before they expire. This can be enabled with the `--sidecar` flag or by setting the `SIDECAR` environment variable to `true`.
