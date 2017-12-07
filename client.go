package flows

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"net"
	"net/http"
	"net/textproto"
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
	var completerURL string
	var ok bool
	if completerURL, ok = os.LookupEnv("COMPLETER_BASE_URL"); !ok {
		log.Fatal("Missing COMPLETER_BASE_URL configuration in environment!")
	}

	return &completerServiceClient{
		url:      completerURL,
		protocol: newCompleterProtocol(completerURL),
		hc: &http.Client{
			Transport: &http.Transport{
				Dial: (&net.Dialer{
					Timeout:   30 * time.Second,
					KeepAlive: 30 * time.Second,
				}).Dial,
				TLSHandshakeTimeout:   10 * time.Second,
				ResponseHeaderTimeout: 10 * time.Minute,
				ExpectContinueTimeout: 1 * time.Second,
			},
		},
		bsClient: newHTTPBlobStoreClient(fmt.Sprintf("%s/blobs", completerURL)),
	}
}

type flowID string
type stageID string

func stageList(sids ...stageID) []string {
	data := make([]string, len(sids))
	for i, sid := range sids {
		// assuming little endian
		data[i] = string(sid)
	}
	return data
}

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
	invokeFunction(fid flowID, functionID string, req *HTTPRequest, loc *codeLoc) stageID
	allOf(fid flowID, sids []stageID, loc *codeLoc) stageID
	anyOf(fid flowID, sids []stageID, loc *codeLoc) stageID
	handle(fid flowID, sid stageID, fn interface{}, loc *codeLoc) stageID
	exceptionally(fid flowID, sid stageID, fn interface{}, loc *codeLoc) stageID
	exceptionallyCompose(fid flowID, sid stageID, fn interface{}, loc *codeLoc) stageID
	thenCombine(fid flowID, sid stageID, alt stageID, fn interface{}, loc *codeLoc) stageID
	complete(fid flowID, sid stageID, val interface{}, loc *codeLoc) bool
}

type completerServiceClient struct {
	url      string
	protocol *completerProtocol
	hc       *http.Client
	bsClient BlobStoreClient
}

func (cs *completerServiceClient) newHTTPReq(path string, msg interface{}) *http.Request {
	url := fmt.Sprintf("%s%s", cs.url, path)
	body := new(bytes.Buffer)
	if err := json.NewEncoder(body).Encode(msg); err != nil {
		panic("Failed to encode request object")
	}

	req, err := http.NewRequest("POST", url, body)
	if err != nil {
		panic("Failed to create request object")
	}
	return req
}

func (cs *completerServiceClient) datumFromValue(fid flowID, value interface{}) *Datum {
	if value == nil {
		return &Datum{Val: &Datum_Empty{}}
	}

	switch v := value.(type) {
	case FlowFuture:
		f, ok := value.(flowFuture)
		if !ok {
			log.Fatalf("Tried to return unsupported flow future type!")
		}
		return &Datum{Val: &Datum_StageRef{StageRef: &StageRefDatum{StageId: string(f.stageID)}}}

	case HTTPResponse:
		// TODO
		log.Fatalf("Not currently supported!")
		return nil

		//return &Datum{Val: &Datum_HttpResp{HttpResp: &HTTPRespDatum{Body: future.Body, Headers: future.Headers, StatusCode: future.StatusCode}}}

	case HTTPRequest:
		// TODO
		log.Fatalf("Not currently supported!")
		return nil

	default:
		b := cs.bsClient.WriteBlob(string(fid), GobMediaHeader, encodeGob(value))
		return &Datum{Val: &Datum_Blob{Blob: &BlobDatum{BlobId: b.blobId, ContentType: b.contentType, Length: b.blobLength}}}
	}
}

func (cs *completerServiceClient) createFlow(fid string) flowID {
	res := &CreateGraphResponse{}
	req := cs.newHTTPReq("/flows", &CreateGraphRequest{FunctionId: fid})
	cs.makeRequest(req, res)
	return flowID(res.FlowId)
}

func (cs *completerServiceClient) completedValue(fid flowID, value interface{}, loc *codeLoc) stageID {
	res := &AddStageResponse{}
	_, isErr := value.(error)

	msg := &AddCompletedValueStageRequest{
		CodeLocation: loc.String(),
		FlowId:       string(fid),
		Value:        &CompletionResult{Successful: !isErr, Datum: cs.datumFromValue(fid, value)},
	}

	req := cs.newHTTPReq(fmt.Sprintf("/flows/%s/value", fid), msg)
	cs.makeRequest(req, res)
	return stageID(res.StageId)
}

func (cs *completerServiceClient) supply(fid flowID, fn interface{}, loc *codeLoc) stageID {
	return cs.addStageWithClosure(fid, CompletionOperation_supply, fn, loc, []string{})
}

func (cs *completerServiceClient) addStageWithClosure(fid flowID, op CompletionOperation, fn interface{}, loc *codeLoc, deps []string) stageID {
	b := cs.bsClient.WriteBlob(string(fid), JSONMediaHeader, encodeContinuationRef(fn))
	blobDatum := &BlobDatum{BlobId: b.blobId, ContentType: b.contentType, Length: b.blobLength}
	return cs.addStage(fid, op, blobDatum, loc, deps)
}

func (cs *completerServiceClient) addStage(fid flowID, op CompletionOperation, closure *BlobDatum, loc *codeLoc, deps []string) stageID {
	msg := &AddStageRequest{
		Closure:      closure,
		CodeLocation: loc.String(),
		Deps:         deps,
		FlowId:       string(fid),
		Operation:    op,
	}
	req := cs.newHTTPReq(fmt.Sprintf("/flows/%s/stage", fid), msg)

	res := &AddStageResponse{}
	cs.makeRequest(req, res)
	return stageID(res.StageId)
}

func (cs *completerServiceClient) thenApply(fid flowID, sid stageID, fn interface{}, loc *codeLoc) stageID {
	return cs.addStageWithClosure(fid, CompletionOperation_thenApply, fn, loc, stageList(sid))
}

func (cs *completerServiceClient) thenCompose(fid flowID, sid stageID, fn interface{}, loc *codeLoc) stageID {
	return cs.addStageWithClosure(fid, CompletionOperation_thenCompose, fn, loc, stageList(sid))
}

func (cs *completerServiceClient) whenComplete(fid flowID, sid stageID, fn interface{}, loc *codeLoc) stageID {
	return cs.addStageWithClosure(fid, CompletionOperation_whenComplete, fn, loc, stageList(sid))
}

func (cs *completerServiceClient) thenAccept(fid flowID, sid stageID, fn interface{}, loc *codeLoc) stageID {
	return cs.addStageWithClosure(fid, CompletionOperation_thenAccept, fn, loc, stageList(sid))
}

func (cs *completerServiceClient) thenRun(fid flowID, sid stageID, fn interface{}, loc *codeLoc) stageID {
	return cs.addStageWithClosure(fid, CompletionOperation_thenRun, fn, loc, stageList(sid))
}

func (cs *completerServiceClient) acceptEither(fid flowID, sid stageID, alt stageID, fn interface{}, loc *codeLoc) stageID {
	return cs.addStageWithClosure(fid, CompletionOperation_acceptEither, fn, loc, stageList(sid, alt))
}

func (cs *completerServiceClient) applyToEither(fid flowID, sid stageID, alt stageID, fn interface{}, loc *codeLoc) stageID {
	return cs.addStageWithClosure(fid, CompletionOperation_applyToEither, fn, loc, stageList(sid, alt))
}

func (cs *completerServiceClient) thenAcceptBoth(fid flowID, sid stageID, alt stageID, fn interface{}, loc *codeLoc) stageID {
	return cs.addStageWithClosure(fid, CompletionOperation_thenAcceptBoth, fn, loc, stageList(sid, alt))
}

func (cs *completerServiceClient) thenCombine(fid flowID, sid stageID, alt stageID, fn interface{}, loc *codeLoc) stageID {
	return cs.addStageWithClosure(fid, CompletionOperation_thenCombine, fn, loc, stageList(sid, alt))
}

func joinedCids(sids []stageID) string {
	var sidStrs []string
	for _, sid := range sids {
		sidStrs = append(sidStrs, string(sid))
	}
	return strings.Join(sidStrs, ",")
}

func (cs *completerServiceClient) allOf(fid flowID, sids []stageID, loc *codeLoc) stageID {
	return cs.addStage(fid, CompletionOperation_allOf, nil, loc, stageList(sids...))
}

func (cs *completerServiceClient) anyOf(fid flowID, sids []stageID, loc *codeLoc) stageID {
	return cs.addStage(fid, CompletionOperation_anyOf, nil, loc, stageList(sids...))
}

func (cs *completerServiceClient) handle(fid flowID, sid stageID, fn interface{}, loc *codeLoc) stageID {
	return cs.addStageWithClosure(fid, CompletionOperation_handle, fn, loc, stageList(sid))
}

func (cs *completerServiceClient) exceptionally(fid flowID, sid stageID, fn interface{}, loc *codeLoc) stageID {
	return cs.addStageWithClosure(fid, CompletionOperation_exceptionally, fn, loc, stageList(sid))
}

func (cs *completerServiceClient) exceptionallyCompose(fid flowID, sid stageID, fn interface{}, loc *codeLoc) stageID {
	return cs.addStageWithClosure(fid, CompletionOperation_exceptionallyCompose, fn, loc, stageList(sid))
}

func (cs *completerServiceClient) complete(fid flowID, sid stageID, val interface{}, loc *codeLoc) bool {
	// TODO
	panic("Not implemented!")
}

func (cs *completerServiceClient) invokeFunction(fid flowID, functionID string, req *HTTPRequest, loc *codeLoc) stageID {
	// TODO
	panic("Not implemented!")
}

func (cs *completerServiceClient) delay(fid flowID, duration time.Duration, loc *codeLoc) stageID {
	timeMs := int64(duration / time.Millisecond)
	res := &AddStageResponse{}
	req := cs.newHTTPReq("/flows", &AddDelayStageRequest{CodeLocation: loc.String(), DelayMs: timeMs, FlowId: string(fid)})
	cs.makeRequest(req, res)
	return stageID(res.StageId)
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

func (cs *completerServiceClient) safeReq(req *http.Request) *http.Response {
	res, err := hc.Do(req)
	if err != nil {
		panic("Failed request: " + err.Error())
	}
	return res
}

func (cs *completerServiceClient) makeRequest(req *http.Request, resp interface{}) {
	r, err := hc.Do(req)
	if err != nil {
		panic("Failed request: " + err.Error())
	}
	defer r.Body.Close()

	if r.StatusCode != 200 {
		log.Fatalf("Got %d response from blobstore", r.StatusCode)
	}
	err = json.NewDecoder(r.Body).Decode(resp)
	if err != nil {
		panic(fmt.Errorf("Failed to deserialize response to %v", reflect.TypeOf(resp)))
	}
}
