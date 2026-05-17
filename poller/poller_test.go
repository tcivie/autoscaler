package poller

import (
	"errors"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
)

type stubRepos struct{ repos []Repo }

func (s *stubRepos) ActiveRepos() ([]Repo, error) { return s.repos, nil }

type stubHeads struct{ shas map[string]string }

func (s *stubHeads) Resolve(url, branch string) (string, error) {
	if v, ok := s.shas[url+"@"+branch]; ok {
		return v, nil
	}
	return "", errors.New("not found")
}

type stubTrigger struct {
	mu    sync.Mutex
	calls []int64
}

func (s *stubTrigger) Trigger(id int64, _ string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.calls = append(s.calls, id)
	return nil
}

type stubStore struct{ data map[string]string }

func (s *stubStore) Get(k string) (string, error) { return s.data[k], nil }
func (s *stubStore) Set(k, v string) error        { s.data[k] = v; return nil }

func TestPollerPrimesUnknownRepos(t *testing.T) {
	t.Parallel()

	repos := &stubRepos{repos: []Repo{
		{ID: 1, CloneURL: "u1", Branch: "main", FullName: "a/b"},
	}}
	heads := &stubHeads{shas: map[string]string{"u1@main": "abc123"}}
	trig := &stubTrigger{}
	store := &stubStore{data: map[string]string{}}

	NewPoller(repos, heads, trig, store).Tick()

	assert.Equal(t, "abc123", store.data["1"])
	assert.Empty(t, trig.calls)
}

func TestPollerTriggersOnChange(t *testing.T) {
	t.Parallel()

	repos := &stubRepos{repos: []Repo{
		{ID: 7, CloneURL: "u", Branch: "main", FullName: "a/b"},
	}}
	heads := &stubHeads{shas: map[string]string{"u@main": "new"}}
	trig := &stubTrigger{}
	store := &stubStore{data: map[string]string{"7": "old"}}

	NewPoller(repos, heads, trig, store).Tick()

	assert.Equal(t, []int64{7}, trig.calls)
	assert.Equal(t, "new", store.data["7"])
}

func TestPollerSkipsUnchanged(t *testing.T) {
	t.Parallel()

	repos := &stubRepos{repos: []Repo{{ID: 1, CloneURL: "u", Branch: "main"}}}
	heads := &stubHeads{shas: map[string]string{"u@main": "same"}}
	trig := &stubTrigger{}
	store := &stubStore{data: map[string]string{"1": "same"}}

	NewPoller(repos, heads, trig, store).Tick()

	assert.Empty(t, trig.calls)
	assert.Equal(t, "same", store.data["1"])
}

func TestInjectToken(t *testing.T) {
	t.Parallel()

	assert.Equal(t, "https://x-access-token:tk@github.com/o/r", injectToken("https://github.com/o/r", "tk"))
	assert.Equal(t, "git@github.com:o/r", injectToken("git@github.com:o/r", "tk"))
}
