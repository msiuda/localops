package environment

import "testing"

func TestCppUsageScanner_SupportsCppExtensionsOnly(t *testing.T) {
	s := cppUsageScanner{}
	if !s.supports("main.cpp") || !s.supports("main.hpp") {
		t.Error("supports(.cpp/.hpp) = false, want true")
	}
	if s.supports("main.c") || s.supports("main.h") {
		t.Error("supports(.c/.h) = true, want false (owned by cUsageScanner)")
	}
}

func TestCppUsageScanner_StdGetenvAndBareGetenv(t *testing.T) {
	s := cppUsageScanner{}

	hits, _ := s.scan("main.cpp", []byte(`auto v = std::getenv("DATABASE_URL");`), nil)
	if len(hits) != 1 || hits[0].Name != "DATABASE_URL" {
		t.Errorf("hits = %+v, want one DATABASE_URL via std::getenv", hits)
	}

	hits, _ = s.scan("main.cpp", []byte(`auto v = getenv("DATABASE_URL");`), nil)
	if len(hits) != 1 || hits[0].Name != "DATABASE_URL" {
		t.Errorf("hits = %+v, want one DATABASE_URL via bare getenv", hits)
	}
}
