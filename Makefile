all: lint tidy build

lint:
	goimports -w .
	gofmt -w -s .

tidy:
	go mod tidy

build:
	CGO_ENABLED=0 go build -o bin/ .

clean:
	rm -r bin
