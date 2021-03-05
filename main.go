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

var addr = flag.String("listen-address", "127.0.0.1:8080", "The address to listen on for HTTP requests.")
var test = flag.String("test-hosts", "nrk.no,vg.no,example.com", "Comma separated list of hosts to test DNS resolution")
var testInterval = flag.Int("test-interval-seconds", 10, "Interval in seconds for running test DNS resolution")

func init() {
	prometheus.MustRegister(promNameserver)
	prometheus.MustRegister(promSearchdomain)
	prometheus.MustRegister(promHostTest)
}

func main() {
	flag.Parse()

	conf, err := resolvconf.Get()
	if err != nil {
		panic(err)
	}

	nameservers := resolvconf.GetNameservers(conf.Content, types.IPv4)
	for _, server := range nameservers {
		promNameserver.With(prometheus.Labels{"server": server}).Inc()
	}

	searchdomains := resolvconf.GetSearchDomains(conf.Content)
	for _, host := range searchdomains {
		promSearchdomain.With(prometheus.Labels{"host": host}).Inc()
	}

	testhosts := strings.Split(strings.ReplaceAll(*test," ",""), ",")
	for _, host := range testhosts {
		ticker := time.NewTicker(time.Duration(float64(*testInterval)) * time.Second)
		quit := make(chan struct{})
		go func(host string, metric *prometheus.GaugeVec) {
			resultPrev := ""
			resultPrevErr := ""
			for {
				select {
				case <-ticker.C:
					result, err := net.LookupHost(host)

					if err != nil {
						resultErr := err.Error()
						fmt.Printf("Error: %s\n", resultErr)
						
						metric.With(prometheus.Labels{"host": host, "status": "failed", "result": resultErr}).Set(1)
						if resultPrev != "" {
							metric.With(prometheus.Labels{"host": host, "status": "success", "result": resultPrev}).Set(0)
						}
						// Reset previous result if it has changed
						if resultPrevErr != "" && resultErr != resultPrevErr {
							metric.With(prometheus.Labels{"host": host, "status": "failed", "result": resultPrevErr}).Set(0)
							resultErr = ""
						}
						
						resultPrevErr = resultErr

					} else {
						sort.Strings(result)
						fmt.Printf("Resolved %s to %s\n", host, result)
						resultString := strings.Join(result, ",")

						metric.With(prometheus.Labels{"host": host, "status": "success", "result": resultString}).Set(1)
						if resultPrevErr != "" {
							metric.With(prometheus.Labels{"host": host, "status": "failed", "result": resultPrevErr}).Set(0)
						}
						// Reset previous result if it has changed
						if resultPrev != "" && resultString != resultPrev {
							metric.With(prometheus.Labels{"host": host, "status": "success", "result": resultPrev}).Set(0)
						}

						resultPrev = resultString
					}
				case <-quit:
					ticker.Stop()
					return
				}
			}
		}(host, promHostTest)
	}

	http.Handle("/metrics", promhttp.Handler())
	log.Fatal(http.ListenAndServe(*addr, nil))
}
