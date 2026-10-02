package technology

import "encoding/xml"

// javaDetector recognizes Java projects from pom.xml (Maven) or
// build.gradle (Gradle's Groovy DSL — build.gradle.kts belongs to
// kotlinDetector instead, since the Kotlin DSL file itself is the
// stronger, more conventional signal for a Kotlin-tooled project), and
// Spring Boot from Maven dependency artifactIds or Gradle dependency/plugin
// text.
type javaDetector struct{}

type pomXML struct {
	Dependencies []struct {
		ArtifactID string `xml:"artifactId"`
	} `xml:"dependencies>dependency"`
}

func (javaDetector) detect(root string, readFile readFileFunc) ([]Detected, []Finding) {
	var detected []Detected
	var findings []Finding

	pomData, pomExisted, err := readManifest(root, "pom.xml", readFile)
	if err != nil {
		findings = append(findings, Finding{Source: "pom.xml", Detail: "could not be read"})
	}

	gradleData, gradleExisted, err := readManifest(root, "build.gradle", readFile)
	if err != nil {
		findings = append(findings, Finding{Source: "build.gradle", Detail: "could not be read"})
	}

	if !pomExisted && !gradleExisted {
		return nil, findings
	}

	var evidence []Evidence
	hasSpringBoot := false

	if pomExisted {
		evidence = append(evidence, Evidence{File: "pom.xml"})
		var pom pomXML
		if err := xml.Unmarshal(pomData, &pom); err != nil {
			findings = append(findings, Finding{Source: "pom.xml", Detail: "could not be parsed"})
		} else {
			for _, dep := range pom.Dependencies {
				if containsAny(dep.ArtifactID, "spring-boot") {
					hasSpringBoot = true
				}
			}
		}
	}

	if gradleExisted {
		evidence = append(evidence, Evidence{File: "build.gradle"})
		if containsAny(string(gradleData), "org.springframework.boot", "spring-boot-starter") {
			hasSpringBoot = true
		}
	}

	detected = append(detected, Detected{ID: IDJava, Name: "Java", Kind: KindLanguage, Evidence: evidence})

	if hasSpringBoot {
		springEvidence := []Evidence{}
		if pomExisted {
			springEvidence = append(springEvidence, Evidence{File: "pom.xml", Detail: "dependency spring-boot-starter"})
		}
		if gradleExisted {
			springEvidence = append(springEvidence, Evidence{File: "build.gradle", Detail: "plugin org.springframework.boot"})
		}
		detected = append(detected, Detected{ID: IDSpringBoot, Name: "Spring Boot", Kind: KindFramework, Evidence: springEvidence})
	}

	return detected, findings
}
