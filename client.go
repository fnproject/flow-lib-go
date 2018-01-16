package flows

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"net"
	"net/http"
	"net/textproto"
	"net/url"
	"os"
	"reflect"
	"strings"
	"time"

	apiClient "github.com/fnproject/flow-lib-go/client"
	flowSvc "github.com/fnproject/flow-lib-go/client/flow_service"
	flowModels "github.com/fnproject/flow-lib-go/models"
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
	cURL, err := url.Parse(completerURL)
	if err != nil {
		log.Fatal("Invalid COMPLETER_BASE_URL provided!")
	}
	cfg := apiClient.DefaultTransportConfig().
		WithHost(cURL.Host).
		WithBasePath(cURL.Path).
		WithSchemes([]string{cURL.Scheme})
	sc := apiClient.NewHTTPClientWithConfig(nil, cfg)

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
		sc:       sc,
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
	emptyFuture(fid flowID, loc *codeLoc) stageID
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
	sc       *apiClient.Flow
	bsClient BlobStoreClient
}

func (cs *completerServiceClient) newHTTPReq(path string, msg interface{}) *http.Request {
	url := fmt.Sprintf("%s/v1%s", cs.url, path)
	body := new(bytes.Buffer)
	if err := json.NewEncoder(body).Encode(msg); err != nil {
		panic("Failed to encode request object")
	}

	debug(fmt.Sprintf("Posting body %v", body.String()))
	req, err := http.NewRequest("POST", url, body)
	if err != nil {
		panic("Failed to create request object")
	}
	return req
}

func (cs *completerServiceClient) createFlow(fid string) flowID {
	req := &flowModels.ModelCreateGraphRequest{FunctionID: fid}
	p := flowSvc.NewCreateGraphParams().WithBody(req)

	ok, err := cs.sc.FlowService.CreateGraph(p)
	if err != nil {
		log.Fatalf("Failed to create flow: %v", err)
	}
	return flowID(ok.Payload.FlowID)
}

func (cs *completerServiceClient) emptyFuture(fid flowID, loc *codeLoc) stageID {
	return cs.addStage(fid, CompletionOperation_externalCompletion, nil, loc, nil)
}

func (cs *completerServiceClient) resultFromValue(fid flowID, value interface{}) *flowModels.ModelCompletionResult {
	datum := new(flowModels.ModelDatum)
	switch v := value.(type) {
	case *flowFuture:
		datum.StageRef = &flowModels.ModelStageRefDatum{StageID: string(v.stageID)}

	default:
		if value == nil {
			datum.Empty = new(flowModels.ModelEmptyDatum)
		} else {
			b := cs.bsClient.WriteBlob(string(fid), GobMediaHeader, encodeGob(value))
			datum.Blob = &flowModels.ModelBlobDatum{BlobID: b.BlobId, ContentType: b.ContentType, Length: b.BlobLength}
		}
	}

	_, isErr := value.(error)
	return &flowModels.ModelCompletionResult{Successful: !isErr, Datum: datum}
}

func (cs *completerServiceClient) completedValue(fid flowID, value interface{}, loc *codeLoc) stageID {
	req := &flowModels.ModelAddCompletedValueStageRequest{
		CodeLocation: loc.String(),
		FlowID:       string(fid),
		Value:        cs.resultFromValue(fid, value),
	}
	p := flowSvc.NewAddValueStageParams().WithFlowID(string(fid)).WithBody(req)

	ok, err := cs.sc.FlowService.AddValueStage(p)
	if err != nil {
		log.Fatalf("Failed to add value stage: %v", err)
	}
	return stageID(ok.Payload.StageID)
}

func (cs *completerServiceClient) supply(fid flowID, fn interface{}, loc *codeLoc) stageID {
	return cs.addStageWithClosure(fid, CompletionOperation_supply, fn, loc, []string{})
}

func (cs *completerServiceClient) addStageWithClosure(fid flowID, op CompletionOperation, fn interface{}, loc *codeLoc, deps []string) stageID {
	b := cs.bsClient.WriteBlob(string(fid), JSONMediaHeader, encodeContinuationRef(fn))
	blobDatum := &BlobDatum{BlobId: b.BlobId, ContentType: b.ContentType, Length: b.BlobLength}
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

func (cs *completerServiceClient) completionResultForValue(fid flowID, value interface{}) *CompletionResult {
	datum := new(Datum)
	switch v := value.(type) {
	case *flowFuture:
		datum.Val = &Datum_StageRef{&StageRefDatum{StageId: string(v.stageID)}}

	default:
		if value == nil {
			datum = &Datum{&Datum_Empty{&EmptyDatum{}}}
		} else {
			b := cs.bsClient.WriteBlob(string(fid), GobMediaHeader, encodeGob(value))
			datum.Val = &Datum_Blob{&BlobDatum{BlobId: b.BlobId, ContentType: b.ContentType, Length: b.BlobLength}}
		}
	}

	_, isErr := value.(error)
	return &CompletionResult{Successful: !isErr, Datum: datum}
}

func (cs *completerServiceClient) complete(fid flowID, sid stageID, value interface{}, loc *codeLoc) bool {

	res := &CompleteStageExternallyResponse{}
	msg := &CompleteStageExternallyRequest{
		CodeLocation: loc.String(),
		FlowId:       string(fid),
		StageId:      string(sid),
		Value:        cs.completionResultForValue(fid, value),
	}

	path := fmt.Sprintf("/flows/%s/stages/%s/complete", string(fid), string(sid))
	req := cs.newHTTPReq(path, msg)
	cs.makeRequest(req, res)
	return res.Successful
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
		log.Fatalf("Got %d response from flow server", r.StatusCode)
	}
	err = json.NewDecoder(r.Body).Decode(resp)
	if err != nil {
		panic(fmt.Errorf("Failed to deserialize response to %v", reflect.TypeOf(resp)))
	}
}
