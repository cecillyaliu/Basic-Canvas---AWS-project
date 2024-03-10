package metrics

import "github.com/smira/go-statsd"

var (
	client           *statsd.Client
	throughputPrefix = "throughput"
)

func init() {
	client = statsd.NewClient("localhost:8080",
		statsd.MaxPacketSize(1400),
		statsd.MetricPrefix("webapp."))
}

func ThroughPut(handler string) {
	client.Incr(throughputPrefix, 1, statsd.StringTag("handler", handler))
}

func Close() {
	err := client.Close()
	if err != nil {
		return
	}
}
