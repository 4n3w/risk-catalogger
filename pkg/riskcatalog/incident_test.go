package riskcatalog_test

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	ui "github.com/gizak/termui/v3"
	"github.com/stretchr/testify/require"
	"github.com/vmware-tanzu-labs/risk-catalogger/pkg/riskcatalog"
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

// The incidents and risk factors in fixtures/floats.yml, worked out by hand.
const (
	restartPerYear = 365.25 / 7.5               // 48.7
	restartBM      = 0.3333333 * restartPerYear // 20 seconds a time: 16.23
	nightlyBM      = (10 + 20.5) * 0.25 * 10    // 76.25
	diskBM         = 60 + 120                   // once a year, everyone
	sumOfProducts  = restartPerYear + 0.25*10 + 1
	slowerBM       = 0.5 * sumOfProducts                    // ETTD +0.5 minutes
	moreOftenBM    = 0.1 * (restartBM + nightlyBM + diskBM) // ETTF 10% shorter
	floatsTotalBM  = restartBM + nightlyBM + diskBM + slowerBM + moreOftenBM
	budget999      = (1 - 0.999) * 1440 * 365.25 // 525.96 minutes a year
)

func loadFloats(t *testing.T) riskcatalog.RiskCatalog {
	t.Helper()
	rc, err := riskcatalog.NewRiskCatalogFromFile("./fixtures/floats.yml")
	require.NoError(t, err)
	return rc
}

func TestFloatValues(t *testing.T) {
	rc := loadFloats(t)
	require.Len(t, rc.Incidents, 3)
	require.Len(t, rc.Risks, 2)

	restart := rc.Incidents[0]
	require.Equal(t, float32(0), restart.EttdMinutes)
	require.Equal(t, float32(0.3333333), restart.EttrMinutes)
	require.Equal(t, float32(7.5), restart.EttfDays)
	require.InDelta(t, restartPerYear, restart.IncidentsPerYear(), 0.0001)
	require.InDelta(t, restartBM, restart.BadMinutesPerYear(), 0.001)

	nightly := rc.Incidents[1]
	require.Equal(t, float32(20.5), nightly.EttrMinutes)
	require.Equal(t, float32(36.525), nightly.EttfDays)
	require.InDelta(t, nightlyBM, nightly.BadMinutesPerYear(), 0.001)

	// Whole numbers in the same file still load.
	disk := rc.Incidents[2]
	require.Equal(t, float32(60), disk.EttdMinutes)
	require.Equal(t, float32(120), disk.EttrMinutes)
	require.InDelta(t, diskBM, disk.BadMinutesPerYear(), 0.001)

	require.Equal(t, float32(0.5), rc.Risks[0].EttdPenaltyMinutes)
	require.Equal(t, float32(0), rc.Risks[0].EttrPenaltyMinutes)
	require.Equal(t, float32(0.1), rc.Risks[1].EttfPenaltyPercentageIncrease)

	ranked := rc.ComputeRisk()
	require.Len(t, ranked, 5)
	require.InDelta(t, slowerBM, rc.Risks[0].BadMinutes, 0.001)
	require.InDelta(t, moreOftenBM, rc.Risks[1].BadMinutes, 0.001)
}

func TestTableCellsFormatFloats(t *testing.T) {
	rc := loadFloats(t)
	rc.Incidents[0].EttrMinutes = float32(1) / 3 // 0.33333334 as a float32
	rc.ComputeRisk()

	table := riskcatalog.BuildRiskCatalogTable(rc)
	require.Equal(t, []string{"A restart drops every request for 20 seconds, weekly and a bit", "0", "0.33", "100%", "7.5", "48.7", "16.2"}, table.Rows[1])
	require.Equal(t, []string{"A nightly job overloads the service | a quarter of requests fail", "10", "20.5", "25%", "36.53", "10", "76.2"}, table.Rows[2])
	require.Equal(t, []string{"The disk fills up and the service stops", "60", "120", "100%", "365.25", "1", "180"}, table.Rows[3])

	factors := riskcatalog.BuildRiskFactorTable(rc)
	require.Equal(t, []string{"Detection takes half a minute longer", "0.5", "0", "0%", "26.1"}, factors.Rows[1])
	require.Equal(t, []string{"Every incident happens 10% more often", "0", "0", "10%", "27.2"}, factors.Rows[2])
}

func TestVerdictThresholds(t *testing.T) {
	eb := riskcatalog.ErrorBudget{AvailabilityTarget: 99.5, AcceptableThresholdPercentagePerRisk: 0.25}
	threshold := eb.TooBigThreshold()
	require.InDelta(t, 657.45, threshold, 0.01)

	for _, c := range []struct {
		badMinutes float32
		want       riskcatalog.Verdict
	}{
		{0, riskcatalog.Fits},
		{threshold / 2, riskcatalog.Fits},
		{threshold/2 + 0.01, riskcatalog.Cautionary},
		{threshold, riskcatalog.Cautionary},
		{threshold + 0.01, riskcatalog.TooBig},
		{10 * threshold, riskcatalog.TooBig},
	} {
		require.Equal(t, c.want, eb.VerdictFor(c.badMinutes), "%v bad minutes a year", c.badMinutes)
	}

	require.Equal(t, "fits", riskcatalog.Fits.String())
	require.Equal(t, "cautionary", riskcatalog.Cautionary.String())
	require.Equal(t, "too big", riskcatalog.TooBig.String())
}

func TestBudgetNumbers(t *testing.T) {
	eb := riskcatalog.ErrorBudget{AvailabilityTarget: 99.5, AcceptableThresholdPercentagePerRisk: 0.25}
	require.InDelta(t, 2629.8, eb.MinutesPerYear(), 0.01)
	require.InDelta(t, 201.6, eb.MinutesPerFourWeekIteration(), 0.01)

	rc := loadFloats(t)
	require.InDelta(t, budget999, rc.ErrorBudget.MinutesPerYear(), 0.01)
	require.InDelta(t, 40.32, rc.ErrorBudget.MinutesPerFourWeekIteration(), 0.01)
	require.InDelta(t, budget999*0.25, rc.TooBigThreshold(), 0.01)
	require.InDelta(t, floatsTotalBM, rc.TotalBadMinutesPerYear(), 0.001)
	require.InDelta(t, budget999-floatsTotalBM, rc.RemainingBudget(), 0.01)

	require.Equal(t, []string{
		"Availability target: 99.9%",
		"Error budget: 526 minutes per year, 40.3 minutes per 4 weeks",
		"Single-risk threshold: 131 minutes per year (25% of the budget)",
		"Bad minutes of all risks (3 incidents, 2 risk factors): 326 minutes per year (62% of the budget)",
		"Left: 200 minutes per year (38% of the budget)",
	}, rc.BudgetLines())
	require.Equal(t, strings.Join(rc.BudgetLines(), "\n"), riskcatalog.BuildErrorBudgetParagraph(rc).Text)

	// At 99.95% the budget is 262.98 minutes a year, less than the catalog's 325.83.
	rc.ErrorBudget.AvailabilityTarget = 99.95
	require.InDelta(t, 262.98-floatsTotalBM, rc.RemainingBudget(), 0.02)
	require.Equal(t, "Left: none; over budget by 62.8 minutes per year", rc.BudgetLines()[4])
}

func TestBudgetUsageAt(t *testing.T) {
	rc := loadFloats(t)

	// At 99.99% (52.6 minutes a year, a threshold of 13.15) every risk is over the threshold.
	u := rc.BudgetUsageAt(99.99)
	require.InDelta(t, 52.596, u.BudgetMinutes, 0.01)
	require.Equal(t, float32(0), u.UnderPercent)
	require.InDelta(t, floatsTotalBM/52.596*100, u.OverPercent, 2)

	// At 99.95% (262.98, a threshold of 65.75) the nightly job and the disk are over it.
	u = rc.BudgetUsageAt(99.95)
	require.InDelta(t, (restartBM+slowerBM+moreOftenBM)/262.98*100, u.UnderPercent, 0.02)
	require.InDelta(t, (nightlyBM+diskBM)/262.98*100, u.OverPercent, 0.02)

	// At 99.9% (525.96, a threshold of 131.49) only the disk is.
	u = rc.BudgetUsageAt(99.9)
	require.InDelta(t, (restartBM+nightlyBM+slowerBM+moreOftenBM)/budget999*100, u.UnderPercent, 0.01)
	require.InDelta(t, diskBM/budget999*100, u.OverPercent, 0.01)

	// At 99.5% (2629.8, a threshold of 657.45) none is.
	u = rc.BudgetUsageAt(99.5)
	require.InDelta(t, floatsTotalBM/2629.8*100, u.UnderPercent, 0.01)
	require.Equal(t, float32(0), u.OverPercent)
}

func TestStackedBarChart(t *testing.T) {
	rc := loadFloats(t)
	sbc := riskcatalog.BuildStackedBarChart(rc)
	require.Equal(t, []string{"99.99", "99.95", "99.9", "99.5"}, sbc.Labels)
	require.Equal(t, 100.0, sbc.MaxVal)
	require.Len(t, sbc.Data, 4)

	// Over 100%: drawn full, keeping the proportion of under to over.
	require.InDeltaSlice(t, []float64{0, 100}, sbc.Data[0], 1e-6)
	under, over := restartBM+slowerBM+moreOftenBM, float64(nightlyBM+diskBM)
	require.InDeltaSlice(t, []float64{under / (under + over) * 100, over / (under + over) * 100}, sbc.Data[1], 0.01)
	// Under 100%: as is.
	require.InDeltaSlice(t, []float64{(floatsTotalBM - diskBM) / budget999 * 100, diskBM / budget999 * 100}, sbc.Data[2], 0.01)
	require.InDeltaSlice(t, []float64{floatsTotalBM / 2629.8 * 100, 0}, sbc.Data[3], 0.01)

	usage := riskcatalog.BuildBudgetUsageTable(rc)
	require.Equal(t, []string{"99.99", "0%", "619%", "619%"}, usage.Rows[1])
	require.Equal(t, ui.ColorRed, usage.RowStyles[1].Fg)
	require.Equal(t, []string{"99.9", "27.7%", "34.2%", "62%"}, usage.Rows[3])
	_, coloured := usage.RowStyles[3]
	require.False(t, coloured)
}

func TestStackedRisksTableColoursRowsByVerdict(t *testing.T) {
	rc := loadFloats(t)
	table := riskcatalog.BuildStackedRisksTable(rc, rc.ComputeRisk())
	require.Equal(t, []string{"Description", "BM/Y", "% budget", "Verdict"}, table.Rows[0])

	want := []struct {
		description, verdict string
		colour               ui.Color
	}{
		{"The disk fills up and the service stops", "too big", ui.ColorRed},
		{"A nightly job overloads the service | a quarter of requests fail", "cautionary", ui.ColorYellow},
		{"Every incident happens 10% more often", "fits", ui.ColorGreen},
		{"Detection takes half a minute longer", "fits", ui.ColorGreen},
		{"A restart drops every request for 20 seconds, weekly and a bit", "fits", ui.ColorGreen},
	}
	require.Len(t, table.Rows, len(want)+1)
	for i, w := range want {
		require.Equal(t, w.description, table.Rows[i+1][0])
		require.Equal(t, w.verdict, table.Rows[i+1][3])
		require.Equal(t, w.colour, table.RowStyles[i+1].Fg, w.description)
	}

	legend := riskcatalog.BuildParagraph(rc).Text
	require.Contains(t, legend, "[Green](fg:green) fits")
	require.Contains(t, legend, "[Yellow](fg:yellow) is cautionary")
	require.Contains(t, legend, "[Red](fg:red) is too big")
	require.NotContains(t, legend, "Blue")
}

func TestRiskCatalogListHasNoBlankRows(t *testing.T) {
	rc := loadFloats(t)
	require.Equal(t, []string{
		"A restart drops every request for 20 seconds, weekly and a bit",
		"A nightly job overloads the service | a quarter of requests fail",
		"The disk fills up and the service stops",
	}, riskcatalog.BuildRiskCatalogList(rc).Rows)
}

func TestMarkdown(t *testing.T) {
	rc := loadFloats(t)
	require.Equal(t, `## Error budget

- Availability target: 99.9%
- Error budget: 526 minutes per year, 40.3 minutes per 4 weeks
- Single-risk threshold: 131 minutes per year (25% of the budget)
- Bad minutes of all risks (3 incidents, 2 risk factors): 326 minutes per year (62% of the budget)
- Left: 200 minutes per year (38% of the budget)

## Stack rank

| Description | ETTD (min) | ETTR (min) | ETTF (days) | Impact | Incidents/yr | Bad min/yr | % of budget | Verdict |
|---|--:|--:|--:|--:|--:|--:|--:|---|
| The disk fills up and the service stops | 60 | 120 | 365.25 | 100% | 1 | 180 | 34.2% | too big |
| A nightly job overloads the service \| a quarter of requests fail | 10 | 20.5 | 36.53 | 25% | 10 | 76.2 | 14.5% | cautionary |
| Every incident happens 10% more often | — | — | — | — | — | 27.2 | 5.18% | fits |
| Detection takes half a minute longer | — | — | — | — | — | 26.1 | 4.96% | fits |
| A restart drops every request for 20 seconds, weekly and a bit | 0 | 0.33 | 7.5 | 100% | 48.7 | 16.2 | 3.09% | fits |

## Risk factors

| Description | ETTD+ (min) | ETTR+ (min) | ETTF+ | Bad min/yr | % of budget | Verdict |
|---|--:|--:|--:|--:|--:|---|
| Detection takes half a minute longer | 0.5 | 0 | 0% | 26.1 | 4.96% | fits |
| Every incident happens 10% more often | 0 | 0 | 10% | 27.2 | 5.18% | fits |
`, riskcatalog.Markdown(rc))

	empty := riskcatalog.RiskCatalog{ErrorBudget: rc.ErrorBudget}
	md := riskcatalog.Markdown(empty)
	require.Contains(t, md, "- Left: 526 minutes per year (100% of the budget)\n")
	require.Contains(t, md, "## Stack rank\n\nNo incidents or risk factors.\n")
	require.Contains(t, md, "## Risk factors\n\nNo risk factors.\n")
}

func TestValidate(t *testing.T) {
	dir := t.TempDir()
	budget := "error-budget:\n  availability-target: 99.9\n  single-risk-acceptable-threshold: 0.25\n"
	incident := func(fields string) string {
		return budget + "incidents:\n  - {description: x, estimated-time-to-detect: 1, estimated-time-to-repair: 1, " + fields + "}\n"
	}
	for i, c := range []struct{ yaml, want string }{
		{incident("estimated-time-to-fail: 0, percentage-impact: 1"), "estimated-time-to-fail must be more than 0 days"},
		{incident("estimated-time-to-fail: 30, percentage-impact: 20"), "percentage-impact must be a fraction from 0 to 1"},
		{"error-budget:\n  availability-target: 100\n  single-risk-acceptable-threshold: 0.25\n", "availability-target must be"},
		{"error-budget:\n  availability-target: 99.9\n", "single-risk-acceptable-threshold must be"},
	} {
		path := filepath.Join(dir, fmt.Sprintf("catalog-%d.yml", i))
		require.NoError(t, os.WriteFile(path, []byte(c.yaml), 0o600))
		_, err := riskcatalog.NewRiskCatalogFromFile(path)
		require.Error(t, err, c.yaml)
		require.Contains(t, err.Error(), c.want)
	}

	path := filepath.Join(dir, "good.yml")
	require.NoError(t, os.WriteFile(path, []byte(incident("estimated-time-to-fail: 0.5, percentage-impact: 0.5")), 0o600))
	_, err := riskcatalog.NewRiskCatalogFromFile(path)
	require.NoError(t, err)
}
