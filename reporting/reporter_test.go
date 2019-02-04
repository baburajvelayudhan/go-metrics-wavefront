package reporting

import (
	"fmt"
	"math/rand"
	"sync"
	"testing"
	"time"

	metrics "github.com/rcrowley/go-metrics"
	"github.com/wavefronthq/wavefront-sdk-go/application"
	"github.com/wavefronthq/wavefront-sdk-go/histogram"
	"github.com/wavefronthq/wavefront-sdk-go/senders"

	"github.com/stretchr/testify/assert"
)

func TestPrefixAndSuffix(t *testing.T) {
	reporter := &reporter{}

	reporter.prefix = "prefix"
	reporter.addSuffix = true
	name := reporter.prepareName("name", "count")
	assert.Equal(t, name, "prefix.name.count")

	name = reporter.prepareName("name")
	assert.Equal(t, name, "prefix.name")

	reporter.prefix = ""
	reporter.addSuffix = false
	name = reporter.prepareName("name", "count")
	assert.Equal(t, name, "name")
}

func TestError(t *testing.T) {
	metrics.DefaultRegistry.UnregisterAll()

	sender := &MockSender{}
	reporter := NewReporter(sender, application.New("app", "srv"), DisableAutoStart(), LogErrors(true))
	tags := map[string]string{"tag1": "tag"}

	RegisterMetric("", metrics.NewCounter(), tags)

	c := metrics.NewCounter()
	RegisterMetric("m1", c, tags)
	c.Inc(1)

	reporter.Report()
	reporter.Close()

	_, met, _ := sender.Counters()

	assert.Equal(t, 1, met)
	assert.Equal(t, int64(1), reporter.ErrorsCount())

}

func TestBasicCounter(t *testing.T) {
	metrics.DefaultRegistry.UnregisterAll()

	sender := &MockSender{}
	reporter := NewReporter(sender, application.New("app", "srv"), Interval(time.Second), LogErrors(true))
	tags := map[string]string{"tag1": "tag"}

	name := "counter"
	c := GetMetric(name, tags)
	if c == nil {
		c = metrics.NewCounter()
		RegisterMetric(name, c, tags)
	}
	c.(metrics.Counter).Inc(1)

	time.Sleep(time.Second * 3) // wait  3 reporting interval

	reporter.Close()

	_, met, _ := sender.Counters()
	assert.True(t, met >= 2)

}

func TestWFHistogram(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping Histogram tests in short mode")
	}

	metrics.DefaultRegistry.UnregisterAll()

	sender := newMockSender()
	reporter := NewReporter(sender, application.New("app", "srv"), DisableAutoStart(), LogErrors(true))
	tags := map[string]string{"tag1": "tag"}

	h := NewHistogram(histogram.GranularityOption(histogram.MINUTE))
	// h := NewHistogram()
	RegisterMetric("wf.histogram", h, tags)

	for i := 0; i < 1000; i++ {
		h.Update(rand.Int63())
	}

	time.Sleep(time.Minute * 2) // wait until the histogram rotates

	reporter.Report()

	dis, met, _ := sender.Counters()
	assert.Equal(t, 1, dis)
	assert.Equal(t, 0, met)

	reporter.Close()
}

func TestHistogram(t *testing.T) {
	metrics.DefaultRegistry.UnregisterAll()

	sender := newMockSender()
	reporter := NewReporter(sender, application.New("app", "srv"), DisableAutoStart(), LogErrors(true))
	tags := map[string]string{"tag1": "tag"}

	s := metrics.NewExpDecaySample(1028, 0.015) // or metrics.NewUniformSample(1028)
	h := metrics.NewHistogram(s)
	RegisterMetric("mt.histogram", h, tags)

	for i := 0; i < 1000; i++ {
		h.Update(rand.Int63())
	}

	reporter.Report()

	dis, met, _ := sender.Counters()

	assert.Equal(t, 0, dis)
	assert.Equal(t, 10, met)

	reporter.Close()
}

func TestDeltaPoint(t *testing.T) {
	metrics.DefaultRegistry.UnregisterAll()

	sender := newMockSender()
	reporter := NewReporter(sender, application.New("app", "srv"), DisableAutoStart(), LogErrors(true))
	tags := map[string]string{"tag1": "tag"}

	counter := metrics.NewCounter()
	RegisterMetric(DeltaCounterName("foo"), counter, tags)

	counter.Inc(10)
	reporter.Report()
	_, met, del := sender.Counters()
	assert.Equal(t, 1, del)
	assert.Equal(t, 0, met)

	counter.Inc(10)
	reporter.Report()

	_, met, del = sender.Counters()
	assert.Equal(t, 2, del)
	assert.Equal(t, 0, met)

	reporter.Close()
}

func newMockSender() *MockSender {
	return &MockSender{
		distributions: make([]string, 0),
		metrics:       make([]string, 0),
		deltas:        make([]string, 0),
	}
}

type MockSender struct {
	distributions []string
	metrics       []string
	deltas        []string
	sync.Mutex
}

func (s *MockSender) Close() {}

func (s *MockSender) SendEvent(name string, startMillis, endMillis int64, source string, tags map[string]string) error {
	return nil
}

func (s *MockSender) SendSpan(name string, startMillis, durationMillis int64, source, traceID, spanID string, parents, followsFrom []string, tags []senders.SpanTag, spanLogs []senders.SpanLog) error {
	return nil
}

func (s *MockSender) SendDistribution(name string, centroids []histogram.Centroid, hgs map[histogram.Granularity]bool, ts int64, source string, tags map[string]string) error {
	s.Lock()
	defer s.Unlock()
	s.distributions = append(s.distributions, name)
	return nil
}

func (s *MockSender) SendDeltaCounter(name string, value float64, source string, tags map[string]string) error {
	s.Lock()
	defer s.Unlock()
	s.deltas = append(s.deltas, name)
	return nil
}

func (s *MockSender) SendMetric(name string, value float64, ts int64, source string, tags map[string]string) error {
	if name == ".count" {
		return fmt.Errorf("empty metric name")
	}
	s.Lock()
	defer s.Unlock()
	s.metrics = append(s.metrics, name)
	return nil
}

func (s *MockSender) Flush() error {
	return nil
}

func (s *MockSender) GetFailureCount() int64 {
	return 0
}

func (s *MockSender) Start() {}

func (s *MockSender) Counters() (int, int, int) {
	s.Lock()
	defer s.Unlock()
	return len(s.distributions), len(s.metrics), len(s.deltas)
}
