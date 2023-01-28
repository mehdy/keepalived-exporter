package main

import (
	"flag"
	"net/http"
	"os"
	"time"

	"github.com/mehdy/keepalived-exporter/internal/collector"
	"github.com/mehdy/keepalived-exporter/internal/types/container"
	"github.com/mehdy/keepalived-exporter/internal/types/host"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/sirupsen/logrus"
)

var (
	commit    string
	version   string
	buildTime string
)

func main() {
	listenAddr := flag.String("web.listen-address", ":9165", "Address to listen on for web interface and telemetry.")
	metricsPath := flag.String("web.telemetry-path", "/metrics", "A path under which to expose metrics.")
	keepalivedJSON := flag.Bool("ka.json", false, "Send SIGJSON and decode JSON file instead of parsing text files.")
	keepalivedPID := flag.String("ka.pid-path", "/var/run/keepalived.pid", "A path for Keepalived PID")
	keepalivedCheckScript := flag.String("cs", "", "Health Check script path to be execute for each VIP")
	keepalivedContainerName := flag.String("container-name", "", "Keepalived container name")
	keepalivedContainerTmpDir := flag.String("container-tmp-dir", "/tmp", "Keepalived container tmp volume path")
	versionFlag := flag.Bool("version", false, "Show the current keepalived exporter version")

	flag.Parse()

	if *versionFlag {
		logrus.WithFields(logrus.Fields{
			"commit": commit, "version": version, "build_time": buildTime,
		}).Info("Keepalived Exporter")

		return
	}

	var c collector.Collector
	if *keepalivedContainerName != "" {
		c = container.NewKeepalivedContainerCollectorHost(*keepalivedJSON, *keepalivedContainerName, *keepalivedContainerTmpDir)
	} else {
		c = host.NewKeepalivedHostCollectorHost(*keepalivedJSON, *keepalivedPID)
	}

	keepalivedCollector := collector.NewKeepalivedCollector(*keepalivedJSON, *keepalivedCheckScript, c)
	prometheus.MustRegister(keepalivedCollector)

	http.Handle(*metricsPath, promhttp.Handler())
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		_, err := w.Write([]byte(`<html>
		<head><title>Keepalived Exporter</title></head>
		<body>
		<h1>Keepalived Exporter</h1>
		<p><a href='` + *metricsPath + `'>Metrics</a></p>
		</body>
		</html>`))
		if err != nil {
			logrus.WithError(err).Warn("Error on returning home page")
		}
	})

	logrus.Info("Listening on address: ", *listenAddr)

	server := &http.Server{
		Addr:              *listenAddr,
		ReadHeaderTimeout: 5 * time.Second,
	}
	if err := server.ListenAndServe(); err != nil {
		logrus.WithError(err).Error("Error starting HTTP server")
		os.Exit(1)
	}
}
