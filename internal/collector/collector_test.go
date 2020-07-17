package collector

import (
	"testing"

	"github.com/prometheus/client_golang/prometheus"
)

func TestNewConstMetric(t *testing.T) {
	k := KeepalivedCollector{}
	k.fillMetrics()

	for metric := range k.metrics {
		pm := make(chan prometheus.Metric, 1)
		var valueType prometheus.ValueType
		labelValues := []string{"iname", "intf", "vrid", "state"}

		switch metric {
		case "keepalived_advertisements_received_total", "keepalived_advertisements_sent_total", "keepalived_become_master_total",
			"keepalived_release_master_total", "keepalived_packet_length_errors_total", "keepalived_advertisements_interval_errors_total",
			"keepalived_ip_ttl_errors_total", "keepalived_invalid_type_received_total", "keepalived_address_list_errors_total",
			"keepalived_authentication_invalid_total", "keepalived_authentication_mismatch_total", "keepalived_authentication_failure_total",
			"keepalived_priority_zero_received_total", "keepalived_priority_zero_sent_total", "keepalived_gratuitous_arp_delay_total":
			valueType = prometheus.CounterValue
		case "keepalived_up":
			valueType = prometheus.GaugeValue
			labelValues = nil
		case "keepalived_vrrp_state", "keepalived_exporter_check_script_status":
			valueType = prometheus.GaugeValue
			labelValues = []string{"iname", "intf", "vrid", "ip_address"}
		case "keepalived_script_status", "keepalived_script_state":
			valueType = prometheus.GaugeValue
			labelValues = []string{"name"}
		default:
			t.Fail()
		}

		k.newConstMetric(pm, metric, valueType, 10, labelValues...)

		select {
		case _, ok := <-pm:
			if !ok {
				t.Fail()
			}
		default:
			t.Fail()
		}
	}
}
