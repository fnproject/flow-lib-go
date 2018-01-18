package flow

import (
	"fmt"
	"os"
	"reflect"
	"runtime"
	"strings"
	"sync"
	"time"

	"github.com/fnproject/flow-lib-go/api"
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

func CurrentFlow() api.Flow {
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

func (cf *flow) Supply(action interface{}) api.FlowFuture {
	sid := cf.client.supply(cf.flowID, action, newCodeLoc())
	return cf.continuationFuture(sid, action)
}

func (cf *flow) Delay(duration time.Duration) api.FlowFuture {
	sid := cf.client.delay(cf.flowID, duration, newCodeLoc())
	return &flowFuture{flow: cf, stageID: sid}
}

func (cf *flow) CompletedValue(value interface{}) api.FlowFuture {
	sid := cf.client.completedValue(cf.flowID, value, newCodeLoc())
	return &flowFuture{flow: cf, stageID: sid, returnType: reflect.TypeOf(value)}
}

func (cf *flow) InvokeFunction(functionID string, arg *api.HTTPRequest) api.FlowFuture {
	sid := cf.client.invokeFunction(cf.flowID, functionID, arg, newCodeLoc())
	return &flowFuture{
		flow:       cf,
		stageID:    sid,
		returnType: reflect.TypeOf(new(api.HTTPResponse)),
	}
}

func (cf *flow) EmptyFuture() api.FlowFuture {
	sid := cf.client.emptyFuture(cf.flowID, newCodeLoc())
	return &flowFuture{flow: cf, stageID: sid}
}

func futureCids(futures ...api.FlowFuture) []string {
	var sids []string
	for _, f := range futures {
		ff := f.(*flowFuture)
		sids = append(sids, ff.stageID)
	}
	return sids
}

func (cf *flow) AllOf(futures ...api.FlowFuture) api.FlowFuture {
	sid := cf.client.allOf(cf.flowID, futureCids(futures...), newCodeLoc())
	return &flowFuture{flow: cf, stageID: sid}
}

func (cf *flow) AnyOf(futures ...api.FlowFuture) api.FlowFuture {
	sid := cf.client.anyOf(cf.flowID, futureCids(futures...), newCodeLoc())
	return &flowFuture{flow: cf, stageID: sid}
}

func (f *flowFuture) Get() (chan interface{}, chan error) {
	return f.client.getAsync(f.flowID, f.stageID, f.returnType)
}

func (f *flowFuture) GetType(t reflect.Type) (chan interface{}, chan error) {
	return f.client.getAsync(f.flowID, f.stageID, t)
}

func (f *flowFuture) ThenApply(action interface{}) api.FlowFuture {
	sid := f.client.thenApply(f.flowID, f.stageID, action, newCodeLoc())
	return cf.continuationFuture(sid, action)
}

func (f *flowFuture) ThenCompose(action interface{}) api.FlowFuture {
	sid := f.client.thenCompose(f.flowID, f.stageID, action, newCodeLoc())
	// no type information available for inner future
	return &flowFuture{flow: cf, stageID: sid}
}

func (f *flowFuture) ThenCombine(other api.FlowFuture, action interface{}) api.FlowFuture {
	sid := f.client.thenCombine(f.flowID, f.stageID, other.(*flowFuture).stageID, action, newCodeLoc())
	return cf.continuationFuture(sid, action)
}

func (f *flowFuture) WhenComplete(action interface{}) api.FlowFuture {
	sid := f.client.whenComplete(f.flowID, f.stageID, action, newCodeLoc())
	return cf.continuationFuture(sid, action)
}

func (f *flowFuture) ThenAccept(action interface{}) api.FlowFuture {
	sid := f.client.thenAccept(f.flowID, f.stageID, action, newCodeLoc())
	return cf.continuationFuture(sid, action)
}

func (f *flowFuture) AcceptEither(other api.FlowFuture, action interface{}) api.FlowFuture {
	sid := f.client.acceptEither(f.flowID, f.stageID, other.(*flowFuture).stageID, action, newCodeLoc())
	return cf.continuationFuture(sid, action)
}

func (f *flowFuture) ApplyToEither(other api.FlowFuture, action interface{}) api.FlowFuture {
	sid := f.client.applyToEither(f.flowID, f.stageID, other.(*flowFuture).stageID, action, newCodeLoc())
	return cf.continuationFuture(sid, action)
}

func (f *flowFuture) ThenAcceptBoth(other api.FlowFuture, action interface{}) api.FlowFuture {
	sid := f.client.thenAcceptBoth(f.flowID, f.stageID, other.(*flowFuture).stageID, action, newCodeLoc())
	return cf.continuationFuture(sid, action)
}

func (f *flowFuture) ThenRun(action interface{}) api.FlowFuture {
	sid := f.client.thenRun(f.flowID, f.stageID, action, newCodeLoc())
	return cf.continuationFuture(sid, action)
}

func (f *flowFuture) Handle(action interface{}) api.FlowFuture {
	sid := f.client.handle(f.flowID, f.stageID, action, newCodeLoc())
	return cf.continuationFuture(sid, action)
}

func (f *flowFuture) Exceptionally(action interface{}) api.FlowFuture {
	sid := f.client.exceptionally(f.flowID, f.stageID, action, newCodeLoc())
	return cf.continuationFuture(sid, action)
}

func (f *flowFuture) ExceptionallyCompose(action interface{}) api.FlowFuture {
	sid := f.client.exceptionallyCompose(f.flowID, f.stageID, action, newCodeLoc())
	// no type information available for inner future
	return &flowFuture{flow: cf, stageID: sid}
}

func (f *flowFuture) Complete(value interface{}) bool {
	return f.client.complete(f.flowID, f.stageID, value, newCodeLoc())
}
