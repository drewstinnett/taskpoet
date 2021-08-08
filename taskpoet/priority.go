package taskpoet

/*
	These are set in stone
	0 - Unset
	1 - Low Effort, High Impact (Sweet Spot)
	2 - High Effort, High Impact (Homework)
	3 - Low Effort, Low Impact (Busywork)
	4 - High Effort, Low Impact (Charity)
*/

const (
	EffortImpactUnset = iota
	EffortImpactHigh
	EffortImpactMedium
	EffortImpactLow
	EffortImpactAvoid
)

var effortImpactText = map[int]string{
	EffortImpactUnset:  "Unset",
	EffortImpactHigh:   "Low Effort, High Impact",
	EffortImpactMedium: "High Effort, High Impact",
	EffortImpactLow:    "Low Effort, Low Impact",
	EffortImpactAvoid:  "High Effort, Low Impact",
}

func EffortImpactText(code int) string {
	return effortImpactText[code]
}
