package main

import (
	"context"
	"github.com/codecreationlabs/async/task"
	"log"
	"time"
)

func main() {
	t := task.New(context.Background(), task.WithFunc(func(ctx context.Context, values ...interface{}) (interface{}, error) {
		tc := task.MustDecodeCtx(ctx)
		log.Println("inside ", tc.Task.ID)
		return 2, nil
	}))

	foo := task.New(context.Background(), task.WithFunc(func(ctx context.Context, values ...interface{}) (interface{}, error) {
		tc := task.MustDecodeCtx(ctx)
		log.Println("inside ", tc.Task.ID, tc.Parent.ID)
		return 123, nil
	}))
	bar := task.New(context.Background(), task.WithFunc(func(ctx context.Context, values ...interface{}) (interface{}, error) {
		tc := task.MustDecodeCtx(ctx)
		log.Println("inside ", tc.Task.ID, tc.Parent.ID)
		return nil, nil
	}))

	quz := task.New(context.Background(), task.WithFunc(func(ctx context.Context, values ...interface{}) (interface{}, error) {
		tc := task.MustDecodeCtx(ctx)
		log.Println("inside quz", tc.Task.ID, tc.Parent.ID, values)

		time.Sleep(time.Second * 2)
		return []string{"hello", "world"}, nil
	}))

	baz := task.New(context.Background(), task.WithFunc(func(ctx context.Context, values ...interface{}) (interface{}, error) {
		tc := task.MustDecodeCtx(ctx)
		log.Println("inside baz ", tc.Task.ID, tc.Parent.ID, values)

		time.Sleep(time.Second * 2)
		return nil, nil
	}))

	quz.AddSubtasks(baz)
	foo.AddSubtasks(quz)

	t.AddSubtasks(foo, bar)

	err := task.Run([]*task.Task{t})

	if err != nil {
		panic(err)
	}
}
