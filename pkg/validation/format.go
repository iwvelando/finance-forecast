// Package validation provides common validation utilities.
package validation

import "fmt"

// ValidateOutputFormat checks if the output format is one of the supported formats.
func ValidateOutputFormat(format string) error {
	if format != "pretty" && format != "csv" {
		return fmt.Errorf("expected output format of pretty or csv, got %s", format)
	}
	return nil
}
