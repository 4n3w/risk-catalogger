package riskcatalog

import (
	"fmt"
	"gopkg.in/yaml.v3"
	"os"
	"sort"
)

// DaysPerYear is the year the bad-minute figures are measured over.
const DaysPerYear = 365.25

// StackRankTargets are the availability targets the Risk Stack Rank tab compares the catalog against.
var StackRankTargets = []float32{99.99, 99.95, 99.9, 99.5}

type HasRisk interface {
	BadMinutesPerYear() float32
	RiskDescription() string
}

type RiskCatalog struct {
	Incidents     []Incident   `yaml:"incidents"`
	Risks         []RiskFactor `yaml:"risks"`
	ErrorBudget   ErrorBudget  `yaml:"error-budget"`
	acceptedRisks []HasRisk    //todo future state should be map[float32]*HasRisk

}

// Incident Estimate Time to {Detect,Repair,Fail}({Minutes,Minutes,Days}).
// The durations are float32, so a 20-second outage is 0.33 minutes; whole numbers load too.
type Incident struct {
	Description      string  `yaml:"description"`
	EttdMinutes      float32 `yaml:"estimated-time-to-detect"`
	EttrMinutes      float32 `yaml:"estimated-time-to-repair"`
	EttfDays         float32 `yaml:"estimated-time-to-fail"`
	PercentageImpact float32 `yaml:"percentage-impact"`
}

type RiskFactor struct {
	Description                   string  `yaml:"description"`
	EttdPenaltyMinutes            float32 `yaml:"increased-ettd-minutes,omitempty"`
	EttrPenaltyMinutes            float32 `yaml:"increased-ettr-minutes,omitempty"`
	EttfPenaltyPercentageIncrease float32 `yaml:"increased-ettf-percentage,omitempty"`
	BadMinutes                    float32 `yaml:"-"` // set by ComputeRisk
}

type ErrorBudget struct {
	AvailabilityTarget                   float32 `yaml:"availability-target"`
	AcceptableThresholdPercentagePerRisk float32 `yaml:"single-risk-acceptable-threshold"`
}

// Verdict is how a risk's bad minutes per year compare with the single-risk threshold.
type Verdict int

const (
	// Fits: at most half the single-risk threshold.
	Fits Verdict = iota
	// Cautionary: over half the single-risk threshold.
	Cautionary
	// TooBig: over the single-risk threshold.
	TooBig
)

func (v Verdict) String() string {
	switch v {
	case Fits:
		return "fits"
	case Cautionary:
		return "cautionary"
	case TooBig:
		return "too big"
	}
	return fmt.Sprintf("Verdict(%d)", int(v))
}

// BudgetUsage is how much of one availability target's yearly error budget the catalog uses, in percent,
// split between risks under that target's single-risk threshold and risks over it.
type BudgetUsage struct {
	Target        float32
	BudgetMinutes float32 // per year
	UnderPercent  float32
	OverPercent   float32
}

func (u BudgetUsage) TotalPercent() float32 {
	return u.UnderPercent + u.OverPercent
}

func (i Incident) IncidentsPerYear() float32 {
	return i.IncidentsPerIterationLength(DaysPerYear)
}

func (i Incident) RiskDescription() string {
	return i.Description
}

func (i Incident) IncidentsPerIterationLength(l float32) float32 {
	return l / i.EttfDays
}

func (i Incident) BadMinutesPerYear() float32 {
	return (i.EttdMinutes + i.EttrMinutes) * i.PercentageImpact * i.IncidentsPerYear()
}

func (rf RiskFactor) BadMinutesPerYear() float32 {
	return rf.BadMinutes
}

func (rf RiskFactor) RiskDescription() string {
	return rf.Description
}

func (eb ErrorBudget) MinutesPerYear() float32 {
	return (1.0 - (eb.AvailabilityTarget / 100)) * 1440 * DaysPerYear
}

func (eb ErrorBudget) MinutesPerFourWeekIteration() float32 {
	return (1.0 - (eb.AvailabilityTarget / 100)) * 1440 * 28
}

// TooBigThreshold returns the single-risk threshold in minutes/yr.
func (eb ErrorBudget) TooBigThreshold() float32 {
	return eb.MinutesPerYear() * eb.AcceptableThresholdPercentagePerRisk
}

// VerdictFor rates a risk's bad minutes per year against this budget's single-risk threshold.
func (eb ErrorBudget) VerdictFor(badMinutesPerYear float32) Verdict {
	threshold := eb.TooBigThreshold()
	switch {
	case badMinutesPerYear > threshold:
		return TooBig
	case badMinutesPerYear > threshold/2:
		return Cautionary
	}
	return Fits
}

func (rc RiskCatalog) AcceptedMinutesOfRiskPerYear() float32 {
	var totalAcceptedMinutes float32 = 0.0

	for _, a := range rc.acceptedRisks {
		totalAcceptedMinutes += a.BadMinutesPerYear()
	}

	return totalAcceptedMinutes
}

// TooBigThreshold returns the catalog's single-risk threshold in minutes/yr.
func (rc RiskCatalog) TooBigThreshold() float32 {
	return rc.ErrorBudget.TooBigThreshold()
}

// VerdictFor rates a risk's bad minutes per year against the catalog's single-risk threshold.
func (rc RiskCatalog) VerdictFor(badMinutesPerYear float32) Verdict {
	return rc.ErrorBudget.VerdictFor(badMinutesPerYear)
}

// PercentOfBudget returns bad minutes per year as a percentage of the catalog's yearly error budget.
func (rc RiskCatalog) PercentOfBudget(badMinutesPerYear float32) float32 {
	return badMinutesPerYear / rc.ErrorBudget.MinutesPerYear() * 100
}

// UnallocatedBudget  returns minutes/yr
func (rc RiskCatalog) UnallocatedBudget() float32 {
	return rc.ErrorBudget.MinutesPerYear() - rc.AcceptedMinutesOfRiskPerYear()
}

// TotalBadMinutesPerYear returns the bad minutes per year of all incidents and risk factors together.
func (rc RiskCatalog) TotalBadMinutesPerYear() float32 {
	var total float32
	for _, r := range rc.ComputeRisk() {
		total += r.BadMinutesPerYear()
	}
	return total
}

// RemainingBudget returns the error budget left in minutes/yr after all incidents and risk factors;
// it is negative when the catalog is over budget.
func (rc RiskCatalog) RemainingBudget() float32 {
	return rc.ErrorBudget.MinutesPerYear() - rc.TotalBadMinutesPerYear()
}

// BudgetUsageAt returns how much of the yearly error budget at the given availability target the catalog
// uses, splitting its risks by that target's single-risk threshold (the catalog's threshold fraction applied
// to that target's budget).
func (rc RiskCatalog) BudgetUsageAt(target float32) BudgetUsage {
	eb := ErrorBudget{
		AvailabilityTarget:                   target,
		AcceptableThresholdPercentagePerRisk: rc.ErrorBudget.AcceptableThresholdPercentagePerRisk,
	}
	budget := eb.MinutesPerYear()
	threshold := eb.TooBigThreshold()

	var under, over float32
	for _, r := range rc.ComputeRisk() {
		if r.BadMinutesPerYear() > threshold {
			over += r.BadMinutesPerYear()
		} else {
			under += r.BadMinutesPerYear()
		}
	}

	return BudgetUsage{
		Target:        target,
		BudgetMinutes: budget,
		UnderPercent:  under / budget * 100,
		OverPercent:   over / budget * 100,
	}
}

// ComputeRisk returns the incidents and risk factors ranked by bad minutes per year, highest first.
// It also fills in each risk factor's BadMinutes in rc.Risks.
func (rc RiskCatalog) ComputeRisk() []HasRisk {
	var sumOfProducts float32 = 0.0
	var sumOfBadMinutes float32 = 0.0

	var risks []HasRisk

	for _, i := range rc.Incidents {
		sumOfProducts += i.PercentageImpact * i.IncidentsPerYear()
		sumOfBadMinutes += i.BadMinutesPerYear()
		risks = append(risks, i)
	}

	for i, rf := range rc.Risks {
		rf.BadMinutes = (1+rf.EttfPenaltyPercentageIncrease)*((rf.EttdPenaltyMinutes+rf.EttrPenaltyMinutes)*sumOfProducts) + (rf.EttfPenaltyPercentageIncrease * sumOfBadMinutes)
		risks = append(risks, rf)
		rc.Risks[i] = rf
	}

	sort.SliceStable(risks, func(i, j int) bool {
		return risks[i].BadMinutesPerYear() > risks[j].BadMinutesPerYear()
	})

	return risks
}

// Validate checks the values the arithmetic depends on.
func (rc RiskCatalog) Validate() error {
	eb := rc.ErrorBudget
	if eb.AvailabilityTarget <= 0 || eb.AvailabilityTarget >= 100 {
		return fmt.Errorf("error-budget: availability-target must be a percentage between 0 and 100, not %v", eb.AvailabilityTarget)
	}
	if eb.AcceptableThresholdPercentagePerRisk <= 0 || eb.AcceptableThresholdPercentagePerRisk > 1 {
		return fmt.Errorf("error-budget: single-risk-acceptable-threshold must be a fraction of the budget, more than 0 and at most 1, not %v", eb.AcceptableThresholdPercentagePerRisk)
	}
	for n, i := range rc.Incidents {
		where := fmt.Sprintf("incident %d (%q)", n+1, i.Description)
		switch {
		case i.EttdMinutes < 0:
			return fmt.Errorf("%s: estimated-time-to-detect must not be negative", where)
		case i.EttrMinutes < 0:
			return fmt.Errorf("%s: estimated-time-to-repair must not be negative", where)
		case i.EttfDays <= 0:
			return fmt.Errorf("%s: estimated-time-to-fail must be more than 0 days", where)
		case i.PercentageImpact < 0 || i.PercentageImpact > 1:
			return fmt.Errorf("%s: percentage-impact must be a fraction from 0 to 1, not %v", where, i.PercentageImpact)
		}
	}
	return nil
}

func NewRiskCatalogFromFile(configFilePath string) (RiskCatalog, error) {
	c := RiskCatalog{}
	buf, err := os.ReadFile(configFilePath)
	if err != nil {
		return c, err
	}

	err = yaml.Unmarshal(buf, &c)
	if err != nil {
		return c, fmt.Errorf("%s: %w", configFilePath, err)
	}

	if err := c.Validate(); err != nil {
		return c, fmt.Errorf("%s: %w", configFilePath, err)
	}

	return c, nil
}
