package task

import (
	"context"
	"errors"
	"fmt"
	"log"
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
	Revert   TaskFunc
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

func WithRevert(f TaskFunc) TaskConfigFunc {
	return func(t *Task) {
		t.Revert = f
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

func Revert(tasks []*Task, values ...interface{}) {
	if len(tasks) == 0 || tasks == nil {
		return
	}

	errChan := make(chan error, len(tasks))
	wg := sync.WaitGroup{}

	for _, task := range tasks {
		if task.Revert == nil {
			return
		}
		wg.Add(1)
		go func(t *Task) {
			defer wg.Done()

			val, err := t.Revert(t.Context, values...)
			if err != nil {
				errChan <- err
			}
			values = append(values, val)
			if _, err := Run(t.Subtasks, values...); err != nil {
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
		}
	}

	return
}

func Run(tasks []*Task, values ...interface{}) ([]interface{}, error) {
	if len(tasks) == 0 || tasks == nil {
		return nil, nil
	}

	errChan := make(chan error, len(tasks))
	valChan := make(chan []interface{}, 4)
	wg := sync.WaitGroup{}

	for _, task := range tasks {
		wg.Add(1)
		go func(t *Task) {
			defer wg.Done()

			val, err := t.Run(t.Context, values...)
			if err != nil {
				log.Println("err ", err)
				errChan <- err
			} else {
				values = append(values, val)
				valChan <- []interface{}{val}
			}
			vals, err := Run(t.Subtasks, values...)
			if err != nil {
				errChan <- err
			} else {
				valChan <- vals
			}

		}(task)
	}

	go func() {
		wg.Wait()
		close(errChan)
		close(valChan)
	}()

	// Collect any error if they've happened
	for err := range errChan {
		if err != nil {
			Revert(tasks, values...)
			return nil, err
		}
	}

	// Initialize a slice of interface
	var results []interface{}

	for val := range valChan {
		results = append(results, val...)
	}

	// return the results
	return results, nil
}
