package render_test

import (
	"strings"
	"testing"

	"github.com/manjunathrathod/claude-repo-factory/internal/render"
)

func TestString(t *testing.T) {
	e := render.New()

	got, err := e.String("greeting", "Hello {{ .Name }}", map[string]any{"Name": "widget"})
	if err != nil {
		t.Fatalf("String() error = %v", err)
	}
	if got != "Hello widget" {
		t.Errorf("String() = %q, want %q", got, "Hello widget")
	}
}

func TestMissingKeyIsAnError(t *testing.T) {
	e := render.New()

	_, err := e.String("greeting", "Hello {{ .Missing }}", map[string]any{"Name": "widget"})
	if err == nil {
		t.Fatal("String() = nil error, want a failure on a missing key")
	}
	if !strings.Contains(err.Error(), "greeting") {
		t.Errorf("String() error = %v, want it to name the template", err)
	}
}

func TestParseErrorIsWrapped(t *testing.T) {
	e := render.New()

	_, err := e.String("broken", "{{ .Name ", nil)
	if err == nil {
		t.Fatal("String() = nil error, want a parse failure")
	}
	if !strings.Contains(err.Error(), "render: parse broken") {
		t.Errorf("String() error = %v, want a wrapped parse error", err)
	}
}

func TestWithFuncsOverridesDefaults(t *testing.T) {
	e := render.New(render.WithFuncs(map[string]any{
		"shout": strings.ToUpper,
	}))

	got, err := e.String("shout", `{{ shout "hi" }}`, nil)
	if err != nil {
		t.Fatalf("String() error = %v", err)
	}
	if got != "HI" {
		t.Errorf("String() = %q, want HI", got)
	}
}

func TestNamingHelpers(t *testing.T) {
	tests := []struct {
		in     string
		kebab  string
		snake  string
		pascal string
		camel  string
		title  string
	}{
		{"widget", "widget", "widget", "Widget", "widget", "Widget"},
		{"my-widget", "my-widget", "my_widget", "MyWidget", "myWidget", "My Widget"},
		{"my_widget", "my-widget", "my_widget", "MyWidget", "myWidget", "My Widget"},
		{"My Widget", "my-widget", "my_widget", "MyWidget", "myWidget", "My Widget"},
		{"myWidget", "my-widget", "my_widget", "MyWidget", "myWidget", "My Widget"},
		{"HTTPServer", "http-server", "http_server", "HttpServer", "httpServer", "Http Server"},
		{"claude-repo-factory", "claude-repo-factory", "claude_repo_factory", "ClaudeRepoFactory", "claudeRepoFactory", "Claude Repo Factory"},
		{"", "", "", "", "", ""},
	}

	for _, tt := range tests {
		t.Run(tt.in, func(t *testing.T) {
			if got := render.Kebab(tt.in); got != tt.kebab {
				t.Errorf("Kebab(%q) = %q, want %q", tt.in, got, tt.kebab)
			}
			if got := render.Snake(tt.in); got != tt.snake {
				t.Errorf("Snake(%q) = %q, want %q", tt.in, got, tt.snake)
			}
			if got := render.Pascal(tt.in); got != tt.pascal {
				t.Errorf("Pascal(%q) = %q, want %q", tt.in, got, tt.pascal)
			}
			if got := render.Camel(tt.in); got != tt.camel {
				t.Errorf("Camel(%q) = %q, want %q", tt.in, got, tt.camel)
			}
			if got := render.Title(tt.in); got != tt.title {
				t.Errorf("Title(%q) = %q, want %q", tt.in, got, tt.title)
			}
		})
	}
}

func TestIndent(t *testing.T) {
	tests := []struct {
		name string
		n    int
		in   string
		want string
	}{
		{name: "indents each line", n: 2, in: "a\nb", want: "  a\n  b"},
		{name: "leaves blank lines alone", n: 2, in: "a\n\nb", want: "  a\n\n  b"},
		{name: "zero is a no-op", n: 0, in: "a\nb", want: "a\nb"},
		{name: "negative is a no-op", n: -4, in: "a", want: "a"},
		{name: "empty input", n: 4, in: "", want: ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := render.Indent(tt.n, tt.in); got != tt.want {
				t.Errorf("Indent(%d, %q) = %q, want %q", tt.n, tt.in, got, tt.want)
			}
		})
	}
}

func TestHelpersAreAvailableInTemplates(t *testing.T) {
	e := render.New()

	tmpl := `{{ kebab .Name }}|{{ snake .Name }}|{{ pascal .Name }}|{{ camel .Name }}|{{ upper .Name }}|{{ default "fallback" .Empty }}`
	got, err := e.String("helpers", tmpl, map[string]any{"Name": "My Widget", "Empty": ""})
	if err != nil {
		t.Fatalf("String() error = %v", err)
	}

	want := "my-widget|my_widget|MyWidget|myWidget|MY WIDGET|fallback"
	if got != want {
		t.Errorf("String() = %q, want %q", got, want)
	}
}

func TestYearHelperIsPresent(t *testing.T) {
	e := render.New()

	got, err := e.String("year", "{{ year }}", nil)
	if err != nil {
		t.Fatalf("String() error = %v", err)
	}
	if len(got) != 4 {
		t.Errorf("year = %q, want a four digit year", got)
	}
}
