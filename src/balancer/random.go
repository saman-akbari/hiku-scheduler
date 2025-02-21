package balancer

import (
	"math/rand"
	"net/http"
	"net/url"
	"time"

	"hiku/httputil"
	"hiku/lambda"
)

type Random struct {
	workerUrls []url.URL
	rng        *rand.Rand
}

func (b *Random) SelectWorker(r *http.Request, l *lambda.Lambda) (url.URL, *httputil.HttpError) {
	workerUrls := b.workerUrls
	var totalWorkers = len(workerUrls)
	if totalWorkers == 0 {
		return url.URL{}, httputil.New500Error("Can't select worker, Workers empty")
	}

	randomIndex := rand.Intn(totalWorkers)
	return workerUrls[randomIndex], nil
}

func (b *Random) AddWorker(workerURL url.URL) {
	b.workerUrls = append(b.workerUrls, workerURL)
}

func (b *Random) ReleaseWorker(workerURL url.URL, l *lambda.Lambda) {
}

func (b *Random) RemoveWorker(targetURL url.URL) {
	source := b.workerUrls
	targetIndex := FindUrlInSlice(source, targetURL)
	if targetIndex > -1 {
		b.workerUrls = append(source[:targetIndex], source[targetIndex+1:]...)
	}
}

func (b *Random) GetAllWorkers() []url.URL {
	workerUrls := b.workerUrls

	dest := make([]url.URL, len(workerUrls))
	copy(dest, workerUrls)
	return dest
}

func (b *Random) DestroySandbox(workerURL url.URL, l *lambda.Lambda) {

}

func NewRandom(workerUrls []url.URL) Balancer {
	rng := rand.New(rand.NewSource(time.Now().UnixNano()))
	return &Random{workerUrls: workerUrls, rng: rng}
}

func NewRandomFromJSONSlice(jsonSlice []string) Balancer {
	return NewRandom(CreateWorkerURLSlice(jsonSlice))
}
