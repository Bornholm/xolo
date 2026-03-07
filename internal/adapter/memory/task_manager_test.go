package memory

import (
	"context"
	"sync/atomic"
	"testing"
	"time"

	"github.com/bornholm/xolo/internal/core/model"
	"github.com/bornholm/xolo/internal/core/port"
	"github.com/pkg/errors"
)

func TestTaskManager(t *testing.T) {
	tr := NewTaskRunner(10, 24*time.Hour, time.Minute)

	var executed atomic.Int64

	tr.RegisterTask("dummy", port.TaskHandlerFunc(func(ctx context.Context, task model.Task, events chan port.TaskEvent) error {
		t.Logf("[%s] start", task.ID())
		events <- port.NewTaskEvent(port.WithTaskProgress(0.1))
		events <- port.NewTaskEvent(port.WithTaskProgress(0.5))
		events <- port.NewTaskEvent(port.WithTaskProgress(1))
		t.Logf("[%s] done", task.ID())
		executed.Add(1)
		return nil
	}))

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	total := int64(100)

	for range total {
		task := &dummyTask{
			id: model.NewTaskID(),
		}
		t.Logf("Scheduling task %s", task.ID())
		tr.ScheduleTask(ctx, task)
	}

	if err := tr.Run(ctx); err != nil && !errors.Is(err, context.DeadlineExceeded) {
		t.Errorf("%+v", errors.WithStack(err))
	}

	t.Logf("executed: %d", executed.Load())

	if e, g := total, executed.Load(); e != g {
		t.Logf("executed: expected %d, got %d", e, g)
	}

	taskHeaders, err := tr.ListTasks(ctx)
	if err != nil {
		t.Fatalf("%+v", errors.WithStack(err))
	}

	if e, g := int(total), len(taskHeaders); e != g {
		t.Logf("len(taskHeaders): expected %d, got %d", e, g)
	}

	for _, header := range taskHeaders {
		state, err := tr.GetTaskState(ctx, header.ID)
		if err != nil {
			t.Fatalf("%+v", errors.WithStack(err))
		}

		if state.ScheduledAt.IsZero() {
			t.Errorf("task.ScheduledAt should not be zero value")
		}
	}
}

type dummyTask struct {
	id model.TaskID
}

// MarshalJSON implements [model.Task].
func (d *dummyTask) MarshalJSON() ([]byte, error) {
	return nil, nil
}

// UnmarshalJSON implements [model.Task].
func (d *dummyTask) UnmarshalJSON([]byte) error {
	return nil
}

// Owner implements [model.Task].
func (d *dummyTask) Owner() model.User {
	panic("unimplemented")
}

// ID implements port.Task.
func (d *dummyTask) ID() model.TaskID {
	return d.id
}

// Type implements port.Task.
func (d *dummyTask) Type() model.TaskType {
	return "dummy"
}

var _ model.Task = &dummyTask{}
