package flows

import (
	"fmt"
	"net"
	"net/http"
	"net/textproto"
	"net/url"
	"os"
	"reflect"
	"strings"
	"time"
)

var hc = &http.Client{
	Transport: &http.Transport{
		Dial: (&net.Dialer{
			Timeout:   30 * time.Second,
			KeepAlive: 30 * time.Second,
		}).Dial,
		TLSHandshakeTimeout:   10 * time.Second,
		ResponseHeaderTimeout: 10 * time.Minute,
		ExpectContinueTimeout: 1 * time.Second,
	},
}

func newCompleterClient() completerClient {
	if url, ok := os.LookupEnv("COMPLETER_BASE_URL"); !ok {
		panic("Missing COMPLETER_BASE_URL configuration in environment!")
	} else {
		return &completerServiceClient{protocol: newCompleterProtocol(url)}
	}
}

type flowID string
type stageID string

type completerClient interface {
	createFlow(fid string) flowID
	commit(fid flowID)
	getAsync(fid flowID, sid stageID, rType reflect.Type) (chan interface{}, chan error)
	completedValue(fid flowID, value interface{}, loc *codeLoc) stageID
	delay(fid flowID, duration time.Duration, loc *codeLoc) stageID
	supply(fid flowID, fn interface{}, loc *codeLoc) stageID
	thenApply(fid flowID, sid stageID, fn interface{}, loc *codeLoc) stageID
	thenCompose(fid flowID, sid stageID, fn interface{}, loc *codeLoc) stageID
	whenComplete(fid flowID, sid stageID, fn interface{}, loc *codeLoc) stageID
	thenAccept(fid flowID, sid stageID, fn interface{}, loc *codeLoc) stageID
	thenRun(fid flowID, sid stageID, fn interface{}, loc *codeLoc) stageID
	acceptEither(fid flowID, sid stageID, alt stageID, fn interface{}, loc *codeLoc) stageID
	applyToEither(fid flowID, sid stageID, alt stageID, fn interface{}, loc *codeLoc) stageID
	thenAcceptBoth(fid flowID, sid stageID, alt stageID, fn interface{}, loc *codeLoc) stageID
	createExternalCompletion(fid flowID, loc *codeLoc) *externalCompletion
	invokeFunction(fid flowID, functionID string, req *HTTPRequest, loc *codeLoc) stageID
	allOf(fid flowID, sids []stageID, loc *codeLoc) stageID
	anyOf(fid flowID, sids []stageID, loc *codeLoc) stageID
	handle(fid flowID, sid stageID, fn interface{}, loc *codeLoc) stageID
	exceptionally(fid flowID, sid stageID, fn interface{}, loc *codeLoc) stageID
	exceptionallyCompose(fid flowID, sid stageID, fn interface{}, loc *codeLoc) stageID
	thenCombine(fid flowID, sid stageID, alt stageID, fn interface{}, loc *codeLoc) stageID
}

type completerServiceClient struct {
	protocol *completerProtocol
}

func (cs *completerServiceClient) createFlow(fid string) flowID {
	res := cs.safeReq(cs.protocol.createFlowReq(fid))
	return cs.protocol.parseFlowID(res)
}

func (cs *completerServiceClient) completedValue(fid flowID, value interface{}, loc *codeLoc) stageID {
	return cs.addStage(cs.protocol.completedValueReq(fid, value))
}

func (cs *completerServiceClient) supply(fid flowID, fn interface{}, loc *codeLoc) stageID {
	URL := cs.protocol.rootStageURL("supply", fid)
	return cs.addStage(cs.protocol.completionWithBody(URL, fn, loc))
}

func (cs *completerServiceClient) thenApply(fid flowID, sid stageID, fn interface{}, loc *codeLoc) stageID {
	return cs.addStage(cs.protocol.chained("thenApply", fid, sid, fn, loc))
}

func (cs *completerServiceClient) thenCompose(fid flowID, sid stageID, fn interface{}, loc *codeLoc) stageID {
	return cs.addStage(cs.protocol.chained("thenCompose", fid, sid, fn, loc))
}

func (cs *completerServiceClient) whenComplete(fid flowID, sid stageID, fn interface{}, loc *codeLoc) stageID {
	return cs.addStage(cs.protocol.chained("whenComplete", fid, sid, fn, loc))
}

func (cs *completerServiceClient) thenAccept(fid flowID, sid stageID, fn interface{}, loc *codeLoc) stageID {
	return cs.addStage(cs.protocol.chained("thenAccept", fid, sid, fn, loc))
}

func (cs *completerServiceClient) thenRun(fid flowID, sid stageID, fn interface{}, loc *codeLoc) stageID {
	return cs.addStage(cs.protocol.chained("thenRun", fid, sid, fn, loc))
}

func (cs *completerServiceClient) acceptEither(fid flowID, sid stageID, alt stageID, fn interface{}, loc *codeLoc) stageID {
	return cs.addStage(cs.protocol.chainedWithOther("acceptEither", fid, sid, alt, fn, loc))
}

func (cs *completerServiceClient) applyToEither(fid flowID, sid stageID, alt stageID, fn interface{}, loc *codeLoc) stageID {
	return cs.addStage(cs.protocol.chainedWithOther("applyToEither", fid, sid, alt, fn, loc))
}

func (cs *completerServiceClient) thenAcceptBoth(fid flowID, sid stageID, alt stageID, fn interface{}, loc *codeLoc) stageID {
	return cs.addStage(cs.protocol.chainedWithOther("thenAcceptBoth", fid, sid, alt, fn, loc))
}

func (cs *completerServiceClient) thenCombine(fid flowID, sid stageID, alt stageID, fn interface{}, loc *codeLoc) stageID {
	return cs.addStage(cs.protocol.chainedWithOther("thenCombine", fid, sid, alt, fn, loc))
}

func joinedCids(sids []stageID) string {
	var sidStrs []string
	for _, sid := range sids {
		sidStrs = append(sidStrs, string(sid))
	}
	return strings.Join(sidStrs, ",")
}

func (cs *completerServiceClient) allOf(fid flowID, sids []stageID, loc *codeLoc) stageID {
	URL := fmt.Sprintf("%s?sids=%s", cs.protocol.rootStageURL("allOf", fid), joinedCids(sids))
	return cs.addStage(cs.protocol.completion(URL, loc, nil))
}

func (cs *completerServiceClient) anyOf(fid flowID, sids []stageID, loc *codeLoc) stageID {
	URL := fmt.Sprintf("%s?sids=%s", cs.protocol.rootStageURL("anyOf", fid), joinedCids(sids))
	return cs.addStage(cs.protocol.completion(URL, loc, nil))
}

func (cs *completerServiceClient) handle(fid flowID, sid stageID, fn interface{}, loc *codeLoc) stageID {
	return cs.addStage(cs.protocol.chained("handle", fid, sid, fn, loc))
}

func (cs *completerServiceClient) exceptionally(fid flowID, sid stageID, fn interface{}, loc *codeLoc) stageID {
	return cs.addStage(cs.protocol.chained("exceptionally", fid, sid, fn, loc))
}

func (cs *completerServiceClient) exceptionallyCompose(fid flowID, sid stageID, fn interface{}, loc *codeLoc) stageID {
	return cs.addStage(cs.protocol.chained("exceptionallyCompose", fid, sid, fn, loc))
}

type externalCompletion struct {
	sid           stageID
	completionURL *url.URL
	failURL       *url.URL
}

func (cs *completerServiceClient) createExternalCompletion(fid flowID, loc *codeLoc) *externalCompletion {
	URL := cs.protocol.rootStageURL("externalCompletion", fid)
	sid := cs.addStage(cs.protocol.completion(URL, loc, nil))
	cURL, err := url.Parse(cs.protocol.chainedStageURL("complete", fid, sid))
	if err != nil {
		panic("Failed to parse completionURL")
	}
	fURL, err := url.Parse(cs.protocol.chainedStageURL("fail", fid, sid))
	if err != nil {
		panic("Failed to parse failURL")
	}
	return &externalCompletion{sid: sid, completionURL: cURL, failURL: fURL}
}

func (cs *completerServiceClient) invokeFunction(fid flowID, functionID string, req *HTTPRequest, loc *codeLoc) stageID {
	URL := fmt.Sprintf("%s?functionId=%s", cs.protocol.rootStageURL("invokeFunction", fid), functionID)
	return cs.addStage(cs.protocol.invokeFunction(URL, loc, req))
}

func (cs *completerServiceClient) delay(fid flowID, duration time.Duration, loc *codeLoc) stageID {
	timeMs := int64(duration / time.Millisecond)
	URL := fmt.Sprintf("%s?delayMs=%d", cs.protocol.rootStageURL("delay", fid), timeMs)
	return cs.addStage(cs.protocol.completion(URL, loc, nil))
}

func (cs *completerServiceClient) getAsync(fid flowID, sid stageID, rType reflect.Type) (chan interface{}, chan error) {
	valueCh := make(chan interface{}, 1)
	errorCh := make(chan error, 1)
	go cs.get(fid, sid, rType, valueCh, errorCh)
	return valueCh, errorCh
}

func (cs *completerServiceClient) get(fid flowID, sid stageID, rType reflect.Type, valueCh chan interface{}, errorCh chan error) {
	debug(fmt.Sprintf("Getting result for stage %s and flow %s", sid, fid))
	req := cs.protocol.getStageReq(fid, sid)
	res, err := hc.Do(req)
	if err != nil {
		panic("Failed request: " + err.Error())
	}
	defer res.Body.Close()

	debug(fmt.Sprintf("Getting stage value of type %s", res.Header.Get(DatumTypeHeader)))
	hdr := textproto.MIMEHeader(res.Header)
	val := decodeDatum(rType, res.Body, &hdr)
	if err, isErr := val.(error); isErr {
		debug("Getting failed result")
		errorCh <- err
	} else {
		debug("Getting successful result")
		valueCh <- val
	}
}

func (cs *completerServiceClient) commit(fid flowID) {
	cs.safeReq(cs.protocol.commit(fid))
}

func (cs *completerServiceClient) addStage(req *http.Request) stageID {
	return cs.protocol.parseStageID(cs.safeReq(req))
}

func (cs *completerServiceClient) safeReq(req *http.Request) *http.Response {
	res, err := hc.Do(req)
	if err != nil {
		panic("Failed request: " + err.Error())
	}
	return res
}
