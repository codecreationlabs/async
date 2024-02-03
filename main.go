package main

import (
	"context"
	"fmt"
	"github.com/codecreationlabs/async/task"
	"log"
	"time"
)

type User struct {
	ID        string
	UpdatedAt string
}

func main() {
	params := map[string]string{
		"foo": "bar",
	}

	foo := task.New(context.Background(), task.WithFunc(func(ctx context.Context, values ...interface{}) (interface{}, error) {
		tc := task.MustDecodeCtx(ctx)

		log.Println(tc.Task.Values)

		// get my vars
		// create user
		log.Printf("create user.. \n")
		u := User{
			ID: "foobar",
		}
		return u, nil
	}), task.WithRevert(func(ctx context.Context, values ...interface{}) (interface{}, error) {
		// delete user
		log.Printf("rollback and delete user.. %v \n", values)
		return nil, nil
	}), task.WithValues(params))

	quz := task.New(context.Background(), task.WithFunc(func(ctx context.Context, values ...interface{}) (interface{}, error) {
		log.Printf("prepare processing %v ..\n", values)

		user := values[0].(User)

		user.UpdatedAt = time.Now().Format(time.RFC3339)

		return user, nil
	}))

	bar := task.New(context.Background(), task.WithFunc(func(ctx context.Context, values ...interface{}) (interface{}, error) {
		// process user
		log.Printf("process user.. %v \n", values)

		user := values[0].(User)

		if user.ID != "quzbuz" {
			return nil, fmt.Errorf("expected user id to be %s, got %s", "foobar", "quzbuz")
		}

		// what happens if we have an error here?
		return nil, nil
	}), task.WithRevert(func(ctx context.Context, values ...interface{}) (interface{}, error) {
		log.Printf("rollback anything we did while processing..")
		return nil, nil
	}))

	foo.AddSubtasks(quz, bar)
	if _, err := task.Run([]*task.Task{foo}); err != nil {
		panic(err)
	}
}
