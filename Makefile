.PHONY: test lint fuzz build example clean

test:
	go test ./... -race -cover -count=1

lint:
	golangci-lint run

fuzz:
	go test -fuzz=FuzzParse -fuzztime=30s ./providers/...

build:
	go build -o onceo ./cmd/onceo

example:
	go run ./examples/basic

clean:
	rm -f onceo
	rm -rf dist/
