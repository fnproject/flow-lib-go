package flows

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"reflect"
	"runtime"
	"sync"
	"time"

	fdk "github.com/fnproject/fdk-go"
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

func WithFlow(fn fdk.Handler) fdk.Handler {
	return fdk.HandlerFunc(func(ctx context.Context, in io.Reader, out io.Writer) {
		codec := newCodec(ctx, in, out)
		if codec.isContinuation() {
			initFlow(codec, false)
			handleContinuation(codec)
			return
		}
		initFlow(codec, true)
		defer cf.commit()
		debug("Invoking user's main flow function")
		// TODO do we want separate reader/writer here?
		fn.Serve(ctx, in, out)
		debug("Completed invocation of user's main flow function")
	})
}

func initFlow(codec codec, shouldCreate bool) {
	completer := newCompleterClient()
	var flowID flowID
	if shouldCreate {
		flowID = completer.createFlow(getFunctionID(codec))
		debug(fmt.Sprintf("Created new flow %v", flowID))
	} else {
		flowID = codec.getFlowID()
		debug(fmt.Sprintf("Awakened flow %v", flowID))
	}
	cfMtx.Lock()
	defer cfMtx.Unlock()
	cf = &flow{
		completer: completer,
		flowID:    flowID,
		codec:     codec,
	}
}

type Flow interface {
	InvokeFunction(functionID string, req *HTTPRequest) FlowFuture
	Supply(action interface{}) FlowFuture
	Delay(duration time.Duration) FlowFuture
	CompletedValue(value interface{}) FlowFuture // value of error indicates failed future
	ExternalFuture() ExternalFlowFuture
	AllOf(futures ...FlowFuture) FlowFuture
	AnyOf(futures ...FlowFuture) FlowFuture
}

type FlowFuture interface {
	Get() (chan interface{}, chan error)
	// Get result as the given type. E.g. for use with ThenCompose
	GetType(t reflect.Type) (chan interface{}, chan error)
	ThenApply(action interface{}) FlowFuture
	ThenCompose(action interface{}) FlowFuture
	ThenCombine(other FlowFuture, action interface{}) FlowFuture
	WhenComplete(action interface{}) FlowFuture
	ThenAccept(action interface{}) FlowFuture
	AcceptEither(other FlowFuture, action interface{}) FlowFuture
	ApplyToEither(other FlowFuture, action interface{}) FlowFuture
	ThenAcceptBoth(other FlowFuture, action interface{}) FlowFuture
	ThenRun(action interface{}) FlowFuture
	Handle(action interface{}) FlowFuture
	Exceptionally(action interface{}) FlowFuture
	ExceptionallyCompose(action interface{}) FlowFuture
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

func (cf *flow) Supply(action interface{}) FlowFuture {
	sid := cf.completer.supply(cf.flowID, action, newCodeLoc())
	return cf.continuationFuture(sid, action)
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

func (f *flowFuture) Get() (chan interface{}, chan error) {
	return f.completer.getAsync(f.flowID, f.stageID, f.returnType)
}

func (f *flowFuture) GetType(t reflect.Type) (chan interface{}, chan error) {
	return f.completer.getAsync(f.flowID, f.stageID, t)
}

func (f *flowFuture) ThenApply(action interface{}) FlowFuture {
	sid := f.completer.thenApply(f.flowID, f.stageID, action, newCodeLoc())
	return cf.continuationFuture(sid, action)
}

func (f *flowFuture) ThenCompose(action interface{}) FlowFuture {
	sid := f.completer.thenCompose(f.flowID, f.stageID, action, newCodeLoc())
	// no type information available for inner future
	return &flowFuture{flow: cf, stageID: sid}
}

func (f *flowFuture) ThenCombine(other FlowFuture, action interface{}) FlowFuture {
	sid := f.completer.thenCombine(f.flowID, f.stageID, other.(*flowFuture).stageID, action, newCodeLoc())
	return cf.continuationFuture(sid, action)
}

func (f *flowFuture) WhenComplete(action interface{}) FlowFuture {
	sid := f.completer.whenComplete(f.flowID, f.stageID, action, newCodeLoc())
	return cf.continuationFuture(sid, action)
}

func (f *flowFuture) ThenAccept(action interface{}) FlowFuture {
	sid := f.completer.thenAccept(f.flowID, f.stageID, action, newCodeLoc())
	return cf.continuationFuture(sid, action)
}

func (f *flowFuture) AcceptEither(other FlowFuture, action interface{}) FlowFuture {
	sid := f.completer.acceptEither(f.flowID, f.stageID, other.(*flowFuture).stageID, action, newCodeLoc())
	return cf.continuationFuture(sid, action)
}

func (f *flowFuture) ApplyToEither(other FlowFuture, action interface{}) FlowFuture {
	sid := f.completer.applyToEither(f.flowID, f.stageID, other.(*flowFuture).stageID, action, newCodeLoc())
	return cf.continuationFuture(sid, action)
}

func (f *flowFuture) ThenAcceptBoth(other FlowFuture, action interface{}) FlowFuture {
	sid := f.completer.thenAcceptBoth(f.flowID, f.stageID, other.(*flowFuture).stageID, action, newCodeLoc())
	return cf.continuationFuture(sid, action)
}

func (f *flowFuture) ThenRun(action interface{}) FlowFuture {
	sid := f.completer.thenRun(f.flowID, f.stageID, action, newCodeLoc())
	return cf.continuationFuture(sid, action)
}

func (f *flowFuture) Handle(action interface{}) FlowFuture {
	sid := f.completer.handle(f.flowID, f.stageID, action, newCodeLoc())
	return cf.continuationFuture(sid, action)
}

func (f *flowFuture) Exceptionally(action interface{}) FlowFuture {
	sid := f.completer.exceptionally(f.flowID, f.stageID, action, newCodeLoc())
	return cf.continuationFuture(sid, action)
}

func (f *flowFuture) ExceptionallyCompose(action interface{}) FlowFuture {
	sid := f.completer.exceptionallyCompose(f.flowID, f.stageID, action, newCodeLoc())
	// no type information available for inner future
	return &flowFuture{flow: cf, stageID: sid}
}
