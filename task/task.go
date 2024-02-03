package task

import (
	"context"
	"errors"
	"fmt"
	"sync/atomic"
)

// counter is a variable of type atomic.Int64 that keeps track of the number of tasks created. It is used to assign a unique ID to each new task that is created. The counter is incremented
var (
	counter atomic.Int64
)

// TaskConfigFunc represents a function that can be used to configure a Task. It takes a pointer to a Task as its parameter and sets various fields of the Task.
type TaskConfigFunc func(t *Task)

// TaskFunc represents a function that can be executed as a task. It takes a context and a variadic number of values as input, and returns an interface and an error as output.
type TaskFunc func(ctx context.Context, values ...interface{}) (interface{}, error)

// CtxKey represents a key for retrieving values from Go context.
type CtxKey string

// Task represents a unit of work that can be executed and reverted.
//
// Members:
// - ID: the unique identifier of the task
// - Context: the context in which the task runs
// - Subtasks: the list of subtasks that are dependent on this task
// - Run: the function that performs the task
// - Revert: the function that reverts the task
type Task struct {
	ID         string
	Parameters []interface{}
	Context    context.Context
	Subtasks   []*Task
	Run        TaskFunc
	Revert     TaskFunc
}

// TaskContext represents the context of a task and its parent task.
type TaskContext struct {
	Parent *Task
	Task   *Task
}

// MustDecodeCtx takes a context and attempts to decode it into a TaskContext. If decoding fails, it panics.
// It returns the decoded TaskContext.
// It is assumed that the context contains a value of type *TaskContext, stored under the key "ctx".
// If the value is not found or the type assertion fails, it returns an error.
//
// Example usage:
//
//	ctx := context.WithValue(parentContext, CtxKey("ctx"), &TaskContext{Task: task, Parent: parentTask})
//	tc := MustDecodeCtx(ctx)
func MustDecodeCtx(ctx context.Context) *TaskContext {
	tc, err := DecodeCtx(ctx)
	if err != nil {
		panic(err)
	}
	return tc
}

// DecodeCtx decodes the TaskContext from a given context.Context. It retrieves the TaskContext by searching for the value associated with the CtxKey("ctx") key in the context. If the
func DecodeCtx(ctx context.Context) (*TaskContext, error) {
	tc, ok := ctx.Value(CtxKey("ctx")).(*TaskContext)
	if !ok {
		return nil, errors.New("no context found")
	}
	return tc, nil
}

// New creates a new Task with the given context and configuration functions.
// It generates a unique ID for the task, initializes the task with the provided configuration functions,
// creates a new value context with the task, increments the counter, and returns the created task.
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

// WithFunc is a function that takes a TaskFunc as an argument and returns a TaskConfigFunc. A TaskConfigFunc modifies a Task object by setting its Run function to the provided Task
func WithFunc(f TaskFunc) TaskConfigFunc {
	return func(t *Task) {
		t.Run = f
	}
}

// WithRevertFunc is a function that returns a TaskConfigFunc which sets the Revert function of a Task struct to the provided TaskFunc.
// The Revert function is meant to handle the reversal of actions performed by the Run function of the Task.
func WithRevertFunc(f TaskFunc) TaskConfigFunc {
	return func(t *Task) {
		t.Revert = f
	}
}

// WithParameters takes a variadic number of parameters and returns a TaskConfigFunc.
func WithParameters(parameters ...interface{}) TaskConfigFunc {
	return func(t *Task) {
		t.Parameters = parameters
	}
}

// AddSubtasks adds subtasks to the task.
// Each subtask is given a new context derived from the parent task's context using context.WithValue.
// The value associated with the key "ctx" in the parent context is set to a TaskContext struct that contains a reference to the parent task and the subtask.
// The subtasks are then appended to the task's Subtasks slice.
func (t *Task) AddSubtasks(st ...*Task) {
	for _, subtask := range st {
		subtask.Context = context.WithValue(t.Context, CtxKey("ctx"), &TaskContext{
			Task:   subtask,
			Parent: t,
		})
	}
	t.Subtasks = append(t.Subtasks, st...)
}

// Revert iterates over a list of tasks and calls their Revert functions in reverse order.
// It takes a slice of tasks and optional values as arguments.
// The Revert function of each task is called with the provided values.
// If an error occurs during the Revert call, it currently does not handle the error.
// The function also recursively adds the subtasks of each task to the task list.
func Revert(tasks []*Task, values ...interface{}) {
	for len(tasks) > 0 {
		task := tasks[0]
		tasks = tasks[1:]

		if task.Revert != nil {
			_, err := task.Revert(task.Context, values...)
			if err != nil {
				// TODO
			}
		}

		tasks = append(tasks, task.Subtasks...)
	}
}

// Run executes a list of tasks in parallel, returning the results and an error if any task fails.
//
// The function takes a slice of pointers to Task structs and variadic arguments representing the initial input values.
//
// Each task in the list is executed by calling its Run method with the provided values.
// If a task returns an error, the function will attempt to revert the changes made by the tasks that have already succeeded,
// by calling their Revert methods in reverse order. The original input values are passed to the Revert methods.
// If an error occurs during the revert process, it is currently not handled and needs to be implemented.
//
// The return value is a slice of the output values produced by each task. If all tasks succeed, the returned error is nil.
//
// Example usage:
//
//	func TestSimpleTask(t *testing.T) {
//		task := New(context.Background(), WithFunc(func(ctx context.Context, values ...interface{}) (interface{}, error) {
//			return nil, nil
//		}))
//
//		if _, err := Run([]*Task{task}); err != nil {
//			t.Error("should not throw an error")
//		}
//	}
//
// task.Run example:
//
//	foo := task.New(context.Background(), task.WithFunc(func(ctx context.Context, values ...interface{}) (interface{}, error) {
//		// create user
//		log.Printf("create user.. \n")
//		u := User{
//			ID: "foobar",
//		}
//		return u, nil
//	}), task.WithRevertFunc(func(ctx context.Context, values ...interface{}) (interface{}, error) {
//
//		// delete user
//		log.Printf("rollback and delete user.. %v \n", values)
//		return nil, nil
//	}))
//
//	quz := task.New(context.Background(), task.WithFunc(func(ctx context.Context, values ...interface{}) (interface{}, error) {
//		log.Printf("prepare processing ..\n")
//		return nil, nil
//	}))
//
//	bar := task.New(context.Background(), task.WithFunc(func(ctx context.Context, values ...interface{}) (interface{}, error) {
//		// process user
//		log.Printf("process user.. \n")
//
//		user := values[0].(User)
//
//		if user.ID != "quzbuz" {
//			return nil, fmt.Errorf("expected user id to be %s, got %s", "foobar", "quzbuz")
//		}
//
//		// what happens if we have an error here?
//		return nil, nil
//	}), task.WithRevertFunc(func(ctx context.Context, values ...interface{}) (interface{}, error) {
//
//		log.Printf("rollback anything we did while processing..")
//		return nil, nil
//	}))
//
// foo.AddSubtasks(quz, bar)
//
//	if _, err := task.Run([]*task.Task{foo}); err != nil {
//		panic(err)
//	}
func Run(tasks []*Task, values ...interface{}) ([]interface{}, error) {
	result := make([]interface{}, 0, len(tasks))
	successfulTasks := make([]*Task, 0, len(tasks))

	for len(tasks) > 0 {
		task := tasks[0]
		tasks[0] = nil // Clear the pointer for garbage collection
		tasks = tasks[1:]

		val, err := task.Run(task.Context, values...)
		if err != nil {
			Revert(successfulTasks, values...)
			return nil, err
		}
		values = append(values, val)
		result = append(result, val)

		// prepend task to successfulTasks with minimal reallocation
		successfulTasks = append(successfulTasks[:1], successfulTasks...)
		successfulTasks[0] = task

		// append subtasks to tasks
		tasks = append(tasks, task.Subtasks...)
	}

	return result, nil
}
