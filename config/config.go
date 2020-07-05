// Package config defines the data structures related to configuration and
// includes functions for modifying the loading and parsing the config.
package config

import (
	"fmt"
	"github.com/spf13/viper"
	"go.uber.org/zap"
	"math"
	"time"
)

// DateTimeLayout is the format expected in config files and is also the output
// date format.
const DateTimeLayout = "2006-01"

// Configuration holds all configuration for finance-forecast.
type Configuration struct {
	Common    Common
	Scenarios []Scenario
}

// Common holds the shared parameters, events, and loans between all scenarios.
type Common struct {
	StartingValue float64
	DeathDate     string
	Events        []Event
	Loans         []Loan
}

// Scenario holds all events and loans for a given scenario.
type Scenario struct {
	Name   string
	Active bool
	Events []Event
	Loans  []Loan
}

// Event indicates a financial event.
type Event struct {
	Name      string
	Amount    float64
	StartDate string
	EndDate   string
	Frequency int // months
	DateList  []time.Time
}

// Loan indicates a loan and its parameters.
type Loan struct {
	Name                    string
	StartDate               string
	Principal               float64
	InterestRate            float64
	Term                    int // months
	DownPayment             float64
	Escrow                  float64
	MortgageInsurance       float64
	MortgageInsuranceCutoff float64
	EarlyPayoffThreshold    float64
	EarlyPayoffDate         string
	SellProperty            bool
	ValueChange             float64
	AmortizationSchedule    map[string]Payment
}

// Payment holds the values for a given payment
type Payment struct {
	Payment            float64
	Principal          float64
	Interest           float64
	RemainingPrincipal float64
	RefundableEscrow   float64
}

// LoadConfiguration takes a file path as input and loads the YAML-formatted
// configuration there.
func LoadConfiguration(configPath string) (*Configuration, error) {
	viper.SetConfigFile(configPath)
	viper.AutomaticEnv()

	viper.SetConfigType("yml")

	if err := viper.ReadInConfig(); err != nil {
		return nil, fmt.Errorf("Error reading config file, %s", err)
	}

	var configuration Configuration
	err := viper.Unmarshal(&configuration)
	if err != nil {
		return nil, fmt.Errorf("unable to decode into struct, %s", err)
	}

	return &configuration, nil
}

// ParseDateLists looks at every date provided in the configuration and
// parses it into a time.Time which is stored back into an Event.DateList.
func (conf *Configuration) ParseDateLists() error {
	// First handle the parsing for all Events in Scenarios.
	for i, scenario := range conf.Scenarios {
		for j := range scenario.Events {
			err := conf.Scenarios[i].Events[j].FormDateList(*conf)
			if err != nil {
				return err
			}
		}
	}

	// Next handle the parsing for the Common Events.
	for i := range conf.Common.Events {
		err := conf.Common.Events[i].FormDateList(*conf)
		if err != nil {
			return err
		}
	}

	return nil
}

// FormDateList handles the date to time.Time parsing for one given event.
func (event *Event) FormDateList(conf Configuration) error {
	dateList := make([]time.Time, 1)
	var startDateT time.Time
	var err error

	// Unspecified startDate goes to the current time.
	if event.StartDate == "" {
		startDateT, err = time.Parse(DateTimeLayout, time.Now().Format(DateTimeLayout))
		if err != nil {
			return err
		}
	} else {
		startDateT, err = time.Parse(DateTimeLayout, event.StartDate)
		if err != nil {
			return err
		}
	}

	// Unspecified endDate goes to the deathDate.
	if event.EndDate == "" {
		event.EndDate = conf.Common.DeathDate
	}
	endDateT, err := time.Parse(DateTimeLayout, event.EndDate)
	if err != nil {
		return err
	}

	// Identify all dates where an event takes place and aggregate them in
	// dateList.
	dateList[0] = startDateT
	for {
		nextDate := dateList[len(dateList)-1].AddDate(0, event.Frequency, 0)
		if nextDate.Equal(endDateT) {
			dateList = append(dateList, nextDate)
			break
		} else if nextDate.After(endDateT) {
			break
		} else {
			dateList = append(dateList, nextDate)
		}
	}
	event.DateList = dateList

	return nil
}

// IncrementDate returns the next string-formatted date following the input
// date; this is always a 1-month increment.
func IncrementDate(previousDate, layout string) (string, error) {
	t, err := time.Parse(layout, previousDate)
	if err != nil {
		return previousDate, err
	}
	date := t.AddDate(0, 1, 0).Format(layout)
	return date, nil
}

// DecrementDate returns the previous string-formatted date prior to the input
// date; this is always a 1-month decrement.
func DecrementDate(currentDate, layout string) (string, error) {
	t, err := time.Parse(layout, currentDate)
	if err != nil {
		return currentDate, err
	}
	date := t.AddDate(0, -1, 0).Format(layout)
	return date, nil
}

// ProcessLoans iterates through all loans and produces the amortization
// schedules.
func (conf *Configuration) ProcessLoans(logger *zap.Logger) error {
	// First handle the processing for all Loans in Scenarios.
	for i, scenario := range conf.Scenarios {
		for j := range scenario.Loans {
			conf.Scenarios[i].Loans[j].ApplyDownPayment()
			err := conf.Scenarios[i].Loans[j].GetAmortizationSchedule(logger, *conf)
			if err != nil {
				return err
			}
		}
	}

	// Next handle the processing for the Common Loans.
	for i := range conf.Common.Loans {
		conf.Common.Loans[i].ApplyDownPayment()
		err := conf.Common.Loans[i].GetAmortizationSchedule(logger, *conf)
		if err != nil {
			return err
		}
	}

	return nil
}

// ApplyDownPayment modifies the loan principal to reflect any down payment
// so that the amortization schedules is computed correctly.
func (loan *Loan) ApplyDownPayment() {
	loan.Principal -= loan.DownPayment
}

// GetAmortizationSchedule computes the amortization schedule for a given Loan.
// This will also take into account optional configuration such as down
// payments, early payoff dates, and selling property on payoff (for example
// when trading in vehicles or upgrading homes). TODO this function is a mouth
// full and is a candidate for being revised.
func (loan *Loan) GetAmortizationSchedule(logger *zap.Logger, conf Configuration) error {
	// Compute periodic payment fundamentals.
	periodicInterestRate := loan.InterestRate / (100.0 * 12.0)
	power := math.Pow((1.00 + periodicInterestRate), float64(loan.Term))
	discountFactor := (power - 1.00) / power
	loanPayment := loan.Principal * periodicInterestRate / discountFactor

	loan.AmortizationSchedule = make(map[string]Payment)

	// Handle the first month individually. TODO consider using a ghost point
	// to prevent having to treat this differently.
	var firstPayment Payment
	firstPayment.Payment = loanPayment + loan.Escrow + loan.DownPayment
	firstPayment.Interest = loan.Principal * loan.InterestRate / (100.0 * 12.0)
	firstPayment.Principal = loanPayment - firstPayment.Interest
	firstPayment.RemainingPrincipal = loan.Principal - firstPayment.Principal
	firstPayment.RefundableEscrow = loan.Escrow
	loan.AmortizationSchedule[loan.StartDate] = firstPayment

	// Iterate over the remainder of the term.
	previousMonth := loan.StartDate
	currentMonth, err := IncrementDate(previousMonth, DateTimeLayout)
	if err != nil {
		return err
	}

	for month := 2; month <= loan.Term; month++ {
		var currentPayment Payment

		// Calculate refundable escrow
		january, err := CheckMonth(currentMonth, "01")
		if err != nil {
			return err
		}
		if january {
			currentPayment.RefundableEscrow = 0.00
		} else {
			currentPayment.RefundableEscrow = loan.AmortizationSchedule[previousMonth].RefundableEscrow + loan.Escrow
		}

		if loan.EarlyPayoffDate == currentMonth {
			if loan.SellProperty {
				currentPayment.Payment = loan.AmortizationSchedule[previousMonth].RemainingPrincipal - loan.Principal*(1.0+loan.ValueChange/100.0) - currentPayment.RefundableEscrow
				logger.Debug(fmt.Sprintf("%s: paying off asset %s for %.2f and selling for %.2f", loan.EarlyPayoffDate, loan.Name, loan.AmortizationSchedule[previousMonth].RemainingPrincipal, loan.Principal*(1.0+loan.ValueChange/100.0)),
					zap.String("op", "config.GetAmortizationSchedule"),
				)
				loan.AmortizationSchedule[currentMonth] = currentPayment
			} else {
				currentPayment.Payment = loan.AmortizationSchedule[previousMonth].RemainingPrincipal - currentPayment.RefundableEscrow
				logger.Debug(fmt.Sprintf("%s: paying off asset %s for %.2f", loan.EarlyPayoffDate, loan.Name, loan.AmortizationSchedule[previousMonth].RemainingPrincipal),
					zap.String("op", "config.GetAmortizationSchedule"),
				)
				loan.AmortizationSchedule[currentMonth] = currentPayment
				// Since we paid off the loan but did not sell the asset we will
				// extrapolate the escrow to be paid on Decembers.
				for {
					if currentMonth == conf.Common.DeathDate {
						break
					}
					december, err := CheckMonth(currentMonth, "12")
					if err != nil {
						return err
					}
					if december {
						var escrowPayment Payment
						escrowPayment.Payment = loan.Escrow * 12
						loan.AmortizationSchedule[currentMonth] = escrowPayment
					}
					previousMonth = currentMonth
					currentMonth, err = IncrementDate(currentMonth, DateTimeLayout)
					if err != nil {
						return err
					}
				}
			}
			break
		} else {
			currentPayment.Payment = loanPayment + loan.Escrow
			currentPayment.Interest = loan.AmortizationSchedule[previousMonth].RemainingPrincipal * loan.InterestRate / (100.0 * 12.0)
			currentPayment.Principal = loanPayment - currentPayment.Interest
			if month == loan.Term {
				// We will get machine error otherwise so just set to 0.
				currentPayment.RemainingPrincipal = 0.00
				// Incorporate the expected escrow refund; the RedunableEscrow value
				// tracks the refundable amount for early payoffs so we need to reduce
				// further by an escrow payment
				currentPayment.Payment -= (currentPayment.RefundableEscrow + loan.Escrow)
			} else {
				currentPayment.RemainingPrincipal = loan.AmortizationSchedule[previousMonth].RemainingPrincipal - currentPayment.Principal
			}
			if loan.MortgageInsuranceCutoff > 0 {
				if currentPayment.RemainingPrincipal/loan.Principal <= loan.MortgageInsuranceCutoff/100.0 {
					currentPayment.Payment -= loan.MortgageInsurance
				}
			}
			loan.AmortizationSchedule[currentMonth] = currentPayment
			// Since the loan matured we will extrapolate the escrow to be paid on
			// Decembers.
			if month == loan.Term {
				for {
					if currentMonth == conf.Common.DeathDate {
						break
					}
					december, err := CheckMonth(currentMonth, "12")
					if err != nil {
						return err
					}
					if december {
						var escrowPayment Payment
						escrowPayment.Payment = loan.Escrow * 12
						loan.AmortizationSchedule[currentMonth] = escrowPayment
					}
					previousMonth = currentMonth
					currentMonth, err = IncrementDate(currentMonth, DateTimeLayout)
					if err != nil {
						return err
					}
				}
			}
		}
		previousMonth = currentMonth
		currentMonth, err = IncrementDate(currentMonth, DateTimeLayout)
		if err != nil {
			return err
		}
	}

	return nil
}

// CheckEarlyPayoffThreshold checks for whether or not it is time to payoff a
// loan early based on an optionally-configured threshold. Note that escrow
// refunds are not factored into the threshold comparison because in reality
// those can take some time to process (even though the simulation acts as
// though an escrow refund is processed immediately).
func (loan *Loan) CheckEarlyPayoffThreshold(logger *zap.Logger, currentMonth string, deathDate string, balance float64) (string, error) {
	var note string
	started, err := DateBeforeDate(loan.StartDate, currentMonth)
	if err != nil {
		return note, err
	}
	if loan.EarlyPayoffThreshold > 0 && started {
		previousMonth, err := DecrementDate(currentMonth, DateTimeLayout)
		if err != nil {
			return note, err
		}
		if balance-loan.AmortizationSchedule[previousMonth].RemainingPrincipal >= loan.EarlyPayoffThreshold {
			logger.Debug(fmt.Sprintf("%s: based on threshold paying off asset %s for %.2f", currentMonth, loan.Name, loan.AmortizationSchedule[previousMonth].RemainingPrincipal),
				zap.String("op", "config.CheckEarlyPayoffThreshold"),
			)
			var finalPayment Payment
			if loan.SellProperty {
				finalPayment.Payment = loan.AmortizationSchedule[previousMonth].RemainingPrincipal - loan.AmortizationSchedule[currentMonth].RefundableEscrow - loan.Principal*(1.0+loan.ValueChange/100.0)
				loan.AmortizationSchedule[currentMonth] = finalPayment
				note = fmt.Sprintf("paying off asset %s for %.2f and selling for %.2f", loan.Name, loan.AmortizationSchedule[previousMonth].RemainingPrincipal, loan.Principal*(1.0+loan.ValueChange/100.0))
				logger.Debug(fmt.Sprintf("%s: selling asset %s for %.2f", currentMonth, loan.Name, loan.Principal*(1.0+loan.ValueChange/100.0)),
					zap.String("op", "config.CheckEarlyPayoffThreshold"),
				)
			} else {
				note = fmt.Sprintf("paying off asset %s for %.2f", loan.Name, loan.AmortizationSchedule[previousMonth].RemainingPrincipal)
				finalPayment.Payment = loan.AmortizationSchedule[previousMonth].RemainingPrincipal - loan.AmortizationSchedule[currentMonth].RefundableEscrow
				loan.AmortizationSchedule[currentMonth] = finalPayment
			}

			// Modify the remainder of the amortization schedule to null out payments
			// and if we did not sell the property and did declare escrow then handle
			// converting that into equivalent annual payments.
			loan.EarlyPayoffThreshold = 0
			for {
				if currentMonth == deathDate {
					break
				}
				previousMonth = currentMonth
				currentMonth, err = IncrementDate(currentMonth, DateTimeLayout)
				if err != nil {
					return note, err
				}
				december, err := CheckMonth(currentMonth, "12")
				if err != nil {
					return note, err
				}
				if december && loan.Escrow > 0 {
					var escrowPayment Payment
					escrowPayment.Payment = loan.Escrow * 12
					loan.AmortizationSchedule[currentMonth] = escrowPayment
				} else {
					delete(loan.AmortizationSchedule, currentMonth)
				}
			}

		}
	}
	return note, nil
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
