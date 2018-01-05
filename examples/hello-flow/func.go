package main

import (
	"context"
	"errors"
	"fmt"
	"io"
	"reflect"
	"strings"
	"time"

	fdk "github.com/fnproject/fdk-go"
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
	fdk.Handle(stringExample())
	//errorValueExample()
	//errorFuncExample()
	//structExample()
	//composedExample()
	//delayExample()
	//invokeExample()
	//externalExample()
}

func stringExample() fdk.Handler {
	return flows.WithFlow(
		fdk.HandlerFunc(func(ctx context.Context, r io.Reader, w io.Writer) {
			cf := flows.CurrentFlow().CompletedValue("foo")
			valueCh, errorCh := cf.ThenApply(strings.ToUpper).ThenApply(strings.ToLower).Get()
			printResult(w, valueCh, errorCh)
		}))
}

func errorValueExample() {
	flows.WithFlow(
		fdk.HandlerFunc(func(ctx context.Context, r io.Reader, w io.Writer) {
			cf := flows.CurrentFlow().CompletedValue(errors.New("foo"))
			valueCh, errorCh := cf.ThenApply(strings.ToUpper).ThenApply(strings.ToLower).Get()
			printResult(w, valueCh, errorCh)
		}))
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
		fdk.HandlerFunc(func(ctx context.Context, r io.Reader, w io.Writer) {
			cf := flows.CurrentFlow().CompletedValue("hello")
			valueCh, errorCh := cf.ThenApply(FailedFunc).Handle(HandleFunc).Get()
			printResult(w, valueCh, errorCh)
		}))
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
		fdk.HandlerFunc(func(ctx context.Context, r io.Reader, w io.Writer) {
			cf := flows.CurrentFlow().CompletedValue(&foo{Name: "foo"})
			valueCh, errorCh := cf.ThenApply(FooToUpper).Get()
			printResult(w, valueCh, errorCh)
		}))
}

func EmptyFunc() string {
	return "empty func"
}

func delayExample() {
	flows.WithFlow(
		fdk.HandlerFunc(func(ctx context.Context, r io.Reader, w io.Writer) {
			cf := flows.CurrentFlow().Delay(5 * time.Second).ThenApply(EmptyFunc)
			valueCh, errorCh := cf.Get()
			printResult(w, valueCh, errorCh)
		}))
}

func invokeExample() {
	flows.WithFlow(
		fdk.HandlerFunc(func(ctx context.Context, r io.Reader, w io.Writer) {
			req := &flows.HTTPRequest{Method: "POST", Body: []byte("payload")}
			cf := flows.CurrentFlow().InvokeFunction("foo/foofn", req)
			valueCh, errorCh := cf.Get()
			printResult(w, valueCh, errorCh)
		}))
}

func TransformExternalRequest(req *flows.HTTPRequest) string {
	result := "Hello " + string(req.Body)
	flows.Log(fmt.Sprintf("Got result %s", result))
	return result
}

func externalExample() {
	flows.WithFlow(
		fdk.HandlerFunc(func(ctx context.Context, r io.Reader, w io.Writer) {
			f := flows.CurrentFlow().ExternalFuture()
			fmt.Fprintf(w, "Post your payload to %v\n", f.CompletionURL())
			f.ThenApply(TransformExternalRequest)
		}))
}

func ComposedFunc(msg string) flows.FlowFuture {
	return flows.CurrentFlow().CompletedValue("Hello " + msg)
}

func composedExample() {
	flows.WithFlow(
		fdk.HandlerFunc(func(ctx context.Context, r io.Reader, w io.Writer) {
			cf := flows.CurrentFlow().CompletedValue("foo")
			valueCh, errorCh := cf.ThenCompose(ComposedFunc).GetType(reflect.TypeOf(""))
			printResult(w, valueCh, errorCh)
		}))
}

func printResult(w io.Writer, valueCh chan interface{}, errorCh chan error) {
	select {
	case value := <-valueCh:
		fmt.Fprintf(w, "Flow succeeded with value %v", value)
	case err := <-errorCh:
		fmt.Fprintf(w, "Flow failed with error %v", err)
	case <-time.After(time.Minute * 1):
		fmt.Fprintf(w, "Timed out!")
	}
}
