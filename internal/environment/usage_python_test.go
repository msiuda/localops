package environment

import (
	"reflect"
	"testing"
)

func TestPythonEnvUsages_SupportedForms(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  []usageHit
	}{
		{
			name:  "os.getenv",
			input: "import os\nv = os.getenv(\"DATABASE_URL\")\n",
			want:  []usageHit{{Name: "DATABASE_URL", Line: 2}},
		},
		{
			name:  "os.environ bracket access",
			input: "import os\nv = os.environ[\"DATABASE_URL\"]\n",
			want:  []usageHit{{Name: "DATABASE_URL", Line: 2}},
		},
		{
			name:  "os.environ.get",
			input: "import os\nv = os.environ.get(\"DATABASE_URL\")\n",
			want:  []usageHit{{Name: "DATABASE_URL", Line: 2}},
		},
		{
			name:  "single-quoted key",
			input: "import os\nv = os.environ['DATABASE_URL']\n",
			want:  []usageHit{{Name: "DATABASE_URL", Line: 2}},
		},
		{
			name:  "aliased import",
			input: "import os as stdos\nv = stdos.getenv(\"DATABASE_URL\")\n",
			want:  []usageHit{{Name: "DATABASE_URL", Line: 2}},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := pythonEnvUsages([]byte(tt.input))
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("pythonEnvUsages() = %+v, want %+v", got, tt.want)
			}
		})
	}
}

func TestPythonEnvUsages_IgnoredCases(t *testing.T) {
	tests := []struct {
		name  string
		input string
	}{
		{name: "no os import", input: `v = os.getenv("FAKE")`},
		{name: "line comment", input: "import os\n# os.getenv(\"FAKE\")\n"},
		{name: "ordinary string", input: "import os\nnote = \"os.getenv('FAKE')\"\n"},
		{name: "triple-quoted docstring", input: "import os\n\"\"\"\nos.getenv(\"FAKE\")\n\"\"\"\n"},
		{name: "dynamic argument", input: "import os\nname = 'X'\nv = os.getenv(name)\n"},
		{name: "unrelated module named os_utils", input: "import os_utils as os\nv = os.getenv(\"FAKE\")\n"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := pythonEnvUsages([]byte(tt.input))
			if len(got) != 0 {
				t.Errorf("pythonEnvUsages(%q) = %+v, want none", tt.input, got)
			}
		})
	}
}
