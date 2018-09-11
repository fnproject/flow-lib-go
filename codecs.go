package flow

import (
	"context"
	"fmt"
	"io"

	fdk "github.com/fnproject/fdk-go"
)

const (
	fnID        = "FN_FN_ID"
)

type codec interface {
	getFnID() string
	isContinuation() bool
	getFlowID() string
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

func (c *fdkCodec) getFnID() string {
	return fdk.Context(c.ctx).Config[fnID]
}

func (c *fdkCodec) isContinuation() bool {
	_, ok := c.getHeader(StageIDHeader)
	return ok
}

func (c *fdkCodec) getFlowID() string {
	fid, ok := c.getHeader(FlowIDHeader)
	if !ok {
		panic("Missing flow ID in continuation")
	}
	return fid
}

func (c *fdkCodec) getHeader(header string) (string, bool) {
	//debug(fmt.Sprintf("headers: %v", fdk.Context(c.ctx).Header))
	//debug(fmt.Sprintf("env: %v", os.Environ()))

	v := fdk.Context(c.ctx).Header.Get(header)
	debug(fmt.Sprintf("header: %s %v %s", header, v != "", v))
	return v, v != ""
}

func (c *fdkCodec) in() io.Reader {
	return c.input
}

func (c *fdkCodec) out() io.Writer {
	return c.output
}

func getFunctionID(c codec) string {
	return c.getFnID()
}
