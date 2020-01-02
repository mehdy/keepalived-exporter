package collector

import (
	"errors"
	"strconv"
	"strings"
	"sync"
	"syscall"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/sirupsen/logrus"
	"github.com/sparrc/go-ping"
)

//KCollector is Keepalived collector
type KCollector struct {
	mutex   sync.Mutex
	useJSON bool
	metrics map[string]*prometheus.Desc
}

//VRRPStats is Keepalived stats structure
type VRRPStats struct {
	AdvertRcvd        int `json:"advert_rcvd"`
	AdvertSent        int `json:"advert_sent"`
	BecomeMaster      int `json:"become_master"`
	ReleaseMaster     int `json:"release_master"`
	PacketLenErr      int `json:"packet_len_err"`
	AdvertIntervalErr int `json:"advert_interval_err"`
	IPTTLErr          int `json:"ip_ttl_err"`
	InvalidTypeRcvd   int `json:"invalid_type_rcvd"`
	AddrListErr       int `json:"addr_list_err"`
	InvalidAuthType   int `json:"invalid_authtype"`
	AuthTypeMismatch  int `json:"authtype_mismatch"`
	AuthFailure       int `json:"auth_failure"`
	PRIZeroRcvd       int `json:"pri_zero_rcvd"`
	PRIZeroSent       int `json:"pri_zero_sent"`
}

//VRRPData is keepalived data structure
type VRRPData struct {
	IName     string   `json:"iname"`
	State     int      `json:"state"`
	WantState int      `json:"wantstate"`
	Intf      string   `json:"ifp_ifname"`
	GArpDelay int      `json:"garp_delay"`
	VRID      int      `json:"vrid"`
	VIPs      []string `json:"vips"`
}

//VRRPScript is keepalived VRRP Script data section structure
type VRRPScript struct {
	Name   string
	Status string
	State  string
}

//Stats is statistics for keepalived
type Stats struct {
	Data  VRRPData  `json:"data"`
	Stats VRRPStats `json:"stats"`
}

//KStats is Keepalived Exporter structure to be export
type KStats struct {
	Stats   []Stats
	Scripts []VRRPScript
}

//NewKCollector is creating new instance of KCollector
func NewKCollector(useJSON bool) *KCollector {
	k := &KCollector{
		useJSON: useJSON,
	}

	lables := []string{"iname", "intf", "vrid", "state"}
	k.metrics = map[string]*prometheus.Desc{
		"keepalived_up":                  prometheus.NewDesc("keepalived_up", "Status", nil, nil),
		"keepalived_vrrp_state":          prometheus.NewDesc("keepalived_vrrp_state", "State of vrrp", []string{"iname", "intf", "vrid", "ip_address"}, nil),
		"keepalived_ping_packet_loss":    prometheus.NewDesc("keepalived_ping_packet_loss", "Ping packet loss status to each vrrp", []string{"iname", "intf", "vrid", "ip_address"}, nil),
		"keepalived_garp_delay":          prometheus.NewDesc("keepalived_garp_deplay", "Gratuitous ARP delay", lables, nil),
		"keepalived_advert_rcvd":         prometheus.NewDesc("keepalived_advert_rcvd", "Advertisements received", lables, nil),
		"keepalived_advert_sent":         prometheus.NewDesc("keepalived_advert_sent", "Advertisements sent", lables, nil),
		"keepalived_become_master":       prometheus.NewDesc("keepalived_become_master", "Became master", lables, nil),
		"keepalived_release_master":      prometheus.NewDesc("keepalived_release_master", "Released master", lables, nil),
		"keepalived_packet_len_err":      prometheus.NewDesc("keepalived_packet_len_err", "Packet length errors", lables, nil),
		"keepalived_advert_interval_err": prometheus.NewDesc("keepalived_advert_interval_err", "Advertisement interval errors", lables, nil),
		"keepalived_ip_ttl_err":          prometheus.NewDesc("keepalived_ip_ttl_err", "TTL errors", lables, nil),
		"keepalived_invalid_type_rcvd":   prometheus.NewDesc("keepalived_invalid_type_rcvd", "Invalid type errors", lables, nil),
		"keepalived_addr_list_err":       prometheus.NewDesc("keepalived_addr_list_err", "Address list errors", lables, nil),
		"keepalived_invalid_authtype":    prometheus.NewDesc("keepalived_invalid_authtype", "Authentication invalid", lables, nil),
		"keepalived_authtype_mismatch":   prometheus.NewDesc("keepalived_authtype_mismatch", "Authentication mismatch", lables, nil),
		"keepalived_auth_failure":        prometheus.NewDesc("keepalived_auth_failure", "Authentication failure", lables, nil),
		"keepalived_pri_zero_rcvd":       prometheus.NewDesc("keepalived_pri_zero_rcvd", "Priority zero received", lables, nil),
		"keepalived_pri_zero_sent":       prometheus.NewDesc("keepalived_pri_zero_sent", "Priority zero sent", lables, nil),
		"keepalived_script_status":       prometheus.NewDesc("keepalived_script_status", "Tracker Script Status", []string{"name"}, nil),
		"keepalived_script_state":        prometheus.NewDesc("keepalived_script_state", "Tracker Script State", []string{"name"}, nil),
	}

	return k
}

func (k *KCollector) collectMetric(ch chan<- prometheus.Metric, name string, value float64, lableValues ...string) {
	pm, err := prometheus.NewConstMetric(
		k.metrics[name],
		prometheus.GaugeValue,
		value,
		lableValues...,
	)
	if err != nil {
		logrus.Error("Failed on Register metric: ", name, " err: ", err)
		return
	}

	ch <- pm
}

//Collect get metrics and add to prometheus metric channel
func (k *KCollector) Collect(ch chan<- prometheus.Metric) {
	k.mutex.Lock()
	defer k.mutex.Unlock()

	var kStats *KStats
	var err error

	if k.useJSON {
		kStats, err = k.json()
		if err != nil {
			logrus.Error("Keepalived Exporter didn't export anything for json use", " err: ", err)
			metric, err := prometheus.NewConstMetric(k.metrics["keepalived_up"], prometheus.GaugeValue, 0)
			if err != nil {
				ch <- metric
			}
			return
		}
	} else {
		kStats, err = k.text()
		if err != nil {
			logrus.Error("Keepalived Exporter didn't export anything for text use", " err: ", err)
			metric, err := prometheus.NewConstMetric(k.metrics["keepalived_up"], prometheus.GaugeValue, 0)
			if err != nil {
				ch <- metric
			}
			return
		}
	}

	metric, err := prometheus.NewConstMetric(k.metrics["keepalived_up"], prometheus.GaugeValue, 1)
	if err != nil {
		ch <- metric
	}

	for _, st := range kStats.Stats {
		state := ""
		ok := false
		if state, ok = st.Data.state2string(st.Data.State); !ok {
			logrus.Warn("Unknown State found for vrrp: ", st.Data.IName)
		}

		//Keppalived Stats
		k.collectMetric(ch, "keepalived_advert_rcvd", float64(st.Stats.AdvertRcvd), st.Data.IName, st.Data.Intf, strconv.Itoa(st.Data.VRID), state)
		k.collectMetric(ch, "keepalived_advert_sent", float64(st.Stats.AdvertSent), st.Data.IName, st.Data.Intf, strconv.Itoa(st.Data.VRID), state)
		k.collectMetric(ch, "keepalived_become_master", float64(st.Stats.BecomeMaster), st.Data.IName, st.Data.Intf, strconv.Itoa(st.Data.VRID), state)
		k.collectMetric(ch, "keepalived_release_master", float64(st.Stats.ReleaseMaster), st.Data.IName, st.Data.Intf, strconv.Itoa(st.Data.VRID), state)
		k.collectMetric(ch, "keepalived_packet_len_err", float64(st.Stats.PacketLenErr), st.Data.IName, st.Data.Intf, strconv.Itoa(st.Data.VRID), state)
		k.collectMetric(ch, "keepalived_advert_interval_err", float64(st.Stats.AdvertIntervalErr), st.Data.IName, st.Data.Intf, strconv.Itoa(st.Data.VRID), state)
		k.collectMetric(ch, "keepalived_ip_ttl_err", float64(st.Stats.IPTTLErr), st.Data.IName, st.Data.Intf, strconv.Itoa(st.Data.VRID), state)
		k.collectMetric(ch, "keepalived_invalid_type_rcvd", float64(st.Stats.InvalidTypeRcvd), st.Data.IName, st.Data.Intf, strconv.Itoa(st.Data.VRID), state)
		k.collectMetric(ch, "keepalived_addr_list_err", float64(st.Stats.AddrListErr), st.Data.IName, st.Data.Intf, strconv.Itoa(st.Data.VRID), state)
		k.collectMetric(ch, "keepalived_invalid_authtype", float64(st.Stats.InvalidAuthType), st.Data.IName, st.Data.Intf, strconv.Itoa(st.Data.VRID), state)
		k.collectMetric(ch, "keepalived_authtype_mismatch", float64(st.Stats.AuthFailure), st.Data.IName, st.Data.Intf, strconv.Itoa(st.Data.VRID), state)
		k.collectMetric(ch, "keepalived_auth_failure", float64(st.Stats.AuthFailure), st.Data.IName, st.Data.Intf, strconv.Itoa(st.Data.VRID), state)
		k.collectMetric(ch, "keepalived_pri_zero_rcvd", float64(st.Stats.PRIZeroRcvd), st.Data.IName, st.Data.Intf, strconv.Itoa(st.Data.VRID), state)
		k.collectMetric(ch, "keepalived_pri_zero_sent", float64(st.Stats.PRIZeroSent), st.Data.IName, st.Data.Intf, strconv.Itoa(st.Data.VRID), state)

		//Keepalived Data
		k.collectMetric(ch, "keepalived_garp_delay", float64(st.Data.GArpDelay), st.Data.IName, st.Data.Intf, strconv.Itoa(st.Data.VRID), state)
		k.collectVRRPState(ch, st.Data)
		k.collectPing(ch, st.Data)
	}

	for _, script := range kStats.Scripts {
		if scriptStatus, ok := script.string2status(script.Status); ok {
			k.collectMetric(ch, "keepalived_script_status", float64(scriptStatus), script.Name)
		} else {
			logrus.Warn("Unknown status for script: ", script.Name, " status: ", script.Status)
		}

		if scriptState, ok := script.string2state(script.State); ok {
			k.collectMetric(ch, "keepalived_script_state", float64(scriptState), script.Name)
		} else {
			logrus.Warn("Unknown state for script: ", script.Name, " state: ", script.State)
		}
	}
}

func (k *KCollector) collectVRRPState(ch chan<- prometheus.Metric, data VRRPData) {
	for _, ip := range data.VIPs {
		ipAddr := strings.Split(ip, " ")[0]
		intf := strings.Split(ip, " ")[2]

		metric, err := prometheus.NewConstMetric(
			k.metrics["keepalived_vrrp_state"],
			prometheus.GaugeValue,
			float64(data.State),
			data.IName, intf, strconv.Itoa(data.VRID), ipAddr,
		)
		if err != nil {
			logrus.Error("Failed to register metric on job collectVRRPState for vip: ", ipAddr, " intf: ", intf, " err: ", err)
			continue
		}

		ch <- metric
	}
}

func (k *KCollector) collectPing(ch chan<- prometheus.Metric, data VRRPData) {
	for _, ip := range data.VIPs {
		ipAddr := strings.Split(ip, " ")[0]
		intf := strings.Split(ip, " ")[2]

		pinger, err := ping.NewPinger(ipAddr)
		if err != nil {
			logrus.Error("Faild on creating new instance for pinger", " err: ", err)
			continue
		}
		pinger.SetPrivileged(true)
		pinger.Count = 1
		pinger.Run()

		metric, err := prometheus.NewConstMetric(
			k.metrics["keepalived_ping_packet_loss"],
			prometheus.GaugeValue,
			pinger.Statistics().PacketLoss,
			data.IName, intf, strconv.Itoa(data.VRID), ipAddr,
		)
		if err != nil {
			logrus.Error("Failed to register metric on job collectPing for vip: ", ipAddr, " intf: ", intf, " err: ", err)
			continue
		}

		ch <- metric
	}
}

//Describe outputs metrics descriptions
func (k *KCollector) Describe(ch chan<- *prometheus.Desc) {
	for _, m := range k.metrics {
		ch <- m
	}
}

func (k *KCollector) json() (*KStats, error) {
	err := k.signal(syscall.Signal(SIGJSON))
	if err != nil {
		return nil, err
	}

	s, err := k.parseJSON()
	if err != nil {
		return nil, err
	}

	return &KStats{
		Stats:   s,
		Scripts: make([]VRRPScript, 0),
	}, nil
}

func (k *KCollector) text() (*KStats, error) {
	stats := make([]Stats, 0)

	err := k.signal(syscall.Signal(SIGDATA))
	if err != nil {
		logrus.Error("Failed to send DATA signal to keepalived", " err: ", err)
		return nil, err
	}

	err = k.signal(syscall.Signal(SIGSTATS))
	if err != nil {
		logrus.Error("Failed to send STATS signal to keepalived", " err: ", err)
		return nil, err
	}

	vrrpData, err := k.parseVRRPData()
	if err != nil {
		return nil, err
	}

	vrrpStats, err := k.parseStats()
	if err != nil {
		return nil, err
	}

	if len(vrrpData) != len(vrrpStats) {
		logrus.Error("keepalived.data and keepalived.stats datas are not synced")
		return nil, errors.New("keepalived.data and keepalived.stats datas are not synced")
	}

	for i := 0; i < len(vrrpData); i++ {
		s := Stats{
			Data:  vrrpData[i],
			Stats: vrrpStats[i],
		}
		stats = append(stats, s)
	}

	scripts, err := k.parseVRRPScript()
	if err != nil {
		return nil, err
	}

	return &KStats{
		Stats:   stats,
		Scripts: scripts,
	}, nil
}
