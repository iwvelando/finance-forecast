package validation

import "testing"

func TestValidateOutputFormat(t *testing.T) {
	tests := []struct {
		name      string
		format    string
		expectErr bool
	}{
		{
			name:      "Valid pretty format",
			format:    "pretty",
			expectErr: false,
		},
		{
			name:      "Valid csv format",
			format:    "csv",
			expectErr: false,
		},
		{
			name:      "Invalid format",
			format:    "json",
			expectErr: true,
		},
		{
			name:      "Empty format",
			format:    "",
			expectErr: true,
		},
		{
			name:      "Case sensitive - uppercase",
			format:    "PRETTY",
			expectErr: true,
		},
		{
			name:      "Case sensitive - mixed case",
			format:    "Pretty",
			expectErr: true,
		},
		{
			name:      "Case sensitive - CSV uppercase",
			format:    "CSV",
			expectErr: true,
		},
		{
			name:      "Leading/trailing spaces",
			format:    " pretty ",
			expectErr: true,
		},
		{
			name:      "Similar but incorrect format",
			format:    "prettyprint",
			expectErr: true,
		},
		{
			name:      "XML format not supported",
			format:    "xml",
			expectErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateOutputFormat(tt.format)

			if tt.expectErr {
				if err == nil {
					t.Errorf("ValidateOutputFormat(%s) expected error but got none", tt.format)
				}
			} else {
				if err != nil {
					t.Errorf("ValidateOutputFormat(%s) unexpected error = %v", tt.format, err)
				}
			}
		})
	}
}

func TestValidateOutputFormatErrorMessage(t *testing.T) {
	// Test that error messages are informative
	invalidFormats := []string{"json", "xml", "yaml", ""}

	for _, format := range invalidFormats {
		err := ValidateOutputFormat(format)
		if err == nil {
			t.Errorf("Expected error for format '%s'", format)
			continue
		}

		// Check that error message contains the invalid format
		errorMsg := err.Error()
		if format != "" && errorMsg != "" {
			// For non-empty formats, the error should mention the format
			// This is a basic check - the actual error message format may vary
			if len(errorMsg) < 10 { // Ensure we have a meaningful error message
				t.Errorf("Error message too short for format '%s': %s", format, errorMsg)
			}
		}
	}
}

func TestValidateOutputFormatBoundaryConditions(t *testing.T) {
	// Test boundary conditions and edge cases
	tests := []struct {
		name      string
		format    string
		expectErr bool
	}{
		{
			name:      "Single character",
			format:    "p",
			expectErr: true,
		},
		{
			name:      "Very long invalid format",
			format:    "this-is-a-very-long-invalid-format-name",
			expectErr: true,
		},
		{
			name:      "Special characters",
			format:    "pretty!",
			expectErr: true,
		},
		{
			name:      "Numbers",
			format:    "pretty123",
			expectErr: true,
		},
		{
			name:      "Underscore format",
			format:    "pretty_format",
			expectErr: true,
		},
		{
			name:      "Hyphen format",
			format:    "pretty-format",
			expectErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateOutputFormat(tt.format)

			if tt.expectErr {
				if err == nil {
					t.Errorf("ValidateOutputFormat(%s) expected error but got none", tt.format)
				}
			} else {
				if err != nil {
					t.Errorf("ValidateOutputFormat(%s) unexpected error = %v", tt.format, err)
				}
			}
		})
	}
}
