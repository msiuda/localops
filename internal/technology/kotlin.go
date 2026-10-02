package technology

// kotlinDetector recognizes Kotlin projects from build.gradle.kts (the
// Kotlin Gradle DSL) when it actually declares a Kotlin plugin or
// dependency — the file's existence alone is not used, since Gradle's
// Kotlin DSL can configure a non-Kotlin (e.g. plain Java) project too — and
// Spring Boot/Ktor from the same build file's declared plugins/
// dependencies.
type kotlinDetector struct{}

func (kotlinDetector) detect(root string, readFile readFileFunc) ([]Detected, []Finding) {
	data, existed, err := readManifest(root, "build.gradle.kts", readFile)
	if err != nil {
		return nil, []Finding{{Source: "build.gradle.kts", Detail: "could not be read"}}
	}
	if !existed {
		return nil, nil
	}

	text := string(data)
	if !containsAny(text, "org.jetbrains.kotlin", "kotlin-stdlib", `kotlin("jvm")`, `kotlin("multiplatform")`) {
		return nil, nil
	}

	detected := []Detected{{ID: IDKotlin, Name: "Kotlin", Kind: KindLanguage, Evidence: []Evidence{{File: "build.gradle.kts", Detail: "plugin org.jetbrains.kotlin"}}}}

	if containsAny(text, "org.springframework.boot", "spring-boot-starter") {
		detected = append(detected, Detected{ID: IDSpringBoot, Name: "Spring Boot", Kind: KindFramework, Evidence: []Evidence{{File: "build.gradle.kts", Detail: "plugin org.springframework.boot"}}})
	}
	if containsAny(text, "io.ktor", "ktor-server") {
		detected = append(detected, Detected{ID: IDKtor, Name: "Ktor", Kind: KindFramework, Evidence: []Evidence{{File: "build.gradle.kts", Detail: "dependency ktor-server"}}})
	}

	return detected, nil
}
