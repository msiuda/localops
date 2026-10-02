package technology

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

// fileExists reports whether root/name exists as a regular file, without
// reading its contents.
func fileExists(root, name string) (bool, error) {
	info, err := os.Stat(filepath.Join(root, name))
	if err != nil {
		if os.IsNotExist(err) {
			return false, nil
		}
		return false, err
	}
	return info.Mode().IsRegular(), nil
}

// readManifest reads root/name via readFile, reporting existed=false (not
// an error) when the file is simply absent.
func readManifest(root, name string, readFile readFileFunc) (data []byte, existed bool, err error) {
	data, err = readFile(filepath.Join(root, name))
	if err != nil {
		if os.IsNotExist(err) {
			return nil, false, nil
		}
		return nil, false, err
	}
	return data, true, nil
}

// rootEntryExtensions lists the file extensions present among root's
// immediate (non-recursive) entries. It is used only as a last-resort,
// weaker evidence signal when no manifest/build-tool marker exists —
// detection never walks a project's full source tree, that is Environment's
// separate concern.
func rootEntryExtensions(root string) (map[string]bool, error) {
	entries, err := os.ReadDir(root)
	if err != nil {
		return nil, err
	}
	exts := make(map[string]bool)
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		exts[filepath.Ext(e.Name())] = true
	}
	return exts, nil
}

// rootFilesWithSuffix lists root's immediate (non-recursive) entries whose
// name ends with suffix (e.g. ".csproj").
func rootFilesWithSuffix(root, suffix string) ([]string, error) {
	entries, err := os.ReadDir(root)
	if err != nil {
		return nil, err
	}
	var matches []string
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		if strings.HasSuffix(e.Name(), suffix) {
			matches = append(matches, e.Name())
		}
	}
	return matches, nil
}

// containsAny reports whether text contains any of substrs.
func containsAny(text string, substrs ...string) bool {
	for _, s := range substrs {
		if strings.Contains(text, s) {
			return true
		}
	}
	return false
}

// declaresPackageToken conservatively reports whether text (a manifest
// format LocalOps does not fully parse, such as a TOML or Gradle/Ruby
// dependency block) declares name as a standalone token — e.g. `"django"`,
// `django =`, or a line in a requirements.txt file — rather than name
// merely appearing as a substring of something unrelated.
//
// This is deliberately not a full parser for the format: it is a small,
// conservative text check reading only what is needed to confirm a
// dependency declaration, per LocalOps's stdlib-first policy for formats
// with no parser in the standard library.
func declaresPackageToken(text, name string) bool {
	pattern := `(?i)[\s"'(]` + regexp.QuoteMeta(name) + `[\s"'=<>:,)\[]`
	// Also match at the very start of a line/file, since the pattern above
	// requires a preceding delimiter character.
	return regexp.MustCompile(pattern).MatchString(text) ||
		regexp.MustCompile(`(?im)^`+regexp.QuoteMeta(name)+`[\s"'=<>:,)\[]`).MatchString(text)
}
