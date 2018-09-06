# Serverless Workflows with Go
Easily create serverless workflows directly in Go with the power of [Fn Flow](https://github.com/fnproject/flow).

## Quick Intro
Simply import this library into your go function, build and deploy onto Fn. Flows use the [fdk-go](https://github.com/fnproject/fdk-go) to handle interacting with Fn, below is an example flow:

```go
package main

import (
	"fmt"
	"strings"
	"time"

  	fdk "github.com/fnproject/fdk-go"
  	flows "github.com/fnproject/flow-lib-go"
)

func init() {
  	flows.RegisterAction(strings.ToUpper)
  	flows.RegisterAction(strings.ToLower)
}

func main() {
	fdk.Handle(flows.WithFlow(
    		fdk.HandlerFunc(func(ctx context.Context, r io.Reader, w io.Writer) {
      			cf := flows.CurrentFlow().CompletedValue("foo")
      			valueCh, errorCh := cf.ThenApply(strings.ToUpper).ThenApply(strings.ToLower).Get()
      			select {
      			case value := <-valueCh:
        			fmt.Fprintf(w, "Flow succeeded with value %v", value)
      			case err := <-errorCh:
        			fmt.Fprintf(w, "Flow failed with error %v", err)
      			case <-time.After(time.Minute * 1):
        			fmt.Fprintf(w, "Timed out!")
      			}
    		}),
  	)
}
```

## Where do I go from here?

A variety of example use-cases is provided [here](examples/hello-flow/README.md).

## FAQs

### How are values serialized?

Go's [gob](https://golang.org/pkg/encoding/gob/) serialization mechanism is used to encode/decode values for communication with the completer.

### What kinds of values can be serialized?

Booleans, string, structs, arrays and slices are supported. Functions, closures and channels are not.

### How are continuations serialized?

Since Go does not support [serializing closures/functions](https://github.com/golang/go/issues/5514) due to its statically compiled nature, they are in fact not serialized at all. Go functions implementing a continuation need to be explicitly registered by calling `flows.RegisterAction(actionFunction)` typically inside the handler's _init_ function. Registering actions assigns a unique and stable key that can be serialized and used to look up a pointer to the function during a continuation invocation.

### Why do actions need to be registered?

See above.

### Can I use closures or method receivers in my continuations?

No. Only continuation actions implemented with functions are supported, since they are stateless. No state will be serialized with a continuation. Although possible, invoking a method receiver is not currently supported inside continuations.

### How does error-handling work?

Go allows functions to return error types in addition to a result via its support for multiple return values. If a continuation function returns a (non-nil) error as its second return value, its error message will be serialized and form the failed value of that stage.

If a panic occurs while invoking the continuation function, the panic value will be captured and the stage failed with the same value.

### Can I invoke other fn functions?

Yes. `flows.CurrentFlow().InvokeFunction("myapp/myfn", req)`. See [here](examples/hello-flow/func.go) for a full example.
