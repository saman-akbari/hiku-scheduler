package balancer

import (
	"math/rand"
	"net/http"
	"net/url"
	"sync"

	"hiku/httputil"
	"hiku/lambda"
)

type LeastConnections struct {
	workerUrls    []url.URL
	connectionMap map[url.URL]uint
	mutex         *sync.Mutex
}

func (b *LeastConnections) getWorkerLoad(workerUrl url.URL) uint {
	connections, _ := b.connectionMap[workerUrl]
	return connections
}

func (b *LeastConnections) incrementWorkerLoad(workerUrl url.URL) {
	connections, _ := b.connectionMap[workerUrl]
	b.connectionMap[workerUrl] = connections + 1
}

func (b *LeastConnections) decrementWorkerLoad(workerUrl url.URL) {
	connections, _ := b.connectionMap[workerUrl]
	b.connectionMap[workerUrl] = connections - 1
}

func (b *LeastConnections) SelectWorker(r *http.Request, l *lambda.Lambda) (url.URL, *httputil.HttpError) {
	b.mutex.Lock()
	defer b.mutex.Unlock()

	workerUrls := b.workerUrls
	if len(workerUrls) == 0 {
		return url.URL{}, httputil.New500Error("Can't select worker, Workers empty")
	}

	leastConnectionsUrl := workerUrls[0]
	leastConnections := b.getWorkerLoad(leastConnectionsUrl)
	tiedWorkers := []url.URL{leastConnectionsUrl}

	for _, workerUrl := range b.workerUrls[1:] {
		tempConnections := b.getWorkerLoad(workerUrl)
		if tempConnections < leastConnections {
			leastConnectionsUrl = workerUrl
			leastConnections = tempConnections
			tiedWorkers = []url.URL{leastConnectionsUrl}
		} else if tempConnections == leastConnections {
			tiedWorkers = append(tiedWorkers, workerUrl)
		}
	}

	// If there are tied workers, select one randomly
	if len(tiedWorkers) > 1 {
		randomIndex := rand.Intn(len(tiedWorkers))
		leastConnectionsUrl = tiedWorkers[randomIndex]
	}

	b.incrementWorkerLoad(leastConnectionsUrl)
	return leastConnectionsUrl, nil
}

func (b *LeastConnections) ReleaseWorker(workerURL url.URL, l *lambda.Lambda) {
	b.mutex.Lock()
	defer b.mutex.Unlock()

	b.decrementWorkerLoad(workerURL)
}

func (b *LeastConnections) AddWorker(workerURL url.URL) {
	b.workerUrls = append(b.workerUrls, workerURL)
	b.connectionMap[workerURL] = 0
}

func (b *LeastConnections) GetAllWorkers() []url.URL {
	workerUrls := b.workerUrls

	dest := make([]url.URL, len(workerUrls))
	copy(dest, workerUrls)
	return dest
}

func (b *LeastConnections) RemoveWorker(targetURL url.URL) {
	source := b.workerUrls
	targetIndex := FindUrlInSlice(source, targetURL)
	b.workerUrls = append(source[:targetIndex], source[targetIndex+1:]...)
}

func NewLeastConnections(workerUrls []url.URL) Balancer {
	leastConnections := &LeastConnections{workerUrls, make(map[url.URL]uint), &sync.Mutex{}}

	for _, workerURL := range workerUrls {
		leastConnections.connectionMap[workerURL] = 0
	}

	return leastConnections
}

func (b *LeastConnections) DestroySandbox(workerUrl url.URL, l *lambda.Lambda) {
}

func NewLeastConnectionsFromJSONSlice(jsonSlice []string) Balancer {
	return NewLeastConnections(CreateWorkerURLSlice(jsonSlice))
}
