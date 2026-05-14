package ari

// Options configures which ari extensions and embedded source roots are
// installed into a Goal context.
//
// Use DefaultOptions for a sensible everything-but-Fyne setup, or build
// your own from the zero value if you need a minimal context.
type Options struct {
	// Prefix is the namespace prefix forwarded to each extension's
	// Import(ctx, prefix) call. Almost always "".
	Prefix string

	// EnableOSIO registers Goal's os package (print, say, read, open,
	// run, shell, …).
	EnableOSIO bool

	// EnableMath registers Goal's math package (sin, cos, sqrt, …).
	EnableMath bool

	// EnableBase64 registers base64.enc / base64.dec / base64.urlenc /
	// base64.urldec.
	EnableBase64 bool

	// EnableZip registers zip.open / zip.write.
	EnableZip bool

	// EnableRateLimit registers ratelimit.new / ratelimit.take.
	EnableRateLimit bool

	// EnableHTTP registers http.client / http.get / http.post / ….
	EnableHTTP bool

	// EnableSQL registers sql.open / sql.close / sql.q / sql.exec /
	// sql.tx (modernc.org/sqlite driver).
	EnableSQL bool

	// EnableFyne registers Fyne GUI verbs (fyne.app, fyne.window, …).
	// Off by default — pulls in heavy desktop/graphics dependencies.
	EnableFyne bool

	// EnableArilib mounts the embedded ari helper source tree at the
	// global "arilib". Goal scripts can then do: arilib import "util".
	EnableArilib bool

	// EnableGoallib mounts the embedded Goal stdlib source tree at the
	// global "goallib". Goal scripts can then do: goallib import "fmt".
	EnableGoallib bool
}

// DefaultOptions enables everything except Fyne and uses the empty prefix.
// This matches the configuration of the standalone `ari` interpreter.
func DefaultOptions() Options {
	return Options{
		EnableOSIO:      true,
		EnableMath:      true,
		EnableBase64:    true,
		EnableZip:       true,
		EnableRateLimit: true,
		EnableHTTP:      true,
		EnableSQL:       true,
		EnableFyne:      false,
		EnableArilib:    true,
		EnableGoallib:   true,
	}
}

// FullOptions is DefaultOptions plus Fyne. Use this for the full standalone
// `ari` desktop interpreter.
func FullOptions() Options {
	o := DefaultOptions()
	o.EnableFyne = true
	return o
}
