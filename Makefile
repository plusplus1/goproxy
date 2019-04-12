.PHONY: build generate image clean test

export GO111MODULE=on

all: build

build: 
	@go build -o bin/goproxy -ldflags "-s -w" .

generate:
	@go generate
	@go mod tidy

image:
	@docker build -t goproxy/goproxy .

test:
	@go test -v `(go list ./... | grep "pkg/proxy")`

clean:
	@git clean -f -d -X
