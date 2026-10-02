package technology

import (
	"strings"

	"golang.org/x/mod/modfile"
)

// goDetector recognizes Go projects from go.mod (or go.work alone, for a
// multi-module workspace with no root go.mod), and common web frameworks
// from go.mod's require directives.
//
// This intentionally does not import internal/project, even though that
// package already parses go.mod: internal/technology must never depend on
// another LocalOps capability package (see docs/architecture.md), so a
// small amount of duplicated parsing is the correct tradeoff here, not a
// shared dependency that would risk an import cycle.
type goDetector struct{}

func (goDetector) detect(root string, readFile readFileFunc) ([]Detected, []Finding) {
	data, existed, err := readManifest(root, "go.mod", readFile)
	if err != nil {
		return nil, []Finding{{Source: "go.mod", Detail: "could not be read"}}
	}
	if !existed {
		if hasWork, _ := fileExists(root, "go.work"); hasWork {
			return []Detected{{ID: IDGo, Name: "Go", Kind: KindLanguage, Evidence: []Evidence{{File: "go.work"}}}}, nil
		}
		return nil, nil
	}

	modFile, err := modfile.Parse("go.mod", data, nil)
	if err != nil {
		return nil, []Finding{{Source: "go.mod", Detail: "could not be parsed"}}
	}

	detected := []Detected{{ID: IDGo, Name: "Go", Kind: KindLanguage, Evidence: []Evidence{{File: "go.mod"}}}}

	frameworks := []struct {
		prefix string
		id     ID
		name   string
	}{
		{"github.com/gin-gonic/gin", IDGin, "Gin"},
		{"github.com/gofiber/fiber", IDFiber, "Fiber"},
		{"github.com/labstack/echo", IDEcho, "Echo"},
	}

	for _, req := range modFile.Require {
		for _, f := range frameworks {
			if strings.HasPrefix(req.Mod.Path, f.prefix) {
				detected = append(detected, Detected{ID: f.id, Name: f.name, Kind: KindFramework, Evidence: []Evidence{{File: "go.mod", Detail: "require " + req.Mod.Path}}})
			}
		}
	}

	return detected, nil
}
