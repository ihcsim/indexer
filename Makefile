.PHONY: lint test compile build run build-server test-repeat

all: lint test compile build run

lint:
	golint .

test:
	go test -v -cover -race ./...

compile: 
	go build -v ./...

build:
	docker build --rm -t dev/indexer .

run:
	docker run --rm --name indexer -p 8080:8080 dev/indexer

coverage:
	go test -v -coverprofile indexer.cover .
	go test -v -coverprofile server.cover `go list`/cmd/server	
	go tool cover -html indexer.cover
	go tool cover -html server.cover

build-server:
	go build -o indexer `go list`/cmd/server

test-repeat:
	for ((i = 0; i < 15; i++)); do go test -v -cover -race ./... ; done
