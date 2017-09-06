package completions

import (
	"net/url"
	"os"
	"strings"
	"time"
)

func WithCloudThread(fn func(ct CloudThread)) {
	codec := newCodec()
	if codec.isContinuation() {
		handleContinuation(codec)
		return
	}
	ct := newCloudThread(codec)
	defer ct.commit()
	fn(ct)
}

func newCloudThread(codec codec) *cloudThread {
	completer := newCompleterClient()
	return &cloudThread{
		completer: completer,
		threadID:  completer.createThread(getFunctionID(codec)),
		codec:     codec,
	}
}

// case insensitive lookup
func lookupEnv(key string) (string, bool) {
	for _, e := range os.Environ() {
		kv := strings.SplitN(e, "=", 2)
		if strings.ToLower(kv[0]) == strings.ToLower(key) {
			return kv[1], true
		}
	}
	return "", false
}

type CloudThread interface {
	// InvokeFunction(functionID string, method HTTPMethod, headers Headers, data byte[]) CloudFuture
	// InvokeFunction(functionID string, method HTTPMethod, headers Headers) CloudFuture
	//Supply(fn interface{}) CloudFuture
	Delay(duration time.Duration) CloudFuture
	CompletedValue(value interface{}) CloudFuture
	//CreateExternalFuture() ExternalCloudFuture
	//AllOf(futures ...CloudFuture) CloudFuture
	//AnyOf(futures ...CloudFuture) CloudFuture
}

type CloudFuture interface {
	Join(result interface{}) // TODO turn this into a channel
	ThenApply(fn interface{}) CloudFuture
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
	completer completerClient
	threadID  threadID
	codec     codec
}

func (ct *cloudThread) Delay(duration time.Duration) CloudFuture {
	return ct.newCloudFuture(ct.completer.delay(ct.threadID, duration))
}

func (ct *cloudThread) CompletedValue(value interface{}) CloudFuture {
	return ct.newCloudFuture(ct.completer.completedValue(ct.threadID, value))
}

func (ct *cloudThread) commit() {
	ct.completer.commit(ct.threadID)
}

type cloudFuture struct {
	*cloudThread
	completionID completionID
}

func (ct *cloudThread) newCloudFuture(cid completionID) CloudFuture {
	return &cloudFuture{cloudThread: ct, completionID: cid}
}

func (cf *cloudFuture) Join(result interface{}) {
	cf.completer.get(cf.threadID, cf.completionID, result)
}

func (cf *cloudFuture) ThenApply(function interface{}) CloudFuture {
	cid := cf.completer.thenApply(cf.threadID, cf.completionID, function)
	return &cloudFuture{cloudThread: cf.cloudThread, completionID: cid}
}

type externalCloudFuture struct {
	cloudFuture
}
