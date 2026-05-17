package main

import (
	"os"

	"github.com/urfave/cli/v3"
)

var flags = []cli.Flag{
	&cli.StringFlag{
		Name:    "log-level",
		Value:   "info",
		Usage:   "default log level",
		Sources: cli.EnvVars("WOODPECKER_LOG_LEVEL"),
	},
	&cli.StringFlag{
		Name:    "interval",
		Value:   "60s",
		Usage:   "poll interval (Go duration)",
		Sources: cli.EnvVars("WOODPECKER_POLLER_INTERVAL"),
	},
	&cli.StringFlag{
		Name:    "server-url",
		Usage:   "woodpecker server address",
		Sources: cli.EnvVars("WOODPECKER_SERVER"),
	},
	&cli.StringFlag{
		Name:  "server-token",
		Usage: "woodpecker api token (admin)",
		Sources: cli.NewValueSourceChain(
			cli.EnvVar("WOODPECKER_TOKEN"),
			cli.File(os.Getenv("WOODPECKER_TOKEN_FILE")),
		),
	},
	&cli.BoolFlag{
		Name:    "skip-verify",
		Usage:   "skip TLS verification when talking to the woodpecker server",
		Sources: cli.EnvVars("WOODPECKER_SKIP_VERIFY"),
	},
	&cli.StringFlag{
		Name:  "github-token",
		Usage: "github token used to read commit shas",
		Sources: cli.NewValueSourceChain(
			cli.EnvVar("WOODPECKER_GITHUB_TOKEN"),
			cli.File(os.Getenv("WOODPECKER_GITHUB_TOKEN_FILE")),
		),
	},
	&cli.StringFlag{
		Name:     "nomad-addr",
		Required: true,
		Usage:    "nomad http api address",
		Sources:  cli.EnvVars("WOODPECKER_NOMAD_ADDR"),
	},
	&cli.StringFlag{
		Name:  "nomad-token",
		Usage: "nomad acl token",
		Sources: cli.NewValueSourceChain(
			cli.EnvVar("WOODPECKER_NOMAD_TOKEN"),
			cli.File(os.Getenv("WOODPECKER_NOMAD_TOKEN_FILE")),
		),
	},
	&cli.StringFlag{
		Name:    "nomad-namespace",
		Value:   "default",
		Usage:   "nomad namespace where the state variable lives",
		Sources: cli.EnvVars("WOODPECKER_NOMAD_NAMESPACE"),
	},
	&cli.StringFlag{
		Name:    "var-prefix",
		Value:   "wp-poller",
		Usage:   "nomad variable path prefix for last-seen shas",
		Sources: cli.EnvVars("WOODPECKER_POLLER_VAR_PREFIX"),
	},
}
