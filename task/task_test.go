package task

import (
	"context"
	"errors"
	"testing"
)

func TestSimpleTask(t *testing.T) {
	task := New(context.Background(), WithFunc(func(ctx context.Context, values ...interface{}) (interface{}, error) {
		return nil, nil
	}))

	if _, err := Run([]*Task{task}); err != nil {
		t.Error("should not throw an error")
	}
}

func TestSimpleErrorTask(t *testing.T) {
	task := New(context.Background(), WithFunc(func(ctx context.Context, values ...interface{}) (interface{}, error) {
		return nil, errors.New("foobar")
	}))

	if _, err := Run([]*Task{task}); err == nil {
		t.Error("expected an error")
	}
}

func TestSimpleTaskChain(t *testing.T) {
	task := New(context.Background(), WithFunc(func(ctx context.Context, values ...interface{}) (interface{}, error) {
		return 1, nil
	}))

	foo := New(context.Background(), WithFunc(func(ctx context.Context, values ...interface{}) (interface{}, error) {
		num := values[0].(int)
		return 2 + num, nil
	}))

	bar := New(context.Background(), WithFunc(func(ctx context.Context, values ...interface{}) (interface{}, error) {
		num := values[1].(int)
		return 3 + num, nil
	}))

	foo.AddSubtasks(bar)

	task.AddSubtasks(foo)

	result, err := Run([]*Task{task})
	if err != nil {
		t.Fatal("didnt expect error")
	}
	if len(result) != 3 {
		t.Error("expected 3 results")
	}

	sum := 0
	for _, val := range result {
		num := val.(int)
		sum += num
	}

	if sum != 10 {
		t.Fatalf("expected sum of results to be %d, got %d", 10, sum)
	}
}

func TestNestedErrorTask(t *testing.T) {
	task := New(context.Background(), WithFunc(func(ctx context.Context, values ...interface{}) (interface{}, error) {
		return nil, nil
	}))

	task.AddSubtasks(New(context.Background(), WithFunc(func(ctx context.Context, values ...interface{}) (interface{}, error) {
		return nil, errors.New("nested error")
	})))

	if _, err := Run([]*Task{task}); err == nil {
		t.Error("expected an error")
	}
}

func TestLargeTasks(t *testing.T) {
	ctx := context.Background()
	mainTask := New(ctx, WithFunc(func(ctx context.Context, values ...interface{}) (interface{}, error) {
		return nil, nil
	}))

	for i := 0; i < 1000; i++ {
		subTask := New(ctx, WithFunc(func(ctx context.Context, values ...interface{}) (interface{}, error) {
			return i, nil
		}))
		mainTask.AddSubtasks(subTask)
	}

	if _, err := Run([]*Task{mainTask}); err != nil {
		t.Error("should not throw an error")
	}
}

var (
	count = 10000
)

func BenchmarkLarge(b *testing.B) {
	values := []interface{}{}

	for i := 0; i < count; i++ {
		values = append(values, i)
	}
}

func BenchmarkLargeTasks(b *testing.B) {
	ctx := context.Background()
	mainTask := New(ctx, WithFunc(func(ctx context.Context, values ...interface{}) (interface{}, error) {
		return nil, nil
	}))

	for i := 0; i < count; i++ {
		subTask := New(ctx, WithFunc(func(ctx context.Context, values ...interface{}) (interface{}, error) {
			tc := MustDecodeCtx(ctx)
			j := tc.Task.Parameters[0]

			return j, nil
		}), WithParameters(i))
		mainTask.AddSubtasks(subTask)
	}

	_, err := Run([]*Task{mainTask})
	if err != nil {
		b.Error("should not throw an error")
	}
}
