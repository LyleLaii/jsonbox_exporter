package exporter

import (
	"github.com/go-kit/kit/log"
	"github.com/prometheus/client_golang/prometheus"
)

type StaticMetricCollector struct {
	StaticMetric []StaticMetric
	Data        map[string]float64
	Logger      log.Logger
}

type StaticMetric struct {
	Name            string
	Desc            *prometheus.Desc
}

func (mc StaticMetricCollector) Describe(ch chan<- *prometheus.Desc) {
	for _, m := range mc.StaticMetric {
		ch <- m.Desc
	}
}

func (mc StaticMetricCollector) Collect(ch chan<- prometheus.Metric) {
	for _, m := range mc.StaticMetric {
		ch <- prometheus.MustNewConstMetric(
			m.Desc,
			prometheus.UntypedValue,
			mc.Data[m.Name],
		)
	}
}
