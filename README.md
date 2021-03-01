 node-dns-exporter

Prometheus exporter for node level DNS metrics. This is intended to run as a
DaemonSet in your Kubernetes cluster to report DNS client metrics from each node.

## Usage

```
Usage of node-dns-exporter:
  -listen-address string
        The address to listen on for HTTP requests. (default "127.0.0.1:8080")
  -test-hosts string
        Comma separated list of hosts to test DNS resolution (default "nrk.no,vg.no,example.com")
  -test-interval-seconds int
        Interval in seconds for running test DNS resolution (default 10)
```

## Features

* [x] resolvconf metrics (`node_dns_nameserver` and `node_dns_searchdomain`)
* [x] resolv a set of sample domain names evry x seconds to test that name

## Example Metrics

```
# HELP node_dns_nameserver
# TYPE node_dns_nameserver counter
node_dns_nameserver{server="1.1.1.1"} 1
node_dns_nameserver{server="192.168.40.44"} 1
# HELP node_dns_test_result
# TYPE node_dns_test_result gauge
node_dns_test_result{host="example.com",result="",status="failed"} 0
node_dns_test_result{host="example.com",result="2606:2800:220:1:248:1893:25c8:1946,93.184.216.34",status="success"} 1
node_dns_test_result{host="nrk.no",result="",status="failed"} 0
node_dns_test_result{host="nrk.no",result="23.36.77.90,23.36.77.99,2a02:26f0:4300::1724:4cf1,2a02:26f0:4300::1724:4cf8",status="success"} 1
node_dns_test_result{host="vg.no",result="",status="failed"} 0
node_dns_test_result{host="vg.no",result="195.88.54.16,195.88.55.16,2001:67c:21e0::16",status="success"} 1
```
