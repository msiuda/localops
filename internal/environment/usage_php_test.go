package environment

import (
	"reflect"
	"testing"
)

func TestPhpEnvUsages_SupportedForms(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  []usageHit
	}{
		{name: "getenv", input: `$v = getenv("DATABASE_URL");`, want: []usageHit{{Name: "DATABASE_URL", Line: 1}}},
		{name: "_ENV double-quoted", input: `$v = $_ENV["DATABASE_URL"];`, want: []usageHit{{Name: "DATABASE_URL", Line: 1}}},
		{name: "_ENV single-quoted", input: `$v = $_ENV['DATABASE_URL'];`, want: []usageHit{{Name: "DATABASE_URL", Line: 1}}},
		{name: "_SERVER double-quoted", input: `$v = $_SERVER["DATABASE_URL"];`, want: []usageHit{{Name: "DATABASE_URL", Line: 1}}},
		{name: "_SERVER single-quoted", input: `$v = $_SERVER['DATABASE_URL'];`, want: []usageHit{{Name: "DATABASE_URL", Line: 1}}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := phpEnvUsages([]byte(tt.input), false)
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("phpEnvUsages() = %+v, want %+v", got, tt.want)
			}
		})
	}
}

func TestPhpEnvUsages_IgnoredCases(t *testing.T) {
	tests := []string{
		`// getenv("FAKE");`,
		`# getenv("FAKE");`,
		"/*\ngetenv(\"FAKE\");\n*/",
		`$note = "getenv('FAKE')";`,
		`$v = getenv($name);`,
	}
	for _, input := range tests {
		got := phpEnvUsages([]byte(input), false)
		if len(got) != 0 {
			t.Errorf("phpEnvUsages(%q) = %+v, want none", input, got)
		}
	}
}

func TestPhpEnvUsages_LaravelGatedByDetection(t *testing.T) {
	input := `$v = env("DATABASE_URL");`

	got := phpEnvUsages([]byte(input), false)
	if len(got) != 0 {
		t.Errorf("phpEnvUsages(laravelDetected=false) = %+v, want none: env() is not a safe default", got)
	}

	got = phpEnvUsages([]byte(input), true)
	if len(got) != 1 || got[0].Name != "DATABASE_URL" {
		t.Errorf("phpEnvUsages(laravelDetected=true) = %+v, want one DATABASE_URL", got)
	}
}
