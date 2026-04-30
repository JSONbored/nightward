package main

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

var (
	testcasePattern = regexp.MustCompile(`<testcase\b([^>]*)>`)
	attrPattern     = regexp.MustCompile(`\b(classname|name)="([^"]*)"`)
	funcPattern     = regexp.MustCompile(`func\s+([A-Za-z0-9_]+)\s*\(`)
)

func main() {
	if len(os.Args) != 3 {
		fmt.Fprintln(os.Stderr, "usage: normalize-go-junit <input> <output>")
		os.Exit(2)
	}

	modulePath, err := moduleName("go.mod")
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	inputPath := filepath.Clean(os.Args[1])
	outputPath := filepath.Clean(os.Args[2])

	input, err := os.ReadFile(inputPath) // #nosec G703 -- explicit local Makefile report input, not user-served content.
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	cache := map[string]string{}
	output := testcasePattern.ReplaceAllStringFunc(string(input), func(tag string) string {
		attrs := testcasePattern.FindStringSubmatch(tag)
		if len(attrs) != 2 || strings.Contains(attrs[1], ` file=`) || strings.Contains(attrs[1], ` filepath=`) {
			return tag
		}

		className, testName := testcaseAttrs(attrs[1])
		if className == "" || testName == "" {
			return tag
		}

		file := testFileFor(modulePath, className, testName, cache)
		if file == "" {
			return tag
		}
		return strings.TrimSuffix(tag, ">") + ` file="` + escapeAttr(file) + `">`
	})

	if err := os.WriteFile(outputPath, []byte(output), 0600); err != nil { // #nosec G703 -- explicit local Makefile report output with private permissions.
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func moduleName(path string) (string, error) {
	data, err := os.ReadFile(path) // #nosec G304 -- fixed repository go.mod path for local report normalization.
	if err != nil {
		return "", err
	}
	for _, line := range strings.Split(string(data), "\n") {
		fields := strings.Fields(line)
		if len(fields) == 2 && fields[0] == "module" {
			return fields[1], nil
		}
	}
	return "", fmt.Errorf("module directive not found in %s", path)
}

func testcaseAttrs(attrs string) (string, string) {
	values := map[string]string{}
	for _, match := range attrPattern.FindAllStringSubmatch(attrs, -1) {
		if len(match) == 3 {
			values[match[1]] = match[2]
		}
	}
	testName := strings.Split(values["name"], "/")[0]
	return values["classname"], testName
}

func testFileFor(modulePath, className, testName string, cache map[string]string) string {
	pkgDir := packageDir(modulePath, className)
	if pkgDir == "" {
		return ""
	}

	cacheKey := pkgDir + "\x00" + testName
	if cached, ok := cache[cacheKey]; ok {
		return cached
	}

	file := findTestFile(pkgDir, testName)
	cache[cacheKey] = file
	return file
}

func packageDir(modulePath, className string) string {
	if className == modulePath {
		return "."
	}
	if !strings.HasPrefix(className, modulePath+"/") {
		return ""
	}
	return filepath.Clean(strings.TrimPrefix(className, modulePath+"/"))
}

func findTestFile(dir, testName string) string {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return ""
	}

	var fallback string
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), "_test.go") {
			continue
		}
		path := filepath.Join(dir, entry.Name())
		if fallback == "" {
			fallback = path
		}
		data, err := os.ReadFile(path) // #nosec G304 -- path comes from repository test file discovery.
		if err != nil {
			continue
		}
		for _, match := range funcPattern.FindAllStringSubmatch(string(data), -1) {
			if len(match) == 2 && match[1] == testName {
				return filepath.ToSlash(path)
			}
		}
	}
	return filepath.ToSlash(fallback)
}

func escapeAttr(value string) string {
	replacer := strings.NewReplacer(
		`&`, `&amp;`,
		`"`, `&quot;`,
		`<`, `&lt;`,
		`>`, `&gt;`,
	)
	return replacer.Replace(value)
}
