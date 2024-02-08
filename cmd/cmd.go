package cmd

import (
	"errors"
	"fmt"
	"log/slog"

	"github.com/USA-RedDragon/gitea-mirror/internal/config"
	"github.com/USA-RedDragon/gitea-mirror/internal/mirror"
	"github.com/spf13/cobra"
)

var (
	ErrMissingConfig = errors.New("missing configuration")
)

func NewCommand(version, commit string) *cobra.Command {
	cmd := &cobra.Command{
		Use:     "gitea-mirror",
		Version: fmt.Sprintf("%s - %s", version, commit),
		Annotations: map[string]string{
			"version": version,
			"commit":  commit,
		},
		RunE:          run,
		SilenceUsage:  true,
		SilenceErrors: true,
	}
	config.RegisterFlags(cmd)
	return cmd
}

func run(cmd *cobra.Command, _ []string) error {
	slog.Info("Gitea Mirror", "version", cmd.Annotations["version"], "commit", cmd.Annotations["commit"])

	config, err := config.LoadConfig(cmd)
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	return mirror.Run(config)
}
