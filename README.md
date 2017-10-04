# FnFlow Applications for Go

## Introduction
Simply import this library into your go application, deploy it on fn, and start using the power of FnFlow.

## Getting Started
```go
package main

import (
	flows "github.com/fnproject/flow-lib-go"
)

func init() {
	flows.RegisterAction(strings.ToUpper)
	flows.RegisterAction(strings.ToLower)
}

func main() {
	flows.WithFlow(
		func() {
			cf := flows.CurrentFlow().CompletedValue("foo")
			ch := cf.ThenApply(strings.ToUpper).ThenApply(strings.ToLower).Get()
			select {
			case result := <-ch:
				if result.Err() != nil {
					fmt.Printf("GOT ERROR %v", result.Err())
				} else {
					fmt.Printf("GOT RESULT %v\n", result.Value())
				}
			case <-time.After(time.Minute * 1):
				fmt.Fprintln(os.Stderr, "timeout")
				fmt.Printf("Timed out!")
			}
		})
}
```

## More Examples

A variety of example use cases are provided [here](examples/hello-flow/README.md).
