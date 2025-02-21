package balancer

import (
	"log"
	"net/url"
)

func CreateWorkerURLSlice(jsonSlice []string) []url.URL {
	workersLength := len(jsonSlice)
	workerUrls := make([]url.URL, workersLength)

	for i, urlString := range jsonSlice {
		workerUrl, err := url.Parse(urlString)
		if err != nil {
			log.Fatalf("Config file Ill-formed, unable to parse URL " + urlString)
		}
		workerUrls[i] = *workerUrl
	}

	return workerUrls
}

func FindUrlInSlice(urlSlice []url.URL, target url.URL) int {
	totalItems := len(urlSlice)
	for i := 0; i < totalItems; i++ {
		if urlSlice[i] == target {
			return i
		}
	}
	return -1
}
