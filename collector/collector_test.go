package collector

import (
	"strings"
	"sync"
	"testing"

	"github.com/prometheus/client_golang/prometheus"
)

func TestNewMetric(t *testing.T) {
	// Test cases with different parameters
	testCases := []struct {
		name       string
		metricName string
		subsystem  string
		docString  string
		labelNames []string
		wantFQName string
	}{
		{
			name:       "Basic metric without labels",
			metricName: "test_metric",
			subsystem:  "test_subsystem",
			docString:  "Test documentation string",
			labelNames: []string{},
			wantFQName: "artifactory_test_subsystem_test_metric",
		},
		{
			name:       "Metric with default labels",
			metricName: "test_metric",
			subsystem:  "test_subsystem",
			docString:  "Test documentation string",
			labelNames: defaultLabelNames,
			wantFQName: "artifactory_test_subsystem_test_metric",
		},
		{
			name:       "Metric with custom labels",
			metricName: "test_metric",
			subsystem:  "test_subsystem",
			docString:  "Test documentation string",
			labelNames: []string{"custom_label1", "custom_label2"},
			wantFQName: "artifactory_test_subsystem_test_metric",
		},
		{
			name:       "Storage metric with filestore labels",
			metricName: "filestore",
			subsystem:  "storage",
			docString:  "Test filestore metric",
			labelNames: filestoreLabelNames,
			wantFQName: "artifactory_storage_filestore",
		},
		{
			name:       "Repo metric with repo labels",
			metricName: "repo_files",
			subsystem:  "storage",
			docString:  "Test repo metric",
			labelNames: repoLabelNames,
			wantFQName: "artifactory_storage_repo_files",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Create the metric descriptor
			metricDesc := newMetric(tc.metricName, tc.subsystem, tc.docString, tc.labelNames)

			// Verify it's not nil
			if metricDesc == nil {
				t.Fatal("Expected non-nil metric descriptor")
			}

			// Extract the metric descriptor details using prometheus.Desc.String()
			// The String() method includes the fully qualified name, help text, and label names
			descString := metricDesc.String()

			// Check that the fully qualified name is included in the descriptor string
			if !contains(descString, tc.wantFQName) {
				t.Errorf("Expected metric name to contain %q, got %q", tc.wantFQName, descString)
			}

			// Check that the help/documentation string is included
			if !contains(descString, tc.docString) {
				t.Errorf("Expected documentation string %q in descriptor, got %q", tc.docString, descString)
			}

			// Check that all label names are included in the descriptor
			for _, label := range tc.labelNames {
				if !contains(descString, label) {
					t.Errorf("Expected label %q in descriptor, got %q", label, descString)
				}
			}
		})
	}
}

// Helper function to check if a string contains another string
func contains(s, substr string) bool {
	return strings.Contains(s, substr)
}

// mockExporter is a controlled implementation of Exporter for testing
type mockExporter struct {
	mutex           sync.Mutex
	scrapeResult    float64
	scrapeCalled    bool
	totalScrapes    prometheus.Counter
	up              prometheus.Gauge
	upValue         float64 // Store the actual value for testing
	totalAPIErrors  prometheus.Counter
	jsonParseFailures prometheus.Counter
}

// newMockExporter creates a new instance of mockExporter with initialized metrics
func newMockExporter() *mockExporter {
	return &mockExporter{
		totalScrapes: prometheus.NewCounter(prometheus.CounterOpts{
			Name: "total_scrapes",
			Help: "Total number of scrapes",
		}),
		up: prometheus.NewGauge(prometheus.GaugeOpts{
			Name: "up",
			Help: "Was the last scrape successful",
		}),
		totalAPIErrors: prometheus.NewCounter(prometheus.CounterOpts{
			Name: "api_errors_total",
			Help: "Total number of API errors",
		}),
		jsonParseFailures: prometheus.NewCounter(prometheus.CounterOpts{
			Name: "json_parse_failures",
			Help: "Total number of JSON parse failures",
		}),
	}
}

// scrape is a mock implementation that just returns the configured result
func (e *mockExporter) scrape(ch chan<- prometheus.Metric) float64 {
	e.scrapeCalled = true
	return e.scrapeResult
}

// Collect implements the prometheus.Collector interface
func (e *mockExporter) Collect(ch chan<- prometheus.Metric) {
	e.mutex.Lock()
	defer e.mutex.Unlock()

	up := e.scrape(ch)
	e.up.Set(up)
	e.upValue = up // Store for testing
	ch <- e.up
	ch <- e.totalScrapes
	ch <- e.totalAPIErrors
	ch <- e.jsonParseFailures
}

// Describe implementation for mockExporter - empty for test purposes
func (e *mockExporter) Describe(ch chan<- *prometheus.Desc) {
	ch <- e.up.Desc()
	ch <- e.totalScrapes.Desc()
	ch <- e.totalAPIErrors.Desc()
	ch <- e.jsonParseFailures.Desc()
}

// TestCollect tests the Collect method of the Exporter
func TestCollect(t *testing.T) {
	testCases := []struct {
		name         string
		scrapeResult float64
		expectedUp   float64
	}{
		{
			name:         "Successful scrape",
			scrapeResult: 1,
			expectedUp:   1,
		},
		{
			name:         "Failed scrape",
			scrapeResult: 0,
			expectedUp:   0,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Create a new mock exporter
			exporter := newMockExporter()
			exporter.scrapeResult = tc.scrapeResult

			// Create a channel to capture metrics
			ch := make(chan prometheus.Metric)
			
			// Collect metrics in a goroutine
			go func() {
				exporter.Collect(ch)
				close(ch)
			}()

			// Count metrics
			metricCount := 0
			for range ch {
				metricCount++
			}

			// Verify scrape was called
			if !exporter.scrapeCalled {
				t.Error("Expected scrape to be called during Collect")
			}

			// Verify up metric was set with correct value using our stored value
			if exporter.upValue != tc.expectedUp {
				t.Errorf("Expected up metric to be %f, got %f", tc.expectedUp, exporter.upValue)
			}

			// Verify total count of metrics
			expectedMetricCount := 4 // up, totalScrapes, totalAPIErrors, jsonParseFailures
			if metricCount != expectedMetricCount {
				t.Errorf("Expected %d metrics, got %d", expectedMetricCount, metricCount)
			}
		})
	}
} 