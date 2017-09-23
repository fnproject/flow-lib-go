package completions

import (
	"strings"
)

const (
	method     = "FN_METHOD"
	appNameEnv = "FN_APP_NAME"
	pathEnv    = "FN_PATH"
	reqUrlEnv  = "FN_REQUEST_URL"
	formatEnv  = "FN_FORMAT"
)

type codec interface {
	getAppName() string
	getRoute() string
	isContinuation() bool
	getHeader(string) (string, bool)
}

type defaultCodec struct {
	appName string
	route   string
}

func newCodec() codec {
	if format, ok := lookupEnv(formatEnv); ok && strings.ToLower(format) == "http" {
		panic("Hot functions not supported!")
	}
	return &defaultCodec{
		appName: lookupReqEnv(appNameEnv),
		route:   lookupReqEnv(pathEnv),
	}
}

func (c *defaultCodec) getAppName() string {
	return c.appName
}

func (c *defaultCodec) getRoute() string {
	return c.route
}

func (c *defaultCodec) isContinuation() bool {
	_, ok := c.getHeader(StageIDHeader)
	return ok
}

func (c *defaultCodec) getHeader(header string) (string, bool) {
	header = strings.Replace(header, "-", "_", -1)
	header = "HEADER_" + header
	return lookupEnv(strings.ToUpper(header))
}

func getFunctionID(c codec) string {
	return c.getAppName() + c.getRoute()
}

func lookupReqEnv(key string) string {
	v, ok := lookupEnv(key)
	if !ok {
		panic("Missing required environment variable: " + key)
	}
	return v
}
