// woodpecker-github-poller — for every active woodpecker repo, watch the
// remote default-branch HEAD via `git ls-remote` and POST a manual pipeline
// trigger when the SHA moves. Last-seen SHAs are persisted as Nomad
// Variables so the poller stays stateless across restarts.

package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/hashicorp/nomad/api"
	_ "github.com/joho/godotenv/autoload"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"github.com/urfave/cli/v3"

	"go.woodpecker-ci.org/autoscaler/poller"
	"go.woodpecker-ci.org/autoscaler/server"
	"go.woodpecker-ci.org/autoscaler/version"
)

func main() {
	app := &cli.Command{
		Name:    "woodpecker-github-poller",
		Version: version.String(),
		Usage:   "trigger woodpecker pipelines when a watched branch HEAD changes",
		Flags:   flags,
		Before:  setupLogging,
		Action:  runAction,
	}
	if err := app.Run(context.Background(), os.Args); err != nil {
		log.Error().Err(err).Msg("poller exited with error")
		os.Exit(1)
	}
}

func setupLogging(ctx context.Context, cmd *cli.Command) (context.Context, error) {
	zerolog.SetGlobalLevel(zerolog.InfoLevel)
	if cmd.IsSet("log-level") {
		raw := cmd.String("log-level")
		lvl, err := zerolog.ParseLevel(raw)
		if err != nil {
			log.Warn().Str("level", raw).Msg("unknown log level")
		} else {
			zerolog.SetGlobalLevel(lvl)
		}
	}
	return ctx, nil
}

func runAction(ctx context.Context, cmd *cli.Command) error {
	log.Info().Str("version", version.String()).Msg("starting")

	interval, err := time.ParseDuration(cmd.String("interval"))
	if err != nil {
		return fmt.Errorf("interval: %w", err)
	}

	wpClient, err := server.NewClient(ctx, cmd)
	if err != nil {
		return fmt.Errorf("woodpecker client: %w", err)
	}

	nomadClient, err := buildNomadClient(cmd)
	if err != nil {
		return fmt.Errorf("nomad client: %w", err)
	}

	repos := &woodpeckerRepoSource{client: wpClient}
	heads := &poller.GitLsRemote{Token: cmd.String("github-token")}
	trigger := &woodpeckerTrigger{client: wpClient}
	store := &nomadVarStore{client: nomadClient, prefix: cmd.String("var-prefix")}

	p := poller.NewPoller(repos, heads, trigger, store, poller.PollerWithInterval(interval))

	ctx, cancel := signal.NotifyContext(ctx, syscall.SIGINT, syscall.SIGTERM)
	defer cancel()

	log.Info().Dur("interval", interval).Msg("ready")
	return p.Run(ctx)
}

func buildNomadClient(cmd *cli.Command) (*api.Client, error) {
	cfg := api.DefaultConfig()
	cfg.Address = cmd.String("nomad-addr")
	cfg.SecretID = cmd.String("nomad-token")
	cfg.Namespace = cmd.String("nomad-namespace")
	return api.NewClient(cfg)
}
