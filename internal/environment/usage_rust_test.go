package environment

import (
	"reflect"
	"testing"
)

func TestRustEnvUsages_QualifiedFormAlwaysRecognized(t *testing.T) {
	input := `let v = std::env::var("DATABASE_URL").unwrap();`
	got := rustEnvUsages([]byte(input))
	want := []usageHit{{Name: "DATABASE_URL", Line: 1}}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("rustEnvUsages() = %+v, want %+v", got, want)
	}
}

func TestRustEnvUsages_VarOs(t *testing.T) {
	input := `let v = std::env::var_os("DATABASE_URL");`
	got := rustEnvUsages([]byte(input))
	if len(got) != 1 || got[0].Name != "DATABASE_URL" {
		t.Errorf("rustEnvUsages() = %+v, want one DATABASE_URL", got)
	}
}

func TestRustEnvUsages_BareFormRequiresImport(t *testing.T) {
	withImport := "use std::env;\nlet v = env::var(\"DATABASE_URL\");\n"
	got := rustEnvUsages([]byte(withImport))
	if len(got) != 1 || got[0].Name != "DATABASE_URL" {
		t.Errorf("rustEnvUsages() = %+v, want one DATABASE_URL when std::env is imported", got)
	}

	withoutImport := `let v = env::var("DATABASE_URL");`
	got = rustEnvUsages([]byte(withoutImport))
	if len(got) != 0 {
		t.Errorf("rustEnvUsages() = %+v, want none: env::var is not statically known to be std::env without the import", got)
	}
}

func TestRustEnvUsages_CommentAndStringIgnored(t *testing.T) {
	tests := []string{
		"// std::env::var(\"FAKE\");",
		`let note = "std::env::var(\"FAKE\")";`,
	}
	for _, input := range tests {
		got := rustEnvUsages([]byte(input))
		if len(got) != 0 {
			t.Errorf("rustEnvUsages(%q) = %+v, want none", input, got)
		}
	}
}
