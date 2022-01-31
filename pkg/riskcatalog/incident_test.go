package riskcatalog_test

import (
	"github.com/stretchr/testify/require"
	"switcher/pkg/riskcatalog"
	"testing"
)

func TestConfig(t *testing.T) {
	c, err := riskcatalog.NewRiskCatalogFromFile("./fixtures/incidents.yml")
	require.NoError(t, err)
	require.NotNil(t, c)
	expected := riskcatalog.RiskCatalog{
		Incidents: []riskcatalog.Incident{
			{
				Description:      "A configuration mishap reduces capacity; causing overload and dropped requests",
				EttdMinutes:      30,
				EttrMinutes:      120,
				EttfDays:         120,
				PercentageImpact: 0.2,
			},
			{
				Description:      "A new release breaks a small set of requests; not detected for a day; quick rollback when detected.",
				EttdMinutes:      1440,
				EttrMinutes:      30,
				EttfDays:         90,
				PercentageImpact: 0.02,
			},
			{
				Description:      "A new release breaks a sizeable subset of requests; unfamiliar rollback procedure extends outage",
				EttdMinutes:      5,
				EttrMinutes:      120,
				EttfDays:         180,
				PercentageImpact: 0.5,
			},
		},
		Risks: []riskcatalog.RiskFactor{
			{
				Description:                   "ETTD++ per riskcatalog (e.g., +30m due to operational overload)",
				EttdPenaltyMinutes:            30,
				EttrPenaltyMinutes:            0,
				EttfPenaltyPercentageIncrease: 0.0,
			},
			{
				Description:                   "ETTR++ per riskcatalog (e.g., +5m due to lack of playbooks)",
				EttdPenaltyMinutes:            0,
				EttrPenaltyMinutes:            5,
				EttfPenaltyPercentageIncrease: 0.0,
			},
			{
				Description:                   "ETTF increase per risk (e.g, all risks +10% more frequent due to lack of postmortems AI follow-up)",
				EttdPenaltyMinutes:            0,
				EttrPenaltyMinutes:            0,
				EttfPenaltyPercentageIncrease: 0.1,
			},
		},
		ErrorBudget: riskcatalog.ErrorBudget{
			AvailabilityTarget:                   99.50,
			AcceptableThresholdPercentagePerRisk: 0.25,
		},
	}
	require.Equal(t, expected, c)
}
