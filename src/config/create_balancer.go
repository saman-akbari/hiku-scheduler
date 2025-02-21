package config

import (
	"hiku/balancer"
)

func createBalancerFromConfig(c JSONConfig) balancer.Balancer {
	switch c.Balancer {
	case "hashing-bounded":
		return balancer.NewConsistentHashingBoundedFromJSONSlice(c.Workers)
	case "least-connections":
		return balancer.NewLeastConnectionsFromJSONSlice(c.Workers)
	case "pull-based":
		return balancer.NewPullBasedFromJSONSlice(c.Workers)
	case "random":
		return balancer.NewRandomFromJSONSlice(c.Workers)
	}

	panic("Unknown balancer: " + c.Balancer)
}
