package cron

import (
	"fmt"
	"io"
	"strings"
	"unicode"

	"github.com/pkg/errors"
)

type frame struct {
	min, max uint8
}

var frames = map[string]frame{
	"minute":     frame{min: 0, max: 59},
	"hour":       frame{min: 0, max: 23},
	"dayOfMonth": frame{min: 1, max: 31},
	"month":      frame{min: 1, max: 12},
	"dayOfWeek":  frame{min: 0, max: 7},
}

// Schedule stores times of execution for each field and a command
type Schedule struct {
	Minutes     uint64
	Hours       uint64
	DaysOfMonth uint64
	Months      uint64
	DaysOfWeek  uint64
	Command     string
}

// Parse parses a string input, return a Schedule type and stops on invalid input
// by returning an error, accepted input "Minute Hour Day of Month Month Day of Week"
// [] 0/1; {} 0+; | OR
// Line = {Spaces} List Spaces List Spaces List Spaces List Spaces List Spaces Command .
//                  minute      hour        dom         month       dow
// List  = Range { "," Range }.
// Range = "*" ["/" Number] | Number ["-" Number] ["/" Number].
func Parse(seq string) (*Schedule, error) {
	seq, minutes := extractField(seq)
	seq, hours := extractField(seq)
	seq, daysOfMounth := extractField(seq)
	seq, months := extractField(seq)
	seq, daysOfWeek := extractField(seq)

	if seq == "" {
		return nil, errors.New("invalid sequance, expecting Minute Hour Day Month Command")
	}
	command := strings.TrimLeftFunc(seq, tabSpaceFn)

	efp := &errFieldParser{}
	sch := &Schedule{
		Minutes:     efp.parseField(minutes, "minute"),
		Hours:       efp.parseField(hours, "hour"),
		DaysOfMonth: efp.parseField(daysOfMounth, "dayOfMonth"),
		Months:      efp.parseField(months, "month"),
		DaysOfWeek:  efp.parseField(daysOfWeek, "dayOfWeek"),
		Command:     command,
	}

	if efp.err != nil {
		return nil, efp.err
	}

	return sch, nil
}

type errFieldParser struct{ err error }

func (e *errFieldParser) parseField(s, field string) (val uint64) {
	if e.err != nil {
		return
	}
	val, e.err = parseList(s, frames[field])
	return
}

var tabSpaceFn = func(r rune) bool { return r == ' ' || r == '\t' }

func extractField(s string) (string, string) {
	s = strings.TrimLeftFunc(s, tabSpaceFn)

	idx := 0
	for idx < len(s) && !unicode.IsSpace(rune(s[idx])) {
		idx++
	}
	return s[idx:], s[:idx]
}

// PrintTable outputs a table of all supported types
func (s *Schedule) PrintTable(w io.Writer) {
	fmt.Fprintf(w, "%s\n", s.PrintMinutes())
	fmt.Fprintf(w, "%s\n", s.PrintHours())
	fmt.Fprintf(w, "%s\n", s.PrintDaysOfMonth())
	fmt.Fprintf(w, "%s\n", s.PrintMonths())
	fmt.Fprintf(w, "%s\n", s.PrintDaysOfWeek())
	fmt.Fprintf(w, "%s\n", s.PrintCommand())
}

// PrintMinutes outputs the list of minutes with `minute`
// as a prefix, ranges from 0-59
func (s *Schedule) PrintMinutes() string {
	return prefixPrint("minute", s.Minutes, frames["minute"])
}

// PrintHours outputs the list of hours with `hour`
// as a prefix; ranges from 0-23
func (s *Schedule) PrintHours() string {
	return prefixPrint("hour", s.Hours, frames["hour"])
}

// PrintDaysOfMonth outputs the list of days with `day of month`
// as a prefix; ranges from 1-31
func (s *Schedule) PrintDaysOfMonth() string {
	return prefixPrint("day of month", s.DaysOfMonth, frames["dayOfMonth"])
}

// PrintMonths outputs the list of days with `day of month`
// as a prefix; ranges from 1-12
func (s *Schedule) PrintMonths() string {
	return prefixPrint("month", s.Months, frames["month"])
}

// PrintDaysOfWeek outputs the list of days with `day of week`
// as a prefix; ranges from 0-7
func (s *Schedule) PrintDaysOfWeek() string {
	return prefixPrint("day of week", s.DaysOfWeek, frames["dayOfWeek"])
}

// PrintCommand outputs the specified command with `command` as prefix
func (s *Schedule) PrintCommand() string {
	return fmt.Sprintf("%-14s %s", "command", s.Command)
}

func prefixPrint(prefix string, val uint64, fr frame) string {
	var b strings.Builder
	b.WriteString(fmt.Sprintf("%-14s", prefix))
	print(&b, val, fr)
	return b.String()
}

func print(b *strings.Builder, val uint64, fr frame) {
	for i := fr.min; i <= fr.max; i++ {
		if (val>>i)&1 != 0 {
			b.WriteString(fmt.Sprintf(" %d", i))
		}
	}
}

func parseList(s string, fr frame) (uint64, error) {
	var acc uint64
	fields := strings.Split(s, ",")
	for _, field := range fields {
		res, err := parseExp(field, fr)
		if err != nil {
			return 0, err
		}
		acc |= res
	}
	return acc, nil
}

func parseExp(s string, fr frame) (uint64, error) {
	if s == "" {
		return 0, errors.New("empty sequance")
	}

	seqs := strings.Split(s, "/")
	rng := strings.Split(seqs[0], "-")
	if len(rng) == 0 || len(rng) > 2 || rng[0] == "" {
		return 0, errors.Errorf("not supported chars in %v", s)
	}

	var err error
	var from, to, inc uint8
	if len(rng) == 1 {
		if rng[0] == "*" {
			from, to = fr.min, fr.max
		} else {
			from, err = parseInt(rng[0])
			if err != nil {
				return 0, err
			}
			to = from
		}
	} else if len(rng) == 2 {
		from, err = parseInt(rng[0])
		if err != nil {
			return 0, err
		}
		to, err = parseInt(rng[1])
		if err != nil {
			return 0, err
		}
	}

	if len(seqs) == 1 {
		inc = 1
	} else if len(seqs) == 2 {
		inc, err = parseInt(seqs[1])
		if err != nil {
			return 0, err
		}
	}
	return fillSet(from, to, inc, fr)
}

func fillSet(from, to, inc uint8, fr frame) (uint64, error) {
	var acc uint64

	if from < fr.min || fr.max < from {
		return acc, errors.Errorf("out of range %v", from)
	}
	if to < fr.min || fr.max < to {
		return acc, errors.Errorf("out of range %v", to)
	}
	if inc == 0 {
		return acc, errors.New("step needs to be > then 0")
	}

	if to < from {
		acc, err := fillSet(from, fr.max, inc, fr)
		if err != nil {
			return 0, err
		}
		newAcc, err := fillSet(fr.min, to, inc, fr)
		if err != nil {
			return 0, err
		}
		return acc | newAcc, nil
	}

	if inc == 1 {
		acc = (1<<(to-from+1) - 1) << from
	} else {
		for i := from; i <= to; i += inc {
			acc |= 1 << i
		}
	}
	return acc, nil
}

const iNF uint8 = 100

func parseInt(s string) (uint8, error) {
	var acc uint8
	for i, r := range s {
		if r < '0' || '9' < r {
			return 0, errors.Errorf("invalid digit %v", s)
		}
		acc = acc*10 + s[i] - '0'
	}
	if acc > iNF {
		acc = iNF
	}
	return acc, nil
}
