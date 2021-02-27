package main

import (
	"flag"
	"fmt"
	"log"
	"net"
	"net/http"
	"sort"
	"strings"
	"time"

	"github.com/docker/libnetwork/resolvconf"
	"github.com/docker/libnetwork/types"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

var (
	promNameserver = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: "node_dns",
			Name:      "nameserver",
		},
		[]string{"server"},
	)
	promSearchdomain = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: "node_dns",
			Name:      "searchdomain",
		},
		[]string{"host"},
	)
	promHostTest = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Namespace: "node_dns",
			Name:      "test_result",
		},
		[]string{"host", "result", "status"},
	)
)

var addr = flag.String("listen-address", ":8080", "The address to listen on for HTTP requests.")
var test = flag.String("test-hosts", "bt.no,vg.no,example.com", "Comma separated list of hosts to test DNS resolution")
var testInterval = flag.Int("test-interval-seconds", 10, "Interval in seconds for running test DNS resolution")

func init() {
	prometheus.MustRegister(promNameserver)
	prometheus.MustRegister(promSearchdomain)
	prometheus.MustRegister(promHostTest)
}

func main() {
	flag.Parse()

	conf, _ := resolvconf.Get()
	nameservers := resolvconf.GetNameservers(conf.Content, types.IPv4)
	searchdomains := resolvconf.GetSearchDomains(conf.Content)
	testhosts := strings.Split(*test, ",")

	for _, server := range nameservers {
		promNameserver.With(prometheus.Labels{"server": server}).Inc()
	}

	for _, host := range searchdomains {
		promSearchdomain.With(prometheus.Labels{"host": host}).Inc()
	}

	for _, host := range testhosts {
		ticker := time.NewTicker(time.Duration(float64(*testInterval)) * time.Second)
		quit := make(chan struct{})
		go func(host string, metric *prometheus.GaugeVec) {
			for {
				select {
				case <-ticker.C:
					result, _ := net.LookupHost(host)
					sort.Strings(result)
					fmt.Printf("Resolved %s to %s\n", host, result)
					metric.With(prometheus.Labels{"host": host, "status": "success", "result": strings.Join(result, ",")}).Set(1)
					metric.With(prometheus.Labels{"host": host, "status": "failed", "result": ""}).Set(0)
				case <-quit:
					ticker.Stop()
					return
				}
			}
		}(host, promHostTest)
	}

	http.Handle("/metrics", promhttp.Handler())
	log.Fatal(http.ListenAndServe("127.0.0.1:8080", nil))
}
