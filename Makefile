GOFILES = $(shell find . -name '*.go' -not -path './vendor/*')
GOPACKAGES = $(shell go list ./...  | grep -v /vendor/)

# Just builds
all: test build

dep: glide.yaml
	glide install --strip-vendor

dep-up:
	glide up --strip-vendor

test: 
	go test -v $(GOPACKAGES)

build:  $(GOFILES)
	go build -o go-cloudthreads

