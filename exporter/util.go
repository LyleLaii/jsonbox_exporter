package exporter

import (
	"context"
	"errors"
	"fmt"
	"github.com/Masterminds/sprig/v3"
	"github.com/go-kit/kit/log"
	"github.com/go-kit/kit/log/level"
	"github.com/prometheus/client_golang/prometheus"
	pconfig "github.com/prometheus/common/config"
	"io"
	"io/ioutil"
	"jsonbox_exporter/config"
	"math"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"text/template"
	"time"
)

func MakeMetricName(parts ...string) string {
	return strings.Join(parts, "_")
}

func SanitizeValue(s string) (float64, error) {
	var value float64
	var resultErr string

	if value, err := strconv.ParseFloat(s, 64); err == nil {
		return value, nil
	} else {
		resultErr = fmt.Sprintf("%s", err)
	}

	if boolValue, err := strconv.ParseBool(s); err == nil {
		if boolValue {
			return 1.0, nil
		} else {
			return 0.0, nil
		}
	} else {
		resultErr = resultErr + "; " + fmt.Sprintf("%s", err)
	}

	if s == "<nil>" {
		return math.NaN(), nil
	}
	return value, fmt.Errorf(resultErr)
}

func CreateMetricsList(c config.Config, module string) ([]JsonMetric, error) {
	var metrics []JsonMetric
	for _, metric := range c.Modules[module].Metrics {
		switch metric.Type {
		case config.ValueScrape:
			var variableLabels, variableLabelsValues []string
			for k, v := range metric.Labels {
				variableLabels = append(variableLabels, k)
				variableLabelsValues = append(variableLabelsValues, v)
			}
			jsonMetric := JsonMetric{
				Desc: prometheus.NewDesc(
					module + "_" + metric.Name,
					metric.Help,
					variableLabels,
					nil,
				),
				KeyJsonPath:     metric.Path,
				LabelsJsonPaths: variableLabelsValues,
			}
			metrics = append(metrics, jsonMetric)
		case config.ObjectScrape:
			for subName, valuePath := range metric.Values {
				name := MakeMetricName(metric.Name, subName)
				var variableLabels, variableLabelsValues []string
				for k, v := range metric.Labels {
					variableLabels = append(variableLabels, k)
					variableLabelsValues = append(variableLabelsValues, v)
				}
				jsonMetric := JsonMetric{
					Desc: prometheus.NewDesc(
						module + "_" + name,
						metric.Help,
						variableLabels,
						nil,
					),
					KeyJsonPath:     metric.Path,
					ValueJsonPath:   valuePath,
					LabelsJsonPaths: variableLabelsValues,
				}
				metrics = append(metrics, jsonMetric)
			}
		default:
			return nil, fmt.Errorf("Unknown metric type: '%s', for metric: '%s'", metric.Type, metric.Name)
		}
	}
	return metrics, nil
}

func CreateStaticMetricsList(module string) ([]StaticMetric, error) {
	var metrics []StaticMetric
	staticMetric := StaticMetric{
		Name: "status",
		Desc: prometheus.NewDesc(
				module + "_request_status",
				"Request target status",
				[]string {},
				nil,
				),
	}
	metrics = append(metrics, staticMetric)

	staticMetric = StaticMetric{
		Name: "duration",
		Desc: prometheus.NewDesc(
			module + "_request_duration",
			"Request target duration by Millisecond",
			[]string {},
			nil,
		),
	}

	metrics = append(metrics, staticMetric)

	return metrics, nil
}

func FetchJson(ctx context.Context, logger log.Logger, module string, endpoint string, c config.Config, tplValues url.Values) ([]byte, float64, error) {
	var req *http.Request
	httpClientConfig := c.Modules[module].RequestConfig.ClientConfig
	client, err := pconfig.NewClientFromConfig(httpClientConfig, "fetch_json")
	if err != nil {
		level.Error(logger).Log("msg", "Error generating HTTP client", "err", err) //nolint:errcheck
		return nil, -1, err
	}

	if c.Modules[module].RequestConfig.Body.Content == "" {
		req, err = http.NewRequest("GET", endpoint, nil)
	} else {
		br := strings.NewReader(c.Modules[module].RequestConfig.Body.Content)
		if c.Modules[module].RequestConfig.Body.Templatize {
			tpl, err := template.New("base").Funcs(sprig.GenericFuncMap()).Parse(c.Modules[module].RequestConfig.Body.Content)
			if err != nil {

				level.Error(logger).Log("msg", "Failed to create a new template from body content", "err", err, "content", c.Modules[module].RequestConfig.Body.Content) //nolint:errcheck
			}
			var b strings.Builder
			if err := tpl.Execute(&b, tplValues); err != nil {
				level.Error(logger).Log("msg", "Failed to render template with values", "err", err, "tempalte", c.Modules[module].RequestConfig.Body.Content, "values", tplValues) //nolint:errcheck
			}
			br = strings.NewReader(b.String())
		}
		req, err = http.NewRequest("POST", endpoint, br)
	}
	req = req.WithContext(ctx)
	if err != nil {
		level.Error(logger).Log("msg", "Failed to create request", "err", err) //nolint:errcheck
		return nil, -1, err
	}

	if c.Modules[module].RequestConfig.Params != nil {
		p := req.URL.Query()
		for k, v := range c.Modules[module].RequestConfig.Params {
			p.Set(k, v)
		}
		req.URL.RawQuery = p.Encode()
		//level.Debug(logger).Log("msg", "Request ", req.URL)
	}

	for key, value := range c.Modules[module].RequestConfig.Headers {
		req.Header.Add(key, value)
	}
	if req.Header.Get("Accept") == "" {
		req.Header.Add("Accept", "application/json")
	}
	start := time.Now()
	resp, err := client.Do(req)
	end := time.Now()
	d := end.Sub(start) / time.Millisecond
	if err != nil {
		return nil, -1, err
	}

	defer func() {
		if _, err := io.Copy(ioutil.Discard, resp.Body); err != nil {
			level.Error(logger).Log("msg", "Failed to discard body", "err", err) //nolint:errcheck
		}
		resp.Body.Close()
	}()

	if resp.StatusCode/100 != 2 {
		return nil, -1, errors.New(resp.Status)
	}

	data, err := ioutil.ReadAll(resp.Body)
	level.Debug(logger).Log("msg", fmt.Sprintf("Request %s", endpoint), "data", data)
	if err != nil {
		return nil, -1, err
	}

	return data, float64(d), nil
}
