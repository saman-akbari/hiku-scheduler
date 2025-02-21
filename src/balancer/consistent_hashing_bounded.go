package balancer

import (
	"log"
	"net/http"
	"net/url"

	"hiku/httputil"
	"hiku/lambda"

	"github.com/lafikl/consistent"
)

type ConsistentHashingBounded struct {
	hashRing  *consistent.Consistent
	workerMap map[string]url.URL
}

func (b *ConsistentHashingBounded) SelectWorker(r *http.Request, l *lambda.Lambda) (url.URL, *httputil.HttpError) {
	if len(b.workerMap) == 0 {
		return url.URL{}, httputil.New500Error("Can't select worker, Workers empty")
	}

	host, err := b.hashRing.GetLeast(r.URL.String())
	if err != nil {
		log.Fatal(err)
	}

	b.hashRing.Inc(host)
	return b.workerMap[host], nil
}

func (b *ConsistentHashingBounded) ReleaseWorker(workerUrl url.URL, l *lambda.Lambda) {
	b.hashRing.Done(workerUrl.String())
}

func (b *ConsistentHashingBounded) AddWorker(workerUrl url.URL) {
	host := workerUrl.String()
	b.hashRing.Add(host)
	b.workerMap[host] = workerUrl
}

func (b *ConsistentHashingBounded) RemoveWorker(workerUrl url.URL) {
	host := workerUrl.String()
	b.hashRing.Remove(host)
	delete(b.workerMap, host)
}

func (b *ConsistentHashingBounded) GetAllWorkers() []url.URL {
	totalUrls := len(b.workerMap)
	urlSlice := make([]url.URL, totalUrls)
	i := 0
	for _, indexedUrl := range b.workerMap {
		urlSlice[i] = indexedUrl
		i++
	}
	return urlSlice
}

func (b *ConsistentHashingBounded) DestroySandbox(workerUrl url.URL, l *lambda.Lambda) {
}

func NewConsistentHashingBounded(workerUrls []url.URL) Balancer {
	workerMap := make(map[string]url.URL)
	hashRing := consistent.New()
	for _, workerUrl := range workerUrls {
		workerMap[workerUrl.String()] = workerUrl
		hashRing.Add(workerUrl.String())
	}

	return &ConsistentHashingBounded{hashRing, workerMap}
}

func NewConsistentHashingBoundedFromJSONSlice(jsonSlice []string) Balancer {
	return NewConsistentHashingBounded(CreateWorkerURLSlice(jsonSlice))
}
