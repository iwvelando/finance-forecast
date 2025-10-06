package config

import (
	"testing"
	"time"
)

func TestInvestmentFormDateListsWithFixedTime(t *testing.T) {
	conf := Configuration{
		Common: Common{DeathDate: "2025-12"},
	}

	investment := Investment{
		Name: "Growth Fund",
		Contributions: []Event{
			{
				Name:      "Monthly Contribution",
				Amount:    200,
				Frequency: 1,
				StartDate: "2025-07",
				EndDate:   "2025-09",
			},
		},
		Withdrawals: []Event{
			{
				Name:      "Quarterly Withdrawal",
				Amount:    500,
				Frequency: 3,
				StartDate: "2025-08",
				EndDate:   "2025-12",
			},
		},
	}

	fixedTime := time.Date(2025, 6, 1, 0, 0, 0, 0, time.UTC)

	err := investment.FormDateListsWithFixedTime(conf, fixedTime)
	if err != nil {
		t.Fatalf("FormDateListsWithFixedTime returned error: %v", err)
	}

	if len(investment.Contributions[0].DateList) != 3 {
		t.Fatalf("expected 3 contribution dates, got %d", len(investment.Contributions[0].DateList))
	}

	expectedContributionMonths := []string{"2025-07", "2025-08", "2025-09"}
	for i, d := range investment.Contributions[0].DateList {
		if d.Format(DateTimeLayout) != expectedContributionMonths[i] {
			t.Errorf("contribution date %d = %s, want %s", i, d.Format(DateTimeLayout), expectedContributionMonths[i])
		}
	}

	if len(investment.Withdrawals[0].DateList) != 2 {
		t.Fatalf("expected 2 withdrawal dates, got %d", len(investment.Withdrawals[0].DateList))
	}

	expectedWithdrawalMonths := []string{"2025-08", "2025-11"}
	for i, d := range investment.Withdrawals[0].DateList {
		if d.Format(DateTimeLayout) != expectedWithdrawalMonths[i] {
			t.Errorf("withdrawal date %d = %s, want %s", i, d.Format(DateTimeLayout), expectedWithdrawalMonths[i])
		}
	}
}

func TestParseDateListsWithInvestments(t *testing.T) {
	conf := &Configuration{
		Common: Common{
			DeathDate: "2026-01",
			Investments: []Investment{
				{
					Name: "Common Investment",
					Contributions: []Event{{
						Amount:    100,
						Frequency: 1,
					}},
				},
			},
		},
		Scenarios: []Scenario{
			{
				Name:   "Scenario A",
				Active: true,
				Investments: []Investment{
					{
						Name: "Scenario Investment",
						Contributions: []Event{{
							Amount:    150,
							Frequency: 2,
						}},
					},
				},
			},
		},
	}

	fixedTime := time.Date(2025, 7, 1, 0, 0, 0, 0, time.UTC)

	err := conf.ParseDateListsWithFixedTime(fixedTime)
	if err != nil {
		t.Fatalf("ParseDateListsWithFixedTime returned error: %v", err)
	}

	if len(conf.Common.Investments[0].Contributions[0].DateList) == 0 {
		t.Fatalf("expected contribution dates for common investment")
	}

	if len(conf.Scenarios[0].Investments[0].Contributions[0].DateList) == 0 {
		t.Fatalf("expected contribution dates for scenario investment")
	}
}
