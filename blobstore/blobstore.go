package blobstore

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"os"
	"sync"
	"time"

	"github.com/fnproject/flow-lib-go/models"
)

var onceBS sync.Once
var blobStore BlobStoreClient

func GetBlobStore() BlobStoreClient {
	onceBS.Do(func() {
		var completerURL string
		var ok bool
		if completerURL, ok = os.LookupEnv("COMPLETER_BASE_URL"); !ok {
			log.Fatal("Missing COMPLETER_BASE_URL configuration in environment!")
		}
		blobStore = newHTTPBlobStoreClient(fmt.Sprintf("%s/blobs", completerURL))
	})
	return blobStore
}

type BlobResponse struct {
	BlobId      string `json:"blob_id"`
	BlobLength  int64  `json:"length"`
	ContentType string `json:"content_type"`
}

func (br *BlobResponse) BlobDatum() *models.ModelBlobDatum {
	return &models.ModelBlobDatum{BlobID: br.BlobId, ContentType: br.ContentType, Length: br.BlobLength}
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
		log.Fatalf("Write failed, got %d response from blobstore", r.StatusCode)
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
		log.Fatalf("Read failed, got %d response from blobstore", r.StatusCode)
	}

	bodyReader(r.Body)
}
