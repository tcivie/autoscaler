package scaler

import (
	"fmt"

	"github.com/hashicorp/nomad/api"
)

// ScaleTarget is the side-effect surface the reconciler invokes. The
// abstraction lets us swap Nomad for any other count-scaled backend (K8s,
// Docker Swarm, …) without touching the orchestrator.
type ScaleTarget interface {
	Scale(group string, count int, reason string) error
}

// NomadTarget calls the Nomad Scale API on a single pre-existing job. The
// caller is responsible for keeping the job definition in source control.
type NomadTarget struct {
	client *api.Client
	jobID  string
}

func NewNomadTarget(client *api.Client, jobID string) *NomadTarget {
	return &NomadTarget{client: client, jobID: jobID}
}

func (t *NomadTarget) Scale(group string, count int, reason string) error {
	c := count
	if _, _, err := t.client.Jobs().Scale(t.jobID, group, &c, reason, false, nil, nil); err != nil {
		return fmt.Errorf("nomad scale %s/%s -> %d: %w", t.jobID, group, count, err)
	}
	return nil
}
