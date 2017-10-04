package flows

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

var debugMtx = &sync.Mutex{}
var debugLog = false

func Debug(withDebug bool) {
	debugMtx.Lock()
	debugLog = withDebug
	debugMtx.Unlock()
	debug("Enabled debugging")
}

func Log(msg string) {
	debug(msg)
}

func debug(msg string) {
	debugMtx.Lock()
	defer debugMtx.Unlock()
	if debugLog {
		fmt.Fprintln(os.Stderr, msg)
	}
}

var cfMtx = &sync.Mutex{}
var cf *flow

func CurrentFlow() Flow {
	cfMtx.Lock()
	defer cfMtx.Unlock()
	if cf == nil {
		panic("Tried accessing unintialized flow")
	}
	return cf
}

func WithFlow(fn func()) {
	codec := newCodec()
	if codec.isContinuation() {
		initFlow(codec, false)
		handleContinuation(codec)
		return
	}
	initFlow(codec, true)
	defer cf.commit()
	debug("Invoking user's main flow function")
	fn()
	debug("Completed invocation of user's main flow function")
}

func initFlow(codec codec, shouldCreate bool) {
	completer := newCompleterClient()
	var flowID flowID
	if shouldCreate {
		flowID = completer.createFlow(getFunctionID(codec))
		debug(fmt.Sprintf("Created new flow %s", flowID))
	} else {
		flowID = codec.getFlowID()
		debug(fmt.Sprintf("Awakened flow %s", flowID))
	}
	cfMtx.Lock()
	defer cfMtx.Unlock()
	cf = &flow{
		completer: completer,
		flowID:    flowID,
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

type Flow interface {
	InvokeFunction(functionID string, req *HTTPRequest) FlowFuture
	Supply(acfion interface{}) FlowFuture
	Delay(duration time.Duration) FlowFuture
	CompletedValue(value interface{}) FlowFuture // value of error indicates failed future
	ExternalFuture() ExternalFlowFuture
	AllOf(futures ...FlowFuture) FlowFuture
	AnyOf(futures ...FlowFuture) FlowFuture
}

type FutureResult interface {
	Value() interface{}
	Err() error
}

type FlowFuture interface {
	Get() chan FutureResult
	// Get result as the given type. E.g. for use with ThenCompose
	GetType(t reflect.Type) chan FutureResult
	ThenApply(acfion interface{}) FlowFuture
	ThenCompose(acfion interface{}) FlowFuture
	ThenCombine(other FlowFuture, acfion interface{}) FlowFuture
	WhenComplete(acfion interface{}) FlowFuture
	ThenAccept(acfion interface{}) FlowFuture
	AcceptEither(other FlowFuture, acfion interface{}) FlowFuture
	ApplyToEither(other FlowFuture, acfion interface{}) FlowFuture
	ThenAcceptBoth(other FlowFuture, acfion interface{}) FlowFuture
	ThenRun(acfion interface{}) FlowFuture
	Handle(acfion interface{}) FlowFuture
	Exceptionally(acfion interface{}) FlowFuture
	ExceptionallyCompose(acfion interface{}) FlowFuture
}

type ExternalFlowFuture interface {
	FlowFuture
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

type flow struct {
	completer completerClient
	flowID    flowID
	codec     codec
}

type flowFuture struct {
	*flow
	stageID    stageID
	returnType reflect.Type
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

func (cf *flow) commit() {
	cf.completer.commit(cf.flowID)
}

func returnTypeForFunc(fn interface{}) reflect.Type {
	t := reflect.ValueOf(fn).Type()
	if t.NumOut() > 0 {
		return t.Out(0)
	}
	return nil
}

func (cf *flow) continuationFuture(sid stageID, fn interface{}) *flowFuture {
	return &flowFuture{flow: cf, stageID: sid, returnType: returnTypeForFunc(fn)}
}

func (cf *flow) Supply(acfion interface{}) FlowFuture {
	sid := cf.completer.supply(cf.flowID, acfion, newCodeLoc())
	return cf.continuationFuture(sid, acfion)
}

func (cf *flow) Delay(duration time.Duration) FlowFuture {
	sid := cf.completer.delay(cf.flowID, duration, newCodeLoc())
	return &flowFuture{flow: cf, stageID: sid}
}

func (cf *flow) CompletedValue(value interface{}) FlowFuture {
	sid := cf.completer.completedValue(cf.flowID, value, newCodeLoc())
	return &flowFuture{flow: cf, stageID: sid, returnType: reflect.TypeOf(value)}
}

func (cf *flow) InvokeFunction(functionID string, req *HTTPRequest) FlowFuture {
	sid := cf.completer.invokeFunction(cf.flowID, functionID, req, newCodeLoc())
	return &flowFuture{
		flow:       cf,
		stageID:    sid,
		returnType: reflect.TypeOf(new(HTTPResponse)),
	}
}

type externalFlowFuture struct {
	flowFuture
	completionURL *url.URL
	failURL       *url.URL
}

func (ex *externalFlowFuture) CompletionURL() *url.URL {
	return ex.completionURL
}

func (ex *externalFlowFuture) FailURL() *url.URL {
	return ex.failURL
}

func (cf *flow) ExternalFuture() ExternalFlowFuture {
	ec := cf.completer.createExternalCompletion(cf.flowID, newCodeLoc())
	f := flowFuture{
		flow:       cf,
		stageID:    ec.sid,
		returnType: reflect.TypeOf(new(HTTPRequest)),
	}
	return &externalFlowFuture{
		flowFuture:    f,
		completionURL: ec.completionURL,
		failURL:       ec.failURL,
	}
}

func futureCids(futures ...FlowFuture) []stageID {
	var sids []stageID
	for _, f := range futures {
		ff := f.(*flowFuture)
		sids = append(sids, ff.stageID)
	}
	return sids
}

func (cf *flow) AllOf(futures ...FlowFuture) FlowFuture {
	sid := cf.completer.allOf(cf.flowID, futureCids(futures...), newCodeLoc())
	return &flowFuture{flow: cf, stageID: sid}
}

func (cf *flow) AnyOf(futures ...FlowFuture) FlowFuture {
	sid := cf.completer.anyOf(cf.flowID, futureCids(futures...), newCodeLoc())
	return &flowFuture{flow: cf, stageID: sid}
}

func (f *flowFuture) Get() chan FutureResult {
	return f.completer.getAsync(f.flowID, f.stageID, f.returnType)
}

func (f *flowFuture) GetType(t reflect.Type) chan FutureResult {
	return f.completer.getAsync(f.flowID, f.stageID, t)
}

func (f *flowFuture) ThenApply(acfion interface{}) FlowFuture {
	sid := f.completer.thenApply(f.flowID, f.stageID, acfion, newCodeLoc())
	return cf.continuationFuture(sid, acfion)
}

func (f *flowFuture) ThenCompose(acfion interface{}) FlowFuture {
	sid := f.completer.thenCompose(f.flowID, f.stageID, acfion, newCodeLoc())
	// no type information available for inner future
	return &flowFuture{flow: cf, stageID: sid}
}

func (f *flowFuture) ThenCombine(other FlowFuture, acfion interface{}) FlowFuture {
	sid := f.completer.thenCombine(f.flowID, f.stageID, other.(*flowFuture).stageID, acfion, newCodeLoc())
	return cf.continuationFuture(sid, acfion)
}

func (f *flowFuture) WhenComplete(acfion interface{}) FlowFuture {
	sid := f.completer.whenComplete(f.flowID, f.stageID, acfion, newCodeLoc())
	return cf.continuationFuture(sid, acfion)
}

func (f *flowFuture) ThenAccept(acfion interface{}) FlowFuture {
	sid := f.completer.thenAccept(f.flowID, f.stageID, acfion, newCodeLoc())
	return cf.continuationFuture(sid, acfion)
}

func (f *flowFuture) AcceptEither(other FlowFuture, acfion interface{}) FlowFuture {
	sid := f.completer.acceptEither(f.flowID, f.stageID, other.(*flowFuture).stageID, acfion, newCodeLoc())
	return cf.continuationFuture(sid, acfion)
}

func (f *flowFuture) ApplyToEither(other FlowFuture, acfion interface{}) FlowFuture {
	sid := f.completer.applyToEither(f.flowID, f.stageID, other.(*flowFuture).stageID, acfion, newCodeLoc())
	return cf.continuationFuture(sid, acfion)
}

func (f *flowFuture) ThenAcceptBoth(other FlowFuture, acfion interface{}) FlowFuture {
	sid := f.completer.thenAcceptBoth(f.flowID, f.stageID, other.(*flowFuture).stageID, acfion, newCodeLoc())
	return cf.continuationFuture(sid, acfion)
}

func (f *flowFuture) ThenRun(acfion interface{}) FlowFuture {
	sid := f.completer.thenRun(f.flowID, f.stageID, acfion, newCodeLoc())
	return cf.continuationFuture(sid, acfion)
}

func (f *flowFuture) Handle(acfion interface{}) FlowFuture {
	sid := f.completer.handle(f.flowID, f.stageID, acfion, newCodeLoc())
	return cf.continuationFuture(sid, acfion)
}

func (f *flowFuture) Exceptionally(acfion interface{}) FlowFuture {
	sid := f.completer.exceptionally(f.flowID, f.stageID, acfion, newCodeLoc())
	return cf.continuationFuture(sid, acfion)
}

func (f *flowFuture) ExceptionallyCompose(acfion interface{}) FlowFuture {
	sid := f.completer.exceptionallyCompose(f.flowID, f.stageID, acfion, newCodeLoc())
	// no type information available for inner future
	return &flowFuture{flow: cf, stageID: sid}
}
