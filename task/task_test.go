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
