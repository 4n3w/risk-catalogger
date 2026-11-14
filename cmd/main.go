package main

import (
	"flag"
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
	file := flag.String("f", "incidents.yml", "the risk catalog `file` to read")
	markdown := flag.Bool("markdown", false, "print a Markdown report to stdout and exit, instead of starting the terminal UI")
	flag.Usage = func() {
		fmt.Fprintf(flag.CommandLine.Output(), "Usage: %s [-f file] [-markdown]\n\n", os.Args[0])
		fmt.Fprintln(flag.CommandLine.Output(), "Ranks incident risks by bad minutes per year against an error budget.")
		fmt.Fprintln(flag.CommandLine.Output(), "Without -markdown it opens a terminal UI: h/l or the arrow keys switch tabs, q quits.")
		fmt.Fprintln(flag.CommandLine.Output())
		flag.PrintDefaults()
	}
	flag.Parse()
	if flag.NArg() > 0 {
		fmt.Fprintf(os.Stderr, "error: unexpected argument %q\n", flag.Arg(0))
		flag.Usage()
		os.Exit(2)
	}

	rc, err := riskcatalog.NewRiskCatalogFromFile(*file)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %s\n", err)
		os.Exit(1)
	}

	if *markdown {
		fmt.Print(riskcatalog.Markdown(rc))
		return
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
	riskFactorTableParagraph := riskcatalog.CatalogParagraph()
	stackedRisksTable := riskcatalog.BuildStackedRisksTable(rc, rankedRisks)

	sbc := riskcatalog.BuildStackedBarChart(rc)
	budgetUsageTable := riskcatalog.BuildBudgetUsageTable(rc)
	p2 := riskcatalog.BuildParagraph(rc)
	errorBudgetParagraph := riskcatalog.BuildErrorBudgetParagraph(rc)

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
			ui.Render(budgetUsageTable)
			ui.Render(stackedRisksTable)
		case 3:
			ui.Render(riskCatalogList)
		case 4:
			ui.Render(errorBudgetParagraph)
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
