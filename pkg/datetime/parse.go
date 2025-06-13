// Package datetime provides date and time utility functions.
package datetime

import (
	"time"

	"github.com/iwvelando/finance-forecast/pkg/constants"
)

const (
	// DateTimeLayout is the format expected in config files and is also the output
	// date format.
	DateTimeLayout = constants.DateTimeLayout
)

// MustParseTime parses a date string using the given layout and panics on error.
// This is intended for use in tests where the date string is known to be valid.
func MustParseTime(layout, dateStr string) time.Time {
	t, err := time.Parse(layout, dateStr)
	if err != nil {
		panic(err)
	}
	return t
}

// OffsetDate returns the string-formatted date offset by the given number of
// months relative to the given date.
func OffsetDate(date, layout string, months int) (string, error) {
	t, err := time.Parse(layout, date)
	if err != nil {
		return date, err
	}
	return t.AddDate(0, months, 0).Format(layout), nil
}

// CheckMonth identifies whether a given date is in the month indicated by the
// numeric representation e.g. 01 = January and 12 = December.
func CheckMonth(date string, month string) (bool, error) {
	dateT, err := time.Parse(DateTimeLayout, date)
	if err != nil {
		return false, err
	}
	if dateT.Format("01") == month {
		return true, nil
	} else {
		return false, nil
	}
}

// DateBeforeDate returns true if firstDate is strictly before secondDate.
func DateBeforeDate(firstDate string, secondDate string) (bool, error) {
	firstDateT, err := time.Parse(DateTimeLayout, firstDate)
	if err != nil {
		return false, err
	}
	secondDateT, err := time.Parse(DateTimeLayout, secondDate)
	if err != nil {
		return false, err
	}
	return firstDateT.Before(secondDateT), nil
}
