package scaler

import (
	"errors"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
)

type fakeQueue struct {
	tasks []Task
	err   error
}

func (f *fakeQueue) Tasks() ([]Task, error) { return f.tasks, f.err }

type fakeTarget struct {
	mu    sync.Mutex
	calls []scaleCall
	err   error
}

type scaleCall struct {
	group  string
	count  int
	reason string
}

func (f *fakeTarget) Scale(group string, count int, reason string) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.calls = append(f.calls, scaleCall{group, count, reason})
	return f.err
}

func TestTickClampsAndScales(t *testing.T) {
	t.Parallel()

	pools := []Pool{
		{Name: "arm", Group: "arm64", Selector: LabelSelector{"platform", "linux/arm64"}, Min: 0, Max: 3},
		{Name: "all", Group: "all", Min: 0, Max: 2},
	}
	router := NewRouter(pools)

	q := &fakeQueue{tasks: []Task{
		{Labels: map[string]string{"platform": "linux/arm64"}},
		{Labels: map[string]string{"platform": "linux/arm64"}},
		{Labels: map[string]string{"platform": "linux/arm64"}},
		{Labels: map[string]string{"platform": "linux/arm64"}},
		{Labels: map[string]string{"platform": "linux/arm64"}}, // 5 arm -> ceil(5/2)=3, clamped to Max=3
		{Labels: nil}, // 1 catch-all -> ceil(1/2)=1
	}}
	tgt := &fakeTarget{}

	r := NewReconciler(router, q, tgt, WithWorkflowsPerAgent(2))
	r.Tick()

	assert.Equal(t, []scaleCall{
		{group: "arm64", count: 3, reason: "arm"},
		{group: "all", count: 1, reason: "all"},
	}, tgt.calls)
}

func TestTickIdleScalesToZero(t *testing.T) {
	t.Parallel()

	pools := []Pool{{Name: "p", Group: "g", Min: 0, Max: 5}}
	tgt := &fakeTarget{}
	r := NewReconciler(NewRouter(pools), &fakeQueue{}, tgt)
	r.Tick()

	assert.Equal(t, 1, len(tgt.calls))
	assert.Equal(t, 0, tgt.calls[0].count)
}

func TestTickRespectsMin(t *testing.T) {
	t.Parallel()

	pools := []Pool{{Name: "p", Group: "g", Min: 1, Max: 5}}
	tgt := &fakeTarget{}
	r := NewReconciler(NewRouter(pools), &fakeQueue{}, tgt)
	r.Tick()

	assert.Equal(t, 1, tgt.calls[0].count)
}

func TestTickQueueErrorSkips(t *testing.T) {
	t.Parallel()

	tgt := &fakeTarget{}
	r := NewReconciler(NewRouter(nil), &fakeQueue{err: errors.New("boom")}, tgt)
	r.Tick()
	assert.Empty(t, tgt.calls)
}

func TestAgentsFor(t *testing.T) {
	t.Parallel()

	cases := map[string]struct {
		workflows, perAgent, want int
	}{
		"zero":  {0, 2, 0},
		"exact": {4, 2, 2},
		"round": {5, 2, 3},
		"one":   {1, 2, 1},
	}
	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			assert.Equal(t, tc.want, agentsFor(tc.workflows, tc.perAgent))
		})
	}
}
