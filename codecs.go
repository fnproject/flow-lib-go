package flows

import (
	"context"
	"io"

	fdk "github.com/fnproject/fdk-go"
)

const (
	method         = "FN_METHOD"
	appName        = "FN_APP_NAME"
	path           = "FN_PATH"
	reqUrl         = "FN_REQUEST_URL"
	format         = "FN_FORMAT"
	fnHeaderPrefix = "FN_HEADER_"
)

type codec interface {
	getAppName() string
	getRoute() string
	isContinuation() bool
	getHeader(string) (string, bool)
	getFlowID() flowID
	in() io.Reader
	out() io.Writer
}

type fdkCodec struct {
	ctx    context.Context
	input  io.Reader
	output io.Writer
}

func newCodec(ctx context.Context, in io.Reader, out io.Writer) codec {
	return &fdkCodec{ctx, in, out}
}

func (c *fdkCodec) getAppName() string {
	return fdk.Context(c.ctx).Config[appName]
}

func (c *fdkCodec) getRoute() string {
	return fdk.Context(c.ctx).Config[path]
}

func (c *fdkCodec) isContinuation() bool {
	_, ok := fdk.Context(c.ctx).Header[StageIDHeader]
	return ok
}

func (c *fdkCodec) in() io.Reader {
	return c.input
}

func (c *fdkCodec) out() io.Writer {
	return c.output
}

func (c *fdkCodec) getFlowID() flowID {
	fid := fdk.Context(c.ctx).Header.Get(FlowIDHeader)
	if fid == "" {
		panic("Missing flow ID in continuation")
	}
	return flowID(fid)
}

func (c *fdkCodec) getHeader(header string) (string, bool) {
	val := fdk.Context(c.ctx).Header.Get(header)
	return val, val != ""
}

func getFunctionID(c codec) string {
	return c.getAppName() + c.getRoute()
}
