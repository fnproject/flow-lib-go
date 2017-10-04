GOFILES = $(shell find . -name '*.go' -not -path './vendor/*')
GOPACKAGES = $(shell go list ./...  | grep -v /vendor/)
DOCKER_LOCALHOST = $(shell docker inspect --type container -f '{{.NetworkSettings.Gateway}}' functions | tr -d '\n')
COMPLETER_SERVER_IP=`docker inspect --type container -f '{{range .NetworkSettings.Networks}}{{.IPAddress}}{{end}}' completer`

# Just builds
all: deploy

dep: glide.yaml
	glide install --strip-vendor

dep-up:
	glide up --strip-vendor

test: 
	go test -v -cover $(GOPACKAGES)

deploy:
	fn deploy --app go-flow --local
	fn apps config set go-flow COMPLETER_BASE_URL "http://$(DOCKER_LOCALHOST):8081"

call:
	fn call go-flow hello-flow/
