GOFILES=$(shell find . -name '*.go' -not -path './vendor/*')
GOPACKAGES = $(shell go list ./...  | grep -v /vendor/)
FNSERVER_IP?=$(shell docker inspect --type container -f '{{.NetworkSettings.IPAddress}}' fnserver)
FLOWSERVER_IP?=$(shell docker inspect --type container -f '{{.NetworkSettings.IPAddress}}' flowserver)

# Just builds
all: deploy

dep:
	dep ensure

dep-up:
	dep ensure -update

test: 
	go test -v -cover $(GOPACKAGES)

deploy:
	fn deploy --app go-flow 
	fn set config app go-flow COMPLETER_BASE_URL "http://$(FLOWSERVER_IP):8081"

deploy-local:
	fn deploy --app go-flow --local
	fn config app go-flow COMPLETER_BASE_URL "http://$(FLOWSERVER_IP):8081"

invoke:
	fn call go-flow go-flow

