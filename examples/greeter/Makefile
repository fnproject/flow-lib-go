GOFILES=$(shell find . -name '*.go' -not -path './vendor/*')
GOPACKAGES=$(shell go list ./...  | grep -v /vendor/)
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
	fn --verbose deploy --app greeter

deploy-local:
	fn --verbose deploy --app greeter --local

invoke:
	fn invoke greeter greeter

