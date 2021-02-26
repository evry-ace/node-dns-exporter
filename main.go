package main

import (
	"log"
	"net/http"

	"github.com/docker/libnetwork/resolvconf"
	"github.com/docker/libnetwork/types"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

var (
	promNameserver = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "nameserver",
		},
		[]string{"server"},
	)
	promSearchdomain = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "searchdomain",
		},
		[]string{"domain"},
	)
)

func init() {
	prometheus.MustRegister(promNameserver)
	prometheus.MustRegister(promSearchdomain)
}

func main() {
	//fmt.Println(net.LookupHost("example.com"))

	conf, _ := resolvconf.Get()
	nameservers := resolvconf.GetNameservers(conf.Content, types.IPv4)
	searchdomains := resolvconf.GetSearchDomains(conf.Content)

	for _, server := range nameservers {
		promNameserver.With(prometheus.Labels{"server": server}).Inc()
	}

	for _, domain := range searchdomains {
		promSearchdomain.With(prometheus.Labels{"domain": domain}).Inc()
	}

	http.Handle("/metrics", promhttp.Handler())
	log.Fatal(http.ListenAndServe("127.0.0.1:8080", nil))
}
