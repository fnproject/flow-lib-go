package flows

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"time"
)

type BlobResponse struct {
	BlobId      string `json:"blob_id"`
	BlobLength  int64  `json:"length"`
	ContentType string `json:"content_type"`
}

type BlobStoreClient interface {
	WriteBlob(prefix string, contentType string, bytes io.Reader) *BlobResponse
	ReadBlob(prefix string, blobID string, expectedContentType string, bodyReader func(body io.ReadCloser))
}

type HTTPBlobStoreClient struct {
	urlBase string
	hc      *http.Client
}

func newHTTPBlobStoreClient(urlBase string) BlobStoreClient {
	return &HTTPBlobStoreClient{
		urlBase: urlBase,
		hc: &http.Client{
			Transport: &http.Transport{
				Dial: (&net.Dialer{
					Timeout:   30 * time.Second,
					KeepAlive: 30 * time.Second,
				}).Dial,
				TLSHandshakeTimeout:   10 * time.Second,
				ResponseHeaderTimeout: 10 * time.Minute,
				ExpectContinueTimeout: 1 * time.Second,
			},
		},
	}
}

func (c *HTTPBlobStoreClient) WriteBlob(prefix string, contentType string, bytes io.Reader) *BlobResponse {
	r, err := c.hc.Post(fmt.Sprintf("%s/%s", c.urlBase, prefix), contentType, bytes)
	if err != nil {
		log.Fatalf("Failed to write blob: %v", err)
	}
	defer r.Body.Close()

	if r.StatusCode != 200 {
		log.Fatalf("Got %d response from blobstore", r.StatusCode)
	}

	res := &BlobResponse{}
	err = json.NewDecoder(r.Body).Decode(res)
	if err != nil {
		log.Fatal("Failed to deserialize blob response")
	}
	return res
}

func (c *HTTPBlobStoreClient) ReadBlob(prefix string, blobID string, expectedContentType string, bodyReader func(body io.ReadCloser)) {
	req, _ := http.NewRequest("GET", fmt.Sprintf("%s/%s/%s", c.urlBase, prefix, blobID), nil)
	req.Header.Set("Accept", expectedContentType)
	r, err := c.hc.Do(req)
	if err != nil {
		log.Fatalf("Failed to read blob: %v", err)
	}
	defer r.Body.Close()

	if r.StatusCode != 200 {
		log.Fatalf("Got %d response from blobstore", r.StatusCode)
	}

	bodyReader(r.Body)
}
