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
	// EffortImpactUnset is undefined EI
	EffortImpactUnset = iota
	// EffortImpactHigh is the highest
	EffortImpactHigh
	// EffortImpactMedium is medium
	EffortImpactMedium
	// EffortImpactLow is low
	EffortImpactLow
	// EffortImpactAvoid means stay away!
	EffortImpactAvoid
)

var effortImpactText = map[int]string{
	EffortImpactUnset:  "Unset",
	EffortImpactHigh:   "Low Effort, High Impact",
	EffortImpactMedium: "High Effort, High Impact",
	EffortImpactLow:    "Low Effort, Low Impact",
	EffortImpactAvoid:  "High Effort, Low Impact",
}

// EffortImpactText returns text from the code
func EffortImpactText(code int) string {
	return effortImpactText[code]
}
