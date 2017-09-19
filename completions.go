package completions

import (
	"fmt"
	"net/url"
	"os"
	"runtime"
	"strings"
	"sync"
	"time"
)

var debugMutex = &sync.Mutex{}
var debugLog = false

func Debug(withDebug bool) {
	debugMutex.Lock()
	defer debugMutex.Unlock()
	debugLog = withDebug
	if debugLog {
		log("Enabled debugging")
	}
}

func log(msg string) {
	if debugLog {
		os.Stderr.WriteString(fmt.Sprintln(msg))
	}
}

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
	Get(result interface{}) chan interface{}
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
	CreateExternalFuture() ExternalCloudFuture
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

type cloudFuture struct {
	*cloudThread
	completionID completionID
}

func (ct *cloudThread) newCloudFuture(cid completionID) CloudFuture {
	return &cloudFuture{cloudThread: ct, completionID: cid}
}

// wraps result to runtime.Caller()
type codeLoc struct {
	file string
	line int
	ok   bool
}

func (cl *codeLoc) String() string {
	if cl.ok {
		return fmt.Sprintf("%s:%d", cl.file, cl.line)
	}
	return "unknown"
}

func newCodeLoc() *codeLoc {
	_, file, line, ok := runtime.Caller(2)
	return &codeLoc{file: file, line: line, ok: ok}
}

func (ct *cloudThread) commit() {
	ct.completer.commit(ct.threadID)
}

func (cf *cloudFuture) Get(result interface{}) chan interface{} {
	return cf.completer.getAsync(cf.threadID, cf.completionID, result)
}

func (ct *cloudThread) Delay(duration time.Duration) CloudFuture {
	return ct.newCloudFuture(ct.completer.delay(ct.threadID, duration, newCodeLoc()))
}

func (ct *cloudThread) CompletedValue(value interface{}) CloudFuture {
	return ct.newCloudFuture(ct.completer.completedValue(ct.threadID, value, newCodeLoc()))
}

func (cf *cloudFuture) ThenApply(function interface{}) CloudFuture {
	cid := cf.completer.thenApply(cf.threadID, cf.completionID, function, newCodeLoc())
	return &cloudFuture{cloudThread: cf.cloudThread, completionID: cid}
}

type externalCloudFuture struct {
	cloudFuture
	completionURL *url.URL
	failURL       *url.URL
}

func (ex *externalCloudFuture) CompletionURL() *url.URL {
	return ex.completionURL
}

func (ex *externalCloudFuture) FailURL() *url.URL {
	return ex.failURL
}

func (cf *cloudFuture) CreateExternalFuture() ExternalCloudFuture {
	ec := cf.completer.createExternalCompletion(cf.threadID, newCodeLoc())
	return &externalCloudFuture{
		cloudFuture:   cloudFuture{cloudThread: cf.cloudThread, completionID: ec.cid},
		completionURL: ec.completionURL,
		failURL:       ec.failURL,
	}
}
