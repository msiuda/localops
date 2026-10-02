package environment

import "testing"

func TestCUsageScanner_SupportsCAndHExtensions(t *testing.T) {
	s := cUsageScanner{}
	if !s.supports("main.c") || !s.supports("main.h") {
		t.Error("supports(.c/.h) = false, want true")
	}
	if s.supports("main.cpp") {
		t.Error("supports(.cpp) = true, want false")
	}
}

func TestCUsageScanner_Getenv(t *testing.T) {
	s := cUsageScanner{}
	hits, _ := s.scan("main.c", []byte(`char *v = getenv("DATABASE_URL");`), nil)
	if len(hits) != 1 || hits[0].Name != "DATABASE_URL" {
		t.Errorf("hits = %+v, want one DATABASE_URL", hits)
	}
}

func TestCUsageScanner_UnrelatedSuffixIdentifierIgnored(t *testing.T) {
	s := cUsageScanner{}
	hits, _ := s.scan("main.c", []byte(`char *v = fgetenv("FAKE");`), nil)
	if len(hits) != 0 {
		t.Errorf("hits = %+v, want none", hits)
	}
}
