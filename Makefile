.PHONY: help build run test vet fmt tidy gen clean

BIN := bin/api

help:
	@echo "Targets:"
	@echo "  make build      Compile cmd/api into $(BIN)"
	@echo "  make run        Build and run the API server"
	@echo "  make gen        Regenerate internal/api/openapi.gen.go from api/openapi.yaml"
	@echo "  make vet        go vet ./..."
	@echo "  make fmt        gofmt -w ."
	@echo "  make test       go test ./..."
	@echo "  make tidy       go mod tidy"
	@echo "  make clean      Remove $(BIN)"

build:
	go build -o $(BIN) ./cmd/api

run: build
	./$(BIN)

gen:
	go generate ./internal/api/...

vet:
	go vet ./...

fmt:
	gofmt -w .

test:
	go test ./...

tidy:
	go mod tidy

clean:
	rm -f $(BIN)
