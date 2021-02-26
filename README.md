# node-dns-exporter

Prometheus exporter for node level DNS metrics. This is intended to run as a
DaemonSet in your Kubernetes cluster to report DNS client metrics from each node.

## Features

* [x] resolvconf metrics (`nameserver` and `searchdomain`)

## ToDo

* [ ] resolv a set of sample domain names evry x minutes to test that name
  resolution works for the particular node.
