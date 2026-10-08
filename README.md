# gitea-mirror

> [!IMPORTANT]
> **This project is archived and no longer maintained.** Use [RayLabsHQ/gitea-mirror](https://github.com/RayLabsHQ/gitea-mirror) instead. It does everything this did and more: a web UI, organization and starred-repo mirroring, release/LFS/wiki/metadata mirroring, and force-push protection that backs a repo up before a sync rewrites its history.

A simple Go program to mirror repositories from GitHub to Gitea.

## Configuration

## Sidecar Mode

In order to allow mirroring without utilizing a PAT, the program can be run as a sidecar to a Gitea instance. This allows the program to inject app-generated tokens into the Gitea instance before they expire. This can be enabled with the `--sidecar` flag or by setting the `SIDECAR` environment variable to `true`.
