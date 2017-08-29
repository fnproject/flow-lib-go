package completions

import (
	"net/url"
	"os"
	"strings"
	"time"
)

var completionSvc = newCompletionService()

func NewCloudThread() CloudThread {
	if isContinuation() {
	}
	return &cloudThread{threadID: completionSvc.createThread()}
}

func isContinuation() bool {
	for _, e := range os.Environ() {
		kv := strings.Split(e, "=")
		if strings.ToLower(kv[0]) == "header_fnproject-stageid" {
			return true
		}
	}
	return false
}

type CloudThread interface {
	// InvokeFunction(functionID string, method HTTPMethod, headers Headers, data byte[]) CloudFuture
	// InvokeFunction(functionID string, method HTTPMethod, headers Headers) CloudFuture
	//Supply(fn interface{}) CloudFuture
	Delay(duration time.Duration) CloudFuture
	//CompletedValue(value interface{}) CloudFuture
	//CreateExternalFuture() ExternalCloudFuture
	//AllOf(futures ...CloudFuture) CloudFuture
	//AnyOf(futures ...CloudFuture) CloudFuture
}

type CloudFuture interface {
	//ThenApply(fn interface{}) CloudFuture
	//ThenCompose(fn interface{}) CloudFuture
	//ThenCombine(fn interface{}) CloudFuture
	//WhenComplete(fn interface{}) CloudFuture
	//ThenAccept(fn interface{}) CloudFuture
	//AcceptEither(fn interface{}) CloudFuture
	//ApplyToEither(fn interface{}) CloudFuture
	//ThenAcceptBoth(fn interface{}) CloudFuture
	//ThenRun(fn interface{}) CloudFuture
	//Handle(fn interface{}) CloudFuture
	//Exceptionally(fn interface{}) CloudFuture
}

type ExternalCloudFuture interface {
	CloudFuture
	CompletionURL() *url.URL
	FailURL() *url.URL
}

type cloudThread struct {
	threadID threadID
}

func (ct *cloudThread) Delay(duration time.Duration) CloudFuture {
	return &cloudFuture{}
}

type cloudFuture struct {
}

type externalCloudFuture struct {
}
