package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"

	fdk "github.com/fnproject/fdk-go"
)

func main() {
	fdk.Handle(fdk.HandlerFunc(myHandler))
}

type GreetingRequest struct {
        Name string `json:"name"`
}

type GreetingResponse struct {
        Msg string `json:"message"`
}

func myHandler(ctx context.Context, in io.Reader, out io.Writer) {
	p := &GreetingRequest{Name: "World"}
	json.NewDecoder(in).Decode(p)
	resp := &GreetingResponse{Msg: fmt.Sprintf("Hello %v", p.Name)} 
	json.NewEncoder(out).Encode(resp)
}
