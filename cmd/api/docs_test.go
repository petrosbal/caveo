package main

import (
	"fmt"
	"regexp"
	"slices"
	"strings"
	"testing"

	"github.com/petrosbal/caveo/internal/hasher"
)

var specPathLine = regexp.MustCompile(`(?m)^  (/\S*):`)

func specPaths(t *testing.T) []string {
	t.Helper()
	matches := specPathLine.FindAllStringSubmatch(string(openAPISpec), -1)

	var paths []string
	for _, match := range matches {
		paths = append(paths, match[1])
	}

	if len(paths) == 0 {
		t.Fatal("found no paths in openapi.yaml. is the parser broken?")
	}

	slices.Sort(paths)
	return paths
}

func routePaths(t *testing.T) []string {
	t.Helper()

	routes := newTestApp().routeTable()
	seen := map[string]bool{}

	var paths []string
	for _, rt := range routes {
		if strings.HasPrefix(rt.pattern, "/docs") {
			continue
		}
		if seen[rt.pattern] {
			continue
		}
		paths = append(paths, rt.pattern)
		seen[rt.pattern] = true
	}

	if len(paths) == 0 {
		t.Fatal("found no paths in routeTable()")
	}
	slices.Sort(paths)
	return paths
}

func TestDocsMatchRoutes(t *testing.T) {
	fromRoutes := routePaths(t)
	fromSpec := specPaths(t)

	inSpec := map[string]bool{}
	for _, path := range fromSpec {
		inSpec[path] = true
	}

	for _, path := range fromRoutes {
		if !inSpec[path] {
			t.Errorf("route %s not found in openapi.yaml\n(routes: %v,\n  spec: %v)", path, fromRoutes, fromSpec)
		}
	}

	inRoutes := map[string]bool{}
	for _, path := range fromRoutes {
		inRoutes[path] = true
	}

	for _, path := range fromSpec {
		if !inRoutes[path] {
			t.Errorf("openapi.yaml documents %s, which is not in routeTable()\n(routes: %v,\n  spec: %v)", path, fromRoutes, fromSpec)
		}
	}
}

func TestDocsPageMentionsEveryPath(t *testing.T) {
	for _, path := range specPaths(t) {
		if !strings.Contains(docsHTML, path) {
			t.Errorf("docs page does not mention %s", path)
		}
	}
}

func TestSpecDocumentsCurrentBounds(t *testing.T) {
	spec := string(openAPISpec)
	for _, want := range []string{
		fmt.Sprintf("m<=%d", hasher.MaxMemory),
		fmt.Sprintf("t<=%d", hasher.MaxIterations),
		fmt.Sprintf("p<=%d", hasher.MaxParallelism),
	} {
		if !strings.Contains(spec, want) {
			t.Errorf("openapi.yaml does not document %s", want)
		}
	}
}
