package environment

import "testing"

func TestJavaUsageScanner_SupportsJavaExtension(t *testing.T) {
	s := javaUsageScanner{}
	if !s.supports("src/Main.java") {
		t.Error("supports(.java) = false, want true")
	}
	if s.supports("src/main.kt") {
		t.Error("supports(.kt) = true, want false")
	}
}

func TestJavaUsageScanner_SystemGetenv(t *testing.T) {
	s := javaUsageScanner{}
	hits, findings := s.scan("Main.java", []byte(`String v = System.getenv("DATABASE_URL");`), nil)
	if len(findings) != 0 {
		t.Errorf("findings = %+v, want none", findings)
	}
	if len(hits) != 1 || hits[0].Name != "DATABASE_URL" || hits[0].Line != 1 {
		t.Errorf("hits = %+v, want one DATABASE_URL at line 1", hits)
	}
}

func TestJavaUsageScanner_CommentIgnored(t *testing.T) {
	s := javaUsageScanner{}
	hits, _ := s.scan("Main.java", []byte("// System.getenv(\"FAKE\");\n"), nil)
	if len(hits) != 0 {
		t.Errorf("hits = %+v, want none", hits)
	}
}
