package taskpoet

/*
	These are set in stone
	0 - Unset
	1 - Low Effort, High Impact (Sweet Spot)
	2 - High Effort, High Impact (Homework)
	3 - Low Effort, Low Impact (Busywork)
	4 - High Effort, Low Impact (Charity)
*/

// EffortImpact is the type for an effor/impact statement
type EffortImpact int

const (
	// EffortImpactUnset is undefined EI
	EffortImpactUnset EffortImpact = iota
	// EffortImpactHigh is the highest
	EffortImpactHigh
	// EffortImpactMedium is medium
	EffortImpactMedium
	// EffortImpactLow is low
	EffortImpactLow
	// EffortImpactAvoid means stay away!
	EffortImpactAvoid
)

// Emoji returns a nice little visual queue for a given effort/impact statement
func (e EffortImpact) Emoji() string {
	return effortImpactEmoji[e]
}

// String satisfies the Stringer interface and allows us to easily print these out
func (e EffortImpact) String() string {
	return effortImpactText[e]
}

var effortImpactText = map[EffortImpact]string{
	EffortImpactUnset:  "Unset",
	EffortImpactHigh:   "Low Effort, High Impact",
	EffortImpactMedium: "High Effort, High Impact",
	EffortImpactLow:    "Low Effort, Low Impact",
	EffortImpactAvoid:  "High Effort, Low Impact",
}

var effortImpactEmoji = map[EffortImpact]string{
	EffortImpactUnset:  "🟣",
	EffortImpactHigh:   "🟢",
	EffortImpactMedium: "🟡",
	EffortImpactLow:    "🔴",
	EffortImpactAvoid:  "💀",
}
