package main

import "testing"

var orders = sampleOrders(2000)

func TestReportsMatch(t *testing.T) {
	naive := ReportNaive(orders)
	tuned := string(ReportTuned(nil, orders))
	if naive != tuned {
		t.Fatalf("reports differ:\nnaive %d bytes\ntuned %d bytes", len(naive), len(tuned))
	}
}

var (
	sinkString string
	sinkBytes  []byte
)

func BenchmarkReportNaive(b *testing.B) {
	for b.Loop() {
		sinkString = ReportNaive(orders)
	}
}

func BenchmarkReportTuned(b *testing.B) {
	buf := make([]byte, 0, 64<<10)
	for b.Loop() {
		buf = ReportTuned(buf, orders)
	}
	sinkBytes = buf
}
