package scaler

import (
	"fmt"

	"go.woodpecker-ci.org/woodpecker/v3/woodpecker-go/woodpecker"
)

// QueueSource exposes the in-flight workflow tasks the scaler needs to make
// decisions. The interface stays narrow so test doubles are trivial.
type QueueSource interface {
	Tasks() ([]Task, error)
}

// WoodpeckerQueue adapts a Woodpecker client to QueueSource.
type WoodpeckerQueue struct {
	client woodpecker.Client
}

func NewWoodpeckerQueue(client woodpecker.Client) *WoodpeckerQueue {
	return &WoodpeckerQueue{client: client}
}

func (q *WoodpeckerQueue) Tasks() ([]Task, error) {
	info, err := q.client.QueueInfo()
	if err != nil {
		return nil, fmt.Errorf("queue info: %w", err)
	}
	tasks := make([]Task, 0, len(info.Pending)+len(info.Running))
	for _, t := range info.Pending {
		tasks = append(tasks, Task{Labels: t.Labels})
	}
	for _, t := range info.Running {
		tasks = append(tasks, Task{Labels: t.Labels})
	}
	return tasks, nil
}
