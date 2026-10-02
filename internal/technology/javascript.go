package technology

import "encoding/json"

// javascriptDetector recognizes JavaScript/TypeScript/Node.js projects from
// package.json, and common frameworks from its declared dependencies.
//
// package.json by itself only proves the Node.js/npm ecosystem is present —
// it does not prove JavaScript is one of the project's primary source
// languages, since plenty of TypeScript projects never touch a .js file.
// JavaScript is therefore reported only when TypeScript was not detected;
// when tsconfig.json or a typescript dependency is present, TypeScript is
// reported on its own rather than alongside a redundant "JavaScript" the
// user never asked about. This stays a manifest-only rule, deliberately not
// a source-tree scan to statistically decide the "real" primary language.
type javascriptDetector struct{}

type packageJSON struct {
	Dependencies    map[string]string `json:"dependencies"`
	DevDependencies map[string]string `json:"devDependencies"`
}

func (javascriptDetector) detect(root string, readFile readFileFunc) ([]Detected, []Finding) {
	data, existed, err := readManifest(root, "package.json", readFile)
	if err != nil {
		return nil, []Finding{{Source: "package.json", Detail: "could not be read"}}
	}
	if !existed {
		return nil, nil
	}

	var pkg packageJSON
	if err := json.Unmarshal(data, &pkg); err != nil {
		return nil, []Finding{{Source: "package.json", Detail: "could not be parsed"}}
	}

	deps := make(map[string]bool, len(pkg.Dependencies)+len(pkg.DevDependencies))
	for name := range pkg.Dependencies {
		deps[name] = true
	}
	for name := range pkg.DevDependencies {
		deps[name] = true
	}

	detected := []Detected{
		{ID: IDNodeJS, Name: "Node.js", Kind: KindRuntime, Evidence: []Evidence{{File: "package.json"}}},
	}

	hasTSConfig, _ := fileExists(root, "tsconfig.json")
	switch {
	case hasTSConfig:
		detected = append(detected, Detected{ID: IDTypeScript, Name: "TypeScript", Kind: KindLanguage, Evidence: []Evidence{{File: "tsconfig.json"}}})
	case deps["typescript"]:
		detected = append(detected, Detected{ID: IDTypeScript, Name: "TypeScript", Kind: KindLanguage, Evidence: []Evidence{{File: "package.json", Detail: "dependency typescript"}}})
	default:
		detected = append(detected, Detected{ID: IDJavaScript, Name: "JavaScript", Kind: KindLanguage, Evidence: []Evidence{{File: "package.json"}}})
	}

	frameworks := []struct {
		dep  string
		id   ID
		name string
	}{
		{"next", IDNextJS, "Next.js"},
		{"react", IDReact, "React"},
		{"vue", IDVue, "Vue"},
		{"@angular/core", IDAngular, "Angular"},
		{"svelte", IDSvelte, "Svelte"},
		{"express", IDExpress, "Express"},
		{"@nestjs/core", IDNestJS, "NestJS"},
	}
	for _, f := range frameworks {
		if deps[f.dep] {
			detected = append(detected, Detected{ID: f.id, Name: f.name, Kind: KindFramework, Evidence: []Evidence{{File: "package.json", Detail: "dependency " + f.dep}}})
		}
	}

	return detected, nil
}
