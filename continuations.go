package completions

import (
	"fmt"
	"reflect"
)

var continuations = make(map[string]interface{})

func invoke(continuation interface{}, args ...interface{}) (result interface{}, err error) {
	// catch panics and return them as errors
	defer func() {
		if r := recover(); r != nil {
			err = fmt.Errorf("%v", r)
		}
	}()
	values := invokeContinuation(continuation, args...)
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

func invokeContinuation(continuation interface{}, args ...interface{}) []reflect.Value {
	v := reflect.ValueOf(continuation)
	rargs := make([]reflect.Value, len(args))
	for i, a := range args {
		rargs[i] = reflect.ValueOf(a)
	}
	return v.Call(rargs)
}

func continuationKey(continuation interface{}) string {
	rt := reflect.TypeOf(continuation)
	return rt.String()
}

func RegisterContinuation(continuation interface{}) {
	if reflect.TypeOf(continuation).Kind() != reflect.Func {
		panic("Continuation must be a function!")
	}
	continuations[continuationKey(continuation)] = continuation
}

func invokeFromRegistry(continuationKey string, args ...interface{}) (interface{}, error) {
	if e, ok := continuations[continuationKey]; !ok {
		panic("Continuation not registered")
	} else {
		return invoke(e, args...)
	}
}
