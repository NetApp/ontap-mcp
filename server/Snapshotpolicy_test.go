package server

import (
	"github.com/netapp/ontap-mcp/ontap"
	"slices"
	"strings"
	"testing"
)

func TestConvertCron(t *testing.T) {
	tests := []struct {
		name           string
		cronExpression string
		wantErrorText  string
		wantMinutes    []int
		wantHours      []int
		wantDays       []int
		wantMonths     []int
		wantWeekdays   []int
	}{
		{
			name:           "wrong expression",
			cronExpression: "*/5 * * * *",
			wantErrorText:  "wrong cron format",
		},
		{
			name:           "At 5 minutes past the hour",
			cronExpression: "5 * * * *",
			wantErrorText:  "",
			wantMinutes:    []int{5},
		},
		{
			name:           "At 01:00 AM and 02:00 AM",
			cronExpression: "0 1,2 * * *",
			wantErrorText:  "",
			wantMinutes:    []int{0},
			wantHours:      []int{1, 2},
		},
		{
			name:           "Every minute, on day 2 of the month",
			cronExpression: "* * 2 * *",
			wantErrorText:  "",
			wantMinutes:    []int{0},
			wantDays:       []int{2},
		},
		{
			name:           "Every minute, on day 5 of the month, only in February and March",
			cronExpression: "* * 5 2,3 *",
			wantErrorText:  "",
			wantDays:       []int{5},
			wantMonths:     []int{2, 3},
		},
		{
			name:           "Every minute, on day 11 of the month, only in January, February and March",
			cronExpression: "* * 11 1-3 *",
			wantErrorText:  "",
			wantDays:       []int{11},
			wantMonths:     []int{1, 2, 3},
		},
		{
			name:           "Every minute, only on Wednesday",
			cronExpression: "* * * * 3",
			wantErrorText:  "",
			wantWeekdays:   []int{3},
		},
		{
			name:           "four fields are rejected",
			cronExpression: "10 * * *",
			wantErrorText:  "expected 5 fields",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			out := ontap.Schedule{
				Name: tt.name,
				Cron: ontap.Cron{
					Minutes:  make([]int, 0),
					Hours:    make([]int, 0),
					Days:     make([]int, 0),
					Months:   make([]int, 0),
					Weekdays: make([]int, 0),
				},
			}
			got := convertCron(tt.cronExpression, &out)
			if got != nil && !strings.Contains(got.Error(), tt.wantErrorText) {
				t.Errorf("convertCron(%q) = %v", tt.cronExpression, got.Error())
			}

			if len(out.Cron.Minutes) > 0 && !slices.Equal(out.Cron.Minutes, tt.wantMinutes) {
				t.Errorf("%s convertCron(%q) Minutes = %v, want = %v", tt.name, tt.cronExpression, out.Cron.Minutes, tt.wantMinutes)
			}
			if len(out.Cron.Hours) > 0 && !slices.Equal(out.Cron.Hours, tt.wantHours) {
				t.Errorf("%s convertCron(%q) Hours = %v, want = %v", tt.name, tt.cronExpression, out.Cron.Hours, tt.wantHours)
			}
			if len(out.Cron.Days) > 0 && !slices.Equal(out.Cron.Days, tt.wantDays) {
				t.Errorf("%s convertCron(%q) Days = %v, want = %v", tt.name, tt.cronExpression, out.Cron.Days, tt.wantDays)
			}
			if len(out.Cron.Months) > 0 && !slices.Equal(out.Cron.Months, tt.wantMonths) {
				t.Errorf("%s convertCron(%q) Months = %v, want = %v", tt.name, tt.cronExpression, out.Cron.Months, tt.wantMonths)
			}
			if len(out.Cron.Weekdays) > 0 && !slices.Equal(out.Cron.Weekdays, tt.wantWeekdays) {
				t.Errorf("%s convertCron(%q) Weekdays = %v, want = %v", tt.name, tt.cronExpression, out.Cron.Weekdays, tt.wantWeekdays)
			}
		})
	}
}

func TestConvertCronValidation(t *testing.T) {
	tests := []struct {
		name           string
		cronExpression string
		wantErrorText  string
		wantCron       ontap.Cron
	}{
		// boundaries that must be accepted
		{name: "minute 0", cronExpression: "0 * * * *", wantCron: ontap.Cron{Minutes: []int{0}}},
		{name: "minute 59", cronExpression: "59 * * * *", wantCron: ontap.Cron{Minutes: []int{59}}},
		{name: "hour 0", cronExpression: "* 0 * * *", wantCron: ontap.Cron{Hours: []int{0}}},
		{name: "hour 23", cronExpression: "* 23 * * *", wantCron: ontap.Cron{Hours: []int{23}}},
		{name: "day 1", cronExpression: "* * 1 * *", wantCron: ontap.Cron{Days: []int{1}}},
		{name: "day 31", cronExpression: "* * 31 * *", wantCron: ontap.Cron{Days: []int{31}}},
		{name: "month 1", cronExpression: "* * * 1 *", wantCron: ontap.Cron{Months: []int{1}}},
		{name: "month 12", cronExpression: "* * * 12 *", wantCron: ontap.Cron{Months: []int{12}}},
		{name: "weekday 0", cronExpression: "* * * * 0", wantCron: ontap.Cron{Weekdays: []int{0}}},
		{name: "weekday 6", cronExpression: "* * * * 6", wantCron: ontap.Cron{Weekdays: []int{6}}},
		{
			name:           "every field at its upper bound",
			cronExpression: "59 23 31 12 6",
			wantCron:       ontap.Cron{Minutes: []int{59}, Hours: []int{23}, Days: []int{31}, Months: []int{12}, Weekdays: []int{6}},
		},
		{
			name:           "full ranges",
			cronExpression: "0-2 22-23 30-31 11-12 0-6",
			wantCron:       ontap.Cron{Minutes: []int{0, 1, 2}, Hours: []int{22, 23}, Days: []int{30, 31}, Months: []int{11, 12}, Weekdays: []int{0, 1, 2, 3, 4, 5, 6}},
		},
		{name: "single value range", cronExpression: "5-5 * * * *", wantCron: ontap.Cron{Minutes: []int{5}}},
		{name: "mixed list and range", cronExpression: "0,15-17,30 * * * *", wantCron: ontap.Cron{Minutes: []int{0, 15, 16, 17, 30}}},
		{name: "surrounding whitespace", cronExpression: "  5\t1 * * *  ", wantCron: ontap.Cron{Minutes: []int{5}, Hours: []int{1}}},

		// off-by-one neighbours that must be rejected
		{name: "minute 60", cronExpression: "60 0 * * *", wantErrorText: `invalid cron minutes value "60": must be between 0 and 59`},
		{name: "hour 24", cronExpression: "0 24 * * *", wantErrorText: `invalid cron hours value "24": must be between 0 and 23`},
		{name: "day 0", cronExpression: "0 0 0 * *", wantErrorText: `invalid cron days value "0": must be between 1 and 31`},
		{name: "day 32", cronExpression: "0 0 32 * *", wantErrorText: `invalid cron days value "32": must be between 1 and 31`},
		{name: "month 0", cronExpression: "0 0 * 0 *", wantErrorText: `invalid cron months value "0": must be between 1 and 12`},
		{name: "month 13", cronExpression: "0 0 * 13 *", wantErrorText: `invalid cron months value "13": must be between 1 and 12`},
		{name: "weekday 7", cronExpression: "0 0 * * 7", wantErrorText: `invalid cron weekdays value "7": must be between 0 and 6`},
		{name: "range end out of bounds", cronExpression: "0-60 * * * *", wantErrorText: `invalid cron minutes value "0-60": must be between 0 and 59`},
		{name: "range start out of bounds", cronExpression: "* * 0-5 * *", wantErrorText: `invalid cron days value "0-5": must be between 1 and 31`},
		{name: "out of range value in list", cronExpression: "0 1,24 * * *", wantErrorText: `invalid cron hours value "24": must be between 0 and 23`},
		{name: "huge value", cronExpression: "99999999999999999999 * * * *", wantErrorText: `invalid cron minutes value "99999999999999999999": must be between 0 and 59`},

		// malformed tokens
		{name: "trailing garbage", cronExpression: "0 4hours * * *", wantErrorText: `invalid cron hours value "4hours": expected *, a number, or a range such as 1-5`},
		{name: "non-numeric", cronExpression: "0 0 * jan *", wantErrorText: `invalid cron months value "jan": expected *, a number, or a range such as 1-5`},
		{name: "negative value", cronExpression: "-1 * * * *", wantErrorText: `invalid cron minutes value "-1": expected *, a number, or a range such as 1-5`},
		{name: "plus sign", cronExpression: "+5 * * * *", wantErrorText: `invalid cron minutes value "+5": expected *, a number, or a range such as 1-5`},
		{name: "empty token in list", cronExpression: "0,,5 * * * *", wantErrorText: `invalid cron minutes value "": expected *, a number, or a range such as 1-5`},
		{name: "trailing comma", cronExpression: "0 1, * * *", wantErrorText: `invalid cron hours value "": expected *, a number, or a range such as 1-5`},
		{name: "open-ended range", cronExpression: "5- * * * *", wantErrorText: `invalid cron minutes value "5-": expected *, a number, or a range such as 1-5`},
		{name: "double range", cronExpression: "1-2-3 * * * *", wantErrorText: `invalid cron minutes value "1-2-3": expected *, a number, or a range such as 1-5`},
		{name: "star in list", cronExpression: "*,5 * * * *", wantErrorText: `invalid cron minutes value "*": expected *, a number, or a range such as 1-5`},
		{name: "inverted range", cronExpression: "* * * * 5-1", wantErrorText: `invalid cron weekdays range "5-1": start is greater than end`},
		{name: "step syntax", cronExpression: "0 */2 * * *", wantErrorText: "wrong cron format */2 detected"},
		{name: "range step syntax", cronExpression: "0-30/5 * * * *", wantErrorText: "wrong cron format 0-30/5 detected"},

		// field count
		{name: "six fields", cronExpression: "0 0 * * * 2026", wantErrorText: `invalid cron expression "0 0 * * * 2026": expected 5 fields (minutes hours days months weekdays), got 6`},
		{name: "four fields", cronExpression: "0 0 * *", wantErrorText: `invalid cron expression "0 0 * *": expected 5 fields (minutes hours days months weekdays), got 4`},
		{name: "one field", cronExpression: "5", wantErrorText: `invalid cron expression "5": expected 5 fields (minutes hours days months weekdays), got 1`},
		{name: "blank", cronExpression: "   ", wantErrorText: `invalid cron expression "   ": expected 5 fields (minutes hours days months weekdays), got 0`},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var out ontap.Schedule
			err := convertCron(tt.cronExpression, &out)
			if tt.wantErrorText != "" {
				if err == nil {
					t.Fatalf("convertCron(%q) succeeded with %+v, want error %q", tt.cronExpression, out.Cron, tt.wantErrorText)
				}
				if err.Error() != tt.wantErrorText {
					t.Errorf("convertCron(%q) error = %q, want %q", tt.cronExpression, err.Error(), tt.wantErrorText)
				}
				return
			}
			if err != nil {
				t.Fatalf("convertCron(%q) = %v, want no error", tt.cronExpression, err)
			}
			checks := []struct {
				field     string
				got, want []int
			}{
				{"Minutes", out.Cron.Minutes, tt.wantCron.Minutes},
				{"Hours", out.Cron.Hours, tt.wantCron.Hours},
				{"Days", out.Cron.Days, tt.wantCron.Days},
				{"Months", out.Cron.Months, tt.wantCron.Months},
				{"Weekdays", out.Cron.Weekdays, tt.wantCron.Weekdays},
			}
			for _, c := range checks {
				if !slices.Equal(c.got, c.want) {
					t.Errorf("convertCron(%q) %s = %v, want %v", tt.cronExpression, c.field, c.got, c.want)
				}
			}
		})
	}
}
