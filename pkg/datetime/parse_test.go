package datetime

import (
	"testing"
)

func TestMustParseTime(t *testing.T) {
	tests := []struct {
		name     string
		layout   string
		dateStr  string
		expected string
	}{
		{
			name:     "Valid date",
			layout:   DateTimeLayout,
			dateStr:  "2025-01",
			expected: "2025-01",
		},
		{
			name:     "Another valid date",
			layout:   DateTimeLayout,
			dateStr:  "2030-12",
			expected: "2030-12",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := MustParseTime(tt.layout, tt.dateStr)
			if result.Format(tt.layout) != tt.expected {
				t.Errorf("MustParseTime() = %s, expected %s", result.Format(tt.layout), tt.expected)
			}
		})
	}
}

func TestMustParseTimePanic(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Errorf("Expected MustParseTime to panic with invalid date")
		}
	}()

	MustParseTime(DateTimeLayout, "invalid-date")
}

func TestOffsetDateAdvanced(t *testing.T) {
	tests := []struct {
		name     string
		date     string
		layout   string
		months   int
		expected string
		wantErr  bool
	}{
		{
			name:     "Add multiple years",
			date:     "2025-01",
			layout:   DateTimeLayout,
			months:   24,
			expected: "2027-01",
			wantErr:  false,
		},
		{
			name:     "Subtract multiple years",
			date:     "2025-01",
			layout:   DateTimeLayout,
			months:   -24,
			expected: "2023-01",
			wantErr:  false,
		},
		{
			name:     "Cross year boundary forward",
			date:     "2025-06",
			layout:   DateTimeLayout,
			months:   8,
			expected: "2026-02",
			wantErr:  false,
		},
		{
			name:     "Cross year boundary backward",
			date:     "2025-06",
			layout:   DateTimeLayout,
			months:   -8,
			expected: "2024-10",
			wantErr:  false,
		},
		{
			name:     "Zero months",
			date:     "2025-06",
			layout:   DateTimeLayout,
			months:   0,
			expected: "2025-06",
			wantErr:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := OffsetDate(tt.date, tt.layout, tt.months)
			if tt.wantErr {
				if err == nil {
					t.Errorf("OffsetDate() expected error but got none")
				}
				return
			}
			if err != nil {
				t.Errorf("OffsetDate() error = %v", err)
				return
			}
			if result != tt.expected {
				t.Errorf("OffsetDate() = %v, expected %v", result, tt.expected)
			}
		})
	}
}

func TestCheckMonthAdvanced(t *testing.T) {
	tests := []struct {
		name     string
		date     string
		month    string
		expected bool
		wantErr  bool
	}{
		{
			name:     "All months match correctly",
			date:     "2025-03",
			month:    "03",
			expected: true,
			wantErr:  false,
		},
		{
			name:     "Leading zero handling",
			date:     "2025-01",
			month:    "01",
			expected: true,
			wantErr:  false,
		},
		{
			name:     "December match",
			date:     "2025-12",
			month:    "12",
			expected: true,
			wantErr:  false,
		},
		{
			name:     "No match different months",
			date:     "2025-01",
			month:    "02",
			expected: false,
			wantErr:  false,
		},
		{
			name:     "Invalid month format",
			date:     "2025-01",
			month:    "1", // Should be "01"
			expected: false,
			wantErr:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := CheckMonth(tt.date, tt.month)
			if tt.wantErr {
				if err == nil {
					t.Errorf("CheckMonth() expected error but got none")
				}
				return
			}
			if err != nil {
				t.Errorf("CheckMonth() error = %v", err)
				return
			}
			if result != tt.expected {
				t.Errorf("CheckMonth() = %v, expected %v", result, tt.expected)
			}
		})
	}
}

func TestDateBeforeDateAdvanced(t *testing.T) {
	tests := []struct {
		name       string
		firstDate  string
		secondDate string
		expected   bool
		wantErr    bool
	}{
		{
			name:       "Different years",
			firstDate:  "2024-12",
			secondDate: "2025-01",
			expected:   true,
			wantErr:    false,
		},
		{
			name:       "Same year different months",
			firstDate:  "2025-01",
			secondDate: "2025-06",
			expected:   true,
			wantErr:    false,
		},
		{
			name:       "Reverse order",
			firstDate:  "2025-06",
			secondDate: "2025-01",
			expected:   false,
			wantErr:    false,
		},
		{
			name:       "Equal dates",
			firstDate:  "2025-06",
			secondDate: "2025-06",
			expected:   false,
			wantErr:    false,
		},
		{
			name:       "Large time difference",
			firstDate:  "2020-01",
			secondDate: "2030-12",
			expected:   true,
			wantErr:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := DateBeforeDate(tt.firstDate, tt.secondDate)
			if tt.wantErr {
				if err == nil {
					t.Errorf("DateBeforeDate() expected error but got none")
				}
				return
			}
			if err != nil {
				t.Errorf("DateBeforeDate() error = %v", err)
				return
			}
			if result != tt.expected {
				t.Errorf("DateBeforeDate() = %v, expected %v", result, tt.expected)
			}
		})
	}
}

func TestDateTimeLayoutConstant(t *testing.T) {
	// Test that our constant matches the format expected
	testDate := "2025-06"
	parsedTime := MustParseTime(DateTimeLayout, testDate)

	if parsedTime.Format(DateTimeLayout) != testDate {
		t.Errorf("DateTimeLayout constant doesn't work correctly for parsing/formatting")
	}
}

func TestTimeOperations(t *testing.T) {
	// Test various time operations work correctly with our layout
	baseDate := "2025-01"

	// Test forward operations
	future, err := OffsetDate(baseDate, DateTimeLayout, 6)
	if err != nil {
		t.Fatalf("OffsetDate forward failed: %v", err)
	}

	// Test backward operations
	past, err := OffsetDate(future, DateTimeLayout, -6)
	if err != nil {
		t.Fatalf("OffsetDate backward failed: %v", err)
	}

	if past != baseDate {
		t.Errorf("Round trip date operation failed: started with %s, ended with %s", baseDate, past)
	}
}
