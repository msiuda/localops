package environment

import (
	"reflect"
	"testing"
)

func TestGoEnvUsages_SupportedForms(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  []usageHit
	}{
		{
			name:  "os.Getenv",
			input: "package main\nimport \"os\"\nfunc main() { os.Getenv(\"FOO\") }\n",
			want:  []usageHit{{Name: "FOO", Line: 3}},
		},
		{
			name:  "os.LookupEnv",
			input: "package main\nimport \"os\"\nfunc main() { os.LookupEnv(\"BAR\") }\n",
			want:  []usageHit{{Name: "BAR", Line: 3}},
		},
		{
			name:  "explicit single-level import alias",
			input: "package main\nimport stdos \"os\"\nfunc main() { stdos.Getenv(\"FOO\") }\n",
			want:  []usageHit{{Name: "FOO", Line: 3}},
		},
		{
			name: "duplicate usage/location handling: same variable, two call sites",
			input: "package main\nimport \"os\"\nfunc main() {\n" +
				"\tos.Getenv(\"FOO\")\n\tos.Getenv(\"FOO\")\n}\n",
			want: []usageHit{{Name: "FOO", Line: 4}, {Name: "FOO", Line: 5}},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := goEnvUsages([]byte(tt.input))
			if err != nil {
				t.Fatalf("goEnvUsages() error = %v", err)
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("goEnvUsages() = %+v, want %+v", got, tt.want)
			}
		})
	}
}

func TestGoEnvUsages_IgnoredCases(t *testing.T) {
	tests := []struct {
		name  string
		input string
	}{
		{
			name:  "line comment",
			input: "package main\nimport \"os\"\n// os.Getenv(\"FAKE\")\nfunc main() {}\n",
		},
		{
			name:  "block comment",
			input: "package main\nimport \"os\"\n/*\nos.Getenv(\"FAKE\")\n*/\nfunc main() {}\n",
		},
		{
			name:  "dynamic argument is not statically knowable",
			input: "package main\nimport \"os\"\nfunc main() { name := \"FOO\"; os.Getenv(name) }\n",
		},
		{
			name:  "os is not imported",
			input: "package main\nfunc main() { os := struct{}{}; _ = os }\n",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := goEnvUsages([]byte(tt.input))
			if err != nil {
				t.Fatalf("goEnvUsages() error = %v", err)
			}
			if len(got) != 0 {
				t.Errorf("goEnvUsages(%q) = %+v, want none", tt.input, got)
			}
		})
	}
}

func TestGoEnvUsages_UnparseableFileIsAnError(t *testing.T) {
	_, err := goEnvUsages([]byte("package main\nfunc main( {\n"))
	if err == nil {
		t.Fatal("goEnvUsages() error = nil, want a parse error for invalid Go syntax")
	}
}
