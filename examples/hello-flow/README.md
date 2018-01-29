# Writing Flow Applications in Go

## Prerequisites
```
# ensure you have the latest images
$ docker pull fnproject/fnserver:latest
$ docker pull fnproject/flow:latest

# ensure you have the latest fn CLI
$ curl -LSs https://raw.githubusercontent.com/fnproject/cli/master/install | sh
```

## Start Services
```
# start the fn server
$ (fn start > /dev/null 2>&1 &)

sleep 5

# start the Flow Service and point it at the functions server API URL
$ DOCKER_LOCALHOST=$(docker inspect --type container -f '{{.NetworkSettings.Gateway}}' fnserver)

$ docker run --rm  \
       -p 8081:8081 \
       -d \
       -e API_URL="http://$DOCKER_LOCALHOST:8080/r" \
       -e no_proxy=$DOCKER_LOCALHOST \
       --name flow-service \
       fnproject/flow:latest
```

## Deploy Example

Deploy the example application to the functions server:
```
make dep-up deploy
```

## Invoke Example

You are now ready to invoke the example:
```
fn call go-flow hello-flow/
```
You should be able to see the following output: _Flow succeeded with value foo_
