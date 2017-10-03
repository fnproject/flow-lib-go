package completions

import (
	"fmt"
	"net/http"
	"net/url"
	"os"
	"reflect"
	"runtime"
	"strings"
	"sync"
	"time"
)

var debugMutex = &sync.Mutex{}
var debugLog = false

func Debug(withDebug bool) {
	debugMutex.Lock()
	debugLog = withDebug
	debugMutex.Unlock()

	debug("Enabled debugging")
}

func Log(msg string) {
	debug(msg)
}

func debug(msg string) {
	debugMutex.Lock()
	defer debugMutex.Unlock()
	if debugLog {
		fmt.Fprintln(os.Stderr, msg)
	}
}

var ct *cloudThread

func CurrentThread() CloudThread {
	if ct == nil {
		panic("Tried accessing unintialized thread")
	}
	return ct
}

func WithCloudThread(fn func()) {
	codec := newCodec()
	if codec.isContinuation() {
		ct = awakeCloudThread(codec)
		handleContinuation(codec)
		return
	}
	ct = newCloudThread(codec)
	debug(fmt.Sprintf("Created new thread %s", ct.threadID))
	defer ct.commit()
	debug("Invoking user function")
	fn()
	debug("Completed invocation of user function")
}

func newCloudThread(codec codec) *cloudThread {
	completer := newCompleterClient()
	return &cloudThread{
		completer: completer,
		threadID:  completer.createThread(getFunctionID(codec)),
		codec:     codec,
	}
}

func awakeCloudThread(codec codec) *cloudThread {
	completer := newCompleterClient()
	return &cloudThread{
		completer: completer,
		threadID:  codec.getThreadID(),
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
	InvokeFunction(functionID string, req HTTPRequest) CloudFuture
	Supply(function interface{}) CloudFuture
	Delay(duration time.Duration) CloudFuture
	CompletedValue(value interface{}) CloudFuture // value of error indicates failed future
	CreateExternalFuture() ExternalCloudFuture
	AllOf(futures ...CloudFuture) CloudFuture
	AnyOf(futures ...CloudFuture) CloudFuture
}

type FutureResult interface {
	Value() interface{}
	Err() error
}

type CloudFuture interface {
	Get(result interface{}) chan FutureResult
	ThenApply(function interface{}) CloudFuture
	ThenCompose(function interface{}) CloudFuture
	ThenCombine(other CloudFuture, function interface{}) CloudFuture
	WhenComplete(function interface{}) CloudFuture
	ThenAccept(function interface{}) CloudFuture
	AcceptEither(other CloudFuture, function interface{}) CloudFuture
	ApplyToEither(other CloudFuture, function interface{}) CloudFuture
	ThenAcceptBoth(other CloudFuture, function interface{}) CloudFuture
	ThenRun(function interface{}) CloudFuture
	Handle(function interface{}) CloudFuture
	Exceptionally(function interface{}) CloudFuture
	ExceptionallyCompose(function interface{}) CloudFuture
}

type ExternalCloudFuture interface {
	CloudFuture
	CompletionURL() *url.URL
	FailURL() *url.URL
}

type HTTPRequest struct {
	Headers http.Header
	Method  string
	Body    []byte
}

type HTTPResponse struct {
	StatusCode int
	Headers    http.Header
	Body       []byte
}

type cloudThread struct {
	completer completerClient
	threadID  threadID
	codec     codec
}

type cloudFuture struct {
	*cloudThread
	completionID completionID
	returnType   reflect.Type
}

func (ct *cloudThread) newCloudFuture(cid completionID) CloudFuture {
	return &cloudFuture{cloudThread: ct, completionID: cid}
}

func (ct *cloudThread) newTypedCloudFuture(cid completionID, rType reflect.Type) CloudFuture {
	return &cloudFuture{cloudThread: ct, completionID: cid, returnType: rType}
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

func returnTypeForFunc(fn interface{}) reflect.Type {
	t := reflect.ValueOf(fn).Type()
	if t.NumOut() > 0 {
		return t.Out(0)
	}
	return nil
}

func (ct *cloudThread) Supply(function interface{}) CloudFuture {
	return ct.newTypedCloudFuture(ct.completer.supply(ct.threadID, function, newCodeLoc()), returnTypeForFunc(function))
}

func (ct *cloudThread) Delay(duration time.Duration) CloudFuture {
	return ct.newCloudFuture(ct.completer.delay(ct.threadID, duration, newCodeLoc()))
}

func (ct *cloudThread) CompletedValue(value interface{}) CloudFuture {
	return ct.newCloudFuture(ct.completer.completedValue(ct.threadID, value, newCodeLoc()))
}

func (ct *cloudThread) InvokeFunction(functionID string, req HTTPRequest) CloudFuture {
	return ct.newCloudFuture(ct.completer.invokeFunction(ct.threadID, functionID, req, newCodeLoc()))
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

func (ct *cloudThread) CreateExternalFuture() ExternalCloudFuture {
	ec := ct.completer.createExternalCompletion(ct.threadID, newCodeLoc())
	return &externalCloudFuture{
		cloudFuture:   cloudFuture{cloudThread: ct, completionID: ec.cid},
		completionURL: ec.completionURL,
		failURL:       ec.failURL,
	}
}

func futureCids(futures ...CloudFuture) []completionID {
	var cids []completionID
	for _, f := range futures {
		cf := f.(*cloudFuture)
		cids = append(cids, cf.completionID)
	}
	return cids
}

func (ct *cloudThread) AllOf(futures ...CloudFuture) CloudFuture {
	return ct.newCloudFuture(ct.completer.allOf(ct.threadID, futureCids(futures...), newCodeLoc()))
}

func (ct *cloudThread) AnyOf(futures ...CloudFuture) CloudFuture {
	return ct.newCloudFuture(ct.completer.anyOf(ct.threadID, futureCids(futures...), newCodeLoc()))
}

func (cf *cloudFuture) Get(result interface{}) chan FutureResult {
	return cf.completer.getAsync(cf.threadID, cf.completionID, result)
}

func (cf *cloudFuture) GetTyped() chan FutureResult {
	return cf.completer.getAsyncTyped(cf.threadID, cf.completionID, cf.returnType)
}

func (cf *cloudFuture) ThenApply(function interface{}) CloudFuture {
	cid := cf.completer.thenApply(cf.threadID, cf.completionID, function, newCodeLoc())
	return &cloudFuture{cloudThread: cf.cloudThread, completionID: cid, returnType: returnTypeForFunc(function)}
}

func (cf *cloudFuture) ThenCompose(function interface{}) CloudFuture {
	cid := cf.completer.thenCompose(cf.threadID, cf.completionID, function, newCodeLoc())
	return &cloudFuture{cloudThread: cf.cloudThread, completionID: cid}
}

func (cf *cloudFuture) ThenCombine(other CloudFuture, function interface{}) CloudFuture {
	cid := cf.completer.thenCombine(cf.threadID, cf.completionID, other.(*cloudFuture).completionID, function, newCodeLoc())
	return &cloudFuture{cloudThread: cf.cloudThread, completionID: cid}
}

func (cf *cloudFuture) WhenComplete(function interface{}) CloudFuture {
	cid := cf.completer.whenComplete(cf.threadID, cf.completionID, function, newCodeLoc())
	return &cloudFuture{cloudThread: cf.cloudThread, completionID: cid}
}

func (cf *cloudFuture) ThenAccept(function interface{}) CloudFuture {
	cid := cf.completer.thenAccept(cf.threadID, cf.completionID, function, newCodeLoc())
	return &cloudFuture{cloudThread: cf.cloudThread, completionID: cid}
}

func (cf *cloudFuture) AcceptEither(other CloudFuture, function interface{}) CloudFuture {
	cid := cf.completer.acceptEither(cf.threadID, cf.completionID, other.(*cloudFuture).completionID, function, newCodeLoc())
	return &cloudFuture{cloudThread: cf.cloudThread, completionID: cid}
}

func (cf *cloudFuture) ApplyToEither(other CloudFuture, function interface{}) CloudFuture {
	cid := cf.completer.applyToEither(cf.threadID, cf.completionID, other.(*cloudFuture).completionID, function, newCodeLoc())
	return &cloudFuture{cloudThread: cf.cloudThread, completionID: cid}
}

func (cf *cloudFuture) ThenAcceptBoth(other CloudFuture, function interface{}) CloudFuture {
	cid := cf.completer.thenAcceptBoth(cf.threadID, cf.completionID, other.(*cloudFuture).completionID, function, newCodeLoc())
	return &cloudFuture{cloudThread: cf.cloudThread, completionID: cid}
}

func (cf *cloudFuture) ThenRun(function interface{}) CloudFuture {
	cid := cf.completer.thenRun(cf.threadID, cf.completionID, function, newCodeLoc())
	return &cloudFuture{cloudThread: cf.cloudThread, completionID: cid}
}

func (cf *cloudFuture) Handle(function interface{}) CloudFuture {
	cid := cf.completer.handle(cf.threadID, cf.completionID, function, newCodeLoc())
	return &cloudFuture{cloudThread: cf.cloudThread, completionID: cid}
}

func (cf *cloudFuture) Exceptionally(function interface{}) CloudFuture {
	cid := cf.completer.exceptionally(cf.threadID, cf.completionID, function, newCodeLoc())
	return &cloudFuture{cloudThread: cf.cloudThread, completionID: cid}
}

func (cf *cloudFuture) ExceptionallyCompose(function interface{}) CloudFuture {
	cid := cf.completer.exceptionallyCompose(cf.threadID, cf.completionID, function, newCodeLoc())
	return &cloudFuture{cloudThread: cf.cloudThread, completionID: cid}
}
