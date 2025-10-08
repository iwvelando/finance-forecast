package adapters

import (
	"testing"
	"time"

	"github.com/iwvelando/finance-forecast/internal/config"
	"github.com/iwvelando/finance-forecast/pkg/datetime"
)

func TestConfigEventAdapter(t *testing.T) {
	date1 := datetime.MustParseTime(datetime.DateTimeLayout, "2025-01")
	date2 := datetime.MustParseTime(datetime.DateTimeLayout, "2025-02")

	event := config.Event{
		Name:     "Test Event",
		Amount:   100.0,
		DateList: []time.Time{date1, date2},
	}

	adapter := ConfigEventAdapter{Event: event}

	// Test GetName
	if adapter.GetName() != "Test Event" {
		t.Errorf("GetName() = %s, expected 'Test Event'", adapter.GetName())
	}

	// Test GetAmount
	if adapter.GetAmount() != 100.0 {
		t.Errorf("GetAmount() = %f, expected 100.0", adapter.GetAmount())
	}

	// Test GetDateList
	dateList := adapter.GetDateList()
	if len(dateList) != 2 {
		t.Errorf("GetDateList() length = %d, expected 2", len(dateList))
	}

	if !dateList[0].Equal(date1) {
		t.Errorf("GetDateList()[0] = %v, expected %v", dateList[0], date1)
	}

	if !dateList[1].Equal(date2) {
		t.Errorf("GetDateList()[1] = %v, expected %v", dateList[1], date2)
	}
}

func TestConfigEventAdapterWithEmptyDateList(t *testing.T) {
	event := config.Event{
		Name:     "Empty Event",
		Amount:   -50.0,
		DateList: []time.Time{},
	}

	adapter := ConfigEventAdapter{Event: event}

	if adapter.GetName() != "Empty Event" {
		t.Errorf("GetName() = %s, expected 'Empty Event'", adapter.GetName())
	}

	if adapter.GetAmount() != -50.0 {
		t.Errorf("GetAmount() = %f, expected -50.0", adapter.GetAmount())
	}

	dateList := adapter.GetDateList()
	if len(dateList) != 0 {
		t.Errorf("GetDateList() length = %d, expected 0", len(dateList))
	}
}

func TestEventsToFinanceEvents(t *testing.T) {
	date1 := datetime.MustParseTime(datetime.DateTimeLayout, "2025-01")
	date2 := datetime.MustParseTime(datetime.DateTimeLayout, "2025-02")

	events := []config.Event{
		{
			Name:     "Event 1",
			Amount:   100.0,
			DateList: []time.Time{date1},
		},
		{
			Name:     "Event 2",
			Amount:   -200.0,
			DateList: []time.Time{date1, date2},
		},
		{
			Name:     "Event 3",
			Amount:   0.0,
			DateList: []time.Time{},
		},
	}

	financeEvents := EventsToFinanceEvents(events)

	// Test correct number of events
	if len(financeEvents) != 3 {
		t.Errorf("EventsToFinanceEvents() length = %d, expected 3", len(financeEvents))
	}

	// Test first event
	if financeEvents[0].GetName() != "Event 1" {
		t.Errorf("financeEvents[0].GetName() = %s, expected 'Event 1'", financeEvents[0].GetName())
	}
	if financeEvents[0].GetAmount() != 100.0 {
		t.Errorf("financeEvents[0].GetAmount() = %f, expected 100.0", financeEvents[0].GetAmount())
	}
	if len(financeEvents[0].GetDateList()) != 1 {
		t.Errorf("financeEvents[0].GetDateList() length = %d, expected 1", len(financeEvents[0].GetDateList()))
	}

	// Test second event
	if financeEvents[1].GetName() != "Event 2" {
		t.Errorf("financeEvents[1].GetName() = %s, expected 'Event 2'", financeEvents[1].GetName())
	}
	if financeEvents[1].GetAmount() != -200.0 {
		t.Errorf("financeEvents[1].GetAmount() = %f, expected -200.0", financeEvents[1].GetAmount())
	}
	if len(financeEvents[1].GetDateList()) != 2 {
		t.Errorf("financeEvents[1].GetDateList() length = %d, expected 2", len(financeEvents[1].GetDateList()))
	}

	// Test third event (empty date list)
	if financeEvents[2].GetName() != "Event 3" {
		t.Errorf("financeEvents[2].GetName() = %s, expected 'Event 3'", financeEvents[2].GetName())
	}
	if len(financeEvents[2].GetDateList()) != 0 {
		t.Errorf("financeEvents[2].GetDateList() length = %d, expected 0", len(financeEvents[2].GetDateList()))
	}
}

func TestEventsToFinanceEventsEmpty(t *testing.T) {
	// Test with empty slice
	events := []config.Event{}
	financeEvents := EventsToFinanceEvents(events)

	if len(financeEvents) != 0 {
		t.Errorf("EventsToFinanceEvents() with empty input length = %d, expected 0", len(financeEvents))
	}

	// Test with nil slice
	financeEvents = EventsToFinanceEvents(nil)
	if len(financeEvents) != 0 {
		t.Errorf("EventsToFinanceEvents() with nil input length = %d, expected 0", len(financeEvents))
	}
}

func TestConfigLoanAdapter(t *testing.T) {
	loan := config.Loan{
		Name: "Test Loan",
		AmortizationSchedule: map[string]config.Payment{
			"2025-01": {Payment: 1500.0},
			"2025-02": {Payment: 1500.0},
			"2025-03": {Payment: 1400.0},
		},
	}

	adapter := ConfigLoanAdapter{Loan: loan}

	// Test GetName
	if adapter.GetName() != "Test Loan" {
		t.Errorf("GetName() = %s, expected 'Test Loan'", adapter.GetName())
	}

	// Test GetPaymentForDate - existing dates
	payment, exists := adapter.GetPaymentForDate("2025-01")
	if !exists {
		t.Errorf("GetPaymentForDate('2025-01') should exist")
	}
	if payment != 1500.0 {
		t.Errorf("GetPaymentForDate('2025-01') = %f, expected 1500.0", payment)
	}

	payment, exists = adapter.GetPaymentForDate("2025-03")
	if !exists {
		t.Errorf("GetPaymentForDate('2025-03') should exist")
	}
	if payment != 1400.0 {
		t.Errorf("GetPaymentForDate('2025-03') = %f, expected 1400.0", payment)
	}

	// Test GetPaymentForDate - non-existing date
	payment, exists = adapter.GetPaymentForDate("2025-12")
	if exists {
		t.Errorf("GetPaymentForDate('2025-12') should not exist")
	}
	if payment != 0.0 {
		t.Errorf("GetPaymentForDate('2025-12') payment = %f, expected 0.0", payment)
	}
}

func TestConfigLoanAdapterEmpty(t *testing.T) {
	loan := config.Loan{
		Name:                 "Empty Loan",
		AmortizationSchedule: map[string]config.Payment{},
	}

	adapter := ConfigLoanAdapter{Loan: loan}

	if adapter.GetName() != "Empty Loan" {
		t.Errorf("GetName() = %s, expected 'Empty Loan'", adapter.GetName())
	}

	// Test with empty schedule
	payment, exists := adapter.GetPaymentForDate("2025-01")
	if exists {
		t.Errorf("GetPaymentForDate() with empty schedule should not exist")
	}
	if payment != 0.0 {
		t.Errorf("GetPaymentForDate() with empty schedule payment = %f, expected 0.0", payment)
	}
}

func TestLoansToFinanceLoans(t *testing.T) {
	loans := []config.Loan{
		{
			Name: "Loan 1",
			AmortizationSchedule: map[string]config.Payment{
				"2025-01": {Payment: 1000.0},
			},
		},
		{
			Name: "Loan 2",
			AmortizationSchedule: map[string]config.Payment{
				"2025-01": {Payment: 500.0},
				"2025-02": {Payment: 500.0},
			},
		},
		{
			Name:                 "Loan 3",
			AmortizationSchedule: map[string]config.Payment{},
		},
	}

	financeLoans := LoansToFinanceLoans(loans)

	// Test correct number of loans
	if len(financeLoans) != 3 {
		t.Errorf("LoansToFinanceLoans() length = %d, expected 3", len(financeLoans))
	}

	// Test first loan
	if financeLoans[0].GetName() != "Loan 1" {
		t.Errorf("financeLoans[0].GetName() = %s, expected 'Loan 1'", financeLoans[0].GetName())
	}
	payment, exists := financeLoans[0].GetPaymentForDate("2025-01")
	if !exists || payment != 1000.0 {
		t.Errorf("financeLoans[0].GetPaymentForDate('2025-01') = (%f, %t), expected (1000.0, true)", payment, exists)
	}

	// Test second loan
	if financeLoans[1].GetName() != "Loan 2" {
		t.Errorf("financeLoans[1].GetName() = %s, expected 'Loan 2'", financeLoans[1].GetName())
	}
	payment, exists = financeLoans[1].GetPaymentForDate("2025-02")
	if !exists || payment != 500.0 {
		t.Errorf("financeLoans[1].GetPaymentForDate('2025-02') = (%f, %t), expected (500.0, true)", payment, exists)
	}

	// Test third loan (empty schedule)
	if financeLoans[2].GetName() != "Loan 3" {
		t.Errorf("financeLoans[2].GetName() = %s, expected 'Loan 3'", financeLoans[2].GetName())
	}
	payment, exists = financeLoans[2].GetPaymentForDate("2025-01")
	if exists || payment != 0.0 {
		t.Errorf("financeLoans[2].GetPaymentForDate('2025-01') = (%f, %t), expected (0.0, false)", payment, exists)
	}
}

func TestLoansToFinanceLoansEmpty(t *testing.T) {
	// Test with empty slice
	loans := []config.Loan{}
	financeLoans := LoansToFinanceLoans(loans)

	if len(financeLoans) != 0 {
		t.Errorf("LoansToFinanceLoans() with empty input length = %d, expected 0", len(financeLoans))
	}

	// Test with nil slice
	financeLoans = LoansToFinanceLoans(nil)
	if len(financeLoans) != 0 {
		t.Errorf("LoansToFinanceLoans() with nil input length = %d, expected 0", len(financeLoans))
	}
}

func TestInvestmentsToFinanceInvestments(t *testing.T) {
	contributionDate := datetime.MustParseTime(datetime.DateTimeLayout, "2025-07")
	withdrawalDate := datetime.MustParseTime(datetime.DateTimeLayout, "2025-12")

	investment := config.Investment{
		Name:                  "Retirement",
		StartingValue:         5000.0,
		AnnualReturnRate:      6.0,
		TaxRate:               24.0,
		WithdrawalTaxRate:     15.0,
		ContributionsFromCash: true,
		Contributions: []config.Event{
			{
				Name:     "Monthly Contribution",
				Amount:   500,
				DateList: []time.Time{contributionDate},
			},
		},
		Withdrawals: []config.Event{
			{
				Name:     "Year-end Withdrawal",
				Amount:   2000,
				DateList: []time.Time{withdrawalDate},
			},
		},
	}

	financeInvestments := InvestmentsToFinanceInvestments([]config.Investment{investment})
	if len(financeInvestments) != 1 {
		t.Fatalf("expected 1 finance investment, got %d", len(financeInvestments))
	}

	fi := financeInvestments[0]
	if fi.GetName() != "Retirement" {
		t.Errorf("GetName() = %s, want 'Retirement'", fi.GetName())
	}

	if fi.GetStartingValue() != 5000 {
		t.Errorf("GetStartingValue() = %.2f, want 5000", fi.GetStartingValue())
	}

	if fi.GetAnnualReturnRate() != 6.0 {
		t.Errorf("GetAnnualReturnRate() = %.2f, want 6.0", fi.GetAnnualReturnRate())
	}

	if fi.GetTaxRate() != 24.0 {
		t.Errorf("GetTaxRate() = %.2f, want 24.0", fi.GetTaxRate())
	}

	if fi.GetWithdrawalTaxRate() != 15.0 {
		t.Errorf("GetWithdrawalTaxRate() = %.2f, want 15.0", fi.GetWithdrawalTaxRate())
	}

	if fi.GetContributionForDate("2025-07") != 500 {
		t.Errorf("GetContributionForDate(2025-07) = %.2f, want 500", fi.GetContributionForDate("2025-07"))
	}

	if fi.GetContributionForDate("2025-08") != 0 {
		t.Errorf("GetContributionForDate(2025-08) should be 0")
	}

	if fi.GetWithdrawalForDate("2025-12") != 2000 {
		t.Errorf("GetWithdrawalForDate(2025-12) = %.2f, want 2000", fi.GetWithdrawalForDate("2025-12"))
	}

	if fi.GetWithdrawalForDate("2025-11") != 0 {
		t.Errorf("GetWithdrawalForDate(2025-11) should be 0")
	}

	if fi.GetWithdrawalPercentageForDate("2025-12") != 0 {
		t.Errorf("GetWithdrawalPercentageForDate(2025-12) = %.2f, want 0", fi.GetWithdrawalPercentageForDate("2025-12"))
	}

	if !fi.ContributionsFromCash() {
		t.Errorf("ContributionsFromCash() = false, want true")
	}
}

func TestInvestmentsToFinanceInvestments_PercentageWithdrawals(t *testing.T) {
	withdrawalDate, _ := time.Parse(time.RFC3339, "2025-06-01T00:00:00Z")

	investment := config.Investment{
		Name:             "Growth Fund",
		StartingValue:    10000,
		AnnualReturnRate: 7.5,
		Withdrawals: []config.Event{
			{
				Name:       "Annual Withdrawal",
				Percentage: 4.0,
				DateList:   []time.Time{withdrawalDate},
			},
		},
	}

	financeInvestments := InvestmentsToFinanceInvestments([]config.Investment{investment})
	if len(financeInvestments) != 1 {
		t.Fatalf("expected 1 finance investment, got %d", len(financeInvestments))
	}

	fi := financeInvestments[0]

	if fi.GetWithdrawalForDate("2025-06") != 0 {
		t.Fatalf("expected amount withdrawal to be 0 when using percentage")
	}

	if fi.GetWithdrawalPercentageForDate("2025-06") != 4.0 {
		t.Fatalf("expected withdrawal percentage to be 4.0, got %.2f", fi.GetWithdrawalPercentageForDate("2025-06"))
	}
}

func TestInvestmentsToFinanceInvestmentsNil(t *testing.T) {
	if len(InvestmentsToFinanceInvestments(nil)) != 0 {
		t.Fatalf("expected nil investments to return empty slice")
	}

	if len(InvestmentsToFinanceInvestments([]config.Investment{})) != 0 {
		t.Fatalf("expected empty investments slice to return empty result")
	}
}
