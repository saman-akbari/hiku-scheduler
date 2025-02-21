package balancer

import (
	"container/heap"
	"math/rand"
	"net/http"
	"net/url"
	"sync"

	"hiku/httputil"
	"hiku/lambda"
)

type IdleQueue struct {
	functionType string
	queue        PriorityQueue
}

type PriorityQueue []*Item

type Item struct {
	url   url.URL
	load  uint
	index int
}

func (pq *PriorityQueue) Len() int { return len(*pq) }

func (pq *PriorityQueue) Less(i, j int) bool {
	return (*pq)[i].load < (*pq)[j].load
}

func (pq *PriorityQueue) Swap(i, j int) {
	(*pq)[i], (*pq)[j] = (*pq)[j], (*pq)[i]
	(*pq)[i].index = i
	(*pq)[j].index = j
}

func (pq *PriorityQueue) Push(x interface{}) {
	n := len(*pq)
	item := x.(*Item)
	item.index = n
	*pq = append(*pq, item)
}

func (pq *PriorityQueue) Pop() interface{} {
	old := *pq
	n := len(old)
	item := old[n-1]
	old[n-1] = nil
	item.index = -1
	*pq = old[0 : n-1]
	return item
}

func (pq *PriorityQueue) Update(item *Item, newLoad uint) {
	item.load = newLoad
	heap.Fix(pq, item.index)
}

type PullBased struct {
	workerUrls []url.URL
	idleQueues map[string]*IdleQueue
	loadMap    map[url.URL]uint
	mutex      *sync.Mutex
}

func (b *PullBased) getWorkerLoad(workerUrl url.URL) uint {
	workerLoad, _ := b.loadMap[workerUrl]
	return workerLoad
}

func (b *PullBased) incrementWorkerLoad(workerUrl url.URL) {
	workerLoad, _ := b.loadMap[workerUrl]
	b.loadMap[workerUrl] = workerLoad + 1

	for _, idleQueue := range b.idleQueues {
		for _, item := range idleQueue.queue {
			if item.url.Host == workerUrl.Host {
				item.load = b.getWorkerLoad(workerUrl)
				heap.Fix(&idleQueue.queue, item.index)
			}
		}
	}
}

func (b *PullBased) decrementWorkerLoad(workerUrl url.URL) {
	workerLoad, _ := b.loadMap[workerUrl]
	b.loadMap[workerUrl] = workerLoad - 1

	for _, idleQueue := range b.idleQueues {
		for _, item := range idleQueue.queue {
			if item.url.Host == workerUrl.Host {
				item.load = b.getWorkerLoad(workerUrl)
				heap.Fix(&idleQueue.queue, item.index)
			}
		}
	}
}

func (b *PullBased) SelectWorker(r *http.Request, l *lambda.Lambda) (url.URL, *httputil.HttpError) {
	b.mutex.Lock()
	defer b.mutex.Unlock()

	queue := b.getIdleQueue(l.Name)
	for queue.Len() > 0 {
		item := heap.Pop(queue).(*Item)
		workerURL := item.url

		if FindUrlInSlice(b.workerUrls, workerURL) != -1 {
			b.incrementWorkerLoad(workerURL)
			return workerURL, nil
		}
	}

	return b.selectLeastLoadedWorker()
}

func (b *PullBased) selectLeastLoadedWorker() (url.URL, *httputil.HttpError) {
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

func (b *PullBased) ReleaseWorker(workerURL url.URL, l *lambda.Lambda) {
	b.mutex.Lock()
	defer b.mutex.Unlock()

	b.decrementWorkerLoad(workerURL)

	idleQueue := b.getIdleQueue(l.Name)
	item := &Item{
		url:  workerURL,
		load: b.getWorkerLoad(workerURL),
	}
	heap.Push(idleQueue, item)
}

func (b *PullBased) DestroySandbox(workerUrl url.URL, l *lambda.Lambda) {
	b.mutex.Lock()
	defer b.mutex.Unlock()

	idleQueue := b.getIdleQueue(l.Name)
	for i, item := range *idleQueue {
		if item.url.Host == workerUrl.Host {
			heap.Remove(idleQueue, i)
			break
		}
	}
}

func (b *PullBased) getIdleQueue(functionType string) *PriorityQueue {
	idleQueue, ok := b.idleQueues[functionType]
	if !ok {
		idleQueue = &IdleQueue{
			functionType: functionType,
			queue:        make(PriorityQueue, 0, 1024),
		}
		heap.Init(&idleQueue.queue)
		b.idleQueues[functionType] = idleQueue
	}
	return &idleQueue.queue
}

func (b *PullBased) AddWorker(workerURL url.URL) {
	b.mutex.Lock()
	defer b.mutex.Unlock()

	b.workerUrls = append(b.workerUrls, workerURL)
	b.loadMap[workerURL] = 0
}

func (b *PullBased) GetAllWorkers() []url.URL {
	b.mutex.Lock()
	defer b.mutex.Unlock()

	workerUrls := make([]url.URL, len(b.workerUrls))
	copy(workerUrls, b.workerUrls)
	return workerUrls
}

func (b *PullBased) RemoveWorker(targetURL url.URL) {
	b.mutex.Lock()
	defer b.mutex.Unlock()

	index := FindUrlInSlice(b.workerUrls, targetURL)
	b.workerUrls = append(b.workerUrls[:index], b.workerUrls[index+1:]...)
	delete(b.loadMap, targetURL)
}

func NewPullBased(workerUrls []url.URL) Balancer {
	pullBased := &PullBased{
		workerUrls: workerUrls,
		idleQueues: make(map[string]*IdleQueue),
		loadMap:    make(map[url.URL]uint),
		mutex:      &sync.Mutex{},
	}

	for _, workerURL := range workerUrls {
		pullBased.loadMap[workerURL] = 0
	}

	return pullBased
}

func NewPullBasedFromJSONSlice(jsonSlice []string) Balancer {
	return NewPullBased(CreateWorkerURLSlice(jsonSlice))
}
