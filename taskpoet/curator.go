package taskpoet

import (
	"fmt"
	"time"
)

// Weither returns x, coefficient, unit
type weighter func(t Task) (float64, int, string)

type weightMap map[string]weighter

const daysUnit string = "days"

var defaultWeightMap = weightMap{
	"effort": func(t Task) (float64, int, string) {
		switch t.EffortImpact {
		case 1:
			return 3, 1, fmt.Sprint(t.EffortImpact)
		case 2:
			return 2, 1, fmt.Sprint(t.EffortImpact)
		case 3:
			return 1, 1, fmt.Sprint(t.EffortImpact)
		default:
			return 0, 0, ""
		}
	},
	"children": func(t Task) (float64, int, string) {
		if len(t.Children) > 0 {
			return 1, 1, "has children"
		}
		return 0, 0, ""
	},
	"next": func(t Task) (float64, int, string) {
		for _, tag := range t.Tags {
			if tag == "next" {
				return 15, 1, "has next tag"
			}
		}
		return 0, 0, ""
	},
	"due": func(t Task) (float64, int, string) {
		if t.Due == nil {
			return 0, 0, ""
		}
		lateness := int(time.Since(*t.Due).Hours() / 24)
		switch {
		case lateness >= 7:
			// A week overdue
			return 1, 1, "maxed out lateness at 1 week overdue"
		case lateness >= -14:
			// Dueness coming up in 2 weeks
			return ((float64(lateness) + 14.0) * 0.8 / 21.0) + 0.2, 1, "approaching"
		default:
			return 0.2, 1, "due in over 2 weeks"
		}
	},
	"age": func(t Task) (float64, int, string) {
		// return float64(0.004), int(time.Since(t.Added).Hours() / 24), daysUnit
		days := time.Since(t.Added).Hours() / 24
		if days < 1 {
			return 0, 0, "super new"
		}
		return 1 / days, 1, fmt.Sprintf("days (%v)", int(days))
	},
}

// Curator decides how important (or Urgent) something is
type Curator struct {
	weights weightMap
}

// Weigh returns the weight of a task by a curator
func (c Curator) Weigh(t Task) float64 {
	ret := float64(0)
	for _, wr := range c.weights {
		co, multi, _ := wr(t)
		ret += co * float64(multi)
	}
	return ret
}

// WeighAndDescribe returns not only the weight, but descriptions of how we got there
func (c Curator) WeighAndDescribe(t Task) (float64, []WeightDescription) {
	ret := float64(0)
	ds := []WeightDescription{}
	for name, wr := range c.weights {
		c, multi, unit := wr(t)
		ret += c * float64(multi)
		if multi != 0 {
			ds = append(ds, WeightDescription{
				Name:        name,
				Coefficient: c,
				Multiplier:  multi,
				Unit:        unit,
			})
		}
	}
	return ret, ds
}

// WeightDescription describes each weight item
type WeightDescription struct {
	Name        string
	Coefficient float64
	Multiplier  int
	Unit        string
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
