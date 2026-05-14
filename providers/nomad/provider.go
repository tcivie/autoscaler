package nomad

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/hashicorp/nomad/api"
	"github.com/rs/zerolog/log"
	"github.com/urfave/cli/v3"

	"go.woodpecker-ci.org/autoscaler/config"
	"go.woodpecker-ci.org/autoscaler/engine"
	"go.woodpecker-ci.org/autoscaler/engine/types"
	"go.woodpecker-ci.org/autoscaler/providers/nomad/nomadapi"
	"go.woodpecker-ci.org/autoscaler/utils"
	"go.woodpecker-ci.org/woodpecker/v3/woodpecker-go/woodpecker"
)

var ErrAddrRequired = errors.New("nomad-addr is required")

type Provider struct {
	name        string
	image       string
	namespace   string
	region      string
	datacenters []string
	arch        string
	cpu         int
	memory      int
	extraMeta   map[string]string
	config      *config.Config
	client      nomadapi.Client
}

func New(_ context.Context, c *cli.Command, cfg *config.Config) (types.Provider, error) {
	addr := c.String("nomad-addr")
	if addr == "" {
		return nil, ErrAddrRequired
	}

	clientCfg := api.DefaultConfig()
	clientCfg.Address = addr
	clientCfg.Region = c.String("nomad-region")
	clientCfg.Namespace = c.String("nomad-namespace")
	clientCfg.SecretID = c.String("nomad-token")

	client, err := nomadapi.NewClient(clientCfg)
	if err != nil {
		return nil, fmt.Errorf("nomad: %w", err)
	}

	extraMeta, err := utils.SliceToMap(c.StringSlice("nomad-meta"), "=")
	if err != nil {
		return nil, fmt.Errorf("nomad: %w", err)
	}

	return newProvider(c, cfg, client, extraMeta), nil
}

func newProvider(c *cli.Command, cfg *config.Config, client nomadapi.Client, extraMeta map[string]string) *Provider {
	return &Provider{
		name:        "nomad",
		image:       c.String("nomad-image"),
		namespace:   c.String("nomad-namespace"),
		region:      c.String("nomad-region"),
		datacenters: c.StringSlice("nomad-datacenters"),
		arch:        c.String("nomad-arch"),
		cpu:         c.Int("nomad-cpu"),
		memory:      c.Int("nomad-memory"),
		extraMeta:   extraMeta,
		config:      cfg,
		client:      client,
	}
}

func (p *Provider) DeployAgent(_ context.Context, agent *woodpecker.Agent) error {
	if err := p.client.Register(p.buildJob(agent)); err != nil {
		return fmt.Errorf("%s: register: %w", p.name, err)
	}
	log.Info().Str("agent", agent.Name).Str("arch", p.arch).Msgf("%s: deployed agent", p.name)
	return nil
}

func (p *Provider) RemoveAgent(_ context.Context, agent *woodpecker.Agent) error {
	if err := p.client.Deregister(agent.Name, true); err != nil {
		return fmt.Errorf("%s: deregister: %w", p.name, err)
	}
	return nil
}

func (p *Provider) ListDeployedAgentNames(_ context.Context) ([]string, error) {
	prefix := fmt.Sprintf("pool-%s-agent-", p.config.PoolID)
	stubs, err := p.client.List(prefix)
	if err != nil {
		return nil, fmt.Errorf("%s: list: %w", p.name, err)
	}
	names := make([]string, 0, len(stubs))
	for _, stub := range stubs {
		if !strings.HasPrefix(stub.Name, prefix) {
			continue
		}
		if stub.Status == "dead" {
			continue
		}
		names = append(names, stub.Name)
	}
	return names, nil
}

func (p *Provider) buildJob(agent *woodpecker.Agent) *api.Job {
	env := p.agentEnv(agent)
	meta := p.jobMeta()
	cpu, memory := p.cpu, p.memory

	task := &api.Task{
		Name:   "agent",
		Driver: "docker",
		Config: map[string]any{
			"image":        p.image,
			"force_pull":   true,
			"network_mode": "host",
			"volumes":      []string{"/var/run/docker.sock:/var/run/docker.sock"},
		},
		Env: env,
		Resources: &api.Resources{
			CPU:      &cpu,
			MemoryMB: &memory,
		},
	}

	zero := 0
	group := &api.TaskGroup{
		Name:             ptr("agent"),
		Count:            ptr(1),
		Tasks:            []*api.Task{task},
		RestartPolicy:    &api.RestartPolicy{Attempts: &zero},
		ReschedulePolicy: &api.ReschedulePolicy{Attempts: &zero},
	}
	if p.arch != "" {
		group.Constraints = []*api.Constraint{{
			LTarget: "${attr.cpu.arch}",
			RTarget: nomadArch(p.arch),
			Operand: "=",
		}}
	}

	jobType := "batch"
	return &api.Job{
		ID:          ptr(agent.Name),
		Name:        ptr(agent.Name),
		Type:        &jobType,
		Namespace:   ptr(p.namespace),
		Region:      ptr(p.region),
		Datacenters: p.datacenters,
		TaskGroups:  []*api.TaskGroup{group},
		Meta:        meta,
	}
}

func (p *Provider) agentEnv(agent *woodpecker.Agent) map[string]string {
	env := map[string]string{
		"WOODPECKER_SERVER":        p.config.GRPCAddress,
		"WOODPECKER_AGENT_SECRET":  agent.Token,
		"WOODPECKER_MAX_WORKFLOWS": fmt.Sprintf("%d", p.config.WorkflowsPerAgent),
		"WOODPECKER_BACKEND":       "docker",
	}
	if p.config.GRPCSecure {
		env["WOODPECKER_GRPC_SECURE"] = "true"
	}
	for k, v := range p.config.Environment {
		env[k] = v
	}
	env["WOODPECKER_AGENT_LABELS"] = renderLabels(p.agentLabels())
	return env
}

func (p *Provider) agentLabels() map[string]string {
	labels := make(map[string]string, len(p.config.ExtraAgentLabels)+1)
	for k, v := range p.config.ExtraAgentLabels {
		labels[k] = v
	}
	if p.arch != "" {
		labels["platform"] = "linux/" + p.arch
	}
	return labels
}

func (p *Provider) jobMeta() map[string]string {
	meta := map[string]string{
		engine.LabelPool:  p.config.PoolID,
		engine.LabelImage: p.image,
	}
	for k, v := range p.extraMeta {
		meta[k] = v
	}
	return meta
}

func nomadArch(arch string) string {
	return arch
}

func renderLabels(labels map[string]string) string {
	out := make([]string, 0, len(labels))
	for k, v := range labels {
		out = append(out, k+"="+v)
	}
	return strings.Join(out, ",")
}

func ptr[T any](v T) *T { return &v }
