package main

import (
	"fmt"
	"github.com/charmbracelet/bubbles/textinput"
	ui "github.com/gizak/termui/v3"
	"github.com/vmware-tanzu-labs/risk-catalogger/pkg/riskcatalog"
	"log"
	"os"
)

type model struct {
	choices   []riskcatalog.HasRisk
	cursor    int
	selected  map[int]struct{}
	textInput textinput.Model
}

func main() {
	rc, err := riskcatalog.NewRiskCatalogFromFile("incidents.yml")
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %s\n", err)
		os.Exit(1)
	}

	if err := ui.Init(); err != nil {
		log.Fatalf("failed to initialize termui: %v", err)
	}
	defer ui.Close()

	var rankedRisks = rc.ComputeRisk()

	tabpane := riskcatalog.BuildTabs()

	riskCatalogTable := riskcatalog.BuildRiskCatalogTable(rc)
	riskCatalogList := riskcatalog.BuildRiskCatalogList(rc)
	riskFactorTable := riskcatalog.BuildRiskFactorTable(rc)
	riskFactorTableParagraph := riskcatalog.RiskCatalogParagraph()
	stackedRisksTable := riskcatalog.BuildStackedRisksTable(rankedRisks)

	sbc := riskcatalog.BuildStackedBarChart()
	p2 := riskcatalog.BuildParagraph()

	renderTab := func() {
		switch tabpane.ActiveTabIndex {
		case 0:
			ui.Render(riskCatalogTable)
			ui.Render(riskFactorTableParagraph)
		case 1:
			ui.Render(riskFactorTable)
		case 2:
			ui.Render(p2)
			ui.Render(sbc)
			ui.Render(stackedRisksTable)
		case 3:
			ui.Render(riskCatalogList)
		}
	}

	ui.Render(tabpane, riskCatalogTable, riskFactorTableParagraph)

	uiEvents := ui.PollEvents()
	for {
		e := <-uiEvents
		switch e.ID {
		case "q", "<C-c>":
			return
		case "h", "<Left>":
			tabpane.FocusLeft()
			ui.Clear()
			ui.Render(tabpane)
			renderTab()
		case "l", "<Right>":
			tabpane.FocusRight()
			ui.Clear()
			ui.Render(tabpane)
			renderTab()
		case "j", "<Down>":
			if tabpane.ActiveTabIndex == 3 {
				riskCatalogList.ScrollDown()
				ui.Clear()
				ui.Render(tabpane)
				ui.Render(riskCatalogList)
				renderTab()
			}
		case "k", "<Up>":
			if tabpane.ActiveTabIndex == 3 {
				riskCatalogList.ScrollUp()
				ui.Clear()
				ui.Render(tabpane)
				ui.Render(riskCatalogList)
				renderTab()
			}

		}

	}

}
