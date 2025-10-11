package output

import (
	"bytes"
	"io"
	"os"
	"strings"
	"testing"

	"github.com/iwvelando/finance-forecast/internal/forecast"
	"github.com/iwvelando/finance-forecast/pkg/optimization"
)

// Simple temporary implementation to get tests passing
func TestPrettyFormat(t *testing.T) {
	// Create test data
	results := []forecast.Forecast{
		{
			Name: "Test Scenario",
			Data: map[string]float64{
				"2025-01": 1000.00,
			},
			Liquid: map[string]float64{
				"2025-01": 750.00,
			},
			Notes: map[string][]string{
				"2025-01": {"Test note"},
			},
		},
	}

	// Capture stdout
	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	PrettyFormat(results)

	_ = w.Close()
	os.Stdout = oldStdout

	var buf bytes.Buffer
	_, _ = io.Copy(&buf, r)
	output := buf.String()

	// Should contain the expected format elements
	if !strings.Contains(output, "--- Results for scenario Test Scenario ---") {
		t.Errorf("PrettyFormat missing scenario header")
	}
	if !strings.Contains(output, "Date    | Liquid Net Worth | Total Net Worth | Notes") {
		t.Errorf("PrettyFormat missing table header")
	}
	if !strings.Contains(output, "____    | ________________ | _______________ | _____") {
		t.Errorf("PrettyFormat missing table separator")
	}
	if !strings.Contains(output, "$750.00") {
		t.Errorf("PrettyFormat missing liquid column value")
	}
	if !strings.Contains(output, "$1,000.00") {
		t.Errorf("PrettyFormat missing total column value")
	}
	if !strings.Contains(output, "Test note") {
		t.Errorf("PrettyFormat missing note")
	}
}

func TestPrettyFormatEmergencyFundSummary(t *testing.T) {
	results := []forecast.Forecast{
		{
			Name:   "Scenario A",
			Data:   map[string]float64{"2025-01": 1000},
			Liquid: map[string]float64{"2025-01": 800},
			Metrics: forecast.ForecastMetrics{
				EmergencyFund: &forecast.EmergencyFundRecommendation{
					TargetMonths:           6,
					AverageMonthlyExpenses: 1500,
					TargetAmount:           9000,
					InitialLiquid:          8000,
					FundedMonths:           5.3,
					Shortfall:              1000,
				},
			},
		},
	}

	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	PrettyFormat(results)

	_ = w.Close()
	os.Stdout = oldStdout

	var buf bytes.Buffer
	_, _ = io.Copy(&buf, r)
	output := buf.String()

	if !strings.Contains(output, "Emergency fund target (6.0 months)") {
		t.Fatalf("expected emergency fund summary, got %q", output)
	}
	if !strings.Contains(output, "Shortfall: $1,000.00") {
		t.Fatalf("expected shortfall detail in summary")
	}
}

func TestPrettyFormatOptimizationSummary(t *testing.T) {
	results := []forecast.Forecast{
		{
			Name:   "Scenario A",
			Data:   map[string]float64{"2025-01": 1000},
			Liquid: map[string]float64{"2025-01": 800},
			Metrics: forecast.ForecastMetrics{
				Optimizations: []optimization.Summary{
					{
						TargetName:  "New Job",
						Field:       "amount",
						Original:    2000,
						Value:       1200,
						Floor:       15000,
						MinimumCash: 15500,
						Headroom:    500,
						Iterations:  6,
						Converged:   true,
					},
				},
			},
		},
	}

	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	PrettyFormat(results)

	_ = w.Close()
	os.Stdout = oldStdout

	var buf bytes.Buffer
	_, _ = io.Copy(&buf, r)
	output := buf.String()

	if !strings.Contains(output, "Optimization adjustments:") {
		t.Fatalf("expected optimization header in output, got %q", output)
	}
	if !strings.Contains(output, "New Job (amount)") {
		t.Fatalf("expected optimization detail line, got %q", output)
	}
}

func TestPrettyFormatSingleScenario(t *testing.T) {
	// Test with single scenario
	results := []forecast.Forecast{
		{
			Name: "Single Scenario",
			Data: map[string]float64{
				"2025-01": 1000.00,
			},
			Liquid: map[string]float64{
				"2025-01": 600.00,
			},
			Notes: map[string][]string{
				"2025-01": {"Single note"},
			},
		},
	}

	// Capture stdout
	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	PrettyFormat(results)

	_ = w.Close()
	os.Stdout = oldStdout

	var buf bytes.Buffer
	_, _ = io.Copy(&buf, r)
	output := buf.String()

	// Should contain scenario name and data
	if !strings.Contains(output, "--- Results for scenario Single Scenario ---") {
		t.Errorf("PrettyFormat missing scenario header")
	}
	if !strings.Contains(output, "$1,000.00") {
		t.Errorf("PrettyFormat missing formatted amount")
	}
	if !strings.Contains(output, "$600.00") {
		t.Errorf("PrettyFormat missing liquid column value")
	}
	if !strings.Contains(output, "Single note") {
		t.Errorf("PrettyFormat missing note")
	}
	if !strings.Contains(output, "Date    | Liquid Net Worth | Total Net Worth | Notes") {
		t.Errorf("PrettyFormat missing table header")
	}
}

func TestPrettyFormatEmptyResults(t *testing.T) {
	// Test with empty results
	results := []forecast.Forecast{}

	// Shouldn't crash
	defer func() {
		if r := recover(); r != nil {
			t.Errorf("PrettyFormat panicked with empty results: %v", r)
		}
	}()

	// Capture stdout to prevent output during test
	oldStdout := os.Stdout
	_, w, _ := os.Pipe()
	os.Stdout = w

	PrettyFormat(results)

	_ = w.Close()
	os.Stdout = oldStdout

	// Just ensure it doesn't crash
}

func TestCsvFormat(t *testing.T) {
	// Create test data
	results := []forecast.Forecast{
		{
			Name: "Scenario A",
			Data: map[string]float64{
				"2025-01": 1000.00,
				"2025-02": 1500.50,
			},
			Liquid: map[string]float64{
				"2025-01": 700.25,
				"2025-02": 1200.75,
			},
			Notes: map[string][]string{
				"2025-01": {"Note A1"},
				"2025-02": {"Note A2", "Additional note"},
			},
		},
		{
			Name: "Scenario B",
			Data: map[string]float64{
				"2025-01": 900.00,
				"2025-02": 1200.25,
			},
			Liquid: map[string]float64{
				"2025-01": 650.00,
				"2025-02": 950.50,
			},
			Notes: map[string][]string{
				"2025-01": {"Note B1"},
				"2025-02": {},
			},
		},
	}

	// Capture stdout
	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	CsvFormat(results)

	_ = w.Close()
	os.Stdout = oldStdout

	var buf bytes.Buffer
	_, _ = io.Copy(&buf, r)
	output := buf.String()

	// Split into lines
	lines := strings.Split(strings.TrimSpace(output), "\n")

	// Should have header + data lines
	if len(lines) < 3 {
		t.Errorf("CsvFormat should produce at least 3 lines (header + 2 data), got %d", len(lines))
	}

	// Check header
	header := lines[0]
	expectedHeaderElements := []string{
		`"date"`,
		`"liquid (Scenario A)"`,
		`"total (Scenario A)"`,
		`"notes (Scenario A)"`,
		`"liquid (Scenario B)"`,
		`"total (Scenario B)"`,
		`"notes (Scenario B)"`,
	}

	for _, element := range expectedHeaderElements {
		if !strings.Contains(header, element) {
			t.Errorf("CsvFormat header missing: %s", element)
		}
	}

	// Check data lines contain expected values
	dataContent := strings.Join(lines[1:], "\n")
	expectedDataElements := []string{
		`"2025-01"`,
		`"2025-02"`,
		`"700.25"`,
		`"1200.75"`,
		`"1000.00"`,
		`"1500.50"`,
		`"650.00"`,
		`"950.50"`,
		`"900.00"`,
		`"1200.25"`,
		`"Note A1"`,
		`"Note A2,Additional note"`,
		`"Note B1"`,
	}

	for _, element := range expectedDataElements {
		if !strings.Contains(dataContent, element) {
			t.Errorf("CsvFormat data missing: %s", element)
		}
	}
}

func TestCsvStringMatchesCsvFormat(t *testing.T) {
	results := []forecast.Forecast{
		{
			Name: "Scenario A",
			Data: map[string]float64{
				"2025-01": 1000.00,
				"2025-02": 1500.50,
			},
			Liquid: map[string]float64{
				"2025-01": 700.25,
				"2025-02": 1200.75,
			},
			Notes: map[string][]string{
				"2025-01": {"Note A1"},
				"2025-02": {"Note A2"},
			},
		},
		{
			Name: "Scenario B",
			Data: map[string]float64{
				"2025-01": 900.00,
			},
			Liquid: map[string]float64{
				"2025-01": 650.00,
			},
			Notes: map[string][]string{
				"2025-01": {"Note B1"},
			},
		},
	}

	expected := CsvString(results)

	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	CsvFormat(results)

	_ = w.Close()
	os.Stdout = oldStdout

	var buf bytes.Buffer
	_, _ = io.Copy(&buf, r)
	output := buf.String()

	if strings.TrimSpace(expected) != strings.TrimSpace(output) {
		t.Fatalf("CsvString and CsvFormat output mismatch\nCsvString:\n%s\nCsvFormat:\n%s", expected, output)
	}
}

func TestCsvFormatSingleScenario(t *testing.T) {
	// Test CSV with single scenario
	results := []forecast.Forecast{
		{
			Name: "Only Scenario",
			Data: map[string]float64{
				"2025-01": 1000.00,
			},
			Liquid: map[string]float64{
				"2025-01": 720.00,
			},
			Notes: map[string][]string{
				"2025-01": {"Only note"},
			},
		},
	}

	// Capture stdout
	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	CsvFormat(results)

	_ = w.Close()
	os.Stdout = oldStdout

	var buf bytes.Buffer
	_, _ = io.Copy(&buf, r)
	output := buf.String()

	lines := strings.Split(strings.TrimSpace(output), "\n")

	// Should have header + 1 data line
	if len(lines) != 2 {
		t.Errorf("CsvFormat with single scenario should produce 2 lines, got %d", len(lines))
	}

	// Check header format
	if !strings.Contains(lines[0], `"date"`) {
		t.Errorf("CsvFormat header missing date column")
	}
	if !strings.Contains(lines[0], `"liquid (Only Scenario)"`) {
		t.Errorf("CsvFormat header missing liquid column")
	}
	if !strings.Contains(lines[0], `"total (Only Scenario)"`) {
		t.Errorf("CsvFormat header missing total column")
	}
	if !strings.Contains(lines[0], `"notes (Only Scenario)"`) {
		t.Errorf("CsvFormat header missing notes column")
	}
}

func TestCsvFormatEmptyResults(t *testing.T) {
	// Test with empty results - should not crash
	results := []forecast.Forecast{}

	defer func() {
		if r := recover(); r != nil {
			t.Errorf("CsvFormat panicked with empty results: %v", r)
		}
	}()

	// Capture stdout to prevent output during test
	oldStdout := os.Stdout
	_, w, _ := os.Pipe()
	os.Stdout = w

	// This might panic due to accessing results[0], but let's test
	CsvFormat(results)

	_ = w.Close()
	os.Stdout = oldStdout
}

func TestCsvFormatNotesHandling(t *testing.T) {
	// Test various note scenarios
	results := []forecast.Forecast{
		{
			Name: "Notes Test",
			Data: map[string]float64{
				"2025-01": 1000.00,
				"2025-02": 2000.00,
				"2025-03": 3000.00,
			},
			Liquid: map[string]float64{
				"2025-01": 700.00,
				"2025-02": 1600.00,
				"2025-03": 2500.00,
			},
			Notes: map[string][]string{
				"2025-01": {"Single note"},
				"2025-02": {"Multiple", "notes", "here"},
				"2025-03": {}, // Empty notes
			},
		},
	}

	// Capture stdout
	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	CsvFormat(results)

	_ = w.Close()
	os.Stdout = oldStdout

	var buf bytes.Buffer
	_, _ = io.Copy(&buf, r)
	output := buf.String()

	// Check that multiple notes are joined with commas
	if !strings.Contains(output, `"Multiple,notes,here"`) {
		t.Errorf("CsvFormat should join multiple notes with commas")
	}

	// Check that single note is handled correctly
	if !strings.Contains(output, `"Single note"`) {
		t.Errorf("CsvFormat should handle single notes correctly")
	}

	// Check that empty notes result in empty quoted string
	lines := strings.Split(output, "\n")
	for _, line := range lines {
		if strings.Contains(line, "2025-03") {
			if !strings.Contains(line, `""`) {
				t.Errorf("CsvFormat should have empty quoted string for no notes")
			}
			break
		}
	}
}

func TestFormatDateSorting(t *testing.T) {
	// Test that dates are properly sorted in output
	results := []forecast.Forecast{
		{
			Name: "Date Sort Test",
			Data: map[string]float64{
				"2025-12": 1000.00,
				"2025-01": 2000.00,
				"2025-06": 3000.00,
			},
			Notes: map[string][]string{
				"2025-12": {"December"},
				"2025-01": {"January"},
				"2025-06": {"June"},
			},
		},
	}

	// Test both formats
	formats := []struct {
		name string
		fn   func([]forecast.Forecast)
	}{
		{"PrettyFormat", PrettyFormat},
		{"CsvFormat", CsvFormat},
	}

	for _, format := range formats {
		t.Run(format.name, func(t *testing.T) {
			// Capture stdout
			oldStdout := os.Stdout
			r, w, _ := os.Pipe()
			os.Stdout = w

			format.fn(results)

			_ = w.Close()
			os.Stdout = oldStdout

			var buf bytes.Buffer
			_, _ = io.Copy(&buf, r)
			output := buf.String()

			// Find positions of dates in output
			pos01 := strings.Index(output, "2025-01")
			pos06 := strings.Index(output, "2025-06")
			pos12 := strings.Index(output, "2025-12")

			// All should be found
			if pos01 == -1 || pos06 == -1 || pos12 == -1 {
				t.Errorf("%s missing some dates in output", format.name)
				return
			}

			// Should be in chronological order
			if pos01 > pos06 || pos06 > pos12 {
				t.Errorf("%s dates not in chronological order", format.name)
			}
		})
	}
}

// Simple test to check the function signature issue
func TestFormatFunctionSignatures(t *testing.T) {
	// Test that format functions can be called with []forecast.Forecast
	results := []forecast.Forecast{
		{
			Name:  "Test",
			Data:  map[string]float64{"2025-01": 1000.00},
			Notes: map[string][]string{"2025-01": {"test note"}},
		},
	}

	// These should compile without errors
	PrettyFormat(results)
	CsvFormat(results)
}
