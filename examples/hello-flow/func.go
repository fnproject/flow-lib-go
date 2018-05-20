package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"reflect"
	"strings"
	"time"

	flows "github.com/fnproject/flow-lib-go"
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
	//completeExample()
}

func stringExample() {
	flows.WithFlow(
		func() {
			cf := flows.CurrentFlow().CompletedValue("foo")
			valueCh, errorCh := cf.ThenApply(strings.ToUpper).ThenApply(strings.ToLower).Get()
			printResult(valueCh, errorCh)
		})
}

func errorValueExample() {
	flows.WithFlow(
		func() {
			cf := flows.CurrentFlow().CompletedValue(errors.New("foo"))
			valueCh, errorCh := cf.ThenApply(strings.ToUpper).ThenApply(strings.ToLower).Get()
			printResult(valueCh, errorCh)
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
			valueCh, errorCh := cf.ThenApply(FailedFunc).Handle(HandleFunc).Get()
			printResult(valueCh, errorCh)
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
			valueCh, errorCh := cf.ThenApply(FooToUpper).Get()
			printResult(valueCh, errorCh)
		})
}

func EmptyFunc() string {
	return "empty func"
}

func delayExample() {
	flows.WithFlow(
		func() {
			cf := flows.CurrentFlow().Delay(5 * time.Second).ThenApply(EmptyFunc)
			valueCh, errorCh := cf.Get()
			printResult(valueCh, errorCh)
		})
}

type GreetingRequest struct {
	Name string `json:"name"`
}

type GreetingResponse struct {
	Name       string `json:"name"`
	Salutation string `json:"salutation"`
}

func invokeExample() {
	flows.WithFlow(
		func() {
			greeting, err := json.Marshal(GreetingRequest{Name: "Charles"})
			if err != nil {
				panic("failed to marshal greeting")
			}
			req := &flows.HTTPRequest{Method: "POST", Body: greeting}
			cf := flows.CurrentFlow().InvokeFunction("examples/greeter", req)
			valueCh, _ := cf.Get()
			v := (<-valueCh).(*flows.HTTPResponse)

			var res GreetingResponse
			json.Unmarshal(v.Body, &res)
			fmt.Printf("Got HTTP status %v and body %v", v.StatusCode, res)
		})
}

func TransformExternalRequest(req *flows.HTTPRequest) string {
	result := "Hello " + string(req.Body)
	flows.Log(fmt.Sprintf("Got result %s", result))
	return result
}

func completeExample() {
	flows.WithFlow(
		func() {
			cs := flows.CurrentFlow().EmptyFuture()
			cf := cs.ThenApply(strings.ToUpper)
			cs.Complete("foo")
			valueCh, errorCh := cf.Get()
			printResult(valueCh, errorCh)
		})
}

func ComposedFunc(msg string) flows.FlowFuture {
	return flows.CurrentFlow().CompletedValue("Hello " + msg)
}

func composedExample() {
	flows.WithFlow(
		func() {
			cf := flows.CurrentFlow().CompletedValue("foo")
			valueCh, errorCh := cf.ThenCompose(ComposedFunc).GetType(reflect.TypeOf(""))
			printResult(valueCh, errorCh)
		})
}

func printResult(valueCh chan interface{}, errorCh chan error) {
	select {
	case value := <-valueCh:
		fmt.Printf("Flow succeeded with value %v", value)
	case err := <-errorCh:
		fmt.Printf("Flow failed with error %v", err)
	case <-time.After(time.Minute * 1):
		fmt.Printf("Timed out!")
	}
}
