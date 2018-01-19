GOFILES = $(shell find . -name '*.go' -not -path './vendor/*')
GOPACKAGES = $(shell go list ./...  | grep -v /vendor/)

# Just builds
all: test build

dep:
	dep ensure

dep-up:
	dep ensure -update

test: 
	go test -v -cover $(GOPACKAGES)

build:  $(GOFILES)
	go build
