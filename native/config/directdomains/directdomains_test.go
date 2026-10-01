package directdomains

import (
	"os"
	P "path"
	"reflect"
	"strings"
	"testing"
)

func TestParseNormalizesAndDeduplicates(t *testing.T) {
	got := Parse(strings.Join([]string{
		"  Example.COM  ",
		"example.com",
		"api.example.com # 业务接口",
		"",
		"   ",
		"# 整行注释",
	}, "\n"))

	want := []string{"example.com", "api.example.com"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("want %v, got %v", want, got)
	}
}

func TestParseDropsInvalidEntries(t *testing.T) {
	got := Parse(strings.Join([]string{
		"ok.example.com",
		"bad domain.com",
		"bad,comma.com",
		"-leading.example.com",
		"trailing-.example.com",
		"http://scheme.example.com",
		"a..b.com",
		strings.Repeat("a", MaxDomainLabelLength+1) + ".com",
		strings.Repeat("a", 1) + "." + strings.Repeat("b", 300),
		"good.example.net",
	}, "\n"))

	want := []string{"ok.example.com", "good.example.net"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("want %v, got %v", want, got)
	}
}

func TestIsValidAcceptsUnderscoreAndHyphenInsideLabels(t *testing.T) {
	for _, domain := range []string{"a-b.example.com", "a_b.example.com", "1.2.3.4", "x.io"} {
		if !IsValid(domain) {
			t.Fatalf("%q should be valid", domain)
		}
	}

	for _, domain := range []string{"", "-a.com", "a-.com", "a_b-.com", "a. b.com"} {
		if IsValid(domain) {
			t.Fatalf("%q should be invalid", domain)
		}
	}
}

func TestLoadReturnsNilWhenFileIsMissing(t *testing.T) {
	if got := Load(t.TempDir()); got != nil {
		t.Fatalf("missing file must yield nil, got %v", got)
	}
}

func TestLoadReadsFromProfileDirectory(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(P.Join(dir, FileName), []byte("example.com\ncdn.example.com\n"), 0o600); err != nil {
		t.Fatal(err)
	}

	want := []string{"example.com", "cdn.example.com"}
	if got := Load(dir); !reflect.DeepEqual(got, want) {
		t.Fatalf("want %v, got %v", want, got)
	}
}
