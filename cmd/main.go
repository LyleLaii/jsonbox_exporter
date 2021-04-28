package main

import (
	"context"
	"encoding/json"
	"github.com/prometheus/exporter-toolkit/web"
	"net/http"
	"os"

	"jsonbox_exporter/config"
	"jsonbox_exporter/exporter"

	"github.com/go-kit/kit/log"
	"github.com/go-kit/kit/log/level"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/prometheus/common/promlog"
	"github.com/prometheus/common/promlog/flag"
	"github.com/prometheus/common/version"
	"gopkg.in/alecthomas/kingpin.v2"
)

var (
	configFile    = kingpin.Flag("config.file", "JSON exporter configuration file.").Default("config.yml").ExistingFile()
	listenAddress = kingpin.Flag("web.listen-address", "The address to listen on for HTTP requests.").Default(":7979").String()
	configCheck   = kingpin.Flag("config.check", "If true validate the config file and then exit.").Default("false").Bool()
	tlsConfigFile = kingpin.Flag("web.config", "[EXPERIMENTAL] Path to config yaml file that can enable TLS or authentication.").Default("").String()
)

func Run() {

	promlogConfig := &promlog.Config{}

	flag.AddFlags(kingpin.CommandLine, promlogConfig)
	kingpin.Version(version.Print("jsonbox_exporter"))
	kingpin.HelpFlag.Short('h')
	kingpin.Parse()
	logger := promlog.New(promlogConfig)

	level.Info(logger).Log("msg", "Starting jsonbox_exporter", "version", version.Info()) //nolint:errcheck
	level.Info(logger).Log("msg", "Build context", "build", version.BuildContext())    //nolint:errcheck

	level.Info(logger).Log("msg", "Loading config file", "file", *configFile) //nolint:errcheck
	config, err := config.LoadConfig(*configFile)
	if err != nil {
		level.Error(logger).Log("msg", "Error loading config", "err", err) //nolint:errcheck
		os.Exit(1)
	}
	configJson, err := json.Marshal(config)
	if err != nil {
		level.Error(logger).Log("msg", "Failed to marshal config to JSON", "err", err) //nolint:errcheck
	}
	level.Info(logger).Log("msg", "Loaded config file", "config", string(configJson)) //nolint:errcheck

	if *configCheck {
		os.Exit(0)
	}

	http.Handle("/metrics", promhttp.Handler())
	http.HandleFunc("/probe", func(w http.ResponseWriter, req *http.Request) {
		probeHandler(w, req, logger, config)
	})

	server := &http.Server{Addr: *listenAddress}
	if err := web.ListenAndServe(server, *tlsConfigFile, logger); err != nil {
		level.Error(logger).Log("msg", "Failed to start the server", "err", err) //nolint:errcheck
		os.Exit(1)
	}
}

func probeHandler(w http.ResponseWriter, r *http.Request, logger log.Logger, config config.Config) {

	ctx, cancel := context.WithCancel(r.Context())
	defer cancel()
	r = r.WithContext(ctx)

	registry := prometheus.NewPedanticRegistry()

	moduleName := r.URL.Query().Get("module")
	if moduleName == "" {
		http.Error(w, "ModuleName parameter is missing", http.StatusBadRequest)
		return
	}

	target := r.URL.Query().Get("target")
	if target == "" {
		http.Error(w, "Target parameter is missing", http.StatusBadRequest)
		return
	}

	staticMetric, _ := exporter.CreateStaticMetricsList(moduleName)
	staticMetricCollector := exporter.StaticMetricCollector{StaticMetric: staticMetric, Data: make(map[string]float64)}
	staticMetricCollector.Logger = logger

	metrics, err := exporter.CreateMetricsList(config,  moduleName)
	if err != nil {
		level.Error(logger).Log("msg", "Failed to create metrics list from config", "err", err) //nolint:errcheck
	}

	jsonMetricCollector := exporter.JsonMetricCollector{JsonMetrics: metrics}
	jsonMetricCollector.Logger = logger

	data, duration, err := exporter.FetchJson(ctx, logger, moduleName, target, config, r.URL.Query())
	if err != nil {
		//level.Warn(logger).Log("Failed to fetch JSON response. TARGET: "+target+", ERROR: "+err.Error())
		http.Error(w, "Failed to fetch JSON response. TARGET: "+target+", ERROR: "+err.Error(), http.StatusServiceUnavailable)
		return
	}

	staticMetricCollector.Data["duration"] = duration

	jsonMetricCollector.Data = data

	registry.MustRegister(staticMetricCollector)
	registry.MustRegister(jsonMetricCollector)
	h := promhttp.HandlerFor(registry, promhttp.HandlerOpts{})
	h.ServeHTTP(w, r)

}

func main() {
	Run()
}
