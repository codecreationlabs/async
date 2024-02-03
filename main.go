package main

import (
	"context"
	"fmt"
	"github.com/codecreationlabs/async/task"
	"log"
	"time"
)

type CreateUserParams struct {
	Name string
}

type User struct {
	ID        string
	Name      string
	Processed bool
	CreatedAt string
	UpdatedAt string
}

func main() {
	params := CreateUserParams{
		Name: "Foobar Quz",
	}

	foo := task.New(context.Background(), task.WithFunc(func(ctx context.Context, values ...interface{}) (interface{}, error) {
		tc := task.MustDecodeCtx(ctx)

		params := tc.Task.Parameters[0].(CreateUserParams)

		// create user
		now := time.Now().Format(time.RFC3339)
		user := User{
			ID:        "foobar",
			Name:      params.Name,
			Processed: false,
			CreatedAt: now,
			UpdatedAt: now,
		}
		log.Printf("create user.. %v \n", user)

		return user, nil
	}), task.WithRevertFunc(func(ctx context.Context, values ...interface{}) (interface{}, error) {
		// delete user
		log.Printf("rollback and delete user.. %v \n", values)
		return nil, nil
	}), task.WithParameters(params))

	quz := task.New(context.Background(), task.WithFunc(func(ctx context.Context, values ...interface{}) (interface{}, error) {
		log.Printf("prepare processing %v ..\n", values)

		user := values[0].(User)
		user.Processed = true
		user.UpdatedAt = time.Now().Format(time.RFC3339)

		return user, nil
	}))

	bar := task.New(context.Background(), task.WithFunc(func(ctx context.Context, values ...interface{}) (interface{}, error) {
		// process user
		user := values[1].(User)
		log.Printf("process user.. %v \n", user)

		if user.ID != "quzbuz" {
			return nil, fmt.Errorf("expected user id to be %s, got %s", "foobar", "quzbuz")
		}

		return nil, nil
	}))

	foo.AddSubtasks(quz, bar)
	if _, err := task.Run([]*task.Task{foo}); err != nil {
		panic(err)
	}
}
