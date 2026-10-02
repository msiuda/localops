package technology

// rustDetector recognizes Rust projects from Cargo.toml, and common web
// frameworks from its dependency tokens.
//
// Cargo.toml is read as plain text rather than fully parsed, for the same
// reason as pyproject.toml: LocalOps has no TOML parser in its dependency
// tree, and a conservative token check for a known crate name is all this
// detection needs.
type rustDetector struct{}

func (rustDetector) detect(root string, readFile readFileFunc) ([]Detected, []Finding) {
	data, existed, err := readManifest(root, "Cargo.toml", readFile)
	if err != nil {
		return nil, []Finding{{Source: "Cargo.toml", Detail: "could not be read"}}
	}
	if !existed {
		return nil, nil
	}

	evidence := []Evidence{{File: "Cargo.toml"}}
	if hasLock, _ := fileExists(root, "Cargo.lock"); hasLock {
		evidence = append(evidence, Evidence{File: "Cargo.lock"})
	}

	detected := []Detected{{ID: IDRust, Name: "Rust", Kind: KindLanguage, Evidence: evidence}}

	text := string(data)
	frameworks := []struct {
		token string
		id    ID
		name  string
	}{
		{"axum", IDAxum, "Axum"},
		{"actix-web", IDActixWeb, "Actix Web"},
	}
	for _, f := range frameworks {
		if declaresPackageToken(text, f.token) {
			detected = append(detected, Detected{ID: f.id, Name: f.name, Kind: KindFramework, Evidence: []Evidence{{File: "Cargo.toml", Detail: "dependency " + f.token}}})
		}
	}

	return detected, nil
}
