package taskpoet

import (
	"time"

	"gonum.org/v1/gonum/floats"
)

type weighter func(t Task) float64

type weightMap map[string]weighter

var defaultWeightMap = weightMap{
	"due": func(t Task) float64 {
		if t.Due == nil {
			return -100
		}
		lateness := time.Since(*t.Due)
		if lateness > 0 {
			return lateness.Hours() * 24 * .001
		}
		// Default
		return 0
	},
	"age": func(t Task) float64 {
		lateDays := time.Since(t.Added).Hours()
		multiplier := float64(0.00001)
		return lateDays * multiplier
	},
}

// Curator examines tasks and weights them, giving them a semi smart sense of
// urgency
type Curator struct {
	weights weightMap
}

// Weigh returns the weight of a task by a curator
func (c Curator) Weigh(t Task) float64 {
	var items []float64
	for _, wr := range c.weights {
		items = append(items, wr(t))
	}
	return floats.Sum(items)
}

// WithWeights sets the weights in a curator at build time
func WithWeights(w weightMap) func(*Curator) {
	return func(c *Curator) {
		c.weights = w
	}
}

// NewCurator returns a new Curator instance with functional options
func NewCurator(options ...func(*Curator)) *Curator {
	c := &Curator{
		weights: defaultWeightMap,
	}
	for _, opt := range options {
		opt(c)
	}
	return c
}
