package main

import (
	"flag"
	"net/http"
	"os"

	"github.com/cafebazaar/keepalived-exporter/internal/collector"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/sirupsen/logrus"
)

func main() {
	listenAddr := flag.String("web.listen-address", ":2112", "Address to listen on for web interface and telemetry.")
	metricsPath := flag.String("web.telemetry-path", "/metrics", "A path under which to expose metrics.")
	keepalivedJSON := flag.Bool("ka.json", false, "Send SIGJSON and decode JSON file instead of parsing text files.")

	flag.Parse()

	coll := collector.NewKCollector(*keepalivedJSON)
	prometheus.MustRegister(coll)

	logrus.Info("Listening on address: ", *listenAddr)
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
			logrus.Warn("Error on returning home page err: ", err)
		}
	})
	if err := http.ListenAndServe(*listenAddr, nil); err != nil {
		logrus.Error("Error starting HTTP server", " err", err)
		os.Exit(1)
	}
}
