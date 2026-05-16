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
		Name:    "reconciliation-interval",
		Value:   "30s",
		Usage:   "interval between reconcile ticks (Go duration)",
		Sources: cli.EnvVars("WOODPECKER_RECONCILIATION_INTERVAL"),
	},
	&cli.IntFlag{
		Name:    "workflows-per-agent",
		Value:   2,
		Usage:   "how many workflows one agent can run in parallel",
		Sources: cli.EnvVars("WOODPECKER_WORKFLOWS_PER_AGENT"),
	},
	&cli.StringFlag{
		Name:    "server-url",
		Usage:   "woodpecker server address",
		Sources: cli.EnvVars("WOODPECKER_SERVER"),
	},
	&cli.StringFlag{
		Name:  "server-token",
		Usage: "woodpecker api token",
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
		Name:    "socks-proxy",
		Usage:   "optional SOCKS5 proxy address",
		Sources: cli.EnvVars("SOCKS_PROXY"),
	},
	&cli.BoolFlag{
		Name:    "socks-proxy-off",
		Usage:   "disable SOCKS5 proxy",
		Sources: cli.EnvVars("SOCKS_PROXY_OFF"),
	},
	&cli.StringSliceFlag{
		Name:     "pools",
		Usage:    "ordered list of pool names (per-pool config via WOODPECKER_POOL_<NAME>_{GROUP,FILTER,MIN,MAX})",
		Required: true,
		Sources:  cli.EnvVars("WOODPECKER_POOLS"),
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
		Usage:   "nomad namespace the target job lives in",
		Sources: cli.EnvVars("WOODPECKER_NOMAD_NAMESPACE"),
	},
	&cli.StringFlag{
		Name:    "nomad-region",
		Usage:   "nomad region (optional, defaults to cluster default)",
		Sources: cli.EnvVars("WOODPECKER_NOMAD_REGION"),
	},
	&cli.StringFlag{
		Name:     "nomad-job",
		Required: true,
		Usage:    "nomad job id whose task groups will be scaled (e.g. woodpecker-agent)",
		Sources:  cli.EnvVars("WOODPECKER_NOMAD_JOB"),
	},
}
