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
		return &completerServiceClient{protocol: &completerPotocol{baseURL: url}}
	}
}

type threadID string
type completionID string

type completerClient interface {
	createThread(functionID string) threadID
	completedValue(threadID threadID, value interface{}) completionID
	delay(threadID threadID, duration time.Duration) completionID
}

type completerServiceClient struct {
	protocol *completerPotocol
}

func (cs *completerServiceClient) createThread(functionID string) threadID {
	res := cs.safePost(cs.protocol.createThreadReq(functionID))
	return cs.protocol.parseThreadID(res)
}

func (cs *completerServiceClient) completedValue(threadID threadID, value interface{}) completionID {
	//req.Header.Set("Content-Type", bodyType)
	return cs.addStage(cs.protocol.completedValueReq(threadID, value))
}

func (cs *completerServiceClient) delay(threadID threadID, duration time.Duration) completionID {
	//req.Header.Set("Content-Type", bodyType)
	return cs.addStage(cs.protocol.delayReq(threadID, duration))
}

func (cs *completerServiceClient) addStage(req *http.Request) completionID {
	return cs.protocol.parseStageID(cs.safePost(req))
}

func (cs *completerServiceClient) safePost(req *http.Request) *http.Response {
	res, err := hc.Do(req)
	if err != nil {
		panic("Failed request: " + err.Error())
	}
	return res
}
