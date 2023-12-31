/*
Copyright 2023 Alexander Bartolomey (github@alexanderbartolomey.de)

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

	http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package ipfix

import "github.com/prometheus/client_golang/prometheus"

var (
	PacketsTotal = prometheus.NewCounter(prometheus.CounterOpts{
		Name: "decoder_decoded_packets_total",
		Help: "Total number of decoded packets in decoder",
	})
	ErrorsTotal = prometheus.NewCounter(prometheus.CounterOpts{
		Name: "decoder_errors_total",
		Help: "Total number of errors in decoder",
	})
	DurationMicroseconds = prometheus.NewHistogram(prometheus.HistogramOpts{
		Name:    "decoder_duration_microseconds",
		Help:    "Duration of decoding per protocol in microseconds",
		Buckets: []float64{0.005, 0.01, 0.025, 0.05, 0.1, 0.25, 0.5, 1, 2.5, 5, 10, 25, 50, 100, 250, 500, 1000, 2500},
	})
	DecodedSets = prometheus.NewCounterVec(prometheus.CounterOpts{
		Name: "decoder_decoded_sets_total",
		Help: "Total number of decoded sets per type",
	}, []string{"type"})
	DecodedRecords = prometheus.NewCounterVec(prometheus.CounterOpts{
		Namespace: "collector",
		Name:      "decoder_decoded_records_total",
		Help:      "Total number of decoded records per type",
	}, []string{"type"})
	DroppedRecords = prometheus.NewCounterVec(prometheus.CounterOpts{
		Namespace: "collector",
		Name:      "decoder_dropped_records_total",
		Help:      "Total number of records dropped due to filters per type",
	}, []string{"type"})
)

var (
	TCPActiveConnections = prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "tcp_listener_active_connections_total",
		Help: "Total number of active connections currently maintained by the TCP listener",
	})
	TCPErrorsTotal = prometheus.NewCounter(prometheus.CounterOpts{
		Name: "tcp_listener_errors_total",
		Help: "Total number of errors encountered in the TCP listener",
	})
	TCPReceivedBytes = prometheus.NewCounter(prometheus.CounterOpts{
		Name: "tcp_listener_received_bytes",
		Help: "Total number of bytes read in the TCP listener",
	})
)

var (
	UDPPacketsTotal = prometheus.NewCounter(prometheus.CounterOpts{
		Name: "udp_listener_packets_total",
		Help: "Total number of packets received via UDP listener",
	})
	UDPErrorsTotal = prometheus.NewCounter(prometheus.CounterOpts{
		Name: "udp_listener_errors_total",
		Help: "Total number of errors encountered in the UDP listener",
	})
	UDPPacketBytes = prometheus.NewCounter(prometheus.CounterOpts{
		Name: "udp_listener_packet_bytes",
		Help: "Total number of bytes read in the UDP listener",
	})
)
