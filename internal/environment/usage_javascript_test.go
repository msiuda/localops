package environment

import (
	"reflect"
	"testing"
)

func TestJSEnvUsages_SupportedForms(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  []usageHit
	}{
		{
			name:  "dot access",
			input: `const url = process.env.DATABASE_URL;`,
			want:  []usageHit{{Name: "DATABASE_URL", Line: 1}},
		},
		{
			name:  "double-quoted bracket access",
			input: `const url = process.env["DATABASE_URL"];`,
			want:  []usageHit{{Name: "DATABASE_URL", Line: 1}},
		},
		{
			name:  "single-quoted bracket access",
			input: `const url = process.env['DATABASE_URL'];`,
			want:  []usageHit{{Name: "DATABASE_URL", Line: 1}},
		},
		{
			name:  "import.meta.env dot access",
			input: `const url = import.meta.env.VITE_API_BASE_URL;`,
			want:  []usageHit{{Name: "VITE_API_BASE_URL", Line: 1}},
		},
		{
			name:  "import.meta.env double-quoted bracket access",
			input: `const url = import.meta.env["VITE_API_BASE_URL"];`,
			want:  []usageHit{{Name: "VITE_API_BASE_URL", Line: 1}},
		},
		{
			name:  "import.meta.env single-quoted bracket access",
			input: `const url = import.meta.env['VITE_API_BASE_URL'];`,
			want:  []usageHit{{Name: "VITE_API_BASE_URL", Line: 1}},
		},
		{
			name: "line number reflects the usage's own line",
			input: "const a = 1;\n" +
				"const url = process.env.DATABASE_URL;\n",
			want: []usageHit{{Name: "DATABASE_URL", Line: 2}},
		},
		{
			name:  "duplicate usage of the same variable in one file is each reported",
			input: "process.env.FOO;\nprocess.env.FOO;\n",
			want:  []usageHit{{Name: "FOO", Line: 1}, {Name: "FOO", Line: 2}},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := jsEnvUsages([]byte(tt.input))
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("jsEnvUsages() = %+v, want %+v", got, tt.want)
			}
		})
	}
}

func TestJSEnvUsages_ViteBuiltinsExcludedFromImportMetaEnvOnly(t *testing.T) {
	for _, key := range []string{"MODE", "BASE_URL", "PROD", "DEV", "SSR"} {
		t.Run("import.meta.env."+key, func(t *testing.T) {
			got := jsEnvUsages([]byte(`const v = import.meta.env.` + key + `;`))
			if len(got) != 0 {
				t.Errorf("jsEnvUsages() = %+v, want none: %s is a Vite built-in, not a user-defined variable", got, key)
			}
		})

		t.Run("import.meta.env[\""+key+"\"]", func(t *testing.T) {
			got := jsEnvUsages([]byte(`const v = import.meta.env["` + key + `"];`))
			if len(got) != 0 {
				t.Errorf("jsEnvUsages() = %+v, want none: %s is a Vite built-in, not a user-defined variable", got, key)
			}
		})
	}
}

func TestJSEnvUsages_CustomViteVariableStillDetected(t *testing.T) {
	got := jsEnvUsages([]byte(`const v = import.meta.env.VITE_API_URL;`))
	want := []usageHit{{Name: "VITE_API_URL", Line: 1}}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("jsEnvUsages() = %+v, want %+v", got, want)
	}
}

func TestJSEnvUsages_ProcessEnvDevStillDetected(t *testing.T) {
	// The Vite built-in exclusion is scoped to import.meta.env only —
	// process.env.DEV is a different, user-controlled namespace.
	got := jsEnvUsages([]byte(`const v = process.env.DEV;`))
	want := []usageHit{{Name: "DEV", Line: 1}}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("jsEnvUsages() = %+v, want %+v", got, want)
	}
}

func TestJSEnvUsages_IgnoredCases(t *testing.T) {
	tests := []struct {
		name  string
		input string
	}{
		{
			name:  "line comment",
			input: `// process.env.FAKE`,
		},
		{
			name:  "block comment",
			input: "/*\nprocess.env.FAKE\n*/",
		},
		{
			name:  "inside an ordinary string literal",
			input: `const note = "see process.env.FAKE for details";`,
		},
		{
			name:  "inside a template literal",
			input: "const note = `see process.env.FAKE for details`;",
		},
		{
			name:  "dynamic bracket access via a variable",
			input: "const key = 'X'; process.env[key];",
		},
		{
			name:  "dynamic bracket access via concatenation",
			input: `process.env["FOO" + "BAR"];`,
		},
		{
			name:  "unrelated property access",
			input: `process.environment.FOO;`,
		},
		{
			name:  "unrelated identifier containing the prefix as a suffix",
			input: `myprocess.env.FOO;`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := jsEnvUsages([]byte(tt.input))
			if len(got) != 0 {
				t.Errorf("jsEnvUsages(%q) = %+v, want none", tt.input, got)
			}
		})
	}
}

func TestJSEnvUsages_CommentDoesNotHideRealUsageOnNextLine(t *testing.T) {
	input := "// process.env.FAKE\nconst url = process.env.REAL;\n"

	got := jsEnvUsages([]byte(input))
	want := []usageHit{{Name: "REAL", Line: 2}}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("jsEnvUsages() = %+v, want %+v", got, want)
	}
}
