# go-metrics-wavefront [![GoDoc](https://godoc.org/github.com/wavefrontHQ/go-metrics-wavefront?status.svg)](https://godoc.org/github.com/wavefrontHQ/go-metrics-wavefront) [![travis build status](https://travis-ci.com/wavefrontHQ/go-metrics-wavefront.svg?branch=master)](https://travis-ci.com/wavefrontHQ/go-metrics-wavefront)

This is a plugin for [go-metrics](https://github.com/rcrowley/go-metrics) which adds a Wavefront reporter and a simple abstraction that supports tagging at the host and metric level.

## Usage

### Wavefront Reporter

The Wavefront Reporter supports tagging at the host level. Any tags passed to the reporter here will be applied to every metric before being sent to Wavefront.

```go
import (
	metrics "github.com/rcrowley/go-metrics"
	"github.com/wavefronthq/go-metrics-wavefront/reporting"
	"github.com/wavefronthq/wavefront-sdk-go/application"
	"github.com/wavefronthq/wavefront-sdk-go/senders"
)

directCfg := &senders.DirectConfiguration{
  Server:               "https://[INSTANCE].reporting.com",
  Token:                [WF_TOKEN],
  BatchSize:            10000,
  MaxBufferSize:        50000,
  FlushIntervalSeconds: 1,
}

sender, err := senders.NewDirectSender(directCfg)
if err != nil {
  panic(err)
}

reporter := reporting.NewReporter(
  sender,
  application.New("app", "srv"),
  reporting.Source("go-metrics-test"),
  reporting.Prefix("some.prefix"),
  reporting.LogErrors(true),
)
```

### Tagging Metrics

In addition to tagging at the application level, you can add tags to individual metrics.

```go
tags := map[string]string{
  "key2": "val2",
  "key1": "val1",
}
counter := metrics.NewCounter()                //Create a counter
reporting.RegisterMetric("foo", counter, tags) // will create a 'some.prefix.foo.count' metric with tags
counter.Inc(47)
```

`reporting.RegisterMetric()` has the same affect as go-metrics' `metrics.Register()` except that it accepts tags in the form of a string map. The tags are then used by the Wavefront reporter at flush time. The tags become part of the key for a metric within go-metrics' Registry. Every unique combination of metric name+tags is a unique series. You can pass your tags in any order to the Register and Get functions documented below. The Wavefront plugin ensures the tags are always encoded in the same order within the Registry to ensure no duplication of metric series.

[Go Docs](https://github.com/wavefrontHQ/go-metrics-wavefront/blob/master/GODOCS.md)

### Extended Code Example

```go
package main

import (
	"fmt"
	"math/rand"
	"os"
	"time"

	metrics "github.com/rcrowley/go-metrics"
	"github.com/wavefronthq/go-metrics-wavefront/reporting"
	"github.com/wavefronthq/wavefront-sdk-go/application"
	"github.com/wavefronthq/wavefront-sdk-go/senders"
)

func main() {

	//Tags we'll add to the metric
	tags := map[string]string{
		"key2": "val2",
		"key1": "val1",
		"key0": "val0",
		"key4": "val4",
		"key3": "val3",
	}

	counter := metrics.NewCounter()                //Create a counter
	metrics.Register("foo2", counter)              // will create a 'some.prefix.foo2.count' metric with no tags
	reporting.RegisterMetric("foo", counter, tags) // will create a 'some.prefix.foo.count' metric with tags
	counter.Inc(47)

	histogram := reporting.NewHistogram()
	reporting.RegisterMetric("duration", histogram, tags) // will create a 'some.prefix.duration' histogram metric with tags

	histogram2 := reporting.NewHistogram()
	metrics.Register("duration2", histogram2) // will create a 'some.prefix.duration2' histogram metric with no tags

	deltaCounter := metrics.NewCounter()
	reporting.RegisterMetric(reporting.DeltaCounterName("delta.metric"), deltaCounter, tags)
	deltaCounter.Inc(10)

	directCfg := &senders.DirectConfiguration{
		Server:               "https://" + os.Getenv("WF_INSTANCE") + ".reporting.com",
		Token:                os.Getenv("WF_TOKEN"),
		BatchSize:            10000,
		MaxBufferSize:        50000,
		FlushIntervalSeconds: 1,
	}

	sender, err := senders.NewDirectSender(directCfg)
	if err != nil {
		panic(err)
	}

	reporter := reporting.NewReporter(
		sender,
		application.New("app", "srv"),
		reporting.Source("go-metrics-test"),
		reporting.Prefix("some.prefix"),
		reporting.LogErrors(true),
	)

	fmt.Println("Search wavefront: ts(\"some.prefix.foo.count\")")
	fmt.Println("Entering loop to simulate metrics flushing. Hit ctrl+c to cancel")

	for {
		counter.Inc(rand.Int63())
		histogram.Update(rand.Int63())
		histogram2.Update(rand.Int63())
		deltaCounter.Inc(10)
		time.Sleep(time.Second * 10)
	}
}
```
