package completions

import (
	"bufio"
	"bytes"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"time"
)

func getSuccessResponse(reader *bufio.Reader, req *http.Request) *http.Response {
	var buf bytes.Buffer
	l, _ := strconv.Atoi(req.Header.Get("Content-Length"))
	body := make([]byte, l)
	reader.Read(body)
	fmt.Fprintf(&buf, "Hello %s\n", body)
	for k, vs := range req.Header {
		fmt.Fprintf(&buf, "ENV: %s %#v\n", k, vs)
	}
	return responseFromBuffer(&buf)
}

func getErrorResponse(err error) (res *http.Response) {
	var buf bytes.Buffer
	fmt.Fprintln(&buf, err)
	res = responseFromBuffer(&buf)
	res.StatusCode = 500
	res.Status = http.StatusText(res.StatusCode)
	return
}

func NewCloudThread() {
	reader := bufio.NewReader(os.Stdin)
	req, err := http.ReadRequest(reader)

	var res *http.Response
	if err != nil {
		res = getErrorResponse(err)
	} else {
		res = getSuccessResponse(reader, req)
	}
	res.Write(os.Stdout)
}

func responseFromBuffer(buf *bytes.Buffer) (res *http.Response) {
	res = &http.Response{
		Proto:      "HTTP/1.1",
		ProtoMajor: 1,
		ProtoMinor: 1,
		StatusCode: 200,
		Status:     "OK",
	}
	res.Body = ioutil.NopCloser(buf)
	res.ContentLength = int64(buf.Len())
	return
}

type CloudThread interface {
	// InvokeFunction(functionID string, method HTTPMethod, headers Headers, data byte[]) CloudFuture
	// InvokeFunction(functionID string, method HTTPMethod, headers Headers) CloudFuture
	Supply(fn interface{}) CloudFuture
	Delay(duration time.Duration) CloudFuture
	CompletedValue(value interface{}) CloudFuture
	CreateExternalFuture() ExternalCloudFuture
	AllOf(futures ...CloudFuture) CloudFuture
	AnyOf(futures ...CloudFuture) CloudFuture
}

type CloudFuture interface {
	ThenApply(fn interface{}) CloudFuture
	ThenCompose(fn interface{}) CloudFuture
	ThenCombine(fn interface{}) CloudFuture
	WhenComplete(fn interface{}) CloudFuture
	ThenAccept(fn interface{}) CloudFuture
	AcceptEither(fn interface{}) CloudFuture
	ApplyToEither(fn interface{}) CloudFuture
	ThenAcceptBoth(fn interface{}) CloudFuture
	ThenRun(fn interface{}) CloudFuture
	Handle(fn interface{}) CloudFuture
	Exceptionally(fn interface{}) CloudFuture
}

type ExternalCloudFuture interface {
	CloudFuture
	CompletionURL() *url.URL
	FailURL() *url.URL
}

type cloudFuture struct {
}

type externalCloudFuture struct {
}
