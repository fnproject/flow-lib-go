package flow

import (
	"encoding/json"
	"fmt"
	"log"
	"net"
	"net/http"
	"net/url"
	"os"
	"reflect"
	"strings"
	"time"

	"github.com/fnproject/flow-lib-go/blobstore"
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
		bsClient: blobstore.GetBlobStore(),
	}
}

func stageList(stageIDs ...string) []string {
	data := make([]string, len(stageIDs))
	for i, stageID := range stageIDs {
		// assuming little endian
		data[i] = stageID
	}
	return data
}

type completerClient interface {
	createFlow(functionID string) string
	commit(flowID string)
	getAsync(flowID string, stageID string, rType reflect.Type) (chan interface{}, chan error)
	emptyFuture(flowID string, loc *codeLoc) string
	completedValue(flowID string, value interface{}, loc *codeLoc) string
	delay(flowID string, duration time.Duration, loc *codeLoc) string
	supply(flowID string, fn interface{}, loc *codeLoc) string
	thenApply(flowID string, stageID string, fn interface{}, loc *codeLoc) string
	thenCompose(flowID string, stageID string, fn interface{}, loc *codeLoc) string
	whenComplete(flowID string, stageID string, fn interface{}, loc *codeLoc) string
	thenAccept(flowID string, stageID string, fn interface{}, loc *codeLoc) string
	thenRun(flowID string, stageID string, fn interface{}, loc *codeLoc) string
	acceptEither(flowID string, stageID string, altStageID string, fn interface{}, loc *codeLoc) string
	applyToEither(flowID string, stageID string, altStageID string, fn interface{}, loc *codeLoc) string
	thenAcceptBoth(flowID string, stageID string, altStageID string, fn interface{}, loc *codeLoc) string
	invokeFunction(flowID string, functionID string, req *HTTPRequest, loc *codeLoc) string
	allOf(flowID string, stages []string, loc *codeLoc) string
	anyOf(flowID string, stages []string, loc *codeLoc) string
	handle(flowID string, stageID string, fn interface{}, loc *codeLoc) string
	exceptionally(flowID string, stageID string, fn interface{}, loc *codeLoc) string
	exceptionallyCompose(flowID string, stageID string, fn interface{}, loc *codeLoc) string
	thenCombine(flowID string, stageID string, altStageID string, fn interface{}, loc *codeLoc) string
	complete(flowID string, stageID string, val interface{}, loc *codeLoc) bool
}

type completerServiceClient struct {
	url      string
	protocol *completerProtocol
	hc       *http.Client
	sc       *apiClient.Flow
	bsClient blobstore.BlobStoreClient
}

func (cs *completerServiceClient) createFlow(functionID string) string {
	req := &flowModels.ModelCreateGraphRequest{FunctionID: functionID}
	p := flowSvc.NewCreateGraphParams().WithBody(req)

	ok, err := cs.sc.FlowService.CreateGraph(p)
	if err != nil {
		log.Fatalf("Failed to create flow: %v", err)
	}
	return ok.Payload.FlowID
}

func (cs *completerServiceClient) emptyFuture(flowID string, loc *codeLoc) string {
	panic("Not implemented")
}

func (cs *completerServiceClient) completedValue(flowID string, value interface{}, loc *codeLoc) string {
	req := &flowModels.ModelAddCompletedValueStageRequest{
		CodeLocation: loc.String(),
		FlowID:       flowID,
		Value:        valueToModel(value, flowID, cs.bsClient),
	}
	p := flowSvc.NewAddValueStageParams().WithFlowID(flowID).WithBody(req)

	ok, err := cs.sc.FlowService.AddValueStage(p)
	if err != nil {
		log.Fatalf("Failed to add value stage: %v", err)
	}
	return ok.Payload.StageID
}

func (cs *completerServiceClient) supply(flowID string, fn interface{}, loc *codeLoc) string {
	panic("Not implemented")
}

func (cs *completerServiceClient) addStageWithClosure(flowID string, op string, fn interface{}, loc *codeLoc, deps []string) string {
	// b := cs.bsClient.WriteBlob(flowID, JSONMediaHeader, encodeContinuationRef(fn))
	panic("Not implemented")
}

func (cs *completerServiceClient) thenApply(flowID string, stageID string, fn interface{}, loc *codeLoc) string {
	req := &flowModels.ModelAddStageRequest{
		Closure:      closureToModel(fn, flowID, cs.bsClient),
		CodeLocation: loc.String(),
		Deps:         nil,
		FlowID:       flowID,
		Operation:    flowModels.ModelCompletionOperationThenApply,
	}
	p := flowSvc.NewAddStageParams().WithFlowID(flowID).WithBody(req)

	ok, err := cs.sc.FlowService.AddStage(p)
	if err != nil {
		log.Fatalf("Failed to add value stage: %v", err)
	}
	return ok.Payload.StageID
}

func (cs *completerServiceClient) thenCompose(flowID string, stageID string, fn interface{}, loc *codeLoc) string {
	panic("Not implemented")
	//	return cs.addStageWithClosure(flowID, CompletionOperation_thenCompose, fn, loc, stageList(stageID))
}

func (cs *completerServiceClient) whenComplete(flowID string, stageID string, fn interface{}, loc *codeLoc) string {
	panic("Not implemented")
	//	return cs.addStageWithClosure(flowID, CompletionOperation_whenComplete, fn, loc, stageList(stageID))
}

func (cs *completerServiceClient) thenAccept(flowID string, stageID string, fn interface{}, loc *codeLoc) string {
	panic("Not implemented")
	//	return cs.addStageWithClosure(flowID, CompletionOperation_thenAccept, fn, loc, stageList(stageID))
}

func (cs *completerServiceClient) thenRun(flowID string, stageID string, fn interface{}, loc *codeLoc) string {
	panic("Not implemented")
	//	return cs.addStageWithClosure(flowID, CompletionOperation_thenRun, fn, loc, stageList(stageID))
}

func (cs *completerServiceClient) acceptEither(flowID string, stageID string, altStageID string, fn interface{}, loc *codeLoc) string {
	panic("Not implemented")
	//	return cs.addStageWithClosure(flowID, CompletionOperation_acceptEither, fn, loc, stageList(stageID, alt))
}

func (cs *completerServiceClient) applyToEither(flowID string, stageID string, altStageID string, fn interface{}, loc *codeLoc) string {
	panic("Not implemented")
	//	return cs.addStageWithClosure(flowID, CompletionOperation_applyToEither, fn, loc, stageList(stageID, alt))
}

func (cs *completerServiceClient) thenAcceptBoth(flowID string, stageID string, altStageID string, fn interface{}, loc *codeLoc) string {
	panic("Not implemented")
	//	return cs.addStageWithClosure(flowID, CompletionOperation_thenAcceptBoth, fn, loc, stageList(stageID, alt))
}

func (cs *completerServiceClient) thenCombine(flowID string, stageID string, altStageID string, fn interface{}, loc *codeLoc) string {
	panic("Not implemented")
	//	return cs.addStageWithClosure(flowID, CompletionOperation_thenCombine, fn, loc, stageList(stageID, alt))
}

func joinedCids(stageIDs []string) string {
	var stageIDStrs []string
	for _, stageID := range stageIDs {
		stageIDStrs = append(stageIDStrs, stageID)
	}
	return strings.Join(stageIDStrs, ",")
}

func (cs *completerServiceClient) allOf(flowID string, stages []string, loc *codeLoc) string {
	panic("Not implemented")
	//	return cs.addStage(flowID, CompletionOperation_allOf, nil, loc, stageList(stageIDs...))
}

func (cs *completerServiceClient) anyOf(flowID string, stages []string, loc *codeLoc) string {
	panic("Not implemented")
	//	return cs.addStage(flowID, CompletionOperation_anyOf, nil, loc, stageList(stageIDs...))
}

func (cs *completerServiceClient) handle(flowID string, stageID string, fn interface{}, loc *codeLoc) string {
	panic("Not implemented")
	//	return cs.addStageWithClosure(flowID, CompletionOperation_handle, fn, loc, stageList(stageID))
}

func (cs *completerServiceClient) exceptionally(flowID string, stageID string, fn interface{}, loc *codeLoc) string {
	panic("Not implemented")
	//	return cs.addStageWithClosure(flowID, CompletionOperation_exceptionally, fn, loc, stageList(stageID))
}

func (cs *completerServiceClient) exceptionallyCompose(flowID string, stageID string, fn interface{}, loc *codeLoc) string {
	panic("Not implemented")
	//	return cs.addStageWithClosure(flowID, CompletionOperation_exceptionallyCompose, fn, loc, stageList(stageID))
}

func (cs *completerServiceClient) complete(flowID string, stageID string, value interface{}, loc *codeLoc) bool {
	panic("Not implemented")
}

func (cs *completerServiceClient) invokeFunction(flowID string, functionID string, req *HTTPRequest, loc *codeLoc) string {
	// TODO
	panic("Not implemented!")
}

func (cs *completerServiceClient) delay(flowID string, duration time.Duration, loc *codeLoc) string {
	// timeMs := int64(duration / time.Millisecond)
	panic("Not implemented")
	//
}

func (cs *completerServiceClient) getAsync(flowID string, stageID string, rType reflect.Type) (chan interface{}, chan error) {
	valueCh := make(chan interface{}, 1)
	errorCh := make(chan error, 1)
	go cs.get(flowID, stageID, rType, valueCh, errorCh)
	return valueCh, errorCh
}

func (cs *completerServiceClient) get(flowID string, stageID string, rType reflect.Type, valueCh chan interface{}, errorCh chan error) {
	panic("Not implemented")

	/*
		debug(fmt.Sprintf("Getting result for stage %s and flow %s", stageID, flowID))
		req := cs.protocol.getStageReq(flowID, stageID)
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
	*/
}

func (cs *completerServiceClient) commit(flowID string) {
	p := flowSvc.NewCommitParams().WithFlowID(flowID)
	_, err := cs.sc.FlowService.Commit(p)
	if err != nil {
		log.Fatalf("Failed to create flow: %v", err)
	}
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
