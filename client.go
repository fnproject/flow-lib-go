package flow

import (
	"log"
	"net/url"
	"os"
	"reflect"
	"time"

	"github.com/fnproject/flow-lib-go/blobstore"
	api "github.com/fnproject/flow-lib-go/client"
	flowSvc "github.com/fnproject/flow-lib-go/client/flow_service"
	"github.com/fnproject/flow-lib-go/models"
)

type remoteFlowClient struct {
	url       string
	flows     *flowSvc.Client
	blobStore blobstore.BlobStoreClient
}

func newFlowClient() flowClient {
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

	return &remoteFlowClient{
		url:       completerURL,
		flows:     sc.FlowService,
		blobStore: blobstore.GetBlobStore(),
	}
}

type flowClient interface {
	createFlow(functionID string) string
	commit(flowID string)
	getAsync(flowID string, stageID string, rType reflect.Type) (chan interface{}, chan error)
	emptyFuture(flowID string, loc *codeLoc) string
	completedValue(flowID string, value interface{}, loc *codeLoc) string
	delay(flowID string, duration time.Duration, loc *codeLoc) string
	supply(flowID string, actionFunc interface{}, loc *codeLoc) string
	thenApply(flowID string, stageID string, actionFunc interface{}, loc *codeLoc) string
	thenCompose(flowID string, stageID string, actionFunc interface{}, loc *codeLoc) string
	whenComplete(flowID string, stageID string, actionFunc interface{}, loc *codeLoc) string
	thenAccept(flowID string, stageID string, actionFunc interface{}, loc *codeLoc) string
	thenRun(flowID string, stageID string, actionFunc interface{}, loc *codeLoc) string
	acceptEither(flowID string, stageID string, altStageID string, actionFunc interface{}, loc *codeLoc) string
	applyToEither(flowID string, stageID string, altStageID string, actionFunc interface{}, loc *codeLoc) string
	thenAcceptBoth(flowID string, stageID string, altStageID string, actionFunc interface{}, loc *codeLoc) string
	invokeFunction(flowID string, functionID string, req *HTTPRequest, loc *codeLoc) string
	allOf(flowID string, stages []string, loc *codeLoc) string
	anyOf(flowID string, stages []string, loc *codeLoc) string
	handle(flowID string, stageID string, actionFunc interface{}, loc *codeLoc) string
	exceptionally(flowID string, stageID string, actionFunc interface{}, loc *codeLoc) string
	exceptionallyCompose(flowID string, stageID string, actionFunc interface{}, loc *codeLoc) string
	thenCombine(flowID string, stageID string, altStageID string, actionFunc interface{}, loc *codeLoc) string
	complete(flowID string, stageID string, val interface{}, loc *codeLoc) bool
}

func (c *remoteFlowClient) createFlow(functionID string) string {
	req := &models.ModelCreateGraphRequest{FunctionID: functionID}
	p := flowSvc.NewCreateGraphParams().WithBody(req)

	ok, err := c.flows.CreateGraph(p)
	if err != nil {
		log.Fatalf("Failed to create flow: %v", err)
	}
	return ok.Payload.FlowID
}

func (c *remoteFlowClient) emptyFuture(flowID string, loc *codeLoc) string {
	panic("Not implemented")
}

func (c *remoteFlowClient) completedValue(flowID string, value interface{}, loc *codeLoc) string {
	req := &models.ModelAddCompletedValueStageRequest{
		CodeLocation: loc.String(),
		FlowID:       flowID,
		Value:        valueToModel(value, flowID, c.blobStore),
	}
	p := flowSvc.NewAddValueStageParams().WithFlowID(flowID).WithBody(req)

	ok, err := c.flows.AddValueStage(p)
	if err != nil {
		log.Fatalf("Failed to add value stage: %v", err)
	}
	return ok.Payload.StageID
}

func (c *remoteFlowClient) supply(flowID string, actionFunc interface{}, loc *codeLoc) string {
	panic("Not implemented")
}

func (c *remoteFlowClient) addStageWithClosure(flowID string, op models.ModelCompletionOperation, actionFunc interface{}, loc *codeLoc, deps ...string) string {
	req := &models.ModelAddStageRequest{
		Closure:      actionToModel(actionFunc, flowID, c.blobStore),
		CodeLocation: loc.String(),
		Deps:         deps,
		FlowID:       flowID,
		Operation:    op,
	}
	p := flowSvc.NewAddStageParams().WithFlowID(flowID).WithBody(req)

	ok, err := c.flows.AddStage(p)
	if err != nil {
		log.Fatalf("Failed to add value stage: %v", err)
	}
	return ok.Payload.StageID
}

func (c *remoteFlowClient) thenApply(flowID string, stageID string, actionFunc interface{}, loc *codeLoc) string {
	return c.addStageWithClosure(flowID, models.ModelCompletionOperationThenApply, actionFunc, loc, stageID)
}

func (c *remoteFlowClient) thenCompose(flowID string, stageID string, actionFunc interface{}, loc *codeLoc) string {
	return c.addStageWithClosure(flowID, models.ModelCompletionOperationThenCompose, actionFunc, loc, stageID)
}

func (c *remoteFlowClient) whenComplete(flowID string, stageID string, actionFunc interface{}, loc *codeLoc) string {
	return c.addStageWithClosure(flowID, models.ModelCompletionOperationWhenComplete, actionFunc, loc, stageID)
}

func (c *remoteFlowClient) thenAccept(flowID string, stageID string, actionFunc interface{}, loc *codeLoc) string {
	return c.addStageWithClosure(flowID, models.ModelCompletionOperationThenAccept, actionFunc, loc, stageID)
}

func (c *remoteFlowClient) thenRun(flowID string, stageID string, actionFunc interface{}, loc *codeLoc) string {
	return c.addStageWithClosure(flowID, models.ModelCompletionOperationThenRun, actionFunc, loc, stageID)
}

func (c *remoteFlowClient) acceptEither(flowID string, stageID string, altStageID string, actionFunc interface{}, loc *codeLoc) string {
	return c.addStageWithClosure(flowID, models.ModelCompletionOperationAcceptEither, actionFunc, loc, stageID, altStageID)
}

func (c *remoteFlowClient) applyToEither(flowID string, stageID string, altStageID string, actionFunc interface{}, loc *codeLoc) string {
	return c.addStageWithClosure(flowID, models.ModelCompletionOperationApplyToEither, actionFunc, loc, stageID, altStageID)
}

func (c *remoteFlowClient) thenAcceptBoth(flowID string, stageID string, altStageID string, actionFunc interface{}, loc *codeLoc) string {
	return c.addStageWithClosure(flowID, models.ModelCompletionOperationThenAcceptBoth, actionFunc, loc, stageID, altStageID)
}

func (c *remoteFlowClient) thenCombine(flowID string, stageID string, altStageID string, actionFunc interface{}, loc *codeLoc) string {
	return c.addStageWithClosure(flowID, models.ModelCompletionOperationThenCombine, actionFunc, loc, stageID, altStageID)
}

func (c *remoteFlowClient) allOf(flowID string, stages []string, loc *codeLoc) string {
	panic("Not implemented")
	//	return cs.addStage(flowID, CompletionOperation_allOf, nil, loc, stageList(stageIDs...))
}

func (c *remoteFlowClient) anyOf(flowID string, stages []string, loc *codeLoc) string {
	panic("Not implemented")
	//	return cs.addStage(flowID, CompletionOperation_anyOf, nil, loc, stageList(stageIDs...))
}

func (c *remoteFlowClient) handle(flowID string, stageID string, actionFunc interface{}, loc *codeLoc) string {
	return c.addStageWithClosure(flowID, models.ModelCompletionOperationHandle, actionFunc, loc, stageID)
}

func (c *remoteFlowClient) exceptionally(flowID string, stageID string, actionFunc interface{}, loc *codeLoc) string {
	return c.addStageWithClosure(flowID, models.ModelCompletionOperationExceptionally, actionFunc, loc, stageID)
}

func (c *remoteFlowClient) exceptionallyCompose(flowID string, stageID string, actionFunc interface{}, loc *codeLoc) string {
	return c.addStageWithClosure(flowID, models.ModelCompletionOperationExceptionallyCompose, actionFunc, loc, stageID)
}

func (c *remoteFlowClient) complete(flowID string, stageID string, value interface{}, loc *codeLoc) bool {
	panic("Not implemented")
}

func (c *remoteFlowClient) invokeFunction(flowID string, functionID string, req *HTTPRequest, loc *codeLoc) string {
	panic("Not implemented!")
}

func (c *remoteFlowClient) delay(flowID string, duration time.Duration, loc *codeLoc) string {
	// timeMs := int64(duration / time.Millisecond)
	panic("Not implemented")
	//
}

func (c *remoteFlowClient) getAsync(flowID string, stageID string, rType reflect.Type) (chan interface{}, chan error) {
	valueCh := make(chan interface{}, 1)
	errorCh := make(chan error, 1)
	go c.get(flowID, stageID, rType, valueCh, errorCh)
	return valueCh, errorCh
}

func (c *remoteFlowClient) get(flowID string, stageID string, rType reflect.Type, valueCh chan interface{}, errorCh chan error) {
	p := flowSvc.NewAwaitStageResultParams().WithFlowID(flowID).WithStageID(stageID)
	ok, err := c.flows.AwaitStageResult(p)
	if err != nil {
		log.Fatalf("Failed to add value stage: %v", err)
	}

	result := ok.Payload.Result
	val := result.DecodeValue(flowID, rType, c.blobStore)
	if result.Successful {
		debug("Getting successful result")
		valueCh <- val
	} else {
		debug("Getting failed result")
		errorCh <- err
	}
}

func (c *remoteFlowClient) commit(flowID string) {
	p := flowSvc.NewCommitParams().WithFlowID(flowID)
	_, err := c.flows.Commit(p)
	if err != nil {
		log.Fatalf("Failed to create flow: %v", err)
	}
}
