// woodpecker-nomad-scaler — count-based scaler that drives a single
// pre-existing woodpecker-agent Nomad job. Wires the scaler package
// components together; no business logic lives here.

package main

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/hashicorp/nomad/api"
	_ "github.com/joho/godotenv/autoload"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"github.com/urfave/cli/v3"

	"go.woodpecker-ci.org/autoscaler/scaler"
	"go.woodpecker-ci.org/autoscaler/server"
	"go.woodpecker-ci.org/autoscaler/version"
)

func main() {
	app := &cli.Command{
		Name:    "woodpecker-nomad-scaler",
		Version: version.String(),
		Usage:   "count-scale a single woodpecker-agent Nomad job by per-pool queue depth",
		Flags:   flags,
		Before:  setupLogging,
		Action:  runAction,
	}
	if err := app.Run(context.Background(), os.Args); err != nil {
		log.Error().Err(err).Msg("scaler exited with error")
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
	if zerolog.GlobalLevel() <= zerolog.DebugLevel {
		log.Logger = log.With().Caller().Logger()
	}
	return ctx, nil
}

func runAction(ctx context.Context, cmd *cli.Command) error {
	log.Info().Str("version", version.String()).Msg("starting")

	interval, err := time.ParseDuration(cmd.String("reconciliation-interval"))
	if err != nil {
		return fmt.Errorf("reconciliation-interval: %w", err)
	}

	pools, err := loadPools(cmd.StringSlice("pools"))
	if err != nil {
		return err
	}
	router := scaler.NewRouter(pools)

	wpClient, err := server.NewClient(ctx, cmd)
	if err != nil {
		return fmt.Errorf("woodpecker client: %w", err)
	}
	queue := scaler.NewWoodpeckerQueue(wpClient)

	nomadClient, err := buildNomadClient(cmd)
	if err != nil {
		return fmt.Errorf("nomad client: %w", err)
	}
	target := scaler.NewNomadTarget(nomadClient, cmd.String("nomad-job"))

	r := scaler.NewReconciler(router, queue, target,
		scaler.WithInterval(interval),
		scaler.WithWorkflowsPerAgent(cmd.Int("workflows-per-agent")),
	)

	ctx, cancel := signal.NotifyContext(ctx, syscall.SIGINT, syscall.SIGTERM)
	defer cancel()

	log.Info().
		Str("job", cmd.String("nomad-job")).
		Int("pools", len(pools)).
		Dur("interval", interval).
		Msg("ready")

	return r.Run(ctx)
}

func buildNomadClient(cmd *cli.Command) (*api.Client, error) {
	cfg := api.DefaultConfig()
	cfg.Address = cmd.String("nomad-addr")
	cfg.SecretID = cmd.String("nomad-token")
	cfg.Namespace = cmd.String("nomad-namespace")
	cfg.Region = cmd.String("nomad-region")
	return api.NewClient(cfg)
}

// loadPools resolves Pool specs from CLI input plus per-pool env overrides
// (WOODPECKER_POOL_<NAME>_<KEY>). Pool names come from --pools.
func loadPools(names []string) ([]scaler.Pool, error) {
	if len(names) == 0 {
		return nil, errors.New("--pools is required (comma-separated names or repeated flag)")
	}
	out := make([]scaler.Pool, 0, len(names))
	for _, n := range names {
		n = strings.TrimSpace(n)
		if n == "" {
			continue
		}
		p, err := poolFromEnv(n)
		if err != nil {
			return nil, err
		}
		out = append(out, p)
	}
	if len(out) == 0 {
		return nil, errors.New("--pools resolved to no pools")
	}
	return out, nil
}

func poolFromEnv(name string) (scaler.Pool, error) {
	p := scaler.Pool{Name: name, Group: name, Min: 0, Max: 5}
	if v := poolEnv(name, "GROUP"); v != "" {
		p.Group = v
	}
	if v := poolEnv(name, "FILTER"); v != "" {
		sel, err := scaler.ParseSelector(v)
		if err != nil {
			return p, fmt.Errorf("pool %q: %w", name, err)
		}
		p.Selector = sel
	}
	for key, dst := range map[string]*int{"MIN": &p.Min, "MAX": &p.Max} {
		if v := poolEnv(name, key); v != "" {
			n, err := strconv.Atoi(v)
			if err != nil {
				return p, fmt.Errorf("pool %q: %s: %w", name, key, err)
			}
			*dst = n
		}
	}
	return p, nil
}

func poolEnv(pool, key string) string {
	return os.Getenv("WOODPECKER_POOL_" + pool + "_" + key)
}
