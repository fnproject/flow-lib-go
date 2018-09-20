package main

import (
	"context"
	"encoding/json"
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
	// you can override the default http.Client
	// used to communicate with the flow service
	// httpClient := &http.Client{
	//     Timeout: time.Millisecond * 30,
	// }
	// flows.UseHTTPClient(httpClient)

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
	//fdk.Handle(errorValueExample())
	//fdk.Handle(errorFuncExample())
	//fdk.Handle(structExample())
	//fdk.Handle(composedExample())
	//fdk.Handle(delayExample())
	//fdk.Handle(invokeExample())
	//fdk.Handle(completeExample())
	//fdk.Handle(anyOfExample())
	//fdk.Handle(allOfExample())
}

func stringExample() fdk.Handler {
	return flows.WithFlow(
		fdk.HandlerFunc(func(ctx context.Context, r io.Reader, w io.Writer) {
			cf := flows.CurrentFlow().CompletedValue("foo")
			valueCh, errorCh := cf.ThenApply(strings.ToUpper).ThenApply(strings.ToLower).Get()
			printResult(w, valueCh, errorCh)
		}))
}

func errorValueExample() fdk.Handler {
	return flows.WithFlow(
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

func errorFuncExample() fdk.Handler {
	return flows.WithFlow(
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

func structExample() fdk.Handler {
	return flows.WithFlow(
		fdk.HandlerFunc(func(ctx context.Context, r io.Reader, w io.Writer) {
			cf := flows.CurrentFlow().CompletedValue(&foo{Name: "foo"})
			valueCh, errorCh := cf.ThenApply(FooToUpper).Get()
			printResult(w, valueCh, errorCh)
		}))
}

func EmptyFunc() string {
	return "empty func"
}

func delayExample() fdk.Handler {
	return flows.WithFlow(
		fdk.HandlerFunc(func(ctx context.Context, r io.Reader, w io.Writer) {
			cf := flows.CurrentFlow().Delay(5 * time.Second).ThenApply(EmptyFunc)
			valueCh, errorCh := cf.Get()
			printResult(w, valueCh, errorCh)
		}))
}

type GreetingRequest struct {
	Name string `json:"name"`
}

type GreetingResponse struct {
	Msg string `json:"message"`
}

func invokeExample() fdk.Handler {
	return flows.WithFlow(
		fdk.HandlerFunc(func(ctx context.Context, r io.Reader, w io.Writer) {
			greeting, err := json.Marshal(GreetingRequest{Name: "Charles"})
			if err != nil {
				panic("failed to marshal greeting")
			}
			req := &flows.HTTPRequest{Method: "POST", Body: greeting}
                        // TODO replace the ID below with the function ID of target function to invoke
                        // see https://github.com/fnproject/flow-lib-go/tree/master/examples/greeter/README.md
			cf := flows.CurrentFlow().InvokeFunction("01CQV4NEGMNG8G00GZJ0000002", req)
			valueCh, errorCh := cf.Get()

			select {
			case value := <-valueCh:
				resp, ok := value.(*flows.HTTPResponse)
				if !ok {
					panic("received unexpected value from the server")
				}
				var gr GreetingResponse
				json.Unmarshal(resp.Body, &gr)
				fmt.Fprintf(w, "Got HTTP status %v and payload %v", resp.StatusCode, gr)
			case err := <-errorCh:
				fmt.Fprintf(w, "Flow failed with error %v", err)
			case <-time.After(time.Minute * 1):
				fmt.Fprintf(w, "Timed out!")
			}
		}))
}

func anyOfExample() fdk.Handler {
	return flows.WithFlow(
		fdk.HandlerFunc(func(ctx context.Context, r io.Reader, w io.Writer) {

			cf := flows.CurrentFlow()
			s1 := cf.CompletedValue("first")
			s2 := cf.Delay(2 * time.Second).ThenRun(EmptyFunc)

			s3 := cf.AnyOf(s1, s2).ThenApply(strings.ToUpper)
			valueCh, errorCh := s3.Get()
			printResult(w, valueCh, errorCh)
		}))
}

func allOfExample() fdk.Handler {
	return flows.WithFlow(
		fdk.HandlerFunc(func(ctx context.Context, r io.Reader, w io.Writer) {

			cf := flows.CurrentFlow()
			s1 := cf.CompletedValue("first")
			s2 := cf.Delay(2 * time.Second)

			s3 := cf.AllOf(s1, s2).ThenRun(EmptyFunc)
			valueCh, errorCh := s3.Get()
			printResult(w, valueCh, errorCh)
		}))
}

func TransformExternalRequest(req *flows.HTTPRequest) string {
	result := "Hello " + string(req.Body)
	flows.Log(fmt.Sprintf("Got result %s", result))
	return result
}

func completeExample() fdk.Handler {
	return flows.WithFlow(
		fdk.HandlerFunc(func(ctx context.Context, r io.Reader, w io.Writer) {
			cs := flows.CurrentFlow().EmptyFuture()
			cf := cs.ThenApply(strings.ToUpper)
			cs.Complete("foo")
			valueCh, errorCh := cf.Get()
			printResult(w, valueCh, errorCh)
		}))
}

func ComposedFunc(msg string) flows.FlowFuture {
	return flows.CurrentFlow().CompletedValue("Hello " + msg)
}

func composedExample() fdk.Handler {
	return flows.WithFlow(
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
