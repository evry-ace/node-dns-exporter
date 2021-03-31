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
		[]string{"server", "node"},
	)
	promSearchdomain = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: "node_dns",
			Name:      "searchdomain",
		},
		[]string{"host", "node"},
	)
	promHostTest = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Namespace: "node_dns",
			Name:      "test_result",
		},
		[]string{"host", "result", "status", "node"},
	)
)

var addr = flag.String("listen-address", "127.0.0.1:8080", "The address to listen on for HTTP requests.")
var test = flag.String("test-hosts", "nrk.no,vg.no,example.com", "Comma separated list of hosts to test DNS resolution")
var requiredSearchdomains = flag.String("req-searchdomains","", "Comma separated list of required searchdomains")
var testInterval = flag.Int("test-interval-seconds", 10, "Interval in seconds for running test DNS resolution")
var nodeName = flag.String("node-name", "localhost", "The name of the node running the container")

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
		promNameserver.With(prometheus.Labels{"server": server, "node": *nodeName}).Inc()
	}

	searchdomains := resolvconf.GetSearchDomains(conf.Content)
	
	if *requiredSearchdomains != "" {
		sort.Strings(searchdomains)
		reqSearchdomains := strings.Split(strings.ReplaceAll(*requiredSearchdomains," ",""), ",")
		for _, host := range reqSearchdomains{
			i := sort.SearchStrings(searchdomains,host)
			if !(i < len(searchdomains) && searchdomains[i] == host){
				promSearchdomain.With(prometheus.Labels{"host": host, "node": *nodeName}).Desc()
			}
		}
	}

	for _, host := range searchdomains {
		promSearchdomain.With(prometheus.Labels{"host": host, "node": *nodeName}).Inc()
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
						resultErrSlice := strings.Split(resultErr, ":")
						resultErr = strings.TrimPrefix(resultErrSlice[len(resultErrSlice)-1]," ")
						fmt.Printf("Error: %s\n", err.Error())
						
						metric.With(prometheus.Labels{"host": host, "status": "failed", "result": resultErr, "node": *nodeName}).Set(1)
						if resultPrev != "" {
							metric.With(prometheus.Labels{"host": host, "status": "success", "result": resultPrev, "node": *nodeName}).Set(0)
							resultPrev = ""
						}
						// Reset previous error result if it has changed
						if resultPrevErr != "" && resultErr != resultPrevErr {
							metric.With(prometheus.Labels{"host": host, "status": "failed", "result": resultPrevErr, "node": *nodeName}).Set(0)
						}
						
						resultPrevErr = resultErr

					} else {
						sort.Strings(result)
						fmt.Printf("Resolved %s to %s\n", host, result)
						resultString := strings.Join(result, ",")

						metric.With(prometheus.Labels{"host": host, "status": "success", "result": resultString, "node": *nodeName}).Set(1)
						if resultPrevErr != "" {
							metric.With(prometheus.Labels{"host": host, "status": "failed", "result": resultPrevErr, "node": *nodeName}).Set(0)
							resultPrevErr = ""
						}
						// Reset previous success result if it has changed
						if resultPrev != "" && resultString != resultPrev {
							metric.With(prometheus.Labels{"host": host, "status": "success", "result": resultPrev, "node": *nodeName}).Set(0)
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
