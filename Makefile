.PHONY: build install check clean vet

all: build install

BIN=bb

build:
	go build .

install:
	go install .

check: build
	./bin/$(BIN) --check

vet:
	go vet ./...

clean:
	go clean
	rm -f bin/$(BIN)
