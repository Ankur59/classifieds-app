build:
	go build -o bin/main ./cmd/api
run:
	./bin/main
test:
	go test -race ./...
vet:
	go vet ./...
migrate-up:
	go run ./cmd/migration/main.go up
migrate-down:
	go run ./cmd/migration/main.go down $(N)

	 