package riskcatalog

import (
	"fmt"
	"strconv"
	"strings"
)

// trimmed prints n with at most the given number of decimals, dropping trailing zeros: 30, 0.5, 0.33.
func trimmed(n float32, decimals int) string {
	s := strconv.FormatFloat(float64(n), 'f', decimals, 32)
	if strings.Contains(s, ".") {
		s = strings.TrimSuffix(strings.TrimRight(s, "0"), ".")
	}
	if s == "-0" {
		return "0"
	}
	return s
}

// formatInput prints a duration from the catalog file (minutes or days) with at most two decimals.
func formatInput(n float32) string {
	return trimmed(n, 2)
}

// formatAmount prints a computed figure with fewer decimals the bigger it is: 2630, 65.7, 1.33.
func formatAmount(n float32) string {
	a := n
	if a < 0 {
		a = -a
	}
	switch {
	case a >= 100:
		return trimmed(n, 0)
	case a >= 10:
		return trimmed(n, 1)
	}
	return trimmed(n, 2)
}

// formatFraction prints a fraction from 0 to 1 as a percentage: 0.25 is 25%.
func formatFraction(f float32) string {
	return trimmed(f*100, 2) + "%"
}

// formatPercent prints a percentage that has already been multiplied by 100.
func formatPercent(p float32) string {
	return formatAmount(p) + "%"
}

// formatTarget prints an availability target: 99.5, 99.99, 99.999.
func formatTarget(t float32) string {
	return trimmed(t, 4)
}

func plural(n int, singular, pluralForm string) string {
	if n == 1 {
		return fmt.Sprintf("%d %s", n, singular)
	}
	return fmt.Sprintf("%d %s", n, pluralForm)
}

// BudgetLines describes the catalog's error budget and how much of it the catalog uses, a line each.
func (rc RiskCatalog) BudgetLines() []string {
	eb := rc.ErrorBudget
	total := rc.TotalBadMinutesPerYear()
	left := eb.MinutesPerYear() - total

	lines := []string{
		fmt.Sprintf("Availability target: %s%%", formatTarget(eb.AvailabilityTarget)),
		fmt.Sprintf("Error budget: %s minutes per year, %s minutes per 4 weeks",
			formatAmount(eb.MinutesPerYear()), formatAmount(eb.MinutesPerFourWeekIteration())),
		fmt.Sprintf("Single-risk threshold: %s minutes per year (%s of the budget)",
			formatAmount(eb.TooBigThreshold()), formatFraction(eb.AcceptableThresholdPercentagePerRisk)),
		fmt.Sprintf("Bad minutes of all risks (%s, %s): %s minutes per year (%s of the budget)",
			plural(len(rc.Incidents), "incident", "incidents"), plural(len(rc.Risks), "risk factor", "risk factors"),
			formatAmount(total), formatPercent(rc.PercentOfBudget(total))),
	}
	if left >= 0 {
		lines = append(lines, fmt.Sprintf("Left: %s minutes per year (%s of the budget)",
			formatAmount(left), formatPercent(rc.PercentOfBudget(left))))
	} else {
		lines = append(lines, fmt.Sprintf("Left: none; over budget by %s minutes per year", formatAmount(-left)))
	}
	return lines
}

// markdownCell keeps a value inside its table cell.
func markdownCell(s string) string {
	s = strings.ReplaceAll(s, "\r\n", " ")
	s = strings.ReplaceAll(s, "\n", " ")
	return strings.ReplaceAll(s, "|", `\|`)
}

func writeMarkdownRow(b *strings.Builder, cells ...string) {
	b.WriteString("|")
	for _, c := range cells {
		b.WriteString(" ")
		b.WriteString(markdownCell(c))
		b.WriteString(" |")
	}
	b.WriteString("\n")
}

// notApplicable fills a stack-rank cell that only incidents have.
const notApplicable = "—"

// Markdown returns a report of the catalog: its error budget, every risk ranked by bad minutes per year,
// and the risk factors. Its sections are level-2 headings, so it can sit under a page's own title.
func Markdown(rc RiskCatalog) string {
	ranked := rc.ComputeRisk() // also fills in the risk factors' bad minutes
	var b strings.Builder

	b.WriteString("## Error budget\n\n")
	for _, line := range rc.BudgetLines() {
		fmt.Fprintf(&b, "- %s\n", line)
	}

	b.WriteString("\n## Stack rank\n\n")
	if len(ranked) == 0 {
		b.WriteString("No incidents or risk factors.\n")
	} else {
		b.WriteString("| Description | ETTD (min) | ETTR (min) | ETTF (days) | Impact | Incidents/yr | Bad min/yr | % of budget | Verdict |\n")
		b.WriteString("|---|--:|--:|--:|--:|--:|--:|--:|---|\n")
		for _, r := range ranked {
			bm := r.BadMinutesPerYear()
			ettd, ettr, ettf, impact, perYear := notApplicable, notApplicable, notApplicable, notApplicable, notApplicable
			if i, ok := r.(Incident); ok {
				ettd, ettr, ettf = formatInput(i.EttdMinutes), formatInput(i.EttrMinutes), formatInput(i.EttfDays)
				impact, perYear = formatFraction(i.PercentageImpact), formatAmount(i.IncidentsPerYear())
			}
			writeMarkdownRow(&b, r.RiskDescription(), ettd, ettr, ettf, impact, perYear,
				formatAmount(bm), formatPercent(rc.PercentOfBudget(bm)), rc.VerdictFor(bm).String())
		}
	}

	b.WriteString("\n## Risk factors\n\n")
	if len(rc.Risks) == 0 {
		b.WriteString("No risk factors.\n")
	} else {
		b.WriteString("| Description | ETTD+ (min) | ETTR+ (min) | ETTF+ | Bad min/yr | % of budget | Verdict |\n")
		b.WriteString("|---|--:|--:|--:|--:|--:|---|\n")
		for _, rf := range rc.Risks {
			bm := rf.BadMinutesPerYear()
			writeMarkdownRow(&b, rf.Description, formatInput(rf.EttdPenaltyMinutes), formatInput(rf.EttrPenaltyMinutes),
				formatFraction(rf.EttfPenaltyPercentageIncrease), formatAmount(bm), formatPercent(rc.PercentOfBudget(bm)),
				rc.VerdictFor(bm).String())
		}
	}

	return b.String()
}
