package technology

// rubyDetector recognizes Ruby projects from a Gemfile, and Rails from a
// `gem "rails"`/`gem 'rails'` declaration within it.
type rubyDetector struct{}

func (rubyDetector) detect(root string, readFile readFileFunc) ([]Detected, []Finding) {
	data, existed, err := readManifest(root, "Gemfile", readFile)
	if err != nil {
		return nil, []Finding{{Source: "Gemfile", Detail: "could not be read"}}
	}
	if !existed {
		return nil, nil
	}

	evidence := []Evidence{{File: "Gemfile"}}
	if hasLock, _ := fileExists(root, "Gemfile.lock"); hasLock {
		evidence = append(evidence, Evidence{File: "Gemfile.lock"})
	}

	detected := []Detected{{ID: IDRuby, Name: "Ruby", Kind: KindLanguage, Evidence: evidence}}

	if declaresPackageToken(string(data), "rails") {
		detected = append(detected, Detected{ID: IDRails, Name: "Rails", Kind: KindFramework, Evidence: []Evidence{{File: "Gemfile", Detail: "gem rails"}}})
	}

	return detected, nil
}
