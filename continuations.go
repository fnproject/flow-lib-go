package completions

var registry = make(map[string]continuationEntry)

type continuationEntry interface {
	invoke(args ...interface{}) (interface{}, error)
	key() string
}

type functionContinuationEntry struct {
	continuation FunctionContinuation
	arg0         interface{}
}

func (e *functionContinuationEntry) invoke(args ...interface{}) (interface{}, error) {
	return nil, nil
}

func (e *functionContinuationEntry) key() string {
	return ""
}

type FunctionContinuation func(arg0 interface{}) (interface{}, error)

func Register(continuation FunctionContinuation, arg interface{}) {
	e := &functionContinuationEntry{continuation: continuation, arg0: arg}
	registry[e.key()] = e
}
