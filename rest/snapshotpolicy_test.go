package rest

import (
	"github.com/netapp/ontap-mcp/ontap"
	"slices"
	"strings"
	"testing"
)

func TestValidateSchedule(t *testing.T) {
	tests := []struct {
		name         string
		scheduleName string
		wantSchedule ontap.Schedule
		wantInterval string
		wantDays     []string
		wantHours    []string
		wantMins     []string
		wantMonths   []string
		wantWeeks    []string
		wantErr      string
	}{
		{
			name:         "interval schedule full",
			scheduleName: "PT7M30S",
			wantInterval: "PT7M30S",
			wantErr:      "",
		},
		{
			name:         "interval schedule partial",
			scheduleName: "P5D",
			wantInterval: "P5D",
			wantErr:      "",
		},
		{
			name:         "cron schedule 3h",
			scheduleName: "3 hour",
			wantHours:    []string{"3"},
			wantMins:     []string{"0"},
			wantInterval: "",
			wantErr:      "",
		},
		{
			name:         "cron schedule 8h",
			scheduleName: "8hours",
			wantInterval: "",
			wantHours:    []string{"8"},
			wantMins:     []string{"0"},
			wantErr:      "",
		},
		{
			name:         "cron schedule every week 12 hour 30 mins",
			scheduleName: "2 weekday 12 hour 30 minutes",
			wantInterval: "",
			wantHours:    []string{"12"},
			wantMins:     []string{"30"},
			wantWeeks:    []string{"2"},
			wantErr:      "",
		},
		{
			name:         "invalid duration",
			scheduleName: "invalid",
			wantInterval: "",
			wantErr:      "invalid schedule format:",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			newSchedule, got := validateSchedule(tt.scheduleName)
			if got != nil && !strings.Contains(got.Error(), tt.wantErr) {
				t.Errorf("validateSchedule(%q) = %v, want %v", tt.scheduleName, got.Error(), tt.wantErr)
			}

			if newSchedule.Interval != tt.wantInterval {
				t.Errorf("validateSchedule(%q) = %v, want %v", tt.scheduleName, got, tt.wantInterval)
			}

			if !slices.Equal(newSchedule.Cron.Days, tt.wantDays) {
				t.Errorf("validateSchedule(%q) days got = %v, want %v", tt.scheduleName, newSchedule.Cron.Days, tt.wantDays)
			}
			if !slices.Equal(newSchedule.Cron.Hours, tt.wantHours) {
				t.Errorf("validateSchedule(%q) hours got = %v, want %v", tt.scheduleName, newSchedule.Cron.Hours, tt.wantHours)
			}
			if !slices.Equal(newSchedule.Cron.Minutes, tt.wantMins) {
				t.Errorf("validateSchedule(%q) minutes got = %v, want %v", tt.scheduleName, newSchedule.Cron.Minutes, tt.wantMins)
			}
			if !slices.Equal(newSchedule.Cron.Months, tt.wantMonths) {
				t.Errorf("validateSchedule(%q) months got = %v, want %v", tt.scheduleName, newSchedule.Cron.Months, tt.wantMonths)
			}
			if !slices.Equal(newSchedule.Cron.Weekdays, tt.wantWeeks) {
				t.Errorf("validateSchedule(%q) weekdays got = %v, want %v", tt.scheduleName, newSchedule.Cron.Weekdays, tt.wantMonths)
			}

		})
	}
}
