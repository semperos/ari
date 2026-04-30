package help_test

import (
	"strings"
	"testing"

	arihelp "github.com/semperos/ari/help"
)

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

// helpFn obtains the combined help function once per test run.
func helpFn(t *testing.T) func(string) string {
	t.Helper()
	fn := arihelp.HelpFunc()
	if fn == nil {
		t.Fatal("HelpFunc() returned nil")
	}
	return fn
}

// assertContains fails the test if text does not contain all of the given
// substrings.
func assertContains(t *testing.T, topic, text string, want ...string) {
	t.Helper()
	for _, w := range want {
		if !strings.Contains(text, w) {
			t.Errorf("help(%q): expected to contain %q\nfull output:\n%s", topic, w, text)
		}
	}
}

// assertNotEmpty fails the test if text is empty or all whitespace.
func assertNotEmpty(t *testing.T, topic, text string) {
	t.Helper()
	if strings.TrimSpace(text) == "" {
		t.Errorf("help(%q): got empty string", topic)
	}
}

// ---------------------------------------------------------------------------
// TestTopicsIndex – the "" key is the master topics index and must advertise
// all four extension sections so users can discover them.
// ---------------------------------------------------------------------------

func TestTopicsIndex(t *testing.T) {
	h := helpFn(t)
	text := h("")
	assertNotEmpty(t, "", text)
	assertContains(t, "", text,
		"fyne",
		"http",
		"sql",
		"ratelimit",
		// Core language sections that Goal itself defines should also appear
		// because we override the index to include extension rows.
		"TOPICS",
	)
}

// ---------------------------------------------------------------------------
// TestExtensionSectionOverviews – each extension has a dedicated section help
// text reachable by its short name.
// ---------------------------------------------------------------------------

func TestFyneSection(t *testing.T) {
	h := helpFn(t)
	text := h("fyne")
	assertNotEmpty(t, "fyne", text)
	assertContains(t, "fyne", text,
		"fyne.app",
		"fyne.window",
		"fyne.label",
		"fyne.button",
		"fyne.run",
		"fyne.do",
		"fyne.table",
		"fyne.confirm",
	)
}

func TestHTTPSection(t *testing.T) {
	h := helpFn(t)
	text := h("http")
	assertNotEmpty(t, "http", text)
	assertContains(t, "http", text,
		"http.get",
		"http.post",
		"http.client",
		"http.request",
		"AuthToken",
		"QueryParam",
	)
}

func TestSQLSection(t *testing.T) {
	h := helpFn(t)
	text := h("sql")
	assertNotEmpty(t, "sql", text)
	assertContains(t, "sql", text,
		"sql.open",
		"sql.close",
		"sql.q",
		"sql.exec",
		"sql.tx",
		"sqlite://",
	)
}

func TestRateLimitSection(t *testing.T) {
	h := helpFn(t)
	text := h("ratelimit")
	assertNotEmpty(t, "ratelimit", text)
	assertContains(t, "ratelimit", text,
		"ratelimit.new",
		"ratelimit.take",
		"leaky-bucket",
	)
}

// ---------------------------------------------------------------------------
// TestFyneVerbEntries – spot-check individual fyne.* verb entries.
// ---------------------------------------------------------------------------

func TestFyneVerbEntries(t *testing.T) {
	h := helpFn(t)

	cases := []struct {
		topic string
		want  []string
	}{
		{"fyne.app", []string{"fyne.app", "app ID"}},
		{"fyne.window", []string{"fyne.window", "title"}},
		{"fyne.run", []string{"fyne.run", "ShowAndRun"}},
		{"fyne.setcontent", []string{"fyne.setcontent", "widget"}},
		{"fyne.settitle", []string{"fyne.settitle"}},
		{"fyne.resize", []string{"fyne.resize", "width"}},
		{"fyne.title", []string{"fyne.title"}},
		{"fyne.label", []string{"fyne.label", "Label"}},
		{"fyne.entry", []string{"fyne.entry", "Entry"}},
		{"fyne.password", []string{"fyne.password"}},
		{"fyne.multiline", []string{"fyne.multiline"}},
		{"fyne.progress", []string{"fyne.progress", "ProgressBar"}},
		{"fyne.separator", []string{"fyne.separator"}},
		{"fyne.spacer", []string{"fyne.spacer"}},
		{"fyne.button", []string{"fyne.button", "Button"}},
		{"fyne.check", []string{"fyne.check", "Check"}},
		{"fyne.slider", []string{"fyne.slider", "Slider"}},
		{"fyne.select", []string{"fyne.select", "Select"}},
		{"fyne.text", []string{"fyne.text"}},
		{"fyne.settext", []string{"fyne.settext"}},
		{"fyne.value", []string{"fyne.value"}},
		{"fyne.setvalue", []string{"fyne.setvalue"}},
		{"fyne.enable", []string{"fyne.enable"}},
		{"fyne.disable", []string{"fyne.disable"}},
		{"fyne.show", []string{"fyne.show"}},
		{"fyne.hide", []string{"fyne.hide"}},
		{"fyne.refresh", []string{"fyne.refresh"}},
		{"fyne.vbox", []string{"fyne.vbox", "VBox"}},
		{"fyne.hbox", []string{"fyne.hbox", "HBox"}},
		{"fyne.scroll", []string{"fyne.scroll", "ScrollContainer"}},
		{"fyne.padded", []string{"fyne.padded", "Padded"}},
		{"fyne.center", []string{"fyne.center", "Center"}},
		{"fyne.split", []string{"fyne.split", "HSplit"}},
		{"fyne.border", []string{"fyne.border", "Border"}},
		{"fyne.tabs", []string{"fyne.tabs", "AppTabs"}},
		{"fyne.form", []string{"fyne.form", "Form"}},
		{"fyne.toolbar", []string{"fyne.toolbar", "Toolbar"}},
		{"fyne.action", []string{"fyne.action", "ToolbarAction"}},
		{"fyne.do", []string{"fyne.do", "main event thread"}},
		{"fyne.table", []string{"fyne.table", "Table"}},
		{"fyne.showinfo", []string{"fyne.showinfo", "information dialog"}},
		{"fyne.showerr", []string{"fyne.showerr", "error dialog"}},
		{"fyne.confirm", []string{"fyne.confirm", "confirm dialog"}},
	}

	for _, tc := range cases {
		t.Run(tc.topic, func(t *testing.T) {
			text := h(tc.topic)
			assertNotEmpty(t, tc.topic, text)
			assertContains(t, tc.topic, text, tc.want...)
		})
	}
}

// ---------------------------------------------------------------------------
// TestHTTPVerbEntries – spot-check individual http.* verb entries.
// ---------------------------------------------------------------------------

func TestHTTPVerbEntries(t *testing.T) {
	h := helpFn(t)

	cases := []struct {
		topic string
		want  []string
	}{
		{"http.get", []string{"http.get", "GET"}},
		{"http.post", []string{"http.post", "POST"}},
		{"http.put", []string{"http.put", "PUT"}},
		{"http.patch", []string{"http.patch", "PATCH"}},
		{"http.delete", []string{"http.delete", "DELETE"}},
		{"http.head", []string{"http.head", "HEAD"}},
		{"http.options", []string{"http.options", "OPTIONS"}},
		{"http.request", []string{"http.request", "Method"}},
		{"http.client", []string{"http.client", "BaseURL", "AuthToken", "RetryCount"}},
	}

	for _, tc := range cases {
		t.Run(tc.topic, func(t *testing.T) {
			text := h(tc.topic)
			assertNotEmpty(t, tc.topic, text)
			assertContains(t, tc.topic, text, tc.want...)
		})
	}
}

// ---------------------------------------------------------------------------
// TestSQLVerbEntries – spot-check individual sql.* verb entries.
// ---------------------------------------------------------------------------

func TestSQLVerbEntries(t *testing.T) {
	h := helpFn(t)

	cases := []struct {
		topic string
		want  []string
	}{
		{"sql.open", []string{"sql.open", "scheme://"}},
		{"sql.close", []string{"sql.close"}},
		{"sql.q", []string{"sql.q", "SELECT"}},
		{"sql.exec", []string{"sql.exec", "INSERT"}},
		{"sql.tx", []string{"sql.tx", "transaction"}},
	}

	for _, tc := range cases {
		t.Run(tc.topic, func(t *testing.T) {
			text := h(tc.topic)
			assertNotEmpty(t, tc.topic, text)
			assertContains(t, tc.topic, text, tc.want...)
		})
	}
}

// ---------------------------------------------------------------------------
// TestRateLimitVerbEntries – spot-check individual ratelimit.* verb entries.
// ---------------------------------------------------------------------------

func TestRateLimitVerbEntries(t *testing.T) {
	h := helpFn(t)

	cases := []struct {
		topic string
		want  []string
	}{
		{"ratelimit.new", []string{"ratelimit.new", "requests/second"}},
		{"ratelimit.take", []string{"ratelimit.take", "1i"}},
	}

	for _, tc := range cases {
		t.Run(tc.topic, func(t *testing.T) {
			text := h(tc.topic)
			assertNotEmpty(t, tc.topic, text)
			assertContains(t, tc.topic, text, tc.want...)
		})
	}
}

// ---------------------------------------------------------------------------
// TestGoalCoreHelpPassthrough – a core Goal topic (e.g. "s" for syntax) must
// still return useful content: the Wrap call must not swallow Goal's own help.
// ---------------------------------------------------------------------------

func TestGoalCoreHelpPassthrough(t *testing.T) {
	h := helpFn(t)

	// "v" is Goal's verbs topic; "t" is value types; "a" is adverbs.
	for _, topic := range []string{"v", "t", "a", "io", "+"} {
		t.Run(topic, func(t *testing.T) {
			text := h(topic)
			assertNotEmpty(t, topic, text)
		})
	}
}

// ---------------------------------------------------------------------------
// TestUnknownTopicReturnsEmpty – a completely unknown key must return the
// empty string (Goal's help convention for "not found").
// ---------------------------------------------------------------------------

func TestUnknownTopicReturnsEmpty(t *testing.T) {
	h := helpFn(t)
	text := h("zzz.no.such.topic.ever")
	if text != "" {
		t.Errorf(`help("zzz.no.such.topic.ever"): expected "", got %q`, text)
	}
}
