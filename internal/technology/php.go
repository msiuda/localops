package technology

import "encoding/json"

// phpDetector recognizes PHP projects from composer.json, and Laravel/
// Symfony from its "require" dependencies — the authoritative declaration
// of which framework a PHP project actually depends on. A framework's own
// convention file (artisan for Laravel, bin/console for Symfony) is
// recorded only as supporting evidence alongside a confirmed composer.json
// dependency, never as the sole basis for detection, since a stray file
// alone is not enough evidence of a real framework dependency.
type phpDetector struct{}

type composerJSON struct {
	Require    map[string]string `json:"require"`
	RequireDev map[string]string `json:"require-dev"`
}

func (phpDetector) detect(root string, readFile readFileFunc) ([]Detected, []Finding) {
	data, existed, err := readManifest(root, "composer.json", readFile)
	if err != nil {
		return nil, []Finding{{Source: "composer.json", Detail: "could not be read"}}
	}
	if !existed {
		return nil, nil
	}

	var composer composerJSON
	if err := json.Unmarshal(data, &composer); err != nil {
		return nil, []Finding{{Source: "composer.json", Detail: "could not be parsed"}}
	}

	require := make(map[string]bool, len(composer.Require)+len(composer.RequireDev))
	for name := range composer.Require {
		require[name] = true
	}
	for name := range composer.RequireDev {
		require[name] = true
	}

	detected := []Detected{{ID: IDPHP, Name: "PHP", Kind: KindLanguage, Evidence: []Evidence{{File: "composer.json"}}}}

	if require["laravel/framework"] {
		evidence := []Evidence{{File: "composer.json", Detail: "require laravel/framework"}}
		if hasArtisan, _ := fileExists(root, "artisan"); hasArtisan {
			evidence = append(evidence, Evidence{File: "artisan"})
		}
		detected = append(detected, Detected{ID: IDLaravel, Name: "Laravel", Kind: KindFramework, Evidence: evidence})
	}

	if require["symfony/symfony"] || require["symfony/framework-bundle"] {
		evidence := []Evidence{{File: "composer.json", Detail: "require symfony/framework-bundle"}}
		if hasConsole, _ := fileExists(root, "bin/console"); hasConsole {
			evidence = append(evidence, Evidence{File: "bin/console"})
		}
		if hasLock, _ := fileExists(root, "symfony.lock"); hasLock {
			evidence = append(evidence, Evidence{File: "symfony.lock"})
		}
		detected = append(detected, Detected{ID: IDSymfony, Name: "Symfony", Kind: KindFramework, Evidence: evidence})
	}

	return detected, nil
}
