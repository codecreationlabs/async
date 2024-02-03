dev:
	go run .
test:
	go test ./... -v
bench:
	go test -bench=. ./... -v