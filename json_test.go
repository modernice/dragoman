package dragoman_test

import (
	"cmp"
	"slices"
	"testing"

	tcmp "github.com/google/go-cmp/cmp"
	"github.com/modernice/dragoman"
)

func TestJSONDiff(t *testing.T) {
	source := map[string]any{
		"hello": "Hello, World!",
		"bye":   "Goodbye!",
		"$contact": map[string]any{
			"email": "hello@example.com",
			"phone": "123-456-7890",
			"response": map[string]any{
				"message": "Hello!",
			},
		},
	}
	target := map[string]any{
		"hello": "Hello, World!",
		"$contact": map[string]any{
			"email": "hello@example.com",
		},
	}
	want := []dragoman.JSONPath{
		{"bye"},
		{"$contact", "phone"},
		{"$contact", "response", "message"},
	}

	paths, err := dragoman.JSONDiff(source, target)
	if err != nil {
		t.Fatalf("JSONDiff(%s, %s): %v", source, target, err)
	}

	if !equalPaths(want, paths) {
		t.Fatalf("JSONDiff(): got %v; want %v", paths, want)
	}
}

func TestJSONExtract(t *testing.T) {
	data := map[string]any{
		"hello": "Hello, World!",
		"bye":   "Goodbye!",
		"$contact": map[string]any{
			"email": "hello@example.com",
			"phone": "123-456-7890",
			"response": map[string]any{
				"message": "Hello!",
			},
		},
	}
	paths := []dragoman.JSONPath{
		{"bye"},
		{"$contact", "email"},
		{"$contact", "response", "message"},
	}

	want := map[string]any{
		"bye": "Goodbye!",
		"$contact": map[string]any{
			"email": "hello@example.com",
			"response": map[string]any{
				"message": "Hello!",
			},
		},
	}

	got, err := dragoman.JSONExtract(data, paths)
	if err != nil {
		t.Fatalf("JSONExtract(%s, %s): %v", data, paths, err)
	}

	if !tcmp.Equal(want, got) {
		t.Fatalf("JSONExtract(): got %v; want %v", got, want)
	}
}

func equalPaths(a, b []dragoman.JSONPath) bool {
	if len(a) != len(b) {
		return false
	}

	comparer := func(a, b dragoman.JSONPath) int {
		if v := cmp.Compare(len(a), len(b)); v != 0 {
			return v
		}

		for i := range a {
			if v := cmp.Compare(a[i], b[i]); v != 0 {
				return v
			}
		}

		return 0
	}

	slices.SortFunc(a, comparer)
	slices.SortFunc(b, comparer)

	return tcmp.Equal(a, b)
}
