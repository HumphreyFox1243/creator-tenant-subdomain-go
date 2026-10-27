package main

import "testing"

func TestValidTenantLabels(t *testing.T) {
	cases := []struct {
		input string
		want  bool
	}{
		{"studio-7", true}, {"", false}, {"Studio", false}, {"-west", false}, {"west-", false},
	}
	for _, tc := range cases {
		t.Run(tc.input, func(t *testing.T) {
			if got := validLabel(tc.input); got != tc.want {
				t.Fatalf("validLabel(%q) = %v, want %v", tc.input, got, tc.want)
			}
		})
	}
}
