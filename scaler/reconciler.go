package scaler

import (
	"context"
	"math"
	"time"

	"github.com/rs/zerolog/log"
)

// Reconciler periodically aligns the ScaleTarget with current queue demand.
type Reconciler struct {
	router            *Router
	queue             QueueSource
	target            ScaleTarget
	interval          time.Duration
	workflowsPerAgent int
}

// Option configures a Reconciler.
type Option func(*Reconciler)

func WithInterval(d time.Duration) Option { return func(r *Reconciler) { r.interval = d } }
func WithWorkflowsPerAgent(n int) Option  { return func(r *Reconciler) { r.workflowsPerAgent = n } }

// NewReconciler constructs a Reconciler with sane defaults that can be
// overridden via options.
func NewReconciler(router *Router, queue QueueSource, target ScaleTarget, opts ...Option) *Reconciler {
	r := &Reconciler{
		router:            router,
		queue:             queue,
		target:            target,
		interval:          30 * time.Second,
		workflowsPerAgent: 2,
	}
	for _, o := range opts {
		o(r)
	}
	return r
}

// Run blocks until ctx is cancelled, ticking the reconcile loop on the
// configured interval. A reconcile pass also happens immediately.
func (r *Reconciler) Run(ctx context.Context) error {
	t := time.NewTicker(r.interval)
	defer t.Stop()

	r.Tick()
	for {
		select {
		case <-ctx.Done():
			return nil
		case <-t.C:
			r.Tick()
		}
	}
}

// Tick performs a single reconciliation pass. Exported so tests can drive it
// deterministically without spinning a goroutine.
func (r *Reconciler) Tick() {
	tasks, err := r.queue.Tasks()
	if err != nil {
		log.Error().Err(err).Msg("queue tasks failed")
		return
	}
	counts := r.router.Bucket(tasks)
	for i, p := range r.router.Pools() {
		desired := agentsFor(counts[i], r.workflowsPerAgent)
		desired = clamp(desired, p.Min, p.Max)
		log.Info().
			Str("pool", p.Name).
			Str("group", p.Group).
			Int("workflows", counts[i]).
			Int("target", desired).
			Msg("reconcile")
		if err := r.target.Scale(p.Group, desired, p.Name); err != nil {
			log.Error().Err(err).Str("pool", p.Name).Msg("scale failed")
		}
	}
}

func agentsFor(workflows, perAgent int) int {
	if workflows <= 0 || perAgent <= 0 {
		return 0
	}
	return int(math.Ceil(float64(workflows) / float64(perAgent)))
}

func clamp(v, lo, hi int) int {
	if v < lo {
		return lo
	}
	if hi >= lo && v > hi {
		return hi
	}
	return v
}
