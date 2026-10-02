package environment

import (
	"reflect"
	"testing"
)

func TestRubyEnvUsages_SupportedForms(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  []usageHit
	}{
		{name: "double-quoted bracket", input: `v = ENV["DATABASE_URL"]`, want: []usageHit{{Name: "DATABASE_URL", Line: 1}}},
		{name: "single-quoted bracket", input: `v = ENV['DATABASE_URL']`, want: []usageHit{{Name: "DATABASE_URL", Line: 1}}},
		{name: "fetch double-quoted", input: `v = ENV.fetch("DATABASE_URL")`, want: []usageHit{{Name: "DATABASE_URL", Line: 1}}},
		{name: "fetch single-quoted", input: `v = ENV.fetch('DATABASE_URL')`, want: []usageHit{{Name: "DATABASE_URL", Line: 1}}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := rubyEnvUsages([]byte(tt.input))
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("rubyEnvUsages() = %+v, want %+v", got, tt.want)
			}
		})
	}
}

func TestRubyEnvUsages_IgnoredCases(t *testing.T) {
	tests := []string{
		`# ENV["FAKE"]`,
		`note = "ENV['FAKE']"`,
		`key = "X"; v = ENV[key]`,
		`MyENV["FAKE"]`,
	}
	for _, input := range tests {
		got := rubyEnvUsages([]byte(input))
		if len(got) != 0 {
			t.Errorf("rubyEnvUsages(%q) = %+v, want none", input, got)
		}
	}
}
