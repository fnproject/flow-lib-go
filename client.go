package completions

import (
	"net/http"
	"os"
	"time"
)

var hc = &http.Client{
	Timeout: time.Second * 10,
}

func newCompleterClient() completerClient {
	if url, ok := os.LookupEnv("COMPLETER_BASE_URL"); !ok {
		panic("Missing COMPLETER_BASE_URL configuration in environment!")
	} else {
		return &completerServiceClient{protocol: newCompleterProtocol(url)}
	}
}

type threadID string
type completionID string

type completerClient interface {
	createThread(functionID string) threadID
	completedValue(tid threadID, value interface{}) completionID
	delay(tid threadID, duration time.Duration) completionID
	get(tid threadID, cid completionID, val interface{})
	commit(tid threadID)
	thenApply(tid threadID, cid completionID, function interface{}) completionID
}

type completerServiceClient struct {
	protocol *completerProtocol
}

func (cs *completerServiceClient) createThread(functionID string) threadID {
	res := cs.safeReq(cs.protocol.createThreadReq(functionID))
	return cs.protocol.parseThreadID(res)
}

func (cs *completerServiceClient) completedValue(tid threadID, value interface{}) completionID {
	return cs.addStage(cs.protocol.completedValueReq(tid, value))
}

func (cs *completerServiceClient) thenApply(tid threadID, cid completionID, function interface{}) completionID {
	return cs.addStage(cs.protocol.thenApplyReq(tid, cid, function))
}

func (cs *completerServiceClient) delay(tid threadID, duration time.Duration) completionID {
	return cs.addStage(cs.protocol.delayReq(tid, duration))
}

func (cs *completerServiceClient) get(tid threadID, cid completionID, val interface{}) {
	req := cs.protocol.getStageReq(tid, cid)
	res, err := hc.Do(req)
	if err != nil {
		panic("Failed request: " + err.Error())
	}
	defer res.Body.Close()
	decodeGob(res.Body, val)
}

func (cs *completerServiceClient) commit(tid threadID) {
	cs.safeReq(cs.protocol.commit(tid))
}

func (cs *completerServiceClient) addStage(req *http.Request) completionID {
	return cs.protocol.parseStageID(cs.safeReq(req))
}

func (cs *completerServiceClient) safeReq(req *http.Request) *http.Response {
	res, err := hc.Do(req)
	if err != nil {
		panic("Failed request: " + err.Error())
	}
	return res
}
