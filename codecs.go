package completions

import (
	"strings"
)

type codec interface {
	getAppName() string
	getRoute() string
}

type defaultCodec struct {
	appName string
	route   string
}

func newCodec() codec {
	if format, ok := lookupEnv("FN_FORMAT"); ok && strings.ToLower(format) == "http" {
		panic("Hot functions not supported!")
	}
	return &defaultCodec{
		appName: lookupReqEnv("APP_NAME"),
		route:   lookupReqEnv("ROUTE"),
	}
}

func (c *defaultCodec) getAppName() string {
	return c.appName
}

func (c *defaultCodec) getRoute() string {
	return c.route
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
