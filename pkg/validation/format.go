// Package validation provides common validation utilities.
package validation

import (
	"fmt"

	"github.com/iwvelando/finance-forecast/pkg/constants"
)

// ValidateOutputFormat checks if the output format is one of the supported formats.
func ValidateOutputFormat(format string) error {
	if format != constants.OutputFormatPretty && format != constants.OutputFormatCSV {
		return fmt.Errorf("expected output format of %s or %s, got %s",
			constants.OutputFormatPretty, constants.OutputFormatCSV, format)
	}
	return nil
}
