all: fmt build

fmt:
	find . -name '*.go' -exec gofmt -e -s -w {} +

build:
	CGO_ENABLED=0 go build -o bin/graphql_exporter

clean:
	rm -r bin
