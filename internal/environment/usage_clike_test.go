package environment

import (
	"reflect"
	"testing"
)

func TestFindLiteralCallUsages_SupportedFormAndLineNumber(t *testing.T) {
	input := "a();\nSystem.getenv(\"FOO\");\n"
	got := findLiteralCallUsages([]byte(input), []string{"System.getenv("})
	want := []usageHit{{Name: "FOO", Line: 2}}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("findLiteralCallUsages() = %+v, want %+v", got, want)
	}
}

func TestFindLiteralCallUsages_IgnoredCases(t *testing.T) {
	tests := []struct {
		name  string
		input string
	}{
		{name: "line comment", input: `// System.getenv("FAKE");`},
		{name: "block comment", input: "/*\nSystem.getenv(\"FAKE\");\n*/"},
		{name: "inside a string literal", input: `String s = "System.getenv(\"FAKE\")";`},
		{name: "dynamic argument", input: `System.getenv(name);`},
		{name: "unrelated identifier containing the prefix as a suffix", input: `MySystem.getenv("FAKE");`},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := findLiteralCallUsages([]byte(tt.input), []string{"System.getenv("})
			if len(got) != 0 {
				t.Errorf("findLiteralCallUsages(%q) = %+v, want none", tt.input, got)
			}
		})
	}
}

func TestFindLiteralCallUsages_QualifiedPrefixAlsoMatchesUnqualified(t *testing.T) {
	// System.Environment.GetEnvironmentVariable(...) must be recognized via
	// the bare Environment.GetEnvironmentVariable( prefix.
	input := `System.Environment.GetEnvironmentVariable("FOO");`
	got := findLiteralCallUsages([]byte(input), []string{"Environment.GetEnvironmentVariable("})
	want := []usageHit{{Name: "FOO", Line: 1}}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("findLiteralCallUsages() = %+v, want %+v", got, want)
	}
}

func TestFindLiteralCallUsages_DuplicateUsagesDeterministic(t *testing.T) {
	input := "getenv(\"FOO\");\ngetenv(\"FOO\");\n"
	got := findLiteralCallUsages([]byte(input), []string{"getenv("})
	want := []usageHit{{Name: "FOO", Line: 1}, {Name: "FOO", Line: 2}}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("findLiteralCallUsages() = %+v, want %+v", got, want)
	}
}
