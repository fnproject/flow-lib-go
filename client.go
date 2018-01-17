package flow

import (
	"log"
	"net/http"
	"net/url"
	"os"
	"reflect"
	"time"

	"github.com/fnproject/flow-lib-go/blobstore"
	api "github.com/fnproject/flow-lib-go/client"
	flows "github.com/fnproject/flow-lib-go/client/flow_service"
	"github.com/fnproject/flow-lib-go/models"
)

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

	cfg := api.DefaultTransportConfig().
		WithHost(cURL.Host).
		WithBasePath(cURL.Path).
		WithSchemes([]string{cURL.Scheme})

	sc := api.NewHTTPClientWithConfig(nil, cfg)

	return &completerServiceClient{
		url:      completerURL,
		protocol: newCompleterProtocol(completerURL),
		sc:       sc,
		bsClient: blobstore.GetBlobStore(),
	}
}

func stageList(stageIDs ...string) []string {
	data := make([]string, len(stageIDs))
	for i, stageID := range stageIDs {
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
	sc       *api.Flow
	bsClient blobstore.BlobStoreClient
}

func (cs *completerServiceClient) createFlow(functionID string) string {
	req := &models.ModelCreateGraphRequest{FunctionID: functionID}
	p := flows.NewCreateGraphParams().WithBody(req)

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
	req := &models.ModelAddCompletedValueStageRequest{
		CodeLocation: loc.String(),
		FlowID:       flowID,
		Value:        valueToModel(value, flowID, cs.bsClient),
	}
	p := flows.NewAddValueStageParams().WithFlowID(flowID).WithBody(req)

	ok, err := cs.sc.FlowService.AddValueStage(p)
	if err != nil {
		log.Fatalf("Failed to add value stage: %v", err)
	}
	return ok.Payload.StageID
}

func (cs *completerServiceClient) supply(flowID string, fn interface{}, loc *codeLoc) string {
	panic("Not implemented")
}

func (cs *completerServiceClient) addStageWithClosure(flowID string, op models.ModelCompletionOperation, fn interface{}, loc *codeLoc, deps ...string) string {
	req := &models.ModelAddStageRequest{
		Closure:      closureToModel(fn, flowID, cs.bsClient),
		CodeLocation: loc.String(),
		Deps:         deps,
		FlowID:       flowID,
		Operation:    op,
	}
	p := flows.NewAddStageParams().WithFlowID(flowID).WithBody(req)

	ok, err := cs.sc.FlowService.AddStage(p)
	if err != nil {
		log.Fatalf("Failed to add value stage: %v", err)
	}
	return ok.Payload.StageID
}

func (cs *completerServiceClient) thenApply(flowID string, stageID string, fn interface{}, loc *codeLoc) string {
	return cs.addStageWithClosure(flowID, models.ModelCompletionOperationThenApply, fn, loc, stageID)
}

func (cs *completerServiceClient) thenCompose(flowID string, stageID string, fn interface{}, loc *codeLoc) string {
	return cs.addStageWithClosure(flowID, models.ModelCompletionOperationThenCompose, fn, loc, stageID)
}

func (cs *completerServiceClient) whenComplete(flowID string, stageID string, fn interface{}, loc *codeLoc) string {
	return cs.addStageWithClosure(flowID, models.ModelCompletionOperationWhenComplete, fn, loc, stageID)
}

func (cs *completerServiceClient) thenAccept(flowID string, stageID string, fn interface{}, loc *codeLoc) string {
	return cs.addStageWithClosure(flowID, models.ModelCompletionOperationThenAccept, fn, loc, stageID)
}

func (cs *completerServiceClient) thenRun(flowID string, stageID string, fn interface{}, loc *codeLoc) string {
	return cs.addStageWithClosure(flowID, models.ModelCompletionOperationThenRun, fn, loc, stageID)
}

func (cs *completerServiceClient) acceptEither(flowID string, stageID string, altStageID string, fn interface{}, loc *codeLoc) string {
	return cs.addStageWithClosure(flowID, models.ModelCompletionOperationAcceptEither, fn, loc, stageID, altStageID)
}

func (cs *completerServiceClient) applyToEither(flowID string, stageID string, altStageID string, fn interface{}, loc *codeLoc) string {
	return cs.addStageWithClosure(flowID, models.ModelCompletionOperationApplyToEither, fn, loc, stageID, altStageID)
}

func (cs *completerServiceClient) thenAcceptBoth(flowID string, stageID string, altStageID string, fn interface{}, loc *codeLoc) string {
	return cs.addStageWithClosure(flowID, models.ModelCompletionOperationThenAcceptBoth, fn, loc, stageID, altStageID)
}

func (cs *completerServiceClient) thenCombine(flowID string, stageID string, altStageID string, fn interface{}, loc *codeLoc) string {
	return cs.addStageWithClosure(flowID, models.ModelCompletionOperationThenCombine, fn, loc, stageID, altStageID)
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
	return cs.addStageWithClosure(flowID, models.ModelCompletionOperationHandle, fn, loc, stageID)
}

func (cs *completerServiceClient) exceptionally(flowID string, stageID string, fn interface{}, loc *codeLoc) string {
	return cs.addStageWithClosure(flowID, models.ModelCompletionOperationExceptionally, fn, loc, stageID)
}

func (cs *completerServiceClient) exceptionallyCompose(flowID string, stageID string, fn interface{}, loc *codeLoc) string {
	return cs.addStageWithClosure(flowID, models.ModelCompletionOperationExceptionallyCompose, fn, loc, stageID)
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
	p := flows.NewAwaitStageResultParams().WithFlowID(flowID).WithStageID(stageID)
	ok, err := cs.sc.FlowService.AwaitStageResult(p)
	if err != nil {
		log.Fatalf("Failed to add value stage: %v", err)
	}

	result := ok.Payload.Result
	val := result.DecodeValue(flowID, rType, cs.bsClient)
	if result.Successful {
		debug("Getting successful result")
		valueCh <- val
	} else {
		debug("Getting failed result")
		errorCh <- err
	}
}

func (cs *completerServiceClient) commit(flowID string) {
	p := flows.NewCommitParams().WithFlowID(flowID)
	_, err := cs.sc.FlowService.Commit(p)
	if err != nil {
		log.Fatalf("Failed to create flow: %v", err)
	}
}
