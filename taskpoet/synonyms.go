package taskpoet

import (
	"errors"
	"fmt"
	"time"
)

// Synonym is a helper for a relative time period
type Synonym string

// Aliases returns a list of aliases for a Synonym
func (s Synonym) Aliases() []string {
	if got, ok := aliases[s]; ok {
		return got
	}
	return []string{}
}

// Description returns a description for a Synonym
func (s Synonym) Description() string {
	if got, ok := descriptions[s]; ok {
		return got
	}
	return ""
}

var aliases = map[Synonym][]string{
	January:     {"jan"},
	February:    {"feb"},
	March:       {"mar"},
	April:       {"apr"},
	May:         {"may"},
	June:        {"jun"},
	July:        {"jul"},
	August:      {"aug"},
	September:   {"sep"},
	October:     {"oct"},
	November:    {"nov"},
	December:    {"dec"},
	Monday:      {"mon"},
	Tuesday:     {"tue"},
	Wednesday:   {"wed"},
	Thursday:    {"thu"},
	Friday:      {"fri"},
	Saturday:    {"sat"},
	Sunday:      {"sun"},
	Later:       {"someday"},
	EndOfday:    {"eod"},
	StartOfDay:  {"sod"},
	StartOfYear: {"soy"},
	EndOfYear:   {"eoy"},
}

const (
	dateDesc string = "Date for the specified month, starting at the beginning of the 1st day"
	dayDesc  string = "Date for the specified day, after today, starting at the beginning of the day"
)

var descriptions = map[Synonym]string{
	January:     dateDesc,
	February:    dateDesc,
	March:       dateDesc,
	April:       dateDesc,
	May:         dateDesc,
	June:        dateDesc,
	July:        dateDesc,
	August:      dateDesc,
	September:   dateDesc,
	October:     dateDesc,
	November:    dateDesc,
	December:    dateDesc,
	Monday:      dayDesc,
	Tuesday:     dayDesc,
	Wednesday:   dayDesc,
	Thursday:    dayDesc,
	Friday:      dayDesc,
	Saturday:    dayDesc,
	Sunday:      dayDesc,
	Later:       "Super far away date",
	EndOfday:    "End of the day, today",
	StartOfDay:  "Start of the day, today",
	StartOfYear: "Start of this year, beginning of the day",
	EndOfYear:   "End of this year, end of the day",
}

var (
	// Now is exactly now
	Now Synonym = "now"
	// Today is the beginning of the day today
	Today Synonym = "today"
	// EndOfday is the current date, end of the day
	EndOfday Synonym = "endofday"
	// Tomorrow is the next day, beginning of the day
	Tomorrow Synonym = "tomorrow"
	// StartOfDay is the current date, beginning of the day
	StartOfDay Synonym = "startofday"
	// Yesterday is the previous day, beginning of the day
	Yesterday Synonym = "yesterday"
	// StartOfYear is the start of next year, beginning of day
	StartOfYear Synonym = "startofyear"
	// EndOfYear is the end of this year, end of day
	EndOfYear Synonym = "endofyear"
	// Later is a date faaaaar away
	Later Synonym = "later"
	// EOM is the last day of the current month, end of the day
	EOM Synonym = "eom"
	// EOCM is the end of the current month, end of the day
	EOCM Synonym = "eocm"
	// SOM is the start of the next month, beginning of the day
	SOM Synonym = "som"
	// SOCM is the start of the current month, beginning of the day
	SOCM Synonym = "socm"
	// SOW is the start of the next week, beginning of the day
	SOW Synonym = "sow"
	// SOCW is the start of the current week, beginning of the day
	SOCW Synonym = "socw"
	// EOW is the end of the week, Saturday, beginning of the day
	EOW Synonym = "eow"
	// EOCW is the end of the week, Saturday, beginning of the day
	EOCW Synonym = "eocw"
	// SOWW is the start of the work week, beginning of the day
	SOWW Synonym = "soww"
	// EOWW is the end of the work week, end of the day
	EOWW Synonym = "eoww"

	/*
		Nth represents the n'th day of the month
	*/            // nolint
	First         Synonym = "1st"  // nolint
	Second        Synonym = "2nd"  // nolint
	Third         Synonym = "3rd"  // nolint
	Fourth        Synonym = "4th"  // nolint
	Fifth         Synonym = "5th"  // nolint
	Sixth         Synonym = "6th"  // nolint
	Seventh       Synonym = "7th"  // nolint
	Eight         Synonym = "8th"  // nolint
	Ninth         Synonym = "9th"  // nolint
	Tenth         Synonym = "10th" // nolint
	Eleventh      Synonym = "11th" // nolint
	Twelfth       Synonym = "12th" // nolint
	Thirteenth    Synonym = "13th" // nolint
	Fourteenth    Synonym = "14th" // nolint
	Fifthteenth   Synonym = "15th" // nolint
	Sixteenth     Synonym = "16th" // nolint
	Seventeenth   Synonym = "17th" // nolint
	Eightteenth   Synonym = "18th" // nolint
	Nineteenth    Synonym = "19th" // nolint
	Twentith      Synonym = "20th" // nolint
	TwentyFirst   Synonym = "21st" // nolint
	TwentySecond  Synonym = "22nd" // nolint
	TwentyThird   Synonym = "23rd" // nolint
	TwentyFourth  Synonym = "24th" // nolint
	TwentyFifth   Synonym = "25th" // nolint
	TwentySixth   Synonym = "26th" // nolint
	TwentySeventh Synonym = "27th" // nolint
	TwentyEith    Synonym = "28th" // nolint
	TwentyNinth   Synonym = "29th" // nolint
	Thirtyith     Synonym = "30th" // nolint
	ThirtyFirst   Synonym = "31st" // nolint

	January   Synonym = "january"   // nolint
	February  Synonym = "februrary" // nolint
	March     Synonym = "march"     // nolint
	April     Synonym = "april"     // nolint
	May       Synonym = "may"       // nolint
	June      Synonym = "june"      // nolint
	July      Synonym = "july"      // nolint
	August    Synonym = "august"    // nolint
	September Synonym = "september" // nolint
	October   Synonym = "october"   // nolint
	November  Synonym = "november"  // nolint
	December  Synonym = "december"  // nolint
	Monday    Synonym = "monday"    // nolint
	Tuesday   Synonym = "tuesday"   // nolint
	Wednesday Synonym = "wednesday" // nolint
	Thursday  Synonym = "thursday"  // nolint
	Friday    Synonym = "friday"    // nolint
	Saturday  Synonym = "saturday"  // nolint
	Sunday    Synonym = "sunday"    // nolint
)

type synonymer func(t *time.Time) time.Time

var simpleSynonymMap = map[Synonym]synonymer{
	Now:         func(c *time.Time) time.Time { return *c },
	Today:       func(c *time.Time) time.Time { return floorDay(*c) },
	EndOfday:    func(c *time.Time) time.Time { return ceilDay(*c) },
	Tomorrow:    func(c *time.Time) time.Time { return floorDay(c.AddDate(0, 0, 1)) },
	StartOfDay:  func(c *time.Time) time.Time { return floorDay(c.AddDate(0, 0, 1)) },
	Yesterday:   func(c *time.Time) time.Time { return floorDay(c.AddDate(0, 0, -1)) },
	StartOfYear: func(c *time.Time) time.Time { return time.Date(c.Year()+1, 1, 1, 0, 0, 0, 0, c.Location()) },
	EndOfYear:   func(c *time.Time) time.Time { return time.Date(c.Year(), 12, 31, 0, 0, 0, 0, c.Location()) },
	SOM:         func(c *time.Time) time.Time { return time.Date(c.Year(), c.Month()+1, 1, 0, 0, 0, 0, c.Location()) },
	SOCM:        func(c *time.Time) time.Time { return time.Date(c.Year(), c.Month(), 1, 0, 0, 0, 0, c.Location()) },
	Later:       func(c *time.Time) time.Time { return time.Unix(1<<63-1, 0) },
	EOM: func(c *time.Time) time.Time {
		return time.Date(c.Year(), c.Month(), 1, 23, 59, 59, 999999999, c.Location()).AddDate(0, 1, -1)
	},
	EOCM: func(c *time.Time) time.Time {
		return time.Date(c.Year(), c.Month(), 1, 23, 59, 59, 999999999, c.Location()).AddDate(0, 1, -1)
	},
	SOW:  func(c *time.Time) time.Time { return floorDay(c.AddDate(0, 0, 7-int(c.Weekday()))) },
	SOCW: func(c *time.Time) time.Time { return floorDay(c.AddDate(0, 0, -int(c.Weekday()))) },
	EOW:  func(c *time.Time) time.Time { return ceilDay(c.AddDate(0, 0, 6-int(c.Weekday()))) },
	EOCW: func(c *time.Time) time.Time { return ceilDay(c.AddDate(0, 0, 6-int(c.Weekday()))) },
	SOWW: func(c *time.Time) time.Time {
		y, m, d := c.AddDate(0, 0, int((7+(time.Monday-c.Weekday()))%7)).Date()
		return time.Date(y, m, d, 0, 0, 0, 0, c.Location())
	},
	EOWW: func(c *time.Time) time.Time {
		y, m, d := c.AddDate(0, 0, int((7+(time.Friday-c.Weekday()))%7)).Date()
		return time.Date(y, m, d, 23, 59, 59, 999999999, c.Location())
	},
	First:         func(c *time.Time) time.Time { return nthDay(1, *c) },
	Second:        func(c *time.Time) time.Time { return nthDay(2, *c) },
	Third:         func(c *time.Time) time.Time { return nthDay(3, *c) },
	Fourth:        func(c *time.Time) time.Time { return nthDay(4, *c) },
	Fifth:         func(c *time.Time) time.Time { return nthDay(5, *c) },
	Sixth:         func(c *time.Time) time.Time { return nthDay(6, *c) },
	Seventh:       func(c *time.Time) time.Time { return nthDay(7, *c) },
	Eight:         func(c *time.Time) time.Time { return nthDay(8, *c) },
	Ninth:         func(c *time.Time) time.Time { return nthDay(9, *c) },
	Tenth:         func(c *time.Time) time.Time { return nthDay(10, *c) },
	Eleventh:      func(c *time.Time) time.Time { return nthDay(11, *c) },
	Twelfth:       func(c *time.Time) time.Time { return nthDay(12, *c) },
	Thirteenth:    func(c *time.Time) time.Time { return nthDay(13, *c) },
	Fourteenth:    func(c *time.Time) time.Time { return nthDay(14, *c) },
	Fifthteenth:   func(c *time.Time) time.Time { return nthDay(15, *c) },
	Sixteenth:     func(c *time.Time) time.Time { return nthDay(16, *c) },
	Seventeenth:   func(c *time.Time) time.Time { return nthDay(17, *c) },
	Eightteenth:   func(c *time.Time) time.Time { return nthDay(18, *c) },
	Nineteenth:    func(c *time.Time) time.Time { return nthDay(19, *c) },
	Twentith:      func(c *time.Time) time.Time { return nthDay(20, *c) },
	TwentyFirst:   func(c *time.Time) time.Time { return nthDay(21, *c) },
	TwentySecond:  func(c *time.Time) time.Time { return nthDay(22, *c) },
	TwentyThird:   func(c *time.Time) time.Time { return nthDay(23, *c) },
	TwentyFourth:  func(c *time.Time) time.Time { return nthDay(24, *c) },
	TwentyFifth:   func(c *time.Time) time.Time { return nthDay(25, *c) },
	TwentySixth:   func(c *time.Time) time.Time { return nthDay(26, *c) },
	TwentySeventh: func(c *time.Time) time.Time { return nthDay(27, *c) },
	TwentyEith:    func(c *time.Time) time.Time { return nthDay(28, *c) },
	TwentyNinth:   func(c *time.Time) time.Time { return nthDay(29, *c) },
	Thirtyith:     func(c *time.Time) time.Time { return nthDay(30, *c) },
	ThirtyFirst:   func(c *time.Time) time.Time { return nthDay(31, *c) },
}

func nthDay(d int, c time.Time) time.Time {
	if d == c.Day() {
		return floorDay(c.AddDate(0, 1, 0))
	}
	year, month, day := c.AddDate(0, 0, (31+(d-c.Day()))%31).Date()
	return time.Date(year, month, day, 0, 0, 0, 0, c.Location())
}

func (c Calendar) synonymerWithSynoym(s Synonym) (synonymer, error) { // nolint:funlen
	m := map[Synonym]synonymer{
		January:   func(t *time.Time) time.Time { return c.calcMonth(time.January) },
		February:  func(t *time.Time) time.Time { return c.calcMonth(time.February) },
		March:     func(t *time.Time) time.Time { return c.calcMonth(time.March) },
		April:     func(t *time.Time) time.Time { return c.calcMonth(time.April) },
		May:       func(t *time.Time) time.Time { return c.calcMonth(time.May) },
		June:      func(t *time.Time) time.Time { return c.calcMonth(time.June) },
		July:      func(t *time.Time) time.Time { return c.calcMonth(time.July) },
		August:    func(t *time.Time) time.Time { return c.calcMonth(time.August) },
		September: func(t *time.Time) time.Time { return c.calcMonth(time.September) },
		October:   func(t *time.Time) time.Time { return c.calcMonth(time.October) },
		November:  func(t *time.Time) time.Time { return c.calcMonth(time.November) },
		December:  func(t *time.Time) time.Time { return c.calcMonth(time.December) },
		Monday:    func(t *time.Time) time.Time { return c.calcDay(time.Monday) },
		Tuesday:   func(t *time.Time) time.Time { return c.calcDay(time.Tuesday) },
		Wednesday: func(t *time.Time) time.Time { return c.calcDay(time.Wednesday) },
		Thursday:  func(t *time.Time) time.Time { return c.calcDay(time.Thursday) },
		Friday:    func(t *time.Time) time.Time { return c.calcDay(time.Friday) },
		Saturday:  func(t *time.Time) time.Time { return c.calcDay(time.Saturday) },
		Sunday:    func(t *time.Time) time.Time { return c.calcDay(time.Sunday) },
	}
	if sr, ok := m[s]; ok {
		return sr, nil
	}

	if sr, ok := simpleSynonymMap[s]; ok {
		return sr, nil
	}
	return nil, fmt.Errorf("unknown synonymerWithSynonym: %v", s)
}

func simpleSynonymWithAlias(a string) (*Synonym, error) {
	for ak, av := range aliases {
		if string(ak) == a {
			s := Synonym(a)
			return &s, nil
		}
		for _, v := range av {
			if v == a {
				return &ak, nil
			}
		}
	}
	s := Synonym(a)
	if s == "" {
		return nil, errors.New("empty Synonym")
	}
	return &s, nil
}

func synonymWithAlias(a string) (*Synonym, error) {
	for ak, av := range aliases {
		if string(ak) == a {
			s := Synonym(a)
			return &s, nil
		}
		for _, v := range av {
			if v == a {
				return &ak, nil
			}
		}
	}
	return nil, fmt.Errorf("unknown synonym: %v", a)
}

func (c Calendar) getSynonymer(s string) (synonymer, error) {
	syn, err := simpleSynonymWithAlias(s)
	if err != nil {
		return nil, err
	}
	if syn == nil {
		return nil, errors.New("nil synonym")
	}
	simpleSyn, cerr := c.synonymerWithSynoym(*syn)
	if cerr != nil {
		return nil, cerr
	}
	return simpleSyn, nil

	/*
		syn, err = synonymWithAlias(s)
		if err != nil {
			return nil, fmt.Errorf("no synonym aliases for: %v", s)
		}

		if syn == nil {
			return nil, errors.New("nil synonym")
		}

		final, err := c.synonymerWithSynoym(*syn)
		if err != nil {
			return nil, err
		}

		return final, err
	*/
}
