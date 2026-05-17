package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/hashicorp/nomad/api"
	"go.woodpecker-ci.org/woodpecker/v3/woodpecker-go/woodpecker"

	"go.woodpecker-ci.org/autoscaler/poller"
)

type woodpeckerRepoSource struct{ client woodpecker.Client }

func (w *woodpeckerRepoSource) ActiveRepos() ([]poller.Repo, error) {
	repos, err := w.client.RepoList(woodpecker.RepoListOptions{})
	if err != nil {
		return nil, fmt.Errorf("repo list: %w", err)
	}
	out := make([]poller.Repo, 0, len(repos))
	for _, r := range repos {
		if !r.IsActive {
			continue
		}
		branch := r.Branch
		if branch == "" {
			branch = "main"
		}
		out = append(out, poller.Repo{
			ID:       r.ID,
			CloneURL: r.Clone,
			Branch:   branch,
			FullName: r.FullName,
		})
	}
	return out, nil
}

type woodpeckerTrigger struct {
	baseURL string
	token   string
	http    *http.Client
}

func (w *woodpeckerTrigger) Trigger(repoID int64, branch string) error {
	body, err := json.Marshal(map[string]string{
		"branch":  branch,
		"message": "AUTO TRIGGER @ " + branch,
	})
	if err != nil {
		return err
	}
	url := fmt.Sprintf("%s/api/repos/%d/pipelines", w.baseURL, repoID)
	req, err := http.NewRequest(http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "Bearer "+w.token)
	req.Header.Set("Content-Type", "application/json")
	res, err := w.http.Do(req)
	if err != nil {
		return err
	}
	defer res.Body.Close()
	if res.StatusCode >= 300 {
		buf, _ := io.ReadAll(res.Body)
		return fmt.Errorf("trigger %d: %s", res.StatusCode, string(buf))
	}
	return nil
}

type nomadVarStore struct {
	client *api.Client
	prefix string
}

func (s *nomadVarStore) path(key string) string { return s.prefix + "/" + key }

func (s *nomadVarStore) Get(key string) (string, error) {
	v, _, err := s.client.Variables().Read(s.path(key), nil)
	if err != nil {
		if isNotFound(err) {
			return "", nil
		}
		return "", err
	}
	if v == nil {
		return "", nil
	}
	return v.Items["sha"], nil
}

func (s *nomadVarStore) Set(key, value string) error {
	v := &api.Variable{
		Path:  s.path(key),
		Items: api.VariableItems{"sha": value},
	}
	_, _, err := s.client.Variables().Update(v, nil)
	return err
}

func isNotFound(err error) bool {
	return err != nil && (err.Error() == "variable not found" ||
		err.Error() == "Unexpected response code: 404")
}
