package task

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"sync/atomic"
)

var (
	counter atomic.Int64
)

type TaskConfigFunc func(t *Task)

type TaskFunc func(ctx context.Context, values ...interface{}) (interface{}, error)

type CtxKey string

type Task struct {
	ID       string
	Context  context.Context
	Subtasks []*Task
	Run      TaskFunc
}

type TaskContext struct {
	Parent *Task
	Task   *Task
}

func MustDecodeCtx(ctx context.Context) *TaskContext {
	tc, err := DecodeCtx(ctx)
	if err != nil {
		panic(err)
	}
	return tc
}

func DecodeCtx(ctx context.Context) (*TaskContext, error) {
	tc, ok := ctx.Value(CtxKey("ctx")).(*TaskContext)
	if !ok {
		return nil, errors.New("no context found")
	}
	return tc, nil
}

func New(ctx context.Context, cfgs ...TaskConfigFunc) *Task {
	taskID := fmt.Sprintf("task_%d", counter.Load())

	t := &Task{
		ID: taskID,
	}

	for _, cfg := range cfgs {
		cfg(t)
	}

	valueContext := context.WithValue(ctx, CtxKey("ctx"), &TaskContext{
		Task: t,
	})
	t.Context = valueContext

	counter.Add(1)

	return t
}

func WithFunc(f TaskFunc) TaskConfigFunc {
	return func(t *Task) {
		t.Run = f
	}
}

func (t *Task) AddSubtasks(st ...*Task) {
	for _, subtask := range st {
		subtask.Context = context.WithValue(t.Context, CtxKey("ctx"), &TaskContext{
			Task:   subtask,
			Parent: t,
		})
	}
	t.Subtasks = append(t.Subtasks, st...)
}

func Run(tasks []*Task, values ...interface{}) error {
	if len(tasks) == 0 || tasks == nil {
		return nil
	}

	errChan := make(chan error)
	wg := sync.WaitGroup{}

	for _, task := range tasks {
		wg.Add(1)
		go func(t *Task) {
			defer wg.Done()

			// TODO make this call external runnable
			val, err := t.Run(t.Context, values...)
			if err != nil {
				errChan <- err
			}
			values = append(values, val)
			if err := Run(t.Subtasks, values...); err != nil {
				errChan <- err
			}

			errChan <- nil
		}(task)
	}

	go func() {
		wg.Wait()
		close(errChan)
	}()

	// Collect any error if they've happened
	for err := range errChan {
		if err != nil {
			return err
		}
	}
	return nil
}
