package tui

import (
	"strings"
	"testing"
)

func TestRenderMarkdown_Headers(t *testing.T) {
	input := "## Section\n\nBody text here."
	result := renderMarkdown(input, 80)
	// Glamour processes the content: output must contain ANSI escape sequences.
	// The dark style keeps ## markers but colors them; we check that glamour ran.
	if !strings.Contains(result, "\x1b[") {
		t.Errorf("renderMarkdown output contains no ANSI sequences — glamour did not render: %q", result)
	}
	if !strings.Contains(result, "Section") {
		t.Errorf("renderMarkdown dropped header text: %q", result)
	}
}

func TestRenderMarkdown_ZeroWidth(t *testing.T) {
	input := "## Header\n\nBody."
	result := renderMarkdown(input, 0)
	if result == "" {
		t.Error("renderMarkdown returned empty string for zero-width fallback")
	}
}

func TestRenderMarkdown_NegativeWidth(t *testing.T) {
	input := "## Header"
	result := renderMarkdown(input, -6)
	if result != input {
		t.Errorf("expected raw string fallback, got %q", result)
	}
}

func TestRenderMarkdown_EmptyString(t *testing.T) {
	result := renderMarkdown("", 80)
	_ = result // must not panic; any value is acceptable
}

func TestRenderMarkdown_BoldAndItalic(t *testing.T) {
	input := "**bold text** and *italic text*"
	result := renderMarkdown(input, 80)
	if strings.Contains(result, "**") || strings.Contains(result, "*italic*") {
		t.Errorf("renderMarkdown left raw Markdown markers in output: %q", result)
	}
}

func TestRenderMarkdown_CodeBlock(t *testing.T) {
	input := "```go\nfmt.Println(\"hello\")\n```"
	result := renderMarkdown(input, 80)
	// Backtick fences must be consumed by glamour, not passed through literally.
	if strings.Contains(result, "```") {
		t.Errorf("renderMarkdown left raw code fence markers in output: %q", result)
	}
	// Syntax highlighting splits tokens with ANSI codes, so check tokens separately.
	if !strings.Contains(result, "fmt") || !strings.Contains(result, "Println") {
		t.Errorf("renderMarkdown dropped code block content: %q", result)
	}
}

func TestRenderMarkdown_TrailingNewlineStripped(t *testing.T) {
	input := "## Header\n\nBody."
	result := renderMarkdown(input, 80)
	if strings.HasSuffix(result, "\n") {
		t.Errorf("renderMarkdown output has trailing newline that would cause double blank line: %q", result)
	}
}
