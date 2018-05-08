package parser

import (
	"bytes"
	"fmt"
	"reflect"
	"strconv"
	"strings"
	"testing"
)

// Supported values and formats
func TestParseList(t *testing.T) {
	// Supported
	t.Run("anyValue set first bit", th(parseListReq("*", frame{}), 1<<0, false))
	t.Run("list set first bit", th(parseListReq("0", frame{0, 1}), 1<<0, false))
	t.Run("list set multiple bits", th(parseListReq("0,1", frame{0, 1}), 1<<0|1<<1, false))
	t.Run("range set multiple bits", th(parseListReq("1-2", frame{1, 2}), 1<<1|1<<2, false))
	t.Run("anyValue with step", th(parseListReq("*/2", frame{0, 3}), 1<<0|1<<2, false))
	t.Run("range with step", th(parseListReq("0-5/2", frame{0, 9}), 1<<0|1<<2|1<<4, false))
	t.Run("range with step and offset", th(parseListReq("1-5/2", frame{0, 9}), 1<<1|1<<3|1<<5, false))
	t.Run("range with step > frame.max", th(parseListReq("1-5/20", frame{0, 9}), 1<<1, false))
	t.Run("list, range, list", th(parseListReq("4,3-5,2", frame{2, 5}), 1<<4|1<<3|1<<5|1<<2, false))
	t.Run("zero step", th(parseListReq("*/0", frame{0, 3}), 0, false))
	t.Run("reversed range", th(parseListReq("3-1", frame{0, 3}), 1<<0|1<<1|1<<3, false))
	t.Run("reversed range with step", th(parseListReq("3-1/2", frame{0, 3}), 1<<0|1<<3, false))

	// Not supported
	t.Run("empty value", th(parseListReq("", frame{0, 3}), 0, true))
	t.Run("invalid character", th(parseListReq("i", frame{0, 3}), 0, true))
	t.Run("list values out of frame", th(parseListReq("1,2", frame{}), 0, true))
	t.Run("range values out of frame", th(parseListReq("1-2", frame{}), 0, true))
	t.Run("invalid range format", th(parseListReq("1-2-3", frame{0, 3}), 0, true))
	t.Run("incomplete range format", th(parseListReq("-1", frame{0, 3}), 0, true))
	t.Run("invalid chars in before step", th(parseListReq("p/1", frame{0, 3}), 0, true))
	t.Run("invalid chars in step", th(parseListReq("1/p", frame{0, 3}), 0, true))
	t.Run("invalid chars in range from", th(parseListReq("p-1", frame{0, 3}), 0, true))
	t.Run("invalid chars in range to", th(parseListReq("1-p", frame{0, 3}), 0, true))
	t.Run("invalid chars in range with step", th(parseListReq("1-p/4", frame{0, 3}), 0, true))
	t.Run("invalid chars in step with range", th(parseListReq("1-2/p", frame{0, 3}), 0, true))

	// Reset step to 100
	t.Run("reset step to 100", th(parseListReq("*/1000", frame{0, 0}), 1, false))
}

func TestAround(t *testing.T) {
	// 5-2 -> 5 6 7 0 1 2
	t.Run("non-standard step format", th(parseListReq("5-2", frame{0, 7}), 1<<0|1<<1|1<<2|1<<5|1<<6|1<<7, false))
	t.Run("non-standard step format", th(parseListReq("5-2", frame{0, 3}), 0, true))
}

func TestNonStandard(t *testing.T) {
	// 0 1 2 3
	//   1
	t.Run("non-standard step format", th(parseListReq("1/3", frame{0, 3}), 1<<1, false))
	// 0 1 2 3
	//   1 1 1
	t.Run("non-standard step with list", th(parseListReq("1,2,3/8", frame{0, 3}), 1<<1|1<<2|1<<3, false))
}

func TestFieldsFrame(t *testing.T) {
	// 0-59
	t.Run("Minute", func(t *testing.T) {
		fr := frames["minute"]
		t.Run("input in frame", th(parseListReq("0-59", fr), (1<<(fr.max+1)-1), false))
		t.Run("input out of frame", th(parseListReq("60", fr), 0, true))
	})

	// 0-23
	t.Run("Hour", func(t *testing.T) {
		fr := frames["hour"]
		t.Run("input in frame", th(parseListReq("0-23", fr), (1<<(fr.max+1)-1), false))
		t.Run("input out of frame", th(parseListReq("0-24", fr), 0, true))
		t.Run("input 0", th(parseListReq("0", fr), 1, false))
	})

	// 1-31
	t.Run("DayOfMonth", func(t *testing.T) {
		fr := frames["dayOfMonth"]
		t.Run("input in frame", th(parseListReq("1-31", fr), (1<<(fr.max)-1)<<1, false))
		t.Run("input out of frame", th(parseListReq("0,1,2", fr), 0, true))
	})

	// 1-12
	t.Run("Month", func(t *testing.T) {
		fr := frames["month"]
		t.Run("input in frame", th(parseListReq("1,2,11-12", fr), 1<<1|1<<2|1<<11|1<<12, false))
		t.Run("input out of frame", th(parseListReq("1,2,12-13", fr), 0, true))
	})

	// 0-7
	t.Run("DayOfWeek", func(t *testing.T) {
		fr := frames["dayOfWeek"]
		t.Run("input in frame", th(parseListReq("4,3-5,2", fr), 1<<4|1<<3|1<<5|1<<2, false))
		t.Run("input out of frame", th(parseListReq("6,7,8,5", fr), 0, true))
	})
}

func TestParse(t *testing.T) {
	t.Run("Valid input", func(t *testing.T) {
		inp := "*/15   0   1,15   *   1-5   /usr/bin/find"
		exp := &Schedule{
			Minutes:     1<<0 | 1<<15 | 1<<30 | 1<<45,
			Hours:       1 << 0,
			DaysOfMonth: 1<<1 | 1<<15,
			Months:      (1<<(12) - 1) << 1,
			DaysOfWeek:  1<<1 | 1<<2 | 1<<3 | 1<<4 | 1<<5,
			Command:     "/usr/bin/find",
		}
		got, err := Parse(inp)

		if err != nil {
			t.Errorf("exp no err got %v", err)
		}
		if !reflect.DeepEqual(exp, got) {
			t.Errorf("\nexp %v\ngot %v", exp, got)
		}
	})

	t.Run("Missing/invalid input, exp err", func(t *testing.T) {
		tt := []string{
			"a * * * * cmd",
			"* */3 1,-2 * * cmd",
			"* */3 1-2 9- * cmd",
			"* */3 1-2 9 8 cmd",
			"",
		}
		for _, tc := range tt {
			t.Run(tc, func(t *testing.T) {
				got, err := Parse(tc)
				if got != nil {
					t.Errorf("%v exp nil schedule", tc)
				}
				if err == nil {
					t.Errorf("%v exp err got nil", tc)
				}
			})
		}
	})

	t.Run("Command is not modified", func(t *testing.T) {
		exp := "cmd \t     \t    \t \n"
		inp := "* * * * * " + exp
		got, err := Parse(inp)
		if err != nil {
			t.Errorf("exp no err, got %v", err)
		}
		if got.Command != exp {
			t.Errorf("exp %s got %s", exp, got.Command)
		}
	})
}

func rangeToString(min, max uint8) string {
	arr := make([]string, max-min+1)
	for i := range arr {
		arr[i] = strconv.Itoa(int(min) + i)
	}
	return strings.Join(arr, " ")
}

func TestPrintMethodsAndTable(t *testing.T) {
	schedule := &Schedule{
		Minutes:     0xFFFFFFFFFFFFFFF,
		Hours:       0xFFFFFF,
		DaysOfMonth: 0xFFFFFFFF,
		Months:      0xFFFF,
		DaysOfWeek:  0xFF,
		Command:     "bla bla",
	}

	tt := []struct {
		exp string
		inp func() string
		fr  frame
	}{
		{"minute", schedule.PrintMinutes, frames["minute"]},
		{"hour", schedule.PrintHours, frames["hour"]},
		{"day of month", schedule.PrintDaysOfMonth, frames["dayOfMonth"]},
		{"month", schedule.PrintMonths, frames["month"]},
		{"day of week", schedule.PrintDaysOfWeek, frames["dayOfWeek"]},
		{"command", schedule.PrintCommand, frame{0, 0}},
	}

	var expBuf bytes.Buffer
	for _, tc := range tt {
		t.Run(tc.exp, func(t *testing.T) {
			got := tc.inp()
			exp := fmt.Sprintf("%-14s %s", tc.exp, rangeToString(tc.fr.min, tc.fr.max))
			if tc.exp == "command" {
				exp = fmt.Sprintf("%-14s %s", tc.exp, schedule.Command)
			}
			if !strings.HasPrefix(got, exp) {
				t.Errorf("\nexp %v\ngot %v", exp, got)
			}
			expBuf.WriteString(exp)
			expBuf.WriteString("\n")
		})
	}

	var gotBuf bytes.Buffer
	schedule.PrintTable(&gotBuf)
	if gotBuf.String() != expBuf.String() {
		t.Errorf("\ngot\n%v\nexp\n%v", expBuf.String(), gotBuf.String())
	}
}

type parseListResp func() (uint64, error)

func parseListReq(s string, fr frame) parseListResp {
	return func() (uint64, error) {
		return parseList(s, fr)
	}
}

func th(fn parseListResp, exp uint64, expErr bool) func(*testing.T) {
	return func(t *testing.T) {
		got, err := fn()
		if expErr && err == nil {
			t.Errorf("expected err, got nil")
		}

		if got != exp {
			t.Errorf("\nexp %040b\ngot %040b", exp, got)
		}
	}
}

func TestExtractField(t *testing.T) {
	tt := []struct {
		inp, exp string
	}{
		{" */ ", "*/"},
		{"\t*/\t", "*/"},
		{"\t\t  1\t ", "1"},
	}
	for _, tc := range tt {
		t.Run(tc.inp, func(t *testing.T) {
			_, got := extractField(tc.inp)
			if got != tc.exp {
				t.Errorf("got %v exp %v", tc.exp, got)
			}
		})
	}
}
