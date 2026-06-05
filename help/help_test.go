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
		"http",
		"sql",
		"ratelimit",
		// Core language sections that Goal itself defines should also appear
		// because we override the index to include extension rows.
		"TOPICS",
		// helps must appear in the index so users can discover it.
		"helps",
	)
}

// ---------------------------------------------------------------------------
// TestExtensionSectionOverviews – each extension has a dedicated section help
// text reachable by its short name.
// ---------------------------------------------------------------------------

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
// TestGoalBuiltinExtensions – Goal's own optional extensions (zip, base64,
// math) must be reachable now that their HelpFunc() calls are wired in.
// ---------------------------------------------------------------------------

func TestZipSection(t *testing.T) {
	h := helpFn(t)
	for _, topic := range []string{"zip", "archive/zip"} {
		t.Run(topic, func(t *testing.T) {
			text := h(topic)
			assertNotEmpty(t, topic, text)
			assertContains(t, topic, text, "zip.open", "zip.write")
		})
	}
}

func TestZipVerbEntries(t *testing.T) {
	h := helpFn(t)
	cases := []struct {
		topic string
		want  []string
	}{
		{"zip.open", []string{"zip.open", "file system"}},
		{"zip.write", []string{"zip.write", "zip file"}},
	}
	for _, tc := range cases {
		t.Run(tc.topic, func(t *testing.T) {
			text := h(tc.topic)
			assertNotEmpty(t, tc.topic, text)
			assertContains(t, tc.topic, text, tc.want...)
		})
	}
}

func TestBase64Section(t *testing.T) {
	h := helpFn(t)
	for _, topic := range []string{"base64", "encoding/base64"} {
		t.Run(topic, func(t *testing.T) {
			text := h(topic)
			assertNotEmpty(t, topic, text)
			assertContains(t, topic, text, "base64.enc", "base64.dec")
		})
	}
}

func TestBase64VerbEntries(t *testing.T) {
	h := helpFn(t)
	cases := []struct {
		topic string
		want  []string
	}{
		{"base64.enc", []string{"base64.enc", "encode"}},
		{"base64.urlenc", []string{"base64.urlenc", "url"}},
		{"base64.dec", []string{"base64.dec", "decode"}},
		{"base64.urldec", []string{"base64.urldec", "url"}},
	}
	for _, tc := range cases {
		t.Run(tc.topic, func(t *testing.T) {
			text := h(tc.topic)
			assertNotEmpty(t, tc.topic, text)
			assertContains(t, tc.topic, text, tc.want...)
		})
	}
}

func TestMathSection(t *testing.T) {
	h := helpFn(t)
	text := h("math")
	assertNotEmpty(t, "math", text)
	assertContains(t, "math", text, "math.", "acos", "log2")
}

func TestMathVerbEntries(t *testing.T) {
	h := helpFn(t)
	for _, topic := range []string{"math.acos", "math.log2", "math.tanh", "math.cbrt"} {
		t.Run(topic, func(t *testing.T) {
			text := h(topic)
			assertNotEmpty(t, topic, text)
			// All math verbs resolve to the shared math help block.
			assertContains(t, topic, text, "math.")
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
// ---------------------------------------------------------------------------
// TestHelpsVerbEntry – the "helps" key must describe the verb and its purpose.
// ---------------------------------------------------------------------------

func TestHelpsVerbEntry(t *testing.T) {
	h := helpFn(t)
	text := h("helps")
	assertNotEmpty(t, "helps", text)
	assertContains(t, "helps", text,
		"helps",  // verb name present
		"string", // describes the return type
		"help",   // relates it to help
	)
}

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
