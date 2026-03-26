package rest

import (
	"encoding/json"
	"testing"
)

func Test_parseXput(t *testing.T) {
	tests := []struct {
		input string
		want  Xput
		isErr bool
	}{
		// Adaptive QOS uses IOPS/TB and IOPS/GB forms
		{input: "6144IOPS/TB", want: Xput{IOPS: "6144", Mbps: ""}},
		{input: "6144IOPS/GB", want: Xput{IOPS: "6144000", Mbps: ""}},

		{input: "100IOPS", want: Xput{IOPS: "100", Mbps: ""}},
		{input: "100iops", want: Xput{IOPS: "100", Mbps: ""}},
		{input: "111111IOPS", want: Xput{IOPS: "111111", Mbps: ""}},
		{input: "0", want: Xput{IOPS: "", Mbps: ""}},
		{input: "", want: Xput{IOPS: "", Mbps: ""}},
		{input: "INF", want: Xput{IOPS: "", Mbps: ""}},

		{input: "1GB/s", want: Xput{IOPS: "", Mbps: "1000"}},
		{input: "100B/s", want: Xput{IOPS: "", Mbps: "0"}},
		{input: "10KB/s", want: Xput{IOPS: "", Mbps: "0.01"}},
		{input: "1mb/s", want: Xput{IOPS: "", Mbps: "1"}},
		{input: "1tb/s", want: Xput{IOPS: "", Mbps: "1000000"}},
		{input: "1000KB/s", want: Xput{IOPS: "", Mbps: "1"}},
		{input: "15000IOPS,468.8MB/s", want: Xput{IOPS: "15000", Mbps: "468.8"}},
		{input: "50000IOPS,1.53GB/s", want: Xput{IOPS: "50000", Mbps: "1530"}},

		{input: "1 foople/s", isErr: true},
	}
	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got, err := parseXput(tt.input)
			if tt.isErr {
				if err == nil {
					t.Errorf("parseXput(%q) expected error, got nil", tt.input)
				}
				return
			}
			if err != nil {
				t.Errorf("parseXput(%q) unexpected error: %v", tt.input, err)
				return
			}
			if got.IOPS != tt.want.IOPS {
				t.Errorf("parseXput(%q).IOPS = %q, want %q", tt.input, got.IOPS, tt.want.IOPS)
			}
			if got.Mbps != tt.want.Mbps {
				t.Errorf("parseXput(%q).Mbps = %q, want %q", tt.input, got.Mbps, tt.want.Mbps)
			}
		})
	}
}

func Test_xputField_UnmarshalJSON(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
		isErr bool
	}{
		{name: "string value", input: `"50000IOPS"`, want: "50000IOPS"},
		{name: "string INF", input: `"INF"`, want: "INF"},
		{name: "string bps", input: `"1.53GB/s"`, want: "1.53GB/s"},
		{name: "bare zero int", input: `0`, want: "0"},
		{name: "bare integer", input: `12345`, want: "12345"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var f xputField
			err := json.Unmarshal([]byte(tt.input), &f)
			if tt.isErr {
				if err == nil {
					t.Errorf("UnmarshalJSON(%s) expected error, got nil", tt.input)
				}
				return
			}
			if err != nil {
				t.Errorf("UnmarshalJSON(%s) unexpected error: %v", tt.input, err)
				return
			}
			if string(f) != tt.want {
				t.Errorf("UnmarshalJSON(%s) = %q, want %q", tt.input, string(f), tt.want)
			}
		})
	}
}
