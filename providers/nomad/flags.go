package nomad

import (
	"os"

	"github.com/urfave/cli/v3"
)

const (
	category = "Nomad"

	defaultCPU      = 500
	defaultMemoryMB = 512
)

var ProviderFlags = []cli.Flag{
	&cli.StringFlag{
		Name:     "nomad-addr",
		Usage:    "nomad http api address (e.g. http://nomad.example.internal:4646)",
		Sources:  cli.EnvVars("WOODPECKER_NOMAD_ADDR"),
		Category: category,
	},
	&cli.StringFlag{
		Name:  "nomad-token",
		Usage: "nomad acl token",
		Sources: cli.NewValueSourceChain(
			cli.EnvVar("WOODPECKER_NOMAD_TOKEN"),
			cli.File(os.Getenv("WOODPECKER_NOMAD_TOKEN_FILE")),
		),
		Category: category,
	},
	&cli.StringFlag{
		Name:     "nomad-namespace",
		Value:    "default",
		Usage:    "nomad namespace agent jobs are submitted to",
		Sources:  cli.EnvVars("WOODPECKER_NOMAD_NAMESPACE"),
		Category: category,
	},
	&cli.StringFlag{
		Name:     "nomad-region",
		Usage:    "nomad region (optional, defaults to the cluster default)",
		Sources:  cli.EnvVars("WOODPECKER_NOMAD_REGION"),
		Category: category,
	},
	&cli.StringSliceFlag{
		Name:     "nomad-datacenters",
		Usage:    "nomad datacenters the agent job may run in",
		Sources:  cli.EnvVars("WOODPECKER_NOMAD_DATACENTERS"),
		Category: category,
	},
	&cli.StringFlag{
		Name:     "nomad-image",
		Value:    "woodpeckerci/woodpecker-agent:latest",
		Usage:    "container image used for the agent task",
		Sources:  cli.EnvVars("WOODPECKER_NOMAD_IMAGE"),
		Category: category,
	},
	&cli.IntFlag{
		Name:     "nomad-cpu",
		Value:    defaultCPU,
		Usage:    "cpu reservation per agent in mhz",
		Sources:  cli.EnvVars("WOODPECKER_NOMAD_CPU"),
		Category: category,
	},
	&cli.IntFlag{
		Name:     "nomad-memory",
		Value:    defaultMemoryMB,
		Usage:    "memory reservation per agent in mb",
		Sources:  cli.EnvVars("WOODPECKER_NOMAD_MEMORY"),
		Category: category,
	},
	&cli.StringFlag{
		Name:     "nomad-arch",
		Usage:    "constrain agents to nodes of this cpu architecture (e.g. arm64, amd64); also tags agents with platform=linux/<arch>",
		Sources:  cli.EnvVars("WOODPECKER_NOMAD_ARCH"),
		Category: category,
	},
	&cli.StringSliceFlag{
		Name:     "nomad-meta",
		Usage:    "extra nomad job meta entries (key=value)",
		Sources:  cli.EnvVars("WOODPECKER_NOMAD_META"),
		Category: category,
	},
}
