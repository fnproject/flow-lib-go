package main

import (
	"errors"
	"fmt"
	"os"
	"reflect"
	"strings"
	"time"

	flows "github.com/gviedma/flow-lib-go"
)

func init() {
	flows.Debug(true)
	flows.RegisterAction(strings.ToUpper)
	flows.RegisterAction(strings.ToLower)
	flows.RegisterAction(FooToUpper)
	flows.RegisterAction(ComposedFunc)
	flows.RegisterAction(EmptyFunc)
	flows.RegisterAction(TransformExternalRequest)
	flows.RegisterAction(FailedFunc)
	flows.RegisterAction(HandleFunc)
}

func main() {
	stringExample()
	//errorValueExample()
	//errorFuncExample()
	//structExample()
	//composedExample()
	//delayExample()
	//invokeExample()
	//externalExample()
}

func stringExample() {
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

func errorValueExample() {
	flows.WithFlow(
		func() {
			cf := flows.CurrentFlow().CompletedValue(errors.New("foo"))
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

func FailedFunc(arg string) (string, error) {
	return arg + " is ignored", errors.New("failedFunc")
}

func HandleFunc(arg string, err error) string {
	flows.Log(fmt.Sprintf("Got arg %v and err %v", arg, err))
	if err != nil {
		return "Got error " + err.Error()
	} else {
		return "Got success " + arg
	}
}

func errorFuncExample() {
	flows.WithFlow(
		func() {
			cf := flows.CurrentFlow().CompletedValue("hello")
			ch := cf.ThenApply(FailedFunc).Handle(HandleFunc).Get()
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

type foo struct {
	Name string
}

func FooToUpper(f *foo) *foo {
	f.Name = strings.ToUpper(f.Name)
	return f
}

func structExample() {
	flows.WithFlow(
		func() {
			cf := flows.CurrentFlow().CompletedValue(&foo{Name: "foo"})
			ch := cf.ThenApply(FooToUpper).Get()
			select {
			case result := <-ch:
				if result.Err() != nil {
					fmt.Printf("GOT ERROR %v", result.Err())
				} else {
					fmt.Printf("GOT RESULT %v\n", result.Value())
				}
			case <-time.After(time.Minute * 1):
				fmt.Printf("Timed out!")
			}
		})
}

func EmptyFunc() string {
	return "empty func"
}

func delayExample() {
	flows.WithFlow(
		func() {
			cf := flows.CurrentFlow().Delay(5 * time.Second).ThenApply(EmptyFunc)
			ch := cf.Get()
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

func invokeExample() {
	flows.WithFlow(
		func() {
			req := &flows.HTTPRequest{Method: "POST", Body: []byte("payload")}
			cf := flows.CurrentFlow().InvokeFunction("foo/foofn", req)
			ch := cf.Get()
			select {
			case result := <-ch:
				if result.Err() != nil {
					fmt.Printf("GOT ERROR %v", result.Err())
				} else {
					resp := result.Value().(*flows.HTTPResponse)
					fmt.Printf("GOT RESULT %v\n", string(resp.Body))
				}
			case <-time.After(time.Minute * 1):
				fmt.Fprintln(os.Stderr, "timeout")
				fmt.Printf("Timed out!")
			}
		})
}

func TransformExternalRequest(req *flows.HTTPRequest) string {
	result := "Hello " + string(req.Body)
	flows.Log(fmt.Sprintf("Got result %s", result))
	return result
}

func externalExample() {
	flows.WithFlow(
		func() {
			f := flows.CurrentFlow().ExternalFuture()
			fmt.Printf("Post your payload to %v\n", f.CompletionURL())
			f.ThenApply(TransformExternalRequest)
		})
}

func ComposedFunc(msg string) flows.FlowFuture {
	return flows.CurrentFlow().CompletedValue("Hello " + msg)
}

func composedExample() {
	flows.WithFlow(
		func() {
			cf := flows.CurrentFlow().CompletedValue("foo")
			ch := cf.ThenCompose(ComposedFunc).GetType(reflect.TypeOf(""))

			select {
			case result := <-ch:
				if result.Err() != nil {
					fmt.Printf("GOT ERROR %v", result.Err())
				} else {
					res := result.Value().(string)
					fmt.Printf("GOT RESULT %v\n", res)
				}
			case <-time.After(time.Minute * 1):
				fmt.Fprintln(os.Stderr, "timeout")
				fmt.Printf("Timed out!")
			}
		})
}
