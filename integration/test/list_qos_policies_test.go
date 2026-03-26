package main

import (
	"context"
	"log/slog"
	"strings"
	"testing"
	"time"
)

func TestListQoSPolicies(t *testing.T) {
	SkipIfMissing(t, CheckTools)

	tests := []struct {
		name             string
		input            string
		expectedOntapErr string
		mustContain      []string
		mustNotContain   []string
	}{
		{
			name:  "List all QoS policies",
			input: ClusterStr + "list all qos policies",
		},
		{
			name:        "List QoS policies for a specific SVM — cluster-scoped must always be present",
			input:       ClusterStr + "list all qos policies for the vtest svm",
			mustContain: []string{"cluster"},
		},
		{
			name:           "List only fixed QoS policies",
			input:          ClusterStr + "list all fixed qos policies",
			mustNotContain: []string{"adaptive"},
		},
		{
			name:           "List only adaptive QoS policies",
			input:          ClusterStr + "list all adaptive qos policies",
			mustNotContain: []string{"fixed"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			slog.Debug("", slog.String("Input", tt.input))
			ctx, cancel := context.WithTimeout(context.Background(), 90*time.Second)
			defer cancel()

			response, err := testAgent.ChatWithResponse(ctx, t, tt.input, tt.expectedOntapErr)
			if err != nil {
				slog.Error("Error processing input", slog.Any("error", err))
				return
			}

			lower := strings.ToLower(response)
			for _, want := range tt.mustContain {
				if !strings.Contains(lower, strings.ToLower(want)) {
					t.Errorf("response missing expected text %q\nfull response: %s", want, response)
				}
			}
			for _, notWant := range tt.mustNotContain {
				if strings.Contains(lower, strings.ToLower(notWant)) {
					t.Errorf("response unexpectedly contains %q\nfull response: %s", notWant, response)
				}
			}
		})
	}
}
