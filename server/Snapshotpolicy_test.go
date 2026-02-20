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
			name:           "Every 10 minute with half expression",
			cronExpression: "10 * * *",
			wantErrorText:  "",
			wantMinutes:    []int{10},
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
