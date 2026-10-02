package environment

import (
	"reflect"
	"testing"

	"github.com/msiuda/localops/internal/technology"
)

func TestSymfonyUsageScanner_GatedByDetection(t *testing.T) {
	s := symfonyUsageScanner{}
	content := []byte("parameters:\n    database_url: '%env(DATABASE_URL)%'\n")

	hits, _ := s.scan("config/services.yaml", content, map[technology.ID]bool{})
	if len(hits) != 0 {
		t.Errorf("hits = %+v, want none when Symfony is not detected", hits)
	}

	hits, _ = s.scan("config/services.yaml", content, map[technology.ID]bool{technology.IDSymfony: true})
	want := []usageHit{{Name: "DATABASE_URL", Line: 2}}
	if !reflect.DeepEqual(hits, want) {
		t.Errorf("hits = %+v, want %+v when Symfony is detected", hits, want)
	}
}

func TestSymfonyUsageScanner_CommentedLineIgnored(t *testing.T) {
	content := []byte("# database_url: '%env(FAKE)%'\n")
	got := symfonyEnvUsages(content)
	if len(got) != 0 {
		t.Errorf("symfonyEnvUsages() = %+v, want none for a commented line", got)
	}
}

func TestSymfonyUsageScanner_SupportsYAMLAndXML(t *testing.T) {
	s := symfonyUsageScanner{}
	for _, path := range []string{"config/services.yaml", "config/services.yml", "config/services.xml"} {
		if !s.supports(path) {
			t.Errorf("supports(%q) = false, want true", path)
		}
	}
	if s.supports("config/services.php") {
		t.Error("supports(.php) = true, want false")
	}
}
