package technology

// dotnetDetector recognizes C#/.NET projects from a root-level *.csproj
// file, and ASP.NET Core from that project file's SDK/package references.
// A *.fsproj file (F#) is deliberately never treated as C# evidence. A
// *.sln file is recorded only as supporting evidence alongside a confirmed
// *.csproj, never as the sole basis for detection.
type dotnetDetector struct{}

func (dotnetDetector) detect(root string, readFile readFileFunc) ([]Detected, []Finding) {
	csprojFiles, err := rootFilesWithSuffix(root, ".csproj")
	if err != nil {
		return nil, []Finding{{Source: root, Detail: "could not list project directory"}}
	}
	if len(csprojFiles) == 0 {
		return nil, nil
	}

	var findings []Finding
	hasAspNetCore := false
	var evidence []Evidence

	for _, name := range csprojFiles {
		evidence = append(evidence, Evidence{File: name})
		data, _, err := readManifest(root, name, readFile)
		if err != nil {
			findings = append(findings, Finding{Source: name, Detail: "could not be read"})
			continue
		}
		if containsAny(string(data), "Microsoft.NET.Sdk.Web", "Microsoft.AspNetCore") {
			hasAspNetCore = true
		}
	}

	if slnFiles, _ := rootFilesWithSuffix(root, ".sln"); len(slnFiles) > 0 {
		for _, name := range slnFiles {
			evidence = append(evidence, Evidence{File: name})
		}
	}

	detected := []Detected{
		{ID: IDCSharp, Name: "C#", Kind: KindLanguage, Evidence: evidence},
		{ID: IDDotNet, Name: ".NET", Kind: KindPlatform, Evidence: evidence},
	}

	if hasAspNetCore {
		detected = append(detected, Detected{ID: IDAspNetCore, Name: "ASP.NET Core", Kind: KindFramework, Evidence: []Evidence{{File: "*.csproj", Detail: "Microsoft.NET.Sdk.Web"}}})
	}

	return detected, findings
}
