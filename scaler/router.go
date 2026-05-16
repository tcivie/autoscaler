package scaler

// Router assigns each task to exactly one pool. Pools with a specific selector
// have priority; catch-all pools own whatever no specific selector claimed.
type Router struct {
	pools []Pool
}

func NewRouter(pools []Pool) *Router { return &Router{pools: pools} }

// Pools returns the configured pools in the original declaration order.
func (r *Router) Pools() []Pool { return r.pools }

// Bucket returns one slice of task counts aligned with Pools(): bucket[i] is
// the number of tasks owned by Pools()[i].
func (r *Router) Bucket(tasks []Task) []int {
	counts := make([]int, len(r.pools))
	for _, t := range tasks {
		idx := r.poolFor(t.Labels)
		if idx >= 0 {
			counts[idx]++
		}
	}
	return counts
}

func (r *Router) poolFor(labels map[string]string) int {
	specific := -1
	catchAll := -1
	for i, p := range r.pools {
		if p.Selector.IsCatchAll() {
			if catchAll == -1 {
				catchAll = i
			}
			continue
		}
		if p.Selector.Matches(labels) {
			specific = i
			break
		}
	}
	if specific >= 0 {
		return specific
	}
	return catchAll
}

// Task is the minimal queue entry surface the router needs.
type Task struct {
	Labels map[string]string
}
