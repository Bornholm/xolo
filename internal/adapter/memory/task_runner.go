package memory

import (
	"context"
	"log/slog"
	"sync"
	"time"

	"github.com/bornholm/xolo/internal/adapter/memory/syncx"
	"github.com/bornholm/xolo/internal/core/model"
	"github.com/bornholm/xolo/internal/core/port"
	"github.com/bornholm/go-x/slogx"
	"github.com/pkg/errors"
)

type taskEntry struct {
	Task  model.Task
	State port.TaskState
}

type TaskRunner struct {
	runningMutex *sync.Mutex
	runningCond  sync.Cond
	running      bool

	tasks      syncx.Map[model.TaskID, taskEntry]
	stateMutex sync.Mutex

	handlers  syncx.Map[model.TaskType, port.TaskHandler]
	semaphore chan struct{}

	// cancelFuncs stores cancel functions for running/pending tasks
	cancelFuncs syncx.Map[model.TaskID, context.CancelFunc]

	cleanupDelay    time.Duration
	cleanupInterval time.Duration
}

// CancelTask implements [port.TaskRunner].
func (r *TaskRunner) CancelTask(ctx context.Context, id model.TaskID) error {
	entry, exists := r.tasks.Load(id)
	if !exists {
		return errors.WithStack(port.ErrNotFound)
	}

	// Can only cancel pending or running tasks
	if entry.State.Status != port.TaskStatusPending && entry.State.Status != port.TaskStatusRunning {
		return errors.WithStack(port.ErrCanceled)
	}

	// Get the cancel function if it exists
	cancelFn, exists := r.cancelFuncs.Load(id)
	if !exists {
		// No cancel function means the task might have already finished
		// or there's no way to cancel it
		return errors.WithStack(port.ErrCanceled)
	}

	// Call the cancel function
	cancelFn()

	// Update the task state to failed with canceled error
	r.updateState(entry.Task, func(s *port.TaskState) {
		s.Error = errors.WithStack(port.ErrCanceled)
		s.Status = port.TaskStatusFailed
		s.FinishedAt = time.Now()
	})

	r.cancelFuncs.Delete(id)

	return nil
}

// GetTask implements [port.TaskRunner].
func (r *TaskRunner) GetTask(ctx context.Context, id model.TaskID) (model.Task, error) {
	entry, exists := r.tasks.Load(id)
	if !exists {
		return nil, errors.WithStack(port.ErrNotFound)
	}

	return entry.Task, nil
}

// Run implements port.TaskRunner.
func (r *TaskRunner) Run(ctx context.Context) error {
	r.runningMutex.Lock()
	r.running = true
	r.runningCond.Broadcast()
	r.runningMutex.Unlock()

	go func() {
		ticker := time.NewTicker(r.cleanupInterval)
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				slog.DebugContext(ctx, "running task cleaner")

				// Collect IDs to delete first to avoid race conditions during iteration
				var idsToDelete []model.TaskID

				r.tasks.Range(func(id model.TaskID, entry taskEntry) bool {
					if entry.State.FinishedAt.IsZero() || !time.Now().After(entry.State.FinishedAt.Add(r.cleanupDelay)) {
						return true
					}

					idsToDelete = append(idsToDelete, id)

					return true
				})

				// Now delete them
				for _, id := range idsToDelete {
					slog.DebugContext(ctx, "deleting expired task", slog.String("taskID", string(id)))
					r.tasks.Delete(id)
					r.cancelFuncs.Delete(id)
				}
			}
		}
	}()

	<-ctx.Done()

	if err := ctx.Err(); err != nil {
		return errors.WithStack(err)
	}

	return nil
}

// ListTasks implements port.TaskRunner.
func (r *TaskRunner) ListTasks(ctx context.Context) ([]port.TaskStateHeader, error) {
	headers := make([]port.TaskStateHeader, 0)
	r.tasks.Range(func(id model.TaskID, entry taskEntry) bool {
		headers = append(headers, entry.State.TaskStateHeader)
		return true
	})
	return headers, nil
}

// RegisterTask implements port.TaskRunner.
func (r *TaskRunner) RegisterTask(taskType model.TaskType, handler port.TaskHandler) {
	r.handlers.Store(taskType, handler)
}

// ScheduleTask implements port.TaskRunner.
func (r *TaskRunner) ScheduleTask(ctx context.Context, task model.Task) error {
	taskID := task.ID()

	ctx = slogx.WithAttrs(ctx,
		slog.String("taskID", string(taskID)),
		slog.String("taskType", string(task.Type())),
	)

	r.updateState(task, func(s *port.TaskState) {
		s.ID = taskID
		s.ScheduledAt = time.Now()
		s.Status = port.TaskStatusPending
		s.Type = task.Type()
	})

	// Create a cancelable context for this task
	taskCtx, cancelFn := context.WithCancel(context.Background())
	r.cancelFuncs.Store(taskID, cancelFn)

	go func() {
		defer func() {
			// Clean up the cancel function when the task finishes
			r.cancelFuncs.Delete(taskID)

			if recovered := recover(); recovered != nil {
				err, ok := recovered.(error)
				if !ok {
					err = errors.Errorf("%+v", recovered)
				}

				slog.ErrorContext(ctx, "recovered panic while running task", slog.Any("error", errors.WithStack(err)))

				r.updateState(task, func(s *port.TaskState) {
					s.Error = errors.WithStack(err)
					s.Status = port.TaskStatusFailed
				})
			}
		}()

		r.runningMutex.Lock()
		for !r.running {
			r.runningCond.Wait()
		}
		r.runningMutex.Unlock()

		r.semaphore <- struct{}{}
		defer func() {
			<-r.semaphore
		}()

		handler, exists := r.handlers.Load(task.Type())
		if !exists {
			r.updateState(task, func(s *port.TaskState) {
				s.Error = errors.Errorf("no handler registered for task type '%s'", task.Type())
				s.Status = port.TaskStatusFailed
			})

			return
		}

		r.updateState(task, func(s *port.TaskState) {
			s.Status = port.TaskStatusRunning
		})

		events := make(chan port.TaskEvent)

		var eventsWg sync.WaitGroup
		eventsWg.Add(1)
		go func() {
			defer eventsWg.Done()
			for e := range events {
				r.updateState(task, func(s *port.TaskState) {
					if e.Progress != nil {
						s.Progress = float32(max(min(*e.Progress, 1), 0))
					}
					if e.Message != nil {
						s.Message = *e.Message
					}
				})
			}
		}()

		start := time.Now()

		taskCtx := slogx.WithAttrs(taskCtx,
			slog.String("taskID", string(taskID)),
			slog.String("taskType", string(task.Type())),
		)

		slog.DebugContext(taskCtx, "executing task")

		err := handler.Handle(taskCtx, task, events)

		// Check if the task was canceled
		if errors.Is(err, port.ErrCanceled) {
			slog.DebugContext(ctx, "task was canceled")

			r.updateState(task, func(s *port.TaskState) {
				s.Error = errors.WithStack(port.ErrCanceled)
				s.Status = port.TaskStatusFailed
				s.FinishedAt = time.Now()
			})

			// Close the events channel and wait for the events goroutine
			close(events)
			eventsWg.Wait()

			return
		}

		// Close the events channel and wait for the events goroutine to fully
		// drain before writing the final task state, eliminating the race between
		// the events goroutine and the final updateState call.
		close(events)
		eventsWg.Wait()

		if err != nil {
			err = errors.WithStack(err)
			slog.ErrorContext(ctx, "task failed", slog.Any("error", err))

			r.updateState(task, func(s *port.TaskState) {
				s.Error = err
				s.Status = port.TaskStatusFailed
				s.FinishedAt = time.Now()
			})
			return
		}

		slog.DebugContext(ctx, "task finished", slog.Duration("duration", time.Since(start)))

		r.updateState(task, func(s *port.TaskState) {
			s.Status = port.TaskStatusSucceeded
			s.FinishedAt = time.Now()
			s.Progress = 1
		})
	}()
	return nil
}

func (r *TaskRunner) updateState(task model.Task, fn func(s *port.TaskState)) {
	r.stateMutex.Lock()
	defer r.stateMutex.Unlock()

	entry, _ := r.tasks.LoadOrStore(task.ID(), taskEntry{
		Task: task,
		State: port.TaskState{
			TaskStateHeader: port.TaskStateHeader{
				ID: task.ID(),
			},
		},
	})

	fn(&entry.State)

	r.tasks.Store(task.ID(), entry)
}

// GetTaskState implements port.TaskRunner.
func (r *TaskRunner) GetTaskState(ctx context.Context, id model.TaskID) (*port.TaskState, error) {
	entry, exists := r.tasks.Load(id)
	if !exists {
		return nil, errors.WithStack(port.ErrNotFound)
	}

	return &entry.State, nil
}

func NewTaskRunner(parallelism int, cleanupDelay time.Duration, cleanupInterval time.Duration) *TaskRunner {
	runningMutex := &sync.Mutex{}
	return &TaskRunner{
		runningMutex:    runningMutex,
		runningCond:     *sync.NewCond(runningMutex),
		running:         false,
		semaphore:       make(chan struct{}, parallelism),
		tasks:           syncx.Map[model.TaskID, taskEntry]{},
		handlers:        syncx.Map[model.TaskType, port.TaskHandler]{},
		cleanupDelay:    cleanupDelay,
		cleanupInterval: cleanupInterval,
	}
}

var _ port.TaskRunner = &TaskRunner{}
