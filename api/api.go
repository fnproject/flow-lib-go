package api

import (
	"reflect"
	"time"
)

type HTTPRequest struct {
}

type HTTPResponse struct {
}

type Flow interface {
	InvokeFunction(functionID string, arg *HTTPRequest) FlowFuture
	Supply(action interface{}) FlowFuture
	Delay(duration time.Duration) FlowFuture
	CompletedValue(value interface{}) FlowFuture // value can be an error
	EmptyFuture() FlowFuture
	AllOf(futures ...FlowFuture) FlowFuture
	AnyOf(futures ...FlowFuture) FlowFuture
}

type FlowFuture interface {
	Get() (chan interface{}, chan error)
	// Get result as the given type. E.g. for use with ThenCompose
	GetType(t reflect.Type) (chan interface{}, chan error)
	ThenApply(action interface{}) FlowFuture
	ThenCompose(action interface{}) FlowFuture
	ThenCombine(other FlowFuture, action interface{}) FlowFuture
	WhenComplete(action interface{}) FlowFuture
	ThenAccept(action interface{}) FlowFuture
	AcceptEither(other FlowFuture, action interface{}) FlowFuture
	ApplyToEither(other FlowFuture, action interface{}) FlowFuture
	ThenAcceptBoth(other FlowFuture, action interface{}) FlowFuture
	ThenRun(action interface{}) FlowFuture
	Handle(action interface{}) FlowFuture
	Exceptionally(action interface{}) FlowFuture
	ExceptionallyCompose(action interface{}) FlowFuture
	Complete(value interface{}) bool
}
