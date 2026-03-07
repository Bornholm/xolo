package port

import (
	"context"
	"time"

	"github.com/bornholm/xolo/internal/core/model"
)

type TaskStatus string

const (
	TaskStatusPending   = "pending"
	TaskStatusRunning   = "running"
	TaskStatusSucceeded = "succeeded"
	TaskStatusFailed    = "failed"
)

type TaskStateHeader struct {
	ID          model.TaskID
	Type        model.TaskType
	ScheduledAt time.Time
	Status      TaskStatus
}

type TaskState struct {
	TaskStateHeader
	FinishedAt time.Time
	Progress   float32
	Error      error
	Message    string
}

type TaskEvent struct {
	Message  *string
	Progress *float32
}

type TaskEventFunc func(p *TaskEvent)

func WithTaskMessage(message string) TaskEventFunc {
	return func(p *TaskEvent) {
		p.Message = &message
	}
}

func WithTaskProgress(progress float32) TaskEventFunc {
	return func(p *TaskEvent) {
		p.Progress = &progress
	}
}

func NewTaskEvent(funcs ...TaskEventFunc) TaskEvent {
	p := TaskEvent{}
	for _, fn := range funcs {
		fn(&p)
	}
	return p
}

type TaskHandler interface {
	Handle(ctx context.Context, task model.Task, events chan TaskEvent) error
}

type TaskHandlerFunc func(ctx context.Context, task model.Task, events chan TaskEvent) error

func (f TaskHandlerFunc) Handle(ctx context.Context, task model.Task, events chan TaskEvent) error {
	return f(ctx, task, events)
}

type TaskRunner interface {
	ScheduleTask(ctx context.Context, task model.Task) error
	GetTaskState(ctx context.Context, id model.TaskID) (*TaskState, error)
	GetTask(ctx context.Context, id model.TaskID) (model.Task, error)
	ListTasks(ctx context.Context) ([]TaskStateHeader, error)
	RegisterTask(taskType model.TaskType, handler TaskHandler)
	// CancelTask cancels a scheduled or running task
	// A canceled task should return the error ErrCanceled
	CancelTask(ctx context.Context, id model.TaskID) error
	Run(ctx context.Context) error
}
