// Package testutil provides common utility functions for testing.
package testutil

import (
	"github.com/iwvelando/finance-forecast/internal/forecast"
)

// FindScenario finds a scenario by name in the results slice.
// Returns a pointer to the forecast if found, nil otherwise.
func FindScenario(results []forecast.Forecast, name string) *forecast.Forecast {
	for i := range results {
		if results[i].Name == name {
			return &results[i]
		}
	}
	return nil
}
