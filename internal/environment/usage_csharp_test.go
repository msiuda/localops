package environment

import "testing"

func TestCSharpUsageScanner_SupportsCsExtension(t *testing.T) {
	s := csharpUsageScanner{}
	if !s.supports("Program.cs") {
		t.Error("supports(.cs) = false, want true")
	}
	if s.supports("Program.fs") {
		t.Error("supports(.fs) = true, want false")
	}
}

func TestCSharpUsageScanner_UnqualifiedAndQualifiedForms(t *testing.T) {
	s := csharpUsageScanner{}

	hits, _ := s.scan("Program.cs", []byte(`var v = Environment.GetEnvironmentVariable("DATABASE_URL");`), nil)
	if len(hits) != 1 || hits[0].Name != "DATABASE_URL" {
		t.Errorf("hits = %+v, want one DATABASE_URL", hits)
	}

	hits, _ = s.scan("Program.cs", []byte(`var v = System.Environment.GetEnvironmentVariable("DATABASE_URL");`), nil)
	if len(hits) != 1 || hits[0].Name != "DATABASE_URL" {
		t.Errorf("hits = %+v, want one DATABASE_URL via the fully-qualified form", hits)
	}
}

func TestCSharpUsageScanner_StringLiteralIgnored(t *testing.T) {
	s := csharpUsageScanner{}
	hits, _ := s.scan("Program.cs", []byte(`var note = "Environment.GetEnvironmentVariable(\"FAKE\")";`), nil)
	if len(hits) != 0 {
		t.Errorf("hits = %+v, want none", hits)
	}
}
