package riskcatalog

import (
	"gopkg.in/yaml.v3"
	"os"
	"sort"
)

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

// Incident Estimate Time to {Detect,Repair,Fail}({Minutes,Minutes,Days})
type Incident struct {
	Description      string  `yaml:"description"`
	EttdMinutes      int     `yaml:"estimated-time-to-detect"`
	EttrMinutes      int     `yaml:"estimated-time-to-repair"`
	EttfDays         int     `yaml:"estimated-time-to-fail"`
	PercentageImpact float32 `yaml:"percentage-impact"`
}

type RiskFactor struct {
	Description                   string  `yaml:"description"`
	EttdPenaltyMinutes            int     `yaml:"increased-ettd-minutes,omitempty"`
	EttrPenaltyMinutes            int     `yaml:"increased-ettr-minutes,omitempty"`
	EttfPenaltyPercentageIncrease float32 `yaml:"increased-ettf-percentage,omitempty"`
	BadMinutes                    float32
}

type ErrorBudget struct {
	AvailabilityTarget                   float32 `yaml:"availability-target"`
	AcceptableThresholdPercentagePerRisk float32 `yaml:"single-risk-acceptable-threshold"`
}

func (i Incident) IncidentsPerYear() float32 {
	return i.IncidentsPerIterationLength(365.25)
}

func (i Incident) RiskDescription() string {
	return i.Description
}

func (i Incident) IncidentsPerIterationLength(l float32) float32 {
	return l / float32(i.EttfDays)
}

func (i Incident) BadMinutesPerYear() float32 {
	return (float32(i.EttdMinutes) + float32(i.EttrMinutes)) * i.PercentageImpact * i.IncidentsPerYear()
}

func (rf RiskFactor) BadMinutesPerYear() float32 {
	return rf.BadMinutes
}

func (rf RiskFactor) RiskDescription() string {
	return rf.Description
}

func (eb ErrorBudget) MinutesPerYear() float32 {
	return (1.0 - (eb.AvailabilityTarget / 100)) * 1440 * 365.25
}

func (eb ErrorBudget) MinutesPerFourWeekIteration() float32 {
	return (1.0 - (eb.AvailabilityTarget / 100)) * 1440 * 28
}

func (rc RiskCatalog) AcceptedMinutesOfRiskPerYear() float32 {
	var totalAcceptedMinutes float32 = 0.0

	for _, a := range rc.acceptedRisks {
		totalAcceptedMinutes += a.BadMinutesPerYear()
	}

	return totalAcceptedMinutes
}

func (rc RiskCatalog) TooBigThreshold() float32 {
	return rc.ErrorBudget.MinutesPerYear() * rc.ErrorBudget.AcceptableThresholdPercentagePerRisk
}

// UnallocatedBudget  returns minutes/yr
func (rc RiskCatalog) UnallocatedBudget() float32 {
	return rc.ErrorBudget.MinutesPerYear() - rc.AcceptedMinutesOfRiskPerYear()
}

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
		rf.BadMinutes = (1+rf.EttfPenaltyPercentageIncrease)*(float32(rf.EttdPenaltyMinutes+rf.EttrPenaltyMinutes)*sumOfProducts) + (rf.EttfPenaltyPercentageIncrease * sumOfBadMinutes)
		risks = append(risks, rf)
		rc.Risks[i] = rf
	}

	sort.Slice(risks[:], func(i, j int) bool {
		return risks[i].BadMinutesPerYear() > risks[j].BadMinutesPerYear()
	})

	return risks
}

func NewRiskCatalogFromFile(configFilePath string) (RiskCatalog, error) {
	c := RiskCatalog{}
	buf, err := os.ReadFile(configFilePath)
	if err != nil {
		return c, err
	}

	err = yaml.Unmarshal(buf, &c)
	if err != nil {
		return c, err
	}

	return c, nil
}
