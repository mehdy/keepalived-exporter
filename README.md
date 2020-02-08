# Keepalived Exporter
Prometheus exporter for [Keepalived](https://keepalived.org) metrics.

## Getting Started
To run it:
```
sudo ./keepalived-exporter [flags]
```
Help on flags
```
./keepalived-exporter --help
```

## Usage
Name               | Description
-------------------|------------
web.listen-address | Address to listen on for web interface and telemetry, defaults to `:2112`.
web.telemetry-path | A path under which to expose metrics, defaults to `/metrics`.
ka.json            | Send SIGJSON and decode JSON file instead of parsing text files, defaults to `false`.
ping               | Export VIP ping status, defaults to `false`.
ka.pid-path        | A path for Keepalived PID, defaults to `/var/run/keepalived.pid`.

**Note:** For `ka.json` option requirement is to have Keepalived compiled with `--enable-json` configure option.

## Metrics
| Metric                         | Notes
|--------------------------------|------------------------------------------------
| keepalived_up                  | Status of Keepalived service
| keepalived_vrrp_state          | State of vrrp
| keepalived_ping_packet_loss    | Ping packet loss
| keepalived_garp_delay          | Gratuitous ARP delay
| keepalived_advert_rcvd         | Advertisements received
| keepalived_advert_sent         | Advertisements sent
| keepalived_become_master       | Became master
| keepalived_release_master      | Released master
| keepalived_packet_len_err      | Packet length errors
| keepalived_advert_interval_err | Advertisement interval errors
| keepalived_ip_ttl_err          | TTL errors
| keepalived_invalid_type_rcvd   | Invalid type errors
| keepalived_addr_list_err       | Address list errors
| keepalived_invalid_authtype    | Authentication invalid
| keepalived_authtype_mismatch   | Authentication mismatch
| keepalived_auth_failure        | Authentication failure
| keepalived_pri_zero_rcvd       | Priority zero received
| keepalived_pri_zero_sent       | Priority zero sent
| keepalived_script_status       | Tracker Script Status
| keepalived_script_state        | Tracker Script State
