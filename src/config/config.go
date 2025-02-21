package config

import (
	"hiku/balancer"
	"hiku/proxy"
	"net/url"
)

// Config holds then configured values and objects to be used by the scheduler.
type Config struct {
	Host         string
	Port         int
	Balancer     balancer.Balancer
	ReverseProxy proxy.ReverseProxy
}

func CreateDefaultConfig() Config {
	return Config{
		Host:         "localhost",
		Port:         9020,
		Balancer:     balancer.NewPullBased(make([]url.URL, 0)),
		ReverseProxy: proxy.NewHTTPReverseProxy(),
	}
}
