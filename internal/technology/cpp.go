package technology

// cppDetector recognizes C/C++ projects, preferring the strongest
// available evidence: a CMakeLists.txt build description (whose content is
// inspected, best-effort, to distinguish a C-only project from a C++ one),
// falling back to root-level source file extensions only when no
// build-tool manifest exists at all. Source extensions alone are
// deliberately treated as weaker, supporting-only evidence per
// docs/architecture.md: a single stray .c/.cpp file is not the kind of
// strong project marker the other detectors require.
type cppDetector struct{}

func (cppDetector) detect(root string, readFile readFileFunc) ([]Detected, []Finding) {
	data, existed, err := readManifest(root, "CMakeLists.txt", readFile)
	if err != nil {
		return nil, []Finding{{Source: "CMakeLists.txt", Detail: "could not be read"}}
	}

	if existed {
		text := string(data)
		isCXX := containsAny(text, "CXX", "cpp", "c++", "C++")
		if !isCXX && containsAny(text, "LANGUAGES C)", "LANGUAGES C ", " C)") {
			return []Detected{{ID: IDC, Name: "C", Kind: KindLanguage, Evidence: []Evidence{{File: "CMakeLists.txt"}}}}, nil
		}
		// An ambiguous or C++-flavored CMakeLists.txt is still strong
		// evidence of a C-family project; C++ is the more common default
		// for CMake projects in practice.
		return []Detected{{ID: IDCPP, Name: "C++", Kind: KindLanguage, Evidence: []Evidence{{File: "CMakeLists.txt"}}}}, nil
	}

	exts, err := rootEntryExtensions(root)
	if err != nil {
		return nil, nil
	}

	var detected []Detected
	cppExts := []string{".cpp", ".cc", ".cxx", ".hpp", ".hh"}
	for _, ext := range cppExts {
		if exts[ext] {
			detected = append(detected, Detected{ID: IDCPP, Name: "C++", Kind: KindLanguage, Evidence: []Evidence{{File: "*" + ext, Detail: "source file extension"}}})
			break
		}
	}
	if exts[".c"] {
		detected = append(detected, Detected{ID: IDC, Name: "C", Kind: KindLanguage, Evidence: []Evidence{{File: "*.c", Detail: "source file extension"}}})
	}

	return detected, nil
}
