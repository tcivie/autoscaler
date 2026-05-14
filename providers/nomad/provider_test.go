package nomad

import (
	"errors"
	"strings"
	"testing"

	"github.com/hashicorp/nomad/api"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"go.woodpecker-ci.org/autoscaler/config"
	"go.woodpecker-ci.org/autoscaler/engine"
	"go.woodpecker-ci.org/woodpecker/v3/woodpecker-go/woodpecker"
)

type stubClient struct {
	registerErr   error
	deregisterErr error
	listErr       error
	listResp      []*api.JobListStub
	registered    *api.Job
	deregistered  string
	listedPrefix  string
}

func (s *stubClient) Register(job *api.Job) error {
	s.registered = job
	return s.registerErr
}

func (s *stubClient) Deregister(jobID string, _ bool) error {
	s.deregistered = jobID
	return s.deregisterErr
}

func (s *stubClient) List(prefix string) ([]*api.JobListStub, error) {
	s.listedPrefix = prefix
	return s.listResp, s.listErr
}

func newTestProvider(client *stubClient, arch string) *Provider {
	return &Provider{
		name:        "nomad",
		image:       "woodpeckerci/woodpecker-agent:latest",
		namespace:   "default",
		datacenters: []string{"dc1"},
		arch:        arch,
		cpu:         500,
		memory:      512,
		config: &config.Config{
			PoolID:            "test",
			GRPCAddress:       "grpc.example:9000",
			WorkflowsPerAgent: 2,
			ExtraAgentLabels:  map[string]string{"team": "platform"},
		},
		client: client,
	}
}

func TestDeployAgentBuildsJob(t *testing.T) {
	tests := []struct {
		name           string
		arch           string
		wantConstraint bool
		wantLabel      string
	}{
		{name: "without arch", arch: "", wantConstraint: false, wantLabel: "team=platform"},
		{name: "arm64", arch: "arm64", wantConstraint: true, wantLabel: "platform=linux/arm64"},
		{name: "amd64", arch: "amd64", wantConstraint: true, wantLabel: "platform=linux/amd64"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client := &stubClient{}
			p := newTestProvider(client, tt.arch)

			err := p.DeployAgent(t.Context(), &woodpecker.Agent{Name: "pool-test-agent-abc", Token: "secret"})
			require.NoError(t, err)
			require.NotNil(t, client.registered)

			job := client.registered
			assert.Equal(t, "pool-test-agent-abc", *job.ID)
			assert.Equal(t, "batch", *job.Type)
			assert.Equal(t, "test", job.Meta[engine.LabelPool])
			require.Len(t, job.TaskGroups, 1)

			group := job.TaskGroups[0]
			require.Len(t, group.Tasks, 1)
			task := group.Tasks[0]
			assert.Equal(t, "secret", task.Env["WOODPECKER_AGENT_SECRET"])
			assert.Equal(t, "grpc.example:9000", task.Env["WOODPECKER_SERVER"])
			assert.Contains(t, task.Env["WOODPECKER_AGENT_LABELS"], tt.wantLabel)

			if tt.wantConstraint {
				require.Len(t, group.Constraints, 1)
				assert.Equal(t, "${attr.cpu.arch}", group.Constraints[0].LTarget)
				assert.Equal(t, nomadArch(tt.arch), group.Constraints[0].RTarget)
			} else {
				assert.Empty(t, group.Constraints)
			}
		})
	}
}

func TestDeployAgentRegisterError(t *testing.T) {
	client := &stubClient{registerErr: errors.New("boom")}
	p := newTestProvider(client, "")

	err := p.DeployAgent(t.Context(), &woodpecker.Agent{Name: "x", Token: "t"})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "register: boom")
}

func TestRemoveAgent(t *testing.T) {
	client := &stubClient{}
	p := newTestProvider(client, "")

	require.NoError(t, p.RemoveAgent(t.Context(), &woodpecker.Agent{Name: "pool-test-agent-zzz"}))
	assert.Equal(t, "pool-test-agent-zzz", client.deregistered)
}

func TestRemoveAgentError(t *testing.T) {
	client := &stubClient{deregisterErr: errors.New("nope")}
	p := newTestProvider(client, "")

	err := p.RemoveAgent(t.Context(), &woodpecker.Agent{Name: "x"})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "deregister: nope")
}

func TestListDeployedAgentNamesFiltersByPool(t *testing.T) {
	client := &stubClient{
		listResp: []*api.JobListStub{
			{Name: "pool-test-agent-1", Meta: map[string]string{engine.LabelPool: "test"}},
			{Name: "pool-other-agent-2", Meta: map[string]string{engine.LabelPool: "other"}},
			{Name: "pool-test-agent-3", Meta: map[string]string{engine.LabelPool: "test"}},
		},
	}
	p := newTestProvider(client, "")

	names, err := p.ListDeployedAgentNames(t.Context())
	require.NoError(t, err)
	assert.ElementsMatch(t, []string{"pool-test-agent-1", "pool-test-agent-3"}, names)
	assert.True(t, strings.HasPrefix(client.listedPrefix, "pool-test-agent-"))
}

func TestListDeployedAgentNamesError(t *testing.T) {
	client := &stubClient{listErr: errors.New("api down")}
	p := newTestProvider(client, "")

	_, err := p.ListDeployedAgentNames(t.Context())
	require.Error(t, err)
	assert.Contains(t, err.Error(), "list: api down")
}

func TestNomadArchTranslation(t *testing.T) {
	assert.Equal(t, "arm64", nomadArch("arm64"))
	assert.Equal(t, "amd64", nomadArch("amd64"))
	assert.Equal(t, "riscv64", nomadArch("riscv64"))
}
