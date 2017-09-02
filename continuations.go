package completions

import (
	"fmt"
	"reflect"
)

var registry = make(map[string]continuationEntry)

type continuationEntry interface {
	invoke(args ...interface{}) (interface{}, error)
}

type functionContinuationEntry struct {
	continuation interface{}
	args         []reflect.Kind
}

func (e *functionContinuationEntry) invoke(args ...interface{}) (result interface{}, err error) {
	// catch panics and return them as errors
	defer func() {
		if r := recover(); r != nil {
			err = fmt.Errorf("%v", r)
		}
	}()
	values := invokeContinuation(e.continuation, args...)
	switch len(values) {
	case 0:
		return nil, nil
	case 1:
		return valToInterface(values[0]), nil
	case 2:
		return valToInterface(values[0]), valToError(values[1])
	default:
		return nil, fmt.Errorf("Invalid continuation")
	}
}

func valToInterface(v reflect.Value) interface{} {
	return v.Interface()
}

func valToError(v reflect.Value) error {
	if v.IsNil() {
		return nil
	}
	return valToInterface(v).(error)
}

func invokeContinuation(fn interface{}, args ...interface{}) []reflect.Value {
	v := reflect.ValueOf(fn)
	rargs := make([]reflect.Value, len(args))
	for i, a := range args {
		rargs[i] = reflect.ValueOf(a)
	}
	return v.Call(rargs)
}

func key(continuation interface{}) string {
	rt := reflect.TypeOf(continuation)
	return rt.String()
}

//type FunctionContinuation func(arg0 interface{}) (interface{}, error)

func Register(continuation interface{}, args ...reflect.Kind) {
	if reflect.TypeOf(continuation).Kind() != reflect.Func {
		panic("Continuation must be a function!")
	}
	registry[key(continuation)] = &functionContinuationEntry{
		continuation: continuation,
		args:         args,
	}
}

func invoke(continuation interface{}, args ...interface{}) (interface{}, error) {
	if e, ok := registry[key(continuation)]; !ok {
		panic("Continuation not registered")
	} else {
		return e.invoke(args...)
	}
}
