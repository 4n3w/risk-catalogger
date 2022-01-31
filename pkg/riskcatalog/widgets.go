package riskcatalog

import (
	"fmt"
	ui "github.com/gizak/termui/v3"
	"github.com/gizak/termui/v3/widgets"
)

func BuildTabs() *widgets.TabPane {
	tabpane := widgets.NewTabPane("Risk Catalog", "Risk Factors", "Risk Stack Rank", "Risk List", "Error Budget")
	tabpane.SetRect(0, 1, 80, 4)
	tabpane.Border = true
	return tabpane
}

func BuildStackedRisksTable(rankedRisks []HasRisk) *widgets.Table {
	arr := make([][]string, len(rankedRisks)+1)
	arr[0] = []string{"Description", "BM/Y"}
	t := widgets.NewTable()
	t.RowStyles = map[int]ui.Style{
		0: ui.NewStyle(ui.ColorBlue),
	}
	for i, r := range rankedRisks {
		arr[i+1] = []string{
			r.RiskDescription(),
			fmt.Sprintf("%.0f", r.BadMinutesPerYear()),
		}
	}

	t.Rows = arr
	t.SetRect(5, 5, 85, 30)
	t.ColumnWidths = []int{72, 10}
	t.TextAlignment = ui.AlignCenter
	//t.TextStyle.Bg = ui.ColorBlue

	t.Title = "Risk Description, Bad Minutes per year"
	t.TextAlignment = ui.AlignLeft

	return t
}

func BuildRiskCatalogList(rc RiskCatalog) *widgets.List {
	l := widgets.NewList()
	arr := make([]string, len(rc.Incidents))

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
			fmt.Sprintf("%d", r.EttdMinutes),
			fmt.Sprintf("%d", r.EttrMinutes),
			fmt.Sprintf("%.0f%%", r.PercentageImpact*100.0),
			fmt.Sprintf("%d", r.EttfDays),
			fmt.Sprintf("%.1f", r.IncidentsPerYear()),
			fmt.Sprintf("%.0f", r.BadMinutesPerYear()),
		}
	}

	t.Rows = arr
	t.SetRect(5, 5, 120, 30)
	t.ColumnWidths = []int{60, 8, 8, 8, 8, 8, 8}
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
			fmt.Sprintf("%d", r.EttdPenaltyMinutes),
			fmt.Sprintf("%d", r.EttrPenaltyMinutes),
			fmt.Sprintf("%.0f%%", r.EttfPenaltyPercentageIncrease*100.0),
			fmt.Sprintf("%.2f", r.BadMinutesPerYear()),
		}
	}
	t.Rows = arr
	t.SetRect(5, 5, 120, 30)
	t.ColumnWidths = []int{70, 10, 10, 10, 10}

	t.Title = "Risk Factors"
	t.TextAlignment = ui.AlignCenter

	return t
}

func BuildStackedBarChart() *widgets.StackedBarChart {
	sbc := widgets.NewStackedBarChart()

	sbc.Title = "Risks"
	sbc.Labels = []string{"99.99", "99.95", "99.9", "99.5"}

	sbc.Data = make([][]float64, 4)
	sbc.Data[0] = []float64{90, 85, 90, 80}
	sbc.Data[1] = []float64{70, 85, 75, 60}
	sbc.Data[2] = []float64{75, 60, 80, 85}
	sbc.Data[3] = []float64{100, 100, 100, 100}
	sbc.SetRect(85, 5, 120, 30)
	sbc.BarWidth = 10

	return sbc
}

func BuildParagraph() *widgets.Paragraph {
	p := widgets.NewParagraph()
	p.Text = "Green fits within error budget\n\n"
	p.Text += "Yellow is cautionary\n\n"
	p.Text += "Red is unacceptable risk\n\n"
	p.Text += "Blue is accepted risk\n\n"
	p.TextStyle = ui.NewStyle(ui.ColorGreen)
	p.SetRect(5, 40, 120, 30)
	return p
}

func RiskCatalogParagraph() *widgets.Paragraph {
	p := widgets.NewParagraph()
	p.Text = "ETTD: Estimated Time To Detect: (minutes) How long it takes someone to notice the service is broken.\n"
	p.Text += "ETTD: Estimated Time To Repair: (minutes) How long it takes someone to repair the service.\n"
	p.Text += "% Users: Estimate of how many users would be affected by this incident.\n"
	p.Text += "ETTF: Estimated Time To Fail: (days) Time between instances of this incident occurring.\n"
	p.Text += "IP/Y: Incidents per year (estimate).\n"
	p.Text += "BM/Y: Bad minutes per year. Calculated estimate of the operational cost of this occuring.\n"
	p.TextStyle = ui.NewStyle(ui.ColorGreen)
	p.SetRect(5, 40, 120, 30)
	return p
}

func RiskFactorParagraph() *widgets.Paragraph {
	p := widgets.NewParagraph()
	p.Text = "ETTD: Estimated Time To Detect: (minutes) How long it takes someone to notice the service is broken.\n"
	p.Text += "ETTD: Estimated Time To Repair: (minutes) How long it takes someone to repair the service.\n"
	p.Text += "ETTF: Estimated Time To Fail: (days) Time between instances of this incident occurring.\n"
	p.Text += "IP/Y: Incidents per year (estimate).\n"
	p.Text += "BM/Y: Bad minutes per year. Calculated estimate of the operational cost of this occuring.\n"
	p.TextStyle = ui.NewStyle(ui.ColorGreen)
	p.SetRect(5, 40, 120, 30)
	return p
}
