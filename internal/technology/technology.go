// Package technology identifies the technologies, languages, runtimes,
// platforms, and frameworks that make up a project, and the evidence
// supporting each detection.
//
// This package owns technology identity and detection only. It does not own
// Doctor, Validation, Environment, or any other per-capability behavior —
// those capabilities may consult detected technology identities (see
// internal/environment's use of technology.ID to gate framework-specific
// forms), but technology itself must never depend on them.
//
// Like internal/environment, this package never retains a project file's
// full contents or any secret-shaped value — only technology identity and
// short, safe evidence describing why a technology was detected.
package technology

// ID is a stable, unique identifier for a single detected technology. IDs
// are lowercase and hyphenated, and never change once published, since
// other packages (Environment, and future Doctor/Validation) persist or
// compare against them.
type ID string

const (
	IDJavaScript ID = "javascript"
	IDTypeScript ID = "typescript"
	IDNodeJS     ID = "nodejs"
	IDReact      ID = "react"
	IDNextJS     ID = "nextjs"
	IDVue        ID = "vue"
	IDAngular    ID = "angular"
	IDSvelte     ID = "svelte"
	IDExpress    ID = "express"
	IDNestJS     ID = "nestjs"

	IDPython  ID = "python"
	IDDjango  ID = "django"
	IDFlask   ID = "flask"
	IDFastAPI ID = "fastapi"

	IDPHP     ID = "php"
	IDLaravel ID = "laravel"
	IDSymfony ID = "symfony"

	IDGo    ID = "go"
	IDGin   ID = "gin"
	IDFiber ID = "fiber"
	IDEcho  ID = "echo"

	IDRust     ID = "rust"
	IDAxum     ID = "axum"
	IDActixWeb ID = "actix-web"

	IDJava       ID = "java"
	IDSpringBoot ID = "spring-boot"

	IDCSharp     ID = "csharp"
	IDDotNet     ID = "dotnet"
	IDAspNetCore ID = "aspnet-core"

	IDRuby  ID = "ruby"
	IDRails ID = "rails"

	IDKotlin ID = "kotlin"
	IDKtor   ID = "ktor"

	IDC   ID = "c"
	IDCPP ID = "cpp"
)

// Kind is the category a detected technology belongs to. This is the
// smallest set of kinds that reflects how LocalOps actually needs to group
// and display technologies (see docs/architecture.md); it is not a general
// taxonomy.
type Kind string

const (
	// KindLanguage is a programming language (e.g. TypeScript, Python).
	KindLanguage Kind = "language"
	// KindRuntime is a language runtime (e.g. Node.js).
	KindRuntime Kind = "runtime"
	// KindPlatform is a broader development platform/SDK (e.g. .NET).
	KindPlatform Kind = "platform"
	// KindFramework is a framework or library built on a language/runtime
	// (e.g. React, Laravel, Spring Boot).
	KindFramework Kind = "framework"
)

// Evidence is a single safe, human-readable fact supporting a detection: a
// project file and, optionally, a short machine-generated reason. It never
// contains a file's full contents or a secret-shaped value.
type Evidence struct {
	File   string
	Detail string
}

// Detected is one technology LocalOps found evidence for in a project.
type Detected struct {
	ID       ID
	Name     string
	Kind     Kind
	Evidence []Evidence
}

// Finding is a safe, isolated detection issue — a manifest file that exists
// but could not be read or parsed. It never includes the file's contents.
type Finding struct {
	Source string
	Detail string
}

// Result is the technology detection outcome for a single project root. It
// never contains a secret-shaped value or raw manifest contents.
type Result struct {
	Path     string
	Detected []Detected
	Findings []Finding
}
