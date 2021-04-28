# jsonbox_exporter

Refer to prometheus-community/json_exporter, similar to  prometheus/blackbox_exporter, use one exporter to scrape more kind target.

A [prometheus](https://prometheus.io/) exporter which scrapes remote JSON by JSONPath.
For checking the JSONPath configuration supported by this exporter please head over [here](https://kubernetes.io/docs/reference/kubectl/jsonpath/).  
Checkout the [examples](/examples) directory for sample exporter configuration, prometheus configuration and expected data format.

## Example Usage

```console
$ cat examples/data.json
{
    "counter": 1234,
    "values": [
        {
            "id": "id-A",
            "count": 1,
            "some_boolean": true,
            "state": "ACTIVE"
        },
        {
            "id": "id-B",
            "count": 2,
            "some_boolean": true,
            "state": "INACTIVE"
        },
        {
            "id": "id-C",
            "count": 3,
            "some_boolean": false,
            "state": "ACTIVE"
        }
    ],
    "location": "mars"
}

$ cat examples/config.yml
---
modules:
  test:
    request:
      method: GET
      headers:
        X-Dummy: my-test-header
#      params:
#        test: 1
#        test1: 2
    metrics:
      - name: example_global_value
        path: "{ .counter }"
        help: Example of a top-level global value scrape in the json
        labels:
          environment: beta # static label
          location: "planet-{.location}"          # dynamic label
      - name: example_value
        type: object
        help: Example of sub-level value scrapes from a json
        path: '{.values[?(@.state == "ACTIVE")]}'
        labels:
          environment: beta # static label
          id: '{.id}'          # dynamic label
        values:
          active: 1      # static value
          count: '{.count}' # dynamic value
          boolean: '{.some_boolean}'

$ python -m SimpleHTTPServer 8000 &
Serving HTTP on 0.0.0.0 port 8000 ...

$ ./json_exporter --config.file examples/config.yml &

$ curl "http://localhost:7979/probe?module=test&target=http://localhost:8000/examples/data.json"
# HELP test_example_global_value Example of a top-level global value scrape in the json
# TYPE test_example_global_value untyped
test_example_global_value{environment="beta",location="planet-mars"} 1234
# HELP test_example_value_active Example of sub-level value scrapes from a json
# TYPE test_example_value_active untyped
test_example_value_active{environment="beta",id="id-A"} 1
test_example_value_active{environment="beta",id="id-C"} 1
# HELP test_example_value_boolean Example of sub-level value scrapes from a json
# TYPE test_example_value_boolean untyped
test_example_value_boolean{environment="beta",id="id-A"} 1
test_example_value_boolean{environment="beta",id="id-C"} 0
# HELP test_example_value_count Example of sub-level value scrapes from a json
# TYPE test_example_value_count untyped
test_example_value_count{environment="beta",id="id-A"} 1
test_example_value_count{environment="beta",id="id-C"} 3
# HELP test_request_duration Request target duration by Millisecond
# TYPE test_request_duration untyped
test_request_duration 6

# Abandoned This
test_request_status{target="http://localhost:8000/example/data.json"} 1

# To test through prometheus:
$ docker run --rm -it -p 9090:9090 -v $PWD/examples/prometheus.yml:/etc/prometheus/prometheus.yml --network host prom/prometheus
```
Then head over to http://localhost:9090/graph?g0.range_input=1h&g0.expr=example_value_active&g0.tab=1 or http://localhost:9090/targets to check the scraped metrics or the targets.

## Exposing metrics through HTTPS

TLS configuration supported by this exporter can be found at [exporter-toolkit/web](https://github.com/prometheus/exporter-toolkit/blob/v0.5.1/docs/web-configuration.md)


# Sending body content for HTTP POST


If `body` paramater is set in config, it will be sent by the exporter as the body content in the scrape request. The HTTP method will also be set as 'POST' in this case.
```yaml
body:
  content: |
    My static information: {"time_diff": "1m25s", "anotherVar": "some value"}
```

The body content can also be a Go Template (https://golang.org/pkg/text/template). All the functions from the Sprig library (https://masterminds.github.io/sprig/) can be used in the template.
All the query parameters sent by prometheus in the scrape query to the exporter, are available as values while rendering the template.

Example using template functions:
```yaml
body:
  content: |
    {"time_diff": "{{ duration `95` }}","anotherVar": "{{ randInt 12 30 }}"}
  templatize: true
```

Example using template functions with values from the query parameters:
```yaml
body:
  content: |
    {"time_diff": "{{ duration `95` }}","anotherVar": "{{ .myVal | first }}"}
  templatize: true
```
Then `curl "http://exporter:7979/probe?module=test&target=http://scrape_target:8080/test/data.json&myVal=something"`, would result in sending the following body as the HTTP POST payload to `http://scrape_target:8080/test/data.json`:
```
{"time_diff": "1m35s","anotherVar": "something"}.
```

# Docker

TBD
