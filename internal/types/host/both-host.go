package host

import "github.com/cafebazaar/keepalived-exporter/internal/types"

// KeepalivedHostCollectorHost implements KeepalivedCollector for when Keepalived and Keepalived Exporter are both on a same host
type KeepalivedHostCollectorHost struct {
	pidPath string
}

// NewKeepalivedHostCollectorHost is creating new instance of KeepalivedHostCollectorHost
func NewKeepalivedHostCollectorHost(pidPath string) types.KeepalivedCollector {
	khch := &KeepalivedHostCollectorHost{
		pidPath: pidPath,
	}

	return khch
}
