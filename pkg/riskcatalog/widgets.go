package riskcatalog

import (
	"fmt"
	"strings"

	ui "github.com/gizak/termui/v3"
	"github.com/gizak/termui/v3/widgets"
)

// VerdictColors are the colours of the Risk Stack Rank tab's rows.
var VerdictColors = map[Verdict]ui.Color{
	Fits:       ui.ColorGreen,
	Cautionary: ui.ColorYellow,
	TooBig:     ui.ColorRed,
}

func BuildTabs() *widgets.TabPane {
	tabpane := widgets.NewTabPane("Risk Catalog", "Risk Factors", "Risk Stack Rank", "Risk List", "Error Budget")
	tabpane.SetRect(0, 1, 80, 4)
	tabpane.Border = true
	return tabpane
}

// BuildStackedRisksTable lists the ranked risks, each row coloured by its verdict.
func BuildStackedRisksTable(rc RiskCatalog, rankedRisks []HasRisk) *widgets.Table {
	arr := [][]string{{"Description", "BM/Y", "% budget", "Verdict"}}
	t := widgets.NewTable()
	t.RowStyles = map[int]ui.Style{
		0: ui.NewStyle(ui.ColorBlue),
	}
	for i, r := range rankedRisks {
		bm := r.BadMinutesPerYear()
		v := rc.VerdictFor(bm)
		arr = append(arr, []string{
			r.RiskDescription(),
			formatAmount(bm),
			formatPercent(rc.PercentOfBudget(bm)),
			v.String(),
		})
		t.RowStyles[i+1] = ui.NewStyle(VerdictColors[v])
	}

	t.Rows = arr
	t.RowSeparator = false
	t.SetRect(5, 5, 85, 30)
	t.ColumnWidths = []int{45, 8, 9, 11}
	t.Title = "Risks ranked by bad minutes per year"
	t.TextAlignment = ui.AlignLeft

	return t
}

func BuildRiskCatalogList(rc RiskCatalog) *widgets.List {
	l := widgets.NewList()
	arr := make([]string, 0, len(rc.Incidents))

	for _, r := range rc.Incidents {
		arr = append(arr, r.RiskDescription())
	}
	l.Rows = arr
	l.SetRect(5, 5, 85, 30)

	//l.TextStyle.Bg = ui.ColorBlue
	//header.TextStyle.Bg = ui.ColorBlue

	l.Title = "List of Risks (Incidents, Chaos Experiments, etc)"
	//l.TextAlignment = ui.AlignLeft
	return l
}

func BuildRiskCatalogTable(rc RiskCatalog) *widgets.Table {
	arr := make([][]string, len(rc.Incidents)+1)
	arr[0] = []string{
		"Description", "ETTD", "ETTR", "% Users", "ETTF", "IP/Y", "BM/Y",
	}

	t := widgets.NewTable()
	t.RowStyles = map[int]ui.Style{
		0: ui.NewStyle(ui.ColorBlue),
	}

	for i, r := range rc.Incidents {
		arr[i+1] = []string{
			r.RiskDescription(),
			formatInput(r.EttdMinutes),
			formatInput(r.EttrMinutes),
			formatFraction(r.PercentageImpact),
			formatInput(r.EttfDays),
			formatAmount(r.IncidentsPerYear()),
			formatAmount(r.BadMinutesPerYear()),
		}
	}

	t.Rows = arr
	t.SetRect(5, 5, 120, 30)
	t.ColumnWidths = []int{57, 8, 8, 8, 8, 8, 8}
	t.Title = "Risks as Incidents"
	t.TextAlignment = ui.AlignCenter

	return t
}

func BuildRiskFactorTable(rc RiskCatalog) *widgets.Table {
	arr := make([][]string, len(rc.Risks)+1)

	arr[0] = []string{
		"Description", "ETTD+", "ETTR+", "ETTF+", "BM/Y",
	}
	t := widgets.NewTable()
	t.RowStyles = map[int]ui.Style{
		0: ui.NewStyle(ui.ColorBlue),
	}

	for i, r := range rc.Risks {
		arr[i+1] = []string{
			r.Description,
			formatInput(r.EttdPenaltyMinutes),
			formatInput(r.EttrPenaltyMinutes),
			formatFraction(r.EttfPenaltyPercentageIncrease),
			formatAmount(r.BadMinutesPerYear()),
		}
	}
	t.Rows = arr
	t.SetRect(5, 5, 120, 30)
	t.ColumnWidths = []int{67, 10, 10, 10, 10}

	t.Title = "Risk Factors"
	t.TextAlignment = ui.AlignCenter

	return t
}

// cappedUsage returns a target's usage as the two segments of its bar. A bar over 100% is drawn full,
// keeping the proportion between risks under and over the single-risk threshold.
func cappedUsage(u BudgetUsage) (under, over float64) {
	under, over = float64(u.UnderPercent), float64(u.OverPercent)
	if total := under + over; total > 100 {
		under, over = under/total*100, over/total*100
	}
	return under, over
}

// BuildStackedBarChart shows, for each of StackRankTargets, how much of that target's yearly budget the
// catalog uses: green for risks under that target's single-risk threshold, red for risks over it.
func BuildStackedBarChart(rc RiskCatalog) *widgets.StackedBarChart {
	sbc := widgets.NewStackedBarChart()

	sbc.Title = "% of budget"
	for _, target := range StackRankTargets {
		under, over := cappedUsage(rc.BudgetUsageAt(target))
		sbc.Labels = append(sbc.Labels, formatTarget(target))
		sbc.Data = append(sbc.Data, []float64{under, over})
	}
	sbc.MaxVal = 100
	sbc.BarColors = []ui.Color{ui.ColorGreen, ui.ColorRed}
	sbc.LabelStyles = []ui.Style{ui.NewStyle(ui.ColorClear)}
	// The bars can be scaled down to fit; the figures are in BuildBudgetUsageTable.
	sbc.NumFormatter = func(float64) string { return "" }
	sbc.SetRect(85, 5, 120, 23)
	sbc.BarWidth = 7
	sbc.BarGap = 1

	return sbc
}

// BuildBudgetUsageTable has the figures behind BuildStackedBarChart's bars, uncapped; a row over 100% is red.
func BuildBudgetUsageTable(rc RiskCatalog) *widgets.Table {
	arr := [][]string{{"Target", "Under", "Over", "Total"}}
	t := widgets.NewTable()
	t.RowStyles = map[int]ui.Style{
		0: ui.NewStyle(ui.ColorBlue),
	}
	for i, target := range StackRankTargets {
		u := rc.BudgetUsageAt(target)
		arr = append(arr, []string{
			formatTarget(target),
			formatPercent(u.UnderPercent),
			formatPercent(u.OverPercent),
			formatPercent(u.TotalPercent()),
		})
		if u.TotalPercent() > 100 {
			t.RowStyles[i+1] = ui.NewStyle(ui.ColorRed)
		}
	}

	t.Rows = arr
	t.RowSeparator = false
	t.SetRect(85, 23, 120, 30)
	t.ColumnWidths = []int{6, 7, 7, 8}
	t.TextAlignment = ui.AlignRight

	return t
}

// BuildParagraph is the Risk Stack Rank tab's legend.
func BuildParagraph(rc RiskCatalog) *widgets.Paragraph {
	p := widgets.NewParagraph()
	p.Text = fmt.Sprintf("[Green](fg:green) fits: at most half the single-risk threshold of %s minutes per year.\n",
		formatAmount(rc.TooBigThreshold()))
	p.Text += "[Yellow](fg:yellow) is cautionary: over half the single-risk threshold.\n"
	p.Text += "[Red](fg:red) is too big: over the single-risk threshold.\n\n"
	p.Text += "Bars: how much of each availability target's yearly budget the catalog uses, from risks under " +
		"[green](fg:green) and over [red](fg:red) that target's single-risk threshold. " +
		"A bar over 100% is drawn full, in proportion; the table under it has the figures."
	p.SetRect(5, 30, 120, 40)
	return p
}

// BuildErrorBudgetParagraph is the Error Budget tab.
func BuildErrorBudgetParagraph(rc RiskCatalog) *widgets.Paragraph {
	p := widgets.NewParagraph()
	p.Title = "Error Budget"
	p.Text = strings.Join(rc.BudgetLines(), "\n")
	if rc.RemainingBudget() < 0 {
		p.TextStyle = ui.NewStyle(ui.ColorRed)
	}
	p.SetRect(5, 5, 120, 13)
	return p
}

func CatalogParagraph() *widgets.Paragraph {
	p := widgets.NewParagraph()
	p.Text = "ETTD: Estimated Time To Detect: (minutes) How long it takes someone to notice the service is broken.\n"
	p.Text += "ETTR: Estimated Time To Repair: (minutes) How long it takes someone to repair the service.\n"
	p.Text += "% Users: Estimate of how many users would be affected by this incident.\n"
	p.Text += "ETTF: Estimated Time To Fail: (days) Time between instances of this incident occurring.\n"
	p.Text += "IP/Y: Incidents per year (estimate).\n"
	p.Text += "BM/Y: Bad minutes per year. Calculated estimate of the operational cost of this occurring.\n"
	p.TextStyle = ui.NewStyle(ui.ColorGreen)
	p.SetRect(5, 40, 120, 30)
	return p
}

func RiskFactorParagraph() *widgets.Paragraph {
	p := widgets.NewParagraph()
	p.Text = "ETTD: Estimated Time To Detect: (minutes) How long it takes someone to notice the service is broken.\n"
	p.Text += "ETTR: Estimated Time To Repair: (minutes) How long it takes someone to repair the service.\n"
	p.Text += "ETTF: Estimated Time To Fail: (days) Time between instances of this incident occurring.\n"
	p.Text += "IP/Y: Incidents per year (estimate).\n"
	p.Text += "BM/Y: Bad minutes per year. Calculated estimate of the operational cost of this occurring.\n"
	p.TextStyle = ui.NewStyle(ui.ColorGreen)
	p.SetRect(5, 40, 120, 30)
	return p
}
