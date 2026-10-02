package environment

import "testing"

func TestKotlinUsageScanner_SupportsKtAndKts(t *testing.T) {
	s := kotlinUsageScanner{}
	if !s.supports("src/Main.kt") || !s.supports("build.gradle.kts") {
		t.Error("supports(.kt/.kts) = false, want true")
	}
	if s.supports("src/Main.java") {
		t.Error("supports(.java) = true, want false")
	}
}

func TestKotlinUsageScanner_SystemGetenv(t *testing.T) {
	s := kotlinUsageScanner{}
	hits, _ := s.scan("Main.kt", []byte(`val v = System.getenv("DATABASE_URL")`), nil)
	if len(hits) != 1 || hits[0].Name != "DATABASE_URL" {
		t.Errorf("hits = %+v, want one DATABASE_URL", hits)
	}
}

func TestKotlinUsageScanner_DynamicArgumentIgnored(t *testing.T) {
	s := kotlinUsageScanner{}
	hits, _ := s.scan("Main.kt", []byte(`val name = "X"; System.getenv(name)`), nil)
	if len(hits) != 0 {
		t.Errorf("hits = %+v, want none", hits)
	}
}
