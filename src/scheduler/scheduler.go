package scheduler

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"time"

	"hiku/balancer"
	"hiku/config"
	"hiku/httputil"
	"hiku/lambda"
	"hiku/proxy"
)

// Scheduler is an object that can schedule lambda function workloads to a pool of workers.
type Scheduler struct {
	balancer balancer.Balancer
	proxy    proxy.ReverseProxy
}

// Run is an HTTP request handler that expects requests of form
// /run/<lambdaName>. It extracts the lambda name from the request path
// and then chooses a worker to run the lambda workload using the configured
// load balancer. The lambda response is forwarded to the client "as-is"
// without any modifications.
func (s *Scheduler) Run(w http.ResponseWriter, r *http.Request) {
	l, err := s.getLambdaInfoFromRequest(r)

	if err != nil {
		httputil.RespondWithError(w, err)
		return
	}

	// Select worker and serve http
	startTime := time.Now()
	selectedWorkerURL, err := s.balancer.SelectWorker(r, l)
	log.Printf("Selected worker: %s in %d ns [%s]", selectedWorkerURL.String(), time.Since(startTime).Nanoseconds(), r.URL.Path)
	if err != nil {
		httputil.RespondWithError(w, err)
		return
	}

	s.proxy.ProxyRequest(selectedWorkerURL, w, r)
	s.balancer.ReleaseWorker(selectedWorkerURL, l)
}

func (s *Scheduler) AddWorkers(urls []url.URL) {
	for _, workerURL := range urls {
		s.balancer.AddWorker(workerURL)
	}
}
func (s *Scheduler) RemoveWorkers(urls []url.URL) {
	for _, workerURL := range urls {
		s.balancer.RemoveWorker(workerURL)
	}
}

func (s *Scheduler) DestroySandbox(r *http.Request) {
	l, err := s.getLambdaInfoFromRequest(r)

	if err != nil {
		log.Printf("Error destroying sandbox: %v", err)
		return
	}

	var workerUrl *url.URL
	decodingError := json.NewDecoder(r.Body).Decode(&workerUrl)
	if decodingError != nil {
		log.Printf("Error decoding workerUrl: %v", decodingError)
		return
	}

	s.balancer.DestroySandbox(*workerUrl, l)
}

func (s *Scheduler) StatusCheckAllWorkers(w http.ResponseWriter, r *http.Request) {
	for _, workerUrl := range s.balancer.GetAllWorkers() {
		s.proxy.ProxyRequest(workerUrl, w, r)
	}
}

func (s *Scheduler) getLambdaInfoFromRequest(r *http.Request) (*lambda.Lambda, *httputil.HttpError) {
	lambdaName := httputil.Get2ndPathSegment(r, "run")
	if lambdaName == "" {
		lambdaName = httputil.Get2ndPathSegment(r, "destroySandbox")
	}

	if lambdaName == "" {
		return nil, &httputil.HttpError{
			Msg:  fmt.Sprintf("Could not find lambda name in path %s", r.URL.Path),
			Code: http.StatusBadRequest}
	}

	return &lambda.Lambda{Name: lambdaName}, nil
}

func NewScheduler(c config.Config) *Scheduler {
	return &Scheduler{
		c.Balancer,
		c.ReverseProxy,
	}

}
