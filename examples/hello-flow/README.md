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
(fn start > /dev/null 2>&1 &)

sleep 5

# start the Flow Service and point it at the functions server API URL
FNSERVER_IP=$(docker inspect --type container -f '{{.NetworkSettings.IPAddress}}' fnserver)

docker run --rm -d \
     -p 8081:8081 \
     -e API_URL="http://$FNSERVER_IP:8080/invoke" \
     -e no_proxy=$FNSERVER_IP \
     --name flowserver \
     fnproject/flow:latest
```

## Deploy Example

Install [golang dep](https://github.com/golang/dep) if you haven't already.

Deploy the example application to the functions server:
```
make dep-up deploy-local
```

## Invoke Example

You are now ready to invoke the example:
```
fn call go-flow hello-flow/
```
You should be able to see the following output: _Flow succeeded with value foo_
