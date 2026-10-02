package technology

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

func writeFile(t *testing.T, dir, name, content string) {
	t.Helper()
	path := filepath.Join(dir, name)
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("MkdirAll error = %v", err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("WriteFile(%s) error = %v", name, err)
	}
}

func checkDetected(t *testing.T, all []Detected, id ID) Detected {
	t.Helper()
	for _, d := range all {
		if d.ID == id {
			return d
		}
	}
	t.Fatalf("no detected technology for %q in %+v", id, all)
	return Detected{}
}

func hasDetected(all []Detected, id ID) bool {
	for _, d := range all {
		if d.ID == id {
			return true
		}
	}
	return false
}

// --- JavaScript / TypeScript / Node.js ---

func TestDetect_JavaScriptNode(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "package.json", `{"name":"demo","dependencies":{"react":"^18.0.0"}}`)

	result, err := Detect(dir)
	if err != nil {
		t.Fatalf("Detect() error = %v", err)
	}

	checkDetected(t, result.Detected, IDJavaScript)
	checkDetected(t, result.Detected, IDNodeJS)
	checkDetected(t, result.Detected, IDReact)
	if hasDetected(result.Detected, IDTypeScript) {
		t.Error("TypeScript detected without tsconfig.json or a typescript dependency")
	}
}

func TestDetect_TypeScriptViaTSConfig(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "package.json", `{"name":"demo"}`)
	writeFile(t, dir, "tsconfig.json", `{}`)

	result, err := Detect(dir)
	if err != nil {
		t.Fatalf("Detect() error = %v", err)
	}

	checkDetected(t, result.Detected, IDTypeScript)
}

func TestDetect_TypeScriptProjectDoesNotAlsoReportJavaScript(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "package.json", `{"name":"demo","dependencies":{"@nestjs/core":"^10.0.0"}}`)
	writeFile(t, dir, "tsconfig.json", `{}`)

	result, err := Detect(dir)
	if err != nil {
		t.Fatalf("Detect() error = %v", err)
	}

	checkDetected(t, result.Detected, IDTypeScript)
	checkDetected(t, result.Detected, IDNodeJS)
	checkDetected(t, result.Detected, IDNestJS)
	if hasDetected(result.Detected, IDJavaScript) {
		t.Error("JavaScript reported alongside TypeScript: package.json only proves the Node/npm ecosystem, not that JS is a primary language here")
	}
}

func TestDetect_TypeScriptViaDependencyDoesNotAlsoReportJavaScript(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "package.json", `{"name":"demo","devDependencies":{"typescript":"^5.0.0"}}`)

	result, err := Detect(dir)
	if err != nil {
		t.Fatalf("Detect() error = %v", err)
	}

	checkDetected(t, result.Detected, IDTypeScript)
	if hasDetected(result.Detected, IDJavaScript) {
		t.Error("JavaScript reported alongside a TypeScript dependency")
	}
}

func TestDetect_PlainJavaScriptProjectStillReportsJavaScriptAndNode(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "package.json", `{"name":"demo","dependencies":{"express":"^4.0.0"}}`)

	result, err := Detect(dir)
	if err != nil {
		t.Fatalf("Detect() error = %v", err)
	}

	checkDetected(t, result.Detected, IDJavaScript)
	checkDetected(t, result.Detected, IDNodeJS)
	checkDetected(t, result.Detected, IDExpress)
	if hasDetected(result.Detected, IDTypeScript) {
		t.Error("TypeScript detected without tsconfig.json or a typescript dependency")
	}
}

func TestDetect_NestJSAndNext(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "package.json", `{"dependencies":{"@nestjs/core":"^10.0.0","next":"^14.0.0"}}`)

	result, err := Detect(dir)
	if err != nil {
		t.Fatalf("Detect() error = %v", err)
	}

	checkDetected(t, result.Detected, IDNestJS)
	checkDetected(t, result.Detected, IDNextJS)
}

func TestDetect_MalformedPackageJSONIsAFinding(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "package.json", `{not valid json`)

	result, err := Detect(dir)
	if err != nil {
		t.Fatalf("Detect() error = %v", err)
	}
	if len(result.Findings) != 1 || result.Findings[0].Source != "package.json" {
		t.Errorf("Findings = %+v, want exactly one for package.json", result.Findings)
	}
	if hasDetected(result.Detected, IDJavaScript) {
		t.Error("JavaScript detected despite malformed package.json")
	}
}

// --- Python ---

func TestDetect_PythonDjango(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "requirements.txt", "django==4.2\ngunicorn==21.0\n")

	result, err := Detect(dir)
	if err != nil {
		t.Fatalf("Detect() error = %v", err)
	}

	checkDetected(t, result.Detected, IDPython)
	checkDetected(t, result.Detected, IDDjango)
	if hasDetected(result.Detected, IDFlask) {
		t.Error("Flask detected without evidence")
	}
}

func TestDetect_PythonFastAPIViaPyproject(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "pyproject.toml", "[project]\ndependencies = [\"fastapi>=0.100\"]\n")

	result, err := Detect(dir)
	if err != nil {
		t.Fatalf("Detect() error = %v", err)
	}

	checkDetected(t, result.Detected, IDPython)
	checkDetected(t, result.Detected, IDFastAPI)
}

func TestDetect_PythonWeakEvidenceDoesNotImplyFramework(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "requirements.txt", "requests==2.31.0\n")
	// A file name alone (not a dependency declaration) must never count.
	writeFile(t, dir, "my_django_notes.txt", "django is a web framework\n")

	result, err := Detect(dir)
	if err != nil {
		t.Fatalf("Detect() error = %v", err)
	}

	checkDetected(t, result.Detected, IDPython)
	if hasDetected(result.Detected, IDDjango) {
		t.Error("Django detected without a real dependency declaration")
	}
}

// --- PHP ---

func TestDetect_PHPLaravel(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "composer.json", `{"require":{"laravel/framework":"^10.0"}}`)
	writeFile(t, dir, "artisan", "#!/usr/bin/env php\n")

	result, err := Detect(dir)
	if err != nil {
		t.Fatalf("Detect() error = %v", err)
	}

	checkDetected(t, result.Detected, IDPHP)
	laravel := checkDetected(t, result.Detected, IDLaravel)
	if len(laravel.Evidence) != 2 {
		t.Errorf("Laravel evidence = %+v, want composer.json + artisan", laravel.Evidence)
	}
}

func TestDetect_PHPArtisanAloneIsNotLaravel(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "composer.json", `{"require":{"some/other-package":"^1.0"}}`)
	writeFile(t, dir, "artisan", "not actually laravel\n")

	result, err := Detect(dir)
	if err != nil {
		t.Fatalf("Detect() error = %v", err)
	}

	if hasDetected(result.Detected, IDLaravel) {
		t.Error("Laravel detected from artisan file alone, without a composer.json dependency")
	}
}

func TestDetect_PHPSymfony(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "composer.json", `{"require":{"symfony/framework-bundle":"^6.0"}}`)

	result, err := Detect(dir)
	if err != nil {
		t.Fatalf("Detect() error = %v", err)
	}

	checkDetected(t, result.Detected, IDSymfony)
}

// --- Go ---

func TestDetect_GoGin(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "go.mod", "module demo\n\ngo 1.22\n\nrequire github.com/gin-gonic/gin v1.9.1\n")

	result, err := Detect(dir)
	if err != nil {
		t.Fatalf("Detect() error = %v", err)
	}

	checkDetected(t, result.Detected, IDGo)
	checkDetected(t, result.Detected, IDGin)
}

func TestDetect_GoWorkWithoutGoMod(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "go.work", "go 1.22\n")

	result, err := Detect(dir)
	if err != nil {
		t.Fatalf("Detect() error = %v", err)
	}

	checkDetected(t, result.Detected, IDGo)
}

// --- Rust ---

func TestDetect_RustAxum(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "Cargo.toml", "[package]\nname = \"demo\"\n\n[dependencies]\naxum = \"0.7\"\n")

	result, err := Detect(dir)
	if err != nil {
		t.Fatalf("Detect() error = %v", err)
	}

	checkDetected(t, result.Detected, IDRust)
	checkDetected(t, result.Detected, IDAxum)
}

// --- Java ---

func TestDetect_JavaMavenSpringBoot(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "pom.xml", `<project><dependencies><dependency><artifactId>spring-boot-starter-web</artifactId></dependency></dependencies></project>`)

	result, err := Detect(dir)
	if err != nil {
		t.Fatalf("Detect() error = %v", err)
	}

	checkDetected(t, result.Detected, IDJava)
	checkDetected(t, result.Detected, IDSpringBoot)
}

func TestDetect_JavaGradle(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "build.gradle", "plugins { id 'java' }\n")

	result, err := Detect(dir)
	if err != nil {
		t.Fatalf("Detect() error = %v", err)
	}

	checkDetected(t, result.Detected, IDJava)
	if hasDetected(result.Detected, IDSpringBoot) {
		t.Error("Spring Boot detected without evidence")
	}
}

// --- C# / .NET ---

func TestDetect_DotNetAspNetCore(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "Demo.csproj", `<Project Sdk="Microsoft.NET.Sdk.Web"></Project>`)

	result, err := Detect(dir)
	if err != nil {
		t.Fatalf("Detect() error = %v", err)
	}

	checkDetected(t, result.Detected, IDCSharp)
	checkDetected(t, result.Detected, IDDotNet)
	checkDetected(t, result.Detected, IDAspNetCore)
}

func TestDetect_FSharpProjectDoesNotImplyCSharp(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "Demo.fsproj", `<Project Sdk="Microsoft.NET.Sdk"></Project>`)

	result, err := Detect(dir)
	if err != nil {
		t.Fatalf("Detect() error = %v", err)
	}

	if hasDetected(result.Detected, IDCSharp) {
		t.Error("C# detected from a .fsproj file")
	}
}

// --- Ruby ---

func TestDetect_RubyRails(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "Gemfile", "source 'https://rubygems.org'\ngem 'rails', '~> 7.1'\n")

	result, err := Detect(dir)
	if err != nil {
		t.Fatalf("Detect() error = %v", err)
	}

	checkDetected(t, result.Detected, IDRuby)
	checkDetected(t, result.Detected, IDRails)
}

// --- Kotlin ---

func TestDetect_KotlinSpringBoot(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "build.gradle.kts", "plugins {\n    kotlin(\"jvm\") version \"1.9.20\"\n    id(\"org.springframework.boot\") version \"3.2.0\"\n}\n")

	result, err := Detect(dir)
	if err != nil {
		t.Fatalf("Detect() error = %v", err)
	}

	checkDetected(t, result.Detected, IDKotlin)
	checkDetected(t, result.Detected, IDSpringBoot)
}

func TestDetect_PlainGradleKtsWithoutKotlinPluginIsNotKotlin(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "build.gradle.kts", "plugins {\n    id(\"java\")\n}\n")

	result, err := Detect(dir)
	if err != nil {
		t.Fatalf("Detect() error = %v", err)
	}

	if hasDetected(result.Detected, IDKotlin) {
		t.Error("Kotlin detected from build.gradle.kts without an actual Kotlin plugin/dependency")
	}
}

// --- C / C++ ---

func TestDetect_CMakeCXX(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "CMakeLists.txt", "project(demo LANGUAGES CXX)\n")

	result, err := Detect(dir)
	if err != nil {
		t.Fatalf("Detect() error = %v", err)
	}

	checkDetected(t, result.Detected, IDCPP)
}

func TestDetect_CMakeCOnly(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "CMakeLists.txt", "project(demo LANGUAGES C)\n")

	result, err := Detect(dir)
	if err != nil {
		t.Fatalf("Detect() error = %v", err)
	}

	checkDetected(t, result.Detected, IDC)
	if hasDetected(result.Detected, IDCPP) {
		t.Error("C++ detected for a C-only CMakeLists.txt")
	}
}

func TestDetect_CppSourceExtensionFallback(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "main.cpp", "int main() { return 0; }\n")

	result, err := Detect(dir)
	if err != nil {
		t.Fatalf("Detect() error = %v", err)
	}

	cpp := checkDetected(t, result.Detected, IDCPP)
	if cpp.Evidence[0].Detail != "source file extension" {
		t.Errorf("C++ evidence = %+v, want it marked as source file extension (weaker) evidence", cpp.Evidence)
	}
}

// --- Cross-cutting: ordering, dedup, isolation, security ---

func TestDetect_DeterministicOrdering(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "package.json", `{"dependencies":{"react":"^18.0.0","vue":"^3.0.0"}}`)
	writeFile(t, dir, "tsconfig.json", `{}`)

	result, err := Detect(dir)
	if err != nil {
		t.Fatalf("Detect() error = %v", err)
	}

	var kinds []Kind
	var names []string
	for _, d := range result.Detected {
		kinds = append(kinds, d.Kind)
		names = append(names, d.Name)
	}

	// Languages first, then runtimes, then frameworks; alphabetical within
	// each group. TypeScript is present (tsconfig.json), so JavaScript is
	// not redundantly reported alongside it.
	want := []string{"TypeScript", "Node.js", "React", "Vue"}
	if !reflect.DeepEqual(names, want) {
		t.Errorf("order = %v, want %v", names, want)
	}
}

func TestDetect_DeduplicatesSpringBootAcrossDetectors(t *testing.T) {
	dir := t.TempDir()
	// Unrealistic combination, but exercises the dedup path directly: both
	// the Java and Kotlin detectors may independently find Spring Boot
	// evidence in the same project.
	writeFile(t, dir, "pom.xml", `<project><dependencies><dependency><artifactId>spring-boot-starter-web</artifactId></dependency></dependencies></project>`)
	writeFile(t, dir, "build.gradle.kts", "plugins {\n    kotlin(\"jvm\")\n    id(\"org.springframework.boot\")\n}\n")

	result, err := Detect(dir)
	if err != nil {
		t.Fatalf("Detect() error = %v", err)
	}

	var springBootCount int
	for _, d := range result.Detected {
		if d.ID == IDSpringBoot {
			springBootCount++
		}
	}
	if springBootCount != 1 {
		t.Errorf("Spring Boot reported %d times, want exactly once", springBootCount)
	}
}

func TestDetect_UnreadableManifestIsAFindingNotAFailure(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "package.json", `{"name":"demo"}`)
	writeFile(t, dir, "go.mod", "module demo\n\ngo 1.22\n")

	readFile := func(name string) ([]byte, error) {
		if filepath.Base(name) == "go.mod" {
			return nil, errors.New("permission denied")
		}
		return os.ReadFile(name)
	}

	result, err := detect(dir, readFile)
	if err != nil {
		t.Fatalf("detect() error = %v, want the scan to isolate the unreadable manifest", err)
	}
	if !hasDetected(result.Detected, IDJavaScript) {
		t.Error("JavaScript missing; one unreadable manifest should not hide the rest")
	}
	if hasDetected(result.Detected, IDGo) {
		t.Error("Go detected despite an unreadable go.mod")
	}
	if len(result.Findings) != 1 || result.Findings[0].Source != "go.mod" {
		t.Errorf("Findings = %+v, want exactly one for go.mod", result.Findings)
	}
}

func TestDetect_InvalidRootIsAnError(t *testing.T) {
	missing := filepath.Join(t.TempDir(), "missing")

	if _, err := Detect(missing); err == nil {
		t.Fatal("Detect() error = nil, want an error for a missing root")
	}
}

func TestDetect_SecretLookingValuesNeverAppearInResult(t *testing.T) {
	const secret = "SUPER_SECRET_VALUE_SHOULD_NEVER_APPEAR"
	dir := t.TempDir()
	writeFile(t, dir, "composer.json", `{"require":{"laravel/framework":"^10.0"}}`)
	// A manifest can legitimately contain secret-looking strings (e.g. in
	// an unrelated custom field); Detect must never retain or echo them.
	writeFile(t, dir, ".env", "API_KEY="+secret+"\n")

	result, err := Detect(dir)
	if err != nil {
		t.Fatalf("Detect() error = %v", err)
	}

	dump := fmt.Sprintf("%#v", result)
	if strings.Contains(dump, secret) {
		t.Fatalf("Result contains a secret-looking value:\n%s", dump)
	}
}
