package completions

import (
	"fmt"
	"net/http"
	"net/url"
	"os"
	"strings"
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
	createThread(fid string) threadID
	commit(tid threadID)
	getAsync(tid threadID, cid completionID, val interface{}) chan interface{}
	completedValue(tid threadID, value interface{}, loc *codeLoc) completionID
	failedFuture(tid threadID, err error, loc *codeLoc) completionID
	delay(tid threadID, duration time.Duration, loc *codeLoc) completionID
	supply(tid threadID, fn interface{}, loc *codeLoc) completionID
	thenApply(tid threadID, cid completionID, fn interface{}, loc *codeLoc) completionID
	thenCompose(tid threadID, cid completionID, fn interface{}, loc *codeLoc) completionID
	whenComplete(tid threadID, cid completionID, fn interface{}, loc *codeLoc) completionID
	thenAccept(tid threadID, cid completionID, fn interface{}, loc *codeLoc) completionID
	thenRun(tid threadID, cid completionID, fn interface{}, loc *codeLoc) completionID
	acceptEither(tid threadID, cid completionID, alt completionID, fn interface{}, loc *codeLoc) completionID
	applyToEither(tid threadID, cid completionID, alt completionID, fn interface{}, loc *codeLoc) completionID
	thenAcceptBoth(tid threadID, cid completionID, alt completionID, fn interface{}, loc *codeLoc) completionID
	createExternalCompletion(tid threadID, loc *codeLoc) *externalCompletion

	// TODO
	// invokeFunction(tid threadID, cid completionID, fn interface{}, loc *codeLoc) completionID
	allOf(tid threadID, cids []completionID, loc *codeLoc) completionID
	anyOf(tid threadID, cids []completionID, loc *codeLoc) completionID
	handle(tid threadID, cid completionID, fn interface{}, loc *codeLoc) completionID
	exceptionally(tid threadID, cid completionID, fn interface{}, loc *codeLoc) completionID
	exceptionallyCompose(tid threadID, cid completionID, fn interface{}, loc *codeLoc) completionID
	thenCombine(tid threadID, cid completionID, alt completionID, fn interface{}, loc *codeLoc) completionID
}

type completerServiceClient struct {
	protocol *completerProtocol
}

func (cs *completerServiceClient) createThread(fid string) threadID {
	res := cs.safeReq(cs.protocol.createThreadReq(fid))
	return cs.protocol.parseThreadID(res)
}

func (cs *completerServiceClient) completedValue(tid threadID, value interface{}, loc *codeLoc) completionID {
	return cs.addStage(cs.protocol.gobValueReq(tid, true, value))
}

func (cs *completerServiceClient) failedFuture(tid threadID, err error, loc *codeLoc) completionID {
	return cs.addStage(cs.protocol.gobValueReq(tid, false, err))
}

func (cs *completerServiceClient) supply(tid threadID, fn interface{}, loc *codeLoc) completionID {
	URL := cs.protocol.rootStageURL("supply", tid)
	return cs.addStage(cs.protocol.completionWithBody(URL, fn, loc))
}

func (cs *completerServiceClient) thenApply(tid threadID, cid completionID, fn interface{}, loc *codeLoc) completionID {
	return cs.addStage(cs.protocol.chained("thenApply", tid, cid, fn, loc))
}

func (cs *completerServiceClient) thenCompose(tid threadID, cid completionID, fn interface{}, loc *codeLoc) completionID {
	return cs.addStage(cs.protocol.chained("thenCompose", tid, cid, fn, loc))
}

func (cs *completerServiceClient) whenComplete(tid threadID, cid completionID, fn interface{}, loc *codeLoc) completionID {
	return cs.addStage(cs.protocol.chained("whenComplete", tid, cid, fn, loc))
}

func (cs *completerServiceClient) thenAccept(tid threadID, cid completionID, fn interface{}, loc *codeLoc) completionID {
	return cs.addStage(cs.protocol.chained("thenAccept", tid, cid, fn, loc))
}

func (cs *completerServiceClient) thenRun(tid threadID, cid completionID, fn interface{}, loc *codeLoc) completionID {
	return cs.addStage(cs.protocol.chained("thenRun", tid, cid, fn, loc))
}

func (cs *completerServiceClient) acceptEither(tid threadID, cid completionID, alt completionID, fn interface{}, loc *codeLoc) completionID {
	return cs.addStage(cs.protocol.chainedWithOther("acceptEither", tid, cid, alt, fn, loc))
}

func (cs *completerServiceClient) applyToEither(tid threadID, cid completionID, alt completionID, fn interface{}, loc *codeLoc) completionID {
	return cs.addStage(cs.protocol.chainedWithOther("applyToEither", tid, cid, alt, fn, loc))
}

func (cs *completerServiceClient) thenAcceptBoth(tid threadID, cid completionID, alt completionID, fn interface{}, loc *codeLoc) completionID {
	return cs.addStage(cs.protocol.chainedWithOther("thenAcceptBoth", tid, cid, alt, fn, loc))
}

func (cs *completerServiceClient) thenCombine(tid threadID, cid completionID, alt completionID, fn interface{}, loc *codeLoc) completionID {
	return cs.addStage(cs.protocol.chainedWithOther("thenCombine", tid, cid, alt, fn, loc))
}

func joinedCids(cids []completionID) string {
	var cidStrs []string
	for _, cid := range cids {
		cidStrs = append(cidStrs, string(cid))
	}
	return strings.Join(cidStrs, ",")
}

func (cs *completerServiceClient) allOf(tid threadID, cids []completionID, loc *codeLoc) completionID {
	URL := fmt.Sprintf("%s?cids=%s", cs.protocol.rootStageURL("allOf", tid), joinedCids(cids))
	return cs.addStage(cs.protocol.completion(URL, loc, nil))
}

func (cs *completerServiceClient) anyOf(tid threadID, cids []completionID, loc *codeLoc) completionID {
	URL := fmt.Sprintf("%s?cids=%s", cs.protocol.rootStageURL("anyOf", tid), joinedCids(cids))
	return cs.addStage(cs.protocol.completion(URL, loc, nil))
}

func (cs *completerServiceClient) handle(tid threadID, cid completionID, fn interface{}, loc *codeLoc) completionID {
	return cs.addStage(cs.protocol.chained("handle", tid, cid, fn, loc))
}

func (cs *completerServiceClient) exceptionally(tid threadID, cid completionID, fn interface{}, loc *codeLoc) completionID {
	return cs.addStage(cs.protocol.chained("exceptionally", tid, cid, fn, loc))
}

func (cs *completerServiceClient) exceptionallyCompose(tid threadID, cid completionID, fn interface{}, loc *codeLoc) completionID {
	return cs.addStage(cs.protocol.chained("exceptionallyCompose", tid, cid, fn, loc))
}

type externalCompletion struct {
	cid           completionID
	completionURL *url.URL
	failURL       *url.URL
}

func (cs *completerServiceClient) createExternalCompletion(tid threadID, loc *codeLoc) *externalCompletion {
	URL := cs.protocol.rootStageURL("externalCompletion", tid)
	cid := cs.addStage(cs.protocol.completion(URL, loc, nil))
	cURL, err := url.Parse(cs.protocol.chainedStageURL("complete", tid, cid))
	if err != nil {
		panic("Failed to parse completionURL")
	}
	fURL, err := url.Parse(cs.protocol.chainedStageURL("fail", tid, cid))
	if err != nil {
		panic("Failed to parse failURL")
	}
	return &externalCompletion{cid: cid, completionURL: cURL, failURL: fURL}
}

func (cs *completerServiceClient) delay(tid threadID, duration time.Duration, loc *codeLoc) completionID {
	URL := fmt.Sprintf("%s?delayMs=%d", cs.protocol.rootStageURL("delay", tid), int64(duration))
	return cs.addStage(cs.protocol.completion(URL, loc, nil))
}

func (cs *completerServiceClient) getAsync(tid threadID, cid completionID, val interface{}) chan interface{} {
	ch := make(chan interface{})
	go cs.get(tid, cid, val, ch)
	return ch
}

func (cs *completerServiceClient) get(tid threadID, cid completionID, val interface{}, ch chan interface{}) {
	req := cs.protocol.getStageReq(tid, cid)
	res, err := hc.Do(req)
	if err != nil {
		panic("Failed request: " + err.Error())
	}
	defer res.Body.Close()
	decodeGob(res.Body, val)
	ch <- val
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
