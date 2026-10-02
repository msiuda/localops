package technology

// pythonDetector recognizes Python projects from pyproject.toml,
// requirements.txt, Pipfile, poetry.lock, or uv.lock, and common web
// frameworks from declared dependency tokens.
//
// pyproject.toml and Pipfile are read as plain text rather than fully
// parsed: LocalOps has no TOML parser in its dependency tree (see
// docs/architecture.md), and a conservative token check for a known
// framework package name is all this detection needs.
type pythonDetector struct{}

var pythonMarkerFiles = []string{"pyproject.toml", "requirements.txt", "Pipfile", "poetry.lock", "uv.lock"}

func (pythonDetector) detect(root string, readFile readFileFunc) ([]Detected, []Finding) {
	var evidence []Evidence
	var findings []Finding
	var depText string

	for _, name := range pythonMarkerFiles {
		data, existed, err := readManifest(root, name, readFile)
		if err != nil {
			findings = append(findings, Finding{Source: name, Detail: "could not be read"})
			continue
		}
		if !existed {
			continue
		}
		evidence = append(evidence, Evidence{File: name})
		if name == "pyproject.toml" || name == "requirements.txt" || name == "Pipfile" {
			depText += string(data) + "\n"
		}
	}

	if len(evidence) == 0 {
		return nil, findings
	}

	detected := []Detected{{ID: IDPython, Name: "Python", Kind: KindLanguage, Evidence: evidence}}

	frameworks := []struct {
		token string
		id    ID
		name  string
	}{
		{"django", IDDjango, "Django"},
		{"flask", IDFlask, "Flask"},
		{"fastapi", IDFastAPI, "FastAPI"},
	}
	for _, f := range frameworks {
		if declaresPackageToken(depText, f.token) {
			detected = append(detected, Detected{ID: f.id, Name: f.name, Kind: KindFramework, Evidence: []Evidence{{File: "pyproject.toml/requirements.txt", Detail: "dependency " + f.token}}})
		}
	}

	return detected, findings
}
