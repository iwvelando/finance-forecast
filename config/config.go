// Package config defines the data structures related to configuration and
// includes functions for modifying the loading and parsing the config.
package config

import (
	"fmt"
	"github.com/spf13/viper"
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
	Name                 string
	Principal            float64
	InterestRate         float64
	Term                 int // months
	StartDate            string
	EarlyPayoffThreshold float64
	SellProperty         bool
	ValueChange          float64
	AmortizationSchedule Amortization
}

// Payment holds the values for a given payment
type Payment struct {
	Payment            float64
	Principal          float64
	Interest           float64
	RemainingPrincipal float64
}

// Amortization holds the amortization schedule per month
type Amortization struct {
	Payments map[string]Payment
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

// ProcessLoans iterates through all loans and produces the amortization
// schedules.
func (conf *Configuration) ProcessLoans() error {
	// First handle the processing for all Loans in Scenarios.
	for _, scenario := range conf.Scenarios {
		for _, loan := range scenario.Loans {
			err := loan.GetAmortizationSchedule()
			if err != nil {
				return err
			}
		}
	}

	// Next handle the processing for the Common Loans.
	for _, loan := range conf.Common.Loans {
		err := loan.GetAmortizationSchedule()
		if err != nil {
			return err
		}
	}

	return nil
}

// GetAmortizationSchedule computes the amortization schedule for a given Loan.
func (loan *Loan) GetAmortizationSchedule() error {
	periodicInterestRate := loan.InterestRate / (100.0 * 12.0)
	power := math.Pow((1.00 + periodicInterestRate), float64(loan.Term))
	discountFactor := (power - 1.00) / power
	payment := loan.Principal * periodicInterestRate / discountFactor

	// Handle the first month individually.
	loan.AmortizationSchedule.Payments = make(map[string]Payment)
	loan.AmortizationSchedule.Payments[loan.StartDate] = Payment{
		payment,
		loan.Principal * loan.InterestRate / (100.0 * 12.0),
		payment - loan.Principal*loan.InterestRate/(100.0*12.0),
		loan.Principal - (payment - loan.Principal*loan.InterestRate/(100.0*12.0)),
	}

	// Iterate over the remainder of the term.
	previousMonth := loan.StartDate
	currentMonth, err := IncrementDate(previousMonth, DateTimeLayout)
	if err != nil {
		return err
	}

	for month := 2; month <= loan.Term; month++ {
		var currentPayment Payment
		currentPayment.Payment = payment
		currentPayment.Interest = loan.AmortizationSchedule.Payments[previousMonth].RemainingPrincipal * loan.InterestRate / (100.0 * 12.0)
		currentPayment.Principal = payment - currentPayment.Interest
		if month == loan.Term {
			// We will get machine error otherwise so just set to 0
			currentPayment.RemainingPrincipal = 0.00
		} else {
			currentPayment.RemainingPrincipal = loan.AmortizationSchedule.Payments[previousMonth].RemainingPrincipal - currentPayment.Principal
		}
		loan.AmortizationSchedule.Payments[currentMonth] = currentPayment
		previousMonth = currentMonth
		currentMonth, err = IncrementDate(currentMonth, DateTimeLayout)
	}
	fmt.Println(loan.AmortizationSchedule)

	return nil
}
