package nomadapi

import (
	"github.com/hashicorp/nomad/api"
)

type Client interface {
	Register(job *api.Job) error
	Deregister(jobID string, purge bool) error
	List(prefix string) ([]*api.JobListStub, error)
}

type client struct {
	jobs *api.Jobs
}

func NewClient(cfg *api.Config) (Client, error) {
	c, err := api.NewClient(cfg)
	if err != nil {
		return nil, err
	}
	return &client{jobs: c.Jobs()}, nil
}

func (c *client) Register(job *api.Job) error {
	_, _, err := c.jobs.Register(job, nil)
	return err
}

func (c *client) Deregister(jobID string, purge bool) error {
	_, _, err := c.jobs.Deregister(jobID, purge, nil)
	return err
}

func (c *client) List(prefix string) ([]*api.JobListStub, error) {
	jobs, _, err := c.jobs.List(&api.QueryOptions{Prefix: prefix})
	return jobs, err
}
