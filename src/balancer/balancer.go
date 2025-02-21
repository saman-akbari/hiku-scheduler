package balancer

import (
	"net/http"
	"net/url"

	"hiku/httputil"
	"hiku/lambda"
)

type Balancer interface {
	SelectWorker(r *http.Request, l *lambda.Lambda) (url.URL, *httputil.HttpError)
	ReleaseWorker(workerUrl url.URL, l *lambda.Lambda)
	AddWorker(workerUrl url.URL)
	RemoveWorker(workerUrl url.URL)
	GetAllWorkers() []url.URL
	DestroySandbox(workerUrl url.URL, l *lambda.Lambda)
}
