package completions

import (
	"net/http"
	"os"
	"time"
)

var hc = &http.Client{
	Timeout: time.Second * 10,
}

func newCompletionService() completionService {
	if url, ok := os.LookupEnv("COMPLETER_BASE_URL"); !ok {
		panic("Missing COMPLETER_BASE_URL configuration in environment!")
	} else {
		return &httpCompletionService{url: url}
	}
}

type threadID string
type completionID string

type completionService interface {
	createThread() threadID
	completedValue(value interface{}) completionID
}

type httpCompletionService struct {
	url string
}

func (cs *httpCompletionService) createThread() threadID {
	return ""
}

func (cs *httpCompletionService) completedValue(value interface{}) completionID {
	return ""
}
