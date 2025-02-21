package test

import (
	"hiku/balancer"
	"net/http"
	"net/url"
	"testing"

	"hiku/lambda"
)

func createTestUrls(hosts []string) []url.URL {
	urls := make([]url.URL, len(hosts))
	for i, host := range hosts {
		urls[i] = url.URL{
			Scheme: "http",
			Host:   host,
		}
	}
	return urls
}

func createTestRequest(path string) *http.Request {
	req, _ := http.NewRequest("GET", "http://localhost:8080"+path, nil)
	return req
}

func TestBalancerInterface(t *testing.T) {
	testUrls := createTestUrls([]string{"worker1:8080", "worker2:8080"})

	balancers := map[string]func([]url.URL) balancer.Balancer{
		"Random":            balancer.NewRandom,
		"LeastConnections":  balancer.NewLeastConnections,
		"ConsistentHashing": balancer.NewConsistentHashingBounded,
		"PullBased":         balancer.NewPullBased,
	}

	for name, constructor := range balancers {
		t.Run(name, func(t *testing.T) {
			b := constructor(testUrls)

			workers := b.GetAllWorkers()
			if len(workers) != len(testUrls) {
				t.Errorf("expected %d workers, got %d", len(testUrls), len(workers))
			}

			newWorker := url.URL{Scheme: "http", Host: "worker3:8080"}
			b.AddWorker(newWorker)
			workers = b.GetAllWorkers()
			if len(workers) != len(testUrls)+1 {
				t.Errorf("expected %d workers after add, got %d", len(testUrls)+1, len(workers))
			}

			b.RemoveWorker(newWorker)
			workers = b.GetAllWorkers()
			if len(workers) != len(testUrls) {
				t.Errorf("expected %d workers after remove, got %d", len(testUrls), len(workers))
			}
		})
	}
}

func TestRandomBalancer(t *testing.T) {
	testUrls := createTestUrls([]string{"worker1:8080", "worker2:8080"})
	balancer := balancer.NewRandom(testUrls)
	testLambda := &lambda.Lambda{Name: "test"}

	selections := make(map[string]int)
	iterations := 1000

	for i := 0; i < iterations; i++ {
		worker, err := balancer.SelectWorker(createTestRequest("/run/test"), testLambda)
		if err != nil {
			t.Fatalf("failed to select worker: %v", err)
		}
		selections[worker.Host]++
	}

	for _, url := range testUrls {
		if selections[url.Host] == 0 {
			t.Errorf("worker %s was never selected", url.Host)
		}
	}
}

func TestLeastConnectionsBalancer(t *testing.T) {
	testUrls := createTestUrls([]string{"worker1:8080", "worker2:8080"})
	balancer := balancer.NewLeastConnections(testUrls)
	testLambda := &lambda.Lambda{Name: "test"}

	worker1, err := balancer.SelectWorker(createTestRequest("/run/test"), testLambda)
	if err != nil {
		t.Fatalf("failed to select first worker: %v", err)
	}

	worker2, err := balancer.SelectWorker(createTestRequest("/run/test"), testLambda)
	if err != nil {
		t.Fatalf("failed to select second worker: %v", err)
	}

	if worker1.Host == worker2.Host {
		t.Error("expected different workers to be selected")
	}

	balancer.ReleaseWorker(worker1, testLambda)

	worker3, err := balancer.SelectWorker(createTestRequest("/run/test"), testLambda)
	if err != nil {
		t.Fatalf("failed to select third worker: %v", err)
	}

	if worker3.Host != worker1.Host {
		t.Error("expected released worker to be selected again")
	}
}

func TestConsistentHashingBalancer(t *testing.T) {
	testUrls := createTestUrls([]string{"worker1:8080", "worker2:8080"})
	balancer := balancer.NewConsistentHashingBounded(testUrls)
	testLambda := &lambda.Lambda{Name: "test"}

	request1 := createTestRequest("/run/test")
	worker1, err := balancer.SelectWorker(request1, testLambda)
	if err != nil {
		t.Fatalf("failed to select first worker: %v", err)
	}

	worker2, err := balancer.SelectWorker(request1, testLambda)
	if err != nil {
		t.Fatalf("failed to select second worker: %v", err)
	}

	if worker1.Host != worker2.Host {
		t.Error("expected same worker for identical requests")
	}

	request2 := createTestRequest("/run/test2")
	worker3, err := balancer.SelectWorker(request2, testLambda)
	if err != nil {
		t.Fatalf("failed to select worker for different request: %v", err)
	}

	balancer.ReleaseWorker(worker1, testLambda)
	balancer.ReleaseWorker(worker3, testLambda)
}

func TestPullBasedBalancer(t *testing.T) {
	testUrls := createTestUrls([]string{"worker1:8080", "worker2:8080"})
	balancer := balancer.NewPullBased(testUrls)

	lambda1 := &lambda.Lambda{Name: "function1"}
	lambda2 := &lambda.Lambda{Name: "function2"}

	worker1, err := balancer.SelectWorker(createTestRequest("/run/function1"), lambda1)
	if err != nil {
		t.Fatalf("failed to select worker for function1: %v", err)
	}

	balancer.ReleaseWorker(worker1, lambda1)

	worker2, err := balancer.SelectWorker(createTestRequest("/run/function1"), lambda1)
	if err != nil {
		t.Fatalf("failed to select worker for repeated function1: %v", err)
	}
	if worker2.Host != worker1.Host {
		t.Error("expected same worker for repeated function1 request")
	}

	worker3, err := balancer.SelectWorker(createTestRequest("/run/function2"), lambda2)
	if err != nil {
		t.Fatalf("failed to select worker for function2: %v", err)
	}

	balancer.ReleaseWorker(worker3, lambda2)

	worker4, err := balancer.SelectWorker(createTestRequest("/run/function2"), lambda2)
	if err != nil {
		t.Fatalf("failed to select worker for repeated function2: %v", err)
	}
	if worker4.Host != worker3.Host {
		t.Error("expected same worker for repeated function2 request")
	}
}
