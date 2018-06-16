package flow

import (
	"fmt"
	"net/http"
	"os"
	"reflect"
	"runtime"
	"strings"
	"sync"
	"time"
)

type HTTPRequest struct {
	Headers http.Header
	Method  string
	Body    []byte
}

type HTTPResponse struct {
	StatusCode int32
	Headers    http.Header
	Body       []byte
}

type Flow interface {
	InvokeFunction(functionID string, arg *HTTPRequest) FlowFuture
	Supply(action interface{}) FlowFuture
	Delay(duration time.Duration) FlowFuture
	CompletedValue(value interface{}) FlowFuture // value can be an error
	EmptyFuture() FlowFuture
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
	Complete(value interface{}) bool
}

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

var actions = make(map[string]interface{})

func getActionKey(actionFunc interface{}) string {
	return runtime.FuncForPC(reflect.ValueOf(actionFunc).Pointer()).Name()
}

// registers a go function so it can be used as an action
// in a flow stage
func RegisterAction(actionFunc interface{}) {
	if reflect.TypeOf(actionFunc).Kind() != reflect.Func {
		panic("Action must be a function!")
	}
	actions[getActionKey(actionFunc)] = actionFunc
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
		handleInvocation(codec)
		return
	}
	initFlow(codec, true)
	defer cf.commit()
	debug("Invoking user's main flow function")
	fn()
	debug("Completed invocation of user's main flow function")
}

func initFlow(codec codec, shouldCreate bool) {
	client := newFlowClient()
	var flowID string
	if shouldCreate {
		flowID = client.createFlow(getFunctionID(codec))
		debug(fmt.Sprintf("Created new flow %v", flowID))
	} else {
		flowID = codec.getFlowID()
		debug(fmt.Sprintf("Awakened flow %v", flowID))
	}
	cfMtx.Lock()
	defer cfMtx.Unlock()
	cf = &flow{
		client: client,
		flowID: flowID,
		codec:  codec,
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

type flow struct {
	client flowClient
	flowID string
	codec  codec
}

type flowFuture struct {
	*flow
	stageID    string
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
	cf.client.commit(cf.flowID)
}

func returnTypeForFunc(fn interface{}) reflect.Type {
	t := reflect.ValueOf(fn).Type()
	if t.NumOut() > 0 {
		return t.Out(0)
	}
	return nil
}

func (cf *flow) continuationFuture(stageID string, fn interface{}) *flowFuture {
	return &flowFuture{flow: cf, stageID: stageID, returnType: returnTypeForFunc(fn)}
}

func (cf *flow) Supply(action interface{}) FlowFuture {
	sid := cf.client.supply(cf.flowID, action, newCodeLoc())
	return cf.continuationFuture(sid, action)
}

func (cf *flow) Delay(duration time.Duration) FlowFuture {
	sid := cf.client.delay(cf.flowID, duration, newCodeLoc())
	return &flowFuture{flow: cf, stageID: sid}
}

func (cf *flow) CompletedValue(value interface{}) FlowFuture {
	sid := cf.client.completedValue(cf.flowID, value, newCodeLoc())
	return &flowFuture{flow: cf, stageID: sid, returnType: reflect.TypeOf(value)}
}

func (cf *flow) InvokeFunction(functionID string, arg *HTTPRequest) FlowFuture {
	sid := cf.client.invokeFunction(cf.flowID, functionID, arg, newCodeLoc())
	return &flowFuture{
		flow:       cf,
		stageID:    sid,
		returnType: reflect.TypeOf(new(HTTPResponse)),
	}
}

func (cf *flow) EmptyFuture() FlowFuture {
	sid := cf.client.emptyFuture(cf.flowID, newCodeLoc())
	return &flowFuture{flow: cf, stageID: sid}
}

func futureCids(futures ...FlowFuture) []string {
	var sids []string
	for _, f := range futures {
		ff := f.(*flowFuture)
		sids = append(sids, ff.stageID)
	}
	return sids
}

func (cf *flow) AllOf(futures ...FlowFuture) FlowFuture {
	sid := cf.client.allOf(cf.flowID, futureCids(futures...), newCodeLoc())
	return &flowFuture{flow: cf, stageID: sid}
}

func (cf *flow) AnyOf(futures ...FlowFuture) FlowFuture {
	sid := cf.client.anyOf(cf.flowID, futureCids(futures...), newCodeLoc())
	// If all dependent futures are of the same type, we can introspect
	// the type as a convenience. Otherwise, we have no way of determining
	// the return type at runtime
	var introspected reflect.Type
	for i, f := range futures {
		if ff, ok := f.(*flowFuture); ok {
			if i == 0 {
				introspected = ff.returnType
			} else if ff.returnType != introspected {
				// different types
				introspected = nil
				break
			}
			introspected = ff.returnType
			continue
		}
		// unknown type
		introspected = nil
		break
	}
	debug(fmt.Sprintf("Introspected return type %v\n", introspected))
	return &flowFuture{
		flow:       cf,
		stageID:    sid,
		returnType: introspected,
	}
}

func (f *flowFuture) Get() (chan interface{}, chan error) {
	return f.client.getAsync(f.flowID, f.stageID, f.returnType)
}

func (f *flowFuture) GetType(t reflect.Type) (chan interface{}, chan error) {
	return f.client.getAsync(f.flowID, f.stageID, t)
}

func (f *flowFuture) ThenApply(action interface{}) FlowFuture {
	sid := f.client.thenApply(f.flowID, f.stageID, action, newCodeLoc())
	return cf.continuationFuture(sid, action)
}

func (f *flowFuture) ThenCompose(action interface{}) FlowFuture {
	sid := f.client.thenCompose(f.flowID, f.stageID, action, newCodeLoc())
	// no type information available for inner future
	return &flowFuture{flow: cf, stageID: sid}
}

func (f *flowFuture) ThenCombine(other FlowFuture, action interface{}) FlowFuture {
	sid := f.client.thenCombine(f.flowID, f.stageID, other.(*flowFuture).stageID, action, newCodeLoc())
	return cf.continuationFuture(sid, action)
}

func (f *flowFuture) WhenComplete(action interface{}) FlowFuture {
	sid := f.client.whenComplete(f.flowID, f.stageID, action, newCodeLoc())
	return cf.continuationFuture(sid, action)
}

func (f *flowFuture) ThenAccept(action interface{}) FlowFuture {
	sid := f.client.thenAccept(f.flowID, f.stageID, action, newCodeLoc())
	return cf.continuationFuture(sid, action)
}

func (f *flowFuture) AcceptEither(other FlowFuture, action interface{}) FlowFuture {
	sid := f.client.acceptEither(f.flowID, f.stageID, other.(*flowFuture).stageID, action, newCodeLoc())
	return cf.continuationFuture(sid, action)
}

func (f *flowFuture) ApplyToEither(other FlowFuture, action interface{}) FlowFuture {
	sid := f.client.applyToEither(f.flowID, f.stageID, other.(*flowFuture).stageID, action, newCodeLoc())
	return cf.continuationFuture(sid, action)
}

func (f *flowFuture) ThenAcceptBoth(other FlowFuture, action interface{}) FlowFuture {
	sid := f.client.thenAcceptBoth(f.flowID, f.stageID, other.(*flowFuture).stageID, action, newCodeLoc())
	return cf.continuationFuture(sid, action)
}

func (f *flowFuture) ThenRun(action interface{}) FlowFuture {
	sid := f.client.thenRun(f.flowID, f.stageID, action, newCodeLoc())
	return cf.continuationFuture(sid, action)
}

func (f *flowFuture) Handle(action interface{}) FlowFuture {
	sid := f.client.handle(f.flowID, f.stageID, action, newCodeLoc())
	return cf.continuationFuture(sid, action)
}

func (f *flowFuture) Exceptionally(action interface{}) FlowFuture {
	sid := f.client.exceptionally(f.flowID, f.stageID, action, newCodeLoc())
	return cf.continuationFuture(sid, action)
}

func (f *flowFuture) ExceptionallyCompose(action interface{}) FlowFuture {
	sid := f.client.exceptionallyCompose(f.flowID, f.stageID, action, newCodeLoc())
	// no type information available for inner future
	return &flowFuture{flow: cf, stageID: sid}
}

func (f *flowFuture) Complete(value interface{}) bool {
	return f.client.complete(f.flowID, f.stageID, value, newCodeLoc())
}
