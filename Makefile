include .env
export $(shell sed 's/=.*//' .env)

NAME := joinery

all: test

test: generate
	[ -f .env ] && source .env; go test -v ./...

generate:
	golem generate

build: generate
	go build

install: build
	go install

server:
	go run main.go server

docker:
	docker build -t $(NAME) .

docker-run:
	docker run -d --rm --name $(NAME) -p $(PORT):$(PORT) $(NAME)

dotenv:
	npx @dotenvx/dotenvx encrypt

deps:
	go install golang.org/x/tools/cmd/goimports@latest
	go install github.com/dashotv/golem@latest

.PHONY: server receiver test deps docker docker-run
