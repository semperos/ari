// Package fyne provides Goal bindings for the Fyne GUI toolkit.
//
// Import registers Fyne verbs into a Goal context. Call it as:
//
//	fyne.Import(ctx, "")
//
// which registers globals prefixed with "fyne." (e.g. fyne.app, fyne.window).
//
// # Value types
//
// Fyne objects are wrapped as Goal boxed values (BV):
//
//	fyne.app    – a running Fyne application
//	fyne.window – a Fyne window
//	fyne.widget – any CanvasObject (widget, container, canvas element)
//
// # Verb summary
//
// Monads (single arg):
//
//	fyne.app     – create new Fyne application
//	fyne.run     – ShowAndRun on a fyne.window (blocks until closed)
//	fyne.label   – create Label widget from string
//	fyne.entry   – create Entry widget (arg is placeholder string or "")
//	fyne.password – create password Entry
//	fyne.multiline – create multiline Entry
//	fyne.progress – create ProgressBar at given value (0.0..1.0)
//	fyne.separator – create Separator widget
//	fyne.spacer  – create Spacer widget (for toolbar/box layouts)
//	fyne.text    – get text from Label, Entry or Select widget
//	fyne.value   – get numeric value from Slider or ProgressBar widget
//	fyne.enable  – enable a widget
//	fyne.disable – disable a widget
//	fyne.show    – show a widget
//	fyne.hide    – hide a widget
//	fyne.refresh – refresh a widget
//	fyne.title   – get title string from a fyne.window
//	fyne.vbox    – create VBox container from array of widgets
//	fyne.hbox    – create HBox container from array of widgets
//	fyne.scroll  – wrap a widget in a ScrollContainer
//	fyne.padded  – wrap a widget in a Padded container
//	fyne.center  – wrap a widget in a Center container
//	fyne.do      – run a Goal function on the Fyne main event thread
//
// Dyads (two args, notation: left fyne.verb right):
//
//	fyne.window   – "title" fyne.window app  → create Window
//	fyne.setcontent – win fyne.setcontent widget → set window content
//	fyne.settitle – win fyne.settitle "new title"
//	fyne.resize   – win fyne.resize (w;h) → resize window to float size
//	fyne.button   – "label" fyne.button fn  → Button with tap callback
//	fyne.check    – "label" fyne.check fn   → Check with changed callback
//	fyne.slider   – (min;max) fyne.slider fn → Slider with changed callback
//	fyne.select   – options fyne.select fn  → Select with changed callback
//	fyne.settext  – widget fyne.settext "text" → set text
//	fyne.setvalue – widget fyne.setvalue n  → set numeric value
//	fyne.split    – "h"/"v" fyne.split (w1;w2) → HSplit or VSplit
//	fyne.showinfo – "title" fyne.showinfo ("msg";win) → information dialog
//	fyne.showerr  – "title" fyne.showerr ("msg";win) → error dialog
//	fyne.confirm  – "title" fyne.confirm ("msg";fn;win) → confirm dialog
//
// Bracket calls (multi-arg):
//
//	fyne.border[top;bottom;left;right;...rest] → Border container
//	fyne.tabs[("label";widget);...] → AppTabs container
//	fyne.form[("label";widget);...] → Form widget
//	fyne.toolbar[action;...] → Toolbar (actions from fyne.action)
//	fyne.action[icon;fn] → ToolbarAction (icon is a fyne.widget Resource)
//
// # Callbacks
//
// Callbacks receive a single Goal value:
//   - Button tap: fn called with 0i (integer zero) — ignore with {[_] ...}
//   - Check:      fn called with 1i or 0i (true/false)
//   - Slider:     fn called with float64 value
//   - Select:     fn called with selected string
//   - Confirm:    fn called with 1i or 0i
package fyne

import (
	"fmt"
	"math"
	"os"
	"sort"
	"time"

	goal "codeberg.org/anaseto/goal"
	fynesdk "fyne.io/fyne/v2"
	fyneapp "fyne.io/fyne/v2/app"
	fynecont "fyne.io/fyne/v2/container"
	fynedialog "fyne.io/fyne/v2/dialog"
	fynelayout "fyne.io/fyne/v2/layout"
	fynetheme "fyne.io/fyne/v2/theme"
	fynewidget "fyne.io/fyne/v2/widget"
)

// ---------------------------------------------------------------------------
// BV wrapper types
// ---------------------------------------------------------------------------

// App wraps a fyne.App as a Goal boxed value.
type App struct{ app fynesdk.App }

func (a *App) Append(_ *goal.Context, dst []byte, _ bool) []byte {
	return append(dst, "fyne.app"...)
}
func (a *App) Matches(y goal.BV) bool { yv, ok := y.(*App); return ok && a == yv }
func (a *App) Type() string           { return "fyne.app" }

// Win wraps a fyne.Window as a Goal boxed value.
type Win struct{ win fynesdk.Window }

func (w *Win) Append(_ *goal.Context, dst []byte, _ bool) []byte {
	return append(dst, fmt.Sprintf("fyne.window[%q]", w.win.Title())...)
}
func (w *Win) Matches(y goal.BV) bool { yv, ok := y.(*Win); return ok && w == yv }
func (w *Win) Type() string           { return "fyne.window" }

// Widget wraps a fyne.CanvasObject as a Goal boxed value.
type Widget struct{ obj fynesdk.CanvasObject }

func (w *Widget) Append(_ *goal.Context, dst []byte, _ bool) []byte {
	return append(dst, fmt.Sprintf("fyne.widget[%T]", w.obj)...)
}

func (w *Widget) Matches(y goal.BV) bool {
	yv, ok := y.(*Widget)
	return ok && w.obj == yv.obj
}
func (w *Widget) Type() string { return "fyne.widget" }

// newWidget returns a Goal V containing a wrapped CanvasObject.
func newWidget(obj fynesdk.CanvasObject) goal.V {
	return goal.NewV(&Widget{obj})
}

// ---------------------------------------------------------------------------
// Import
// ---------------------------------------------------------------------------

// Import registers all fyne verbs into ctx. When pfx is empty the globals
// are named "fyne.app", "fyne.window", etc. When pfx is non-empty it is
// prepended with a dot, so Import(ctx,"ui") yields "ui.fyne.app", etc.
func Import(ctx *goal.Context, pfx string) { //nolint:funlen
	ctx.RegisterExtension("fyne", "")
	if pfx != "" {
		pfx += "."
	}

	// reg registers a variadic function as both a keyword (so the scanner
	// emits the right token type for infix notation) and a global variable
	// (so bracket notation works too).  The keyword name and global name are
	// identical: e.g. "fyne.window" for both, making
	//   "title" fyne.window a       (infix dyad)
	//   fyne.window["title";a]      (bracket)
	// both valid.
	reg := func(name string, f goal.VariadicFunc, dyad bool) {
		fullname := pfx + name
		var v goal.V
		if dyad {
			v = ctx.RegisterDyad("."+fullname, f)
		} else {
			v = ctx.RegisterMonad("."+fullname, f)
		}
		ctx.AssignGlobal(fullname, v)
	}

	// ---- monads ----
	reg("fyne.app", vfApp, false)
	reg("fyne.run", vfRun, false)
	reg("fyne.label", vfLabel, false)
	reg("fyne.entry", vfEntry, false)
	reg("fyne.password", vfPassword, false)
	reg("fyne.multiline", vfMultiline, false)
	reg("fyne.progress", vfProgress, false)
	reg("fyne.separator", vfSeparator, false)
	reg("fyne.spacer", vfSpacer, false)
	reg("fyne.text", vfText, false)
	reg("fyne.value", vfValue, false)
	reg("fyne.enable", vfEnable, false)
	reg("fyne.disable", vfDisable, false)
	reg("fyne.show", vfShow, false)
	reg("fyne.hide", vfHide, false)
	reg("fyne.refresh", vfRefresh, false)
	reg("fyne.title", vfTitle, false)
	reg("fyne.vbox", vfVBox, false)
	reg("fyne.hbox", vfHBox, false)
	reg("fyne.scroll", vfScroll, false)
	reg("fyne.padded", vfPadded, false)
	reg("fyne.center", vfCenter, false)
	reg("fyne.do", wrapCtx(ctx, vfDo), false)
	reg("fyne.spinner", vfSpinner, false)
	reg("fyne.async", wrapCtx(ctx, vfAsync), false)
	reg("fyne.table", wrapCtx(ctx, vfTable), false)

	// ---- multi-arg (registered as monads so bracket syntax works freely) ----
	reg("fyne.border", wrapCtx(ctx, vfBorder), false)
	reg("fyne.tabs", wrapCtx(ctx, vfTabs), false)
	reg("fyne.form", wrapCtx(ctx, vfForm), false)
	reg("fyne.toolbar", wrapCtx(ctx, vfToolbar), false)
	reg("fyne.action", wrapCtx(ctx, vfAction), false)

	// ---- dyads ----
	reg("fyne.window", vfWindow, true)
	reg("fyne.setcontent", vfSetContent, true)
	reg("fyne.settitle", vfSetTitle, true)
	reg("fyne.resize", vfResize, true)
	reg("fyne.button", wrapCtxD(ctx, vfButton), true)
	reg("fyne.check", wrapCtxD(ctx, vfCheck), true)
	reg("fyne.slider", wrapCtxD(ctx, vfSlider), true)
	reg("fyne.select", wrapCtxD(ctx, vfSelect), true)
	reg("fyne.settext", vfSetText, true)
	reg("fyne.setvalue", vfSetValue, true)
	reg("fyne.split", vfSplit, true)
	reg("fyne.showinfo", vfShowInfo, true)
	reg("fyne.showerr", vfShowErr, true)
	reg("fyne.confirm", wrapCtxD(ctx, vfConfirm), true)
}

// wrapCtx returns a VariadicFunc that carries ctx in its closure (for
// functions that need to call back into Goal code).
func wrapCtx(ctx *goal.Context, f func(*goal.Context, []goal.V) goal.V) goal.VariadicFunc {
	return func(_ *goal.Context, args []goal.V) goal.V { return f(ctx, args) }
}

// wrapCtxD is identical to wrapCtx; the "D" suffix is a reminder that the
// function will be registered as a dyad.
func wrapCtxD(ctx *goal.Context, f func(*goal.Context, []goal.V) goal.V) goal.VariadicFunc {
	return func(_ *goal.Context, args []goal.V) goal.V { return f(ctx, args) }
}

// ---------------------------------------------------------------------------
// Helper: call a Goal function as a Fyne callback
// ---------------------------------------------------------------------------

// callFn calls the Goal function fn with a single argument v.
// If the call panics it prints the error to stderr and returns 0i.
func callFn(ctx *goal.Context, fn goal.V, v goal.V) {
	r := fn.ApplyAt(ctx, v)
	if r.IsPanic() {
		fmt.Fprintln(os.Stderr, "ari callback panic:", r.Sprint(ctx, true))
	}
}

// ---------------------------------------------------------------------------
// Helper: extract CanvasObject array from an AV Goal value
// ---------------------------------------------------------------------------

func toCanvasObjects(v goal.V) ([]fynesdk.CanvasObject, bool) {
	av, ok := v.BV().(*goal.AV)
	if !ok {
		// Maybe a single widget?
		if w, ok2 := v.BV().(*Widget); ok2 {
			return []fynesdk.CanvasObject{w.obj}, true
		}
		return nil, false
	}
	objs := make([]fynesdk.CanvasObject, 0, len(av.Slice))
	for _, el := range av.Slice {
		w, ok := el.BV().(*Widget)
		if !ok {
			return nil, false
		}
		objs = append(objs, w.obj)
	}
	return objs, true
}

// toCanvasObject extracts a single CanvasObject, or nil if the value is 0i.
func toCanvasObject(v goal.V) fynesdk.CanvasObject {
	if v.IsI() && v.I() == 0 {
		return nil
	}
	if w, ok := v.BV().(*Widget); ok {
		return w.obj
	}
	return nil
}

// toStringSlice converts a Goal AS or AV-of-strings to []string.
func toStringSlice(v goal.V) ([]string, bool) {
	switch xv := v.BV().(type) {
	case *goal.AS:
		return xv.Slice, true
	case *goal.AV:
		ss := make([]string, 0, len(xv.Slice))
		for _, el := range xv.Slice {
			s, ok := el.BV().(goal.S)
			if !ok {
				return nil, false
			}
			ss = append(ss, string(s))
		}
		return ss, true
	}
	return nil, false
}

// ---------------------------------------------------------------------------
// fyne.app  (monad)
// ---------------------------------------------------------------------------

// vfApp creates a new Fyne application.
// Usage: a:fyne.app 0.
func vfApp(_ *goal.Context, args []goal.V) goal.V {
	if len(args) > 1 {
		return goal.Panicf("fyne.app x : too many arguments (%d)", len(args))
	}
	// If arg is a non-empty string, use it as the app ID.
	if len(args) == 1 {
		if s, ok := args[0].BV().(goal.S); ok && s != "" {
			return goal.NewV(&App{fyneapp.NewWithID(string(s))})
		}
	}
	return goal.NewV(&App{fyneapp.New()})
}

// ---------------------------------------------------------------------------
// fyne.window  (dyad: "title" fyne.window app)
// ---------------------------------------------------------------------------

func vfWindow(_ *goal.Context, args []goal.V) goal.V {
	switch len(args) {
	case 1:
		// monadic: fyne.window app  (title defaults to "")
		a, ok := args[0].BV().(*App)
		if !ok {
			return goal.Panicf("fyne.window app : expected fyne.app, got %q", args[0].Type())
		}
		return goal.NewV(&Win{a.app.NewWindow("")})
	case 2:
		// dyadic: "title" fyne.window app
		// args[0] = app (right), args[1] = title (left)
		a, ok := args[0].BV().(*App)
		if !ok {
			return goal.Panicf("t fyne.window app : expected fyne.app in right arg, got %q", args[0].Type())
		}
		title, ok := args[1].BV().(goal.S)
		if !ok {
			return goal.Panicf("t fyne.window app : expected string title in left arg, got %q", args[1].Type())
		}
		return goal.NewV(&Win{a.app.NewWindow(string(title))})
	default:
		return goal.Panicf("fyne.window : too many arguments (%d)", len(args))
	}
}

// ---------------------------------------------------------------------------
// fyne.run  (monad: fyne.run win)
// ---------------------------------------------------------------------------

func vfRun(_ *goal.Context, args []goal.V) goal.V {
	if len(args) != 1 {
		return goal.Panicf("fyne.run win : expected 1 argument, got %d", len(args))
	}
	w, ok := args[0].BV().(*Win)
	if !ok {
		return goal.Panicf("fyne.run win : expected fyne.window, got %q", args[0].Type())
	}
	w.win.ShowAndRun()
	return goal.NewI(0)
}

// ---------------------------------------------------------------------------
// fyne.setcontent  (dyad: win fyne.setcontent widget)
// ---------------------------------------------------------------------------

func vfSetContent(_ *goal.Context, args []goal.V) goal.V {
	if len(args) != 2 {
		return goal.Panicf("win fyne.setcontent widget : expected 2 arguments, got %d", len(args))
	}
	// args[1] = win (left), args[0] = widget (right)
	w, ok := args[1].BV().(*Win)
	if !ok {
		return goal.Panicf("win fyne.setcontent widget : expected fyne.window in left arg, got %q", args[1].Type())
	}
	obj, ok := args[0].BV().(*Widget)
	if !ok {
		return goal.Panicf("win fyne.setcontent widget : expected fyne.widget in right arg, got %q", args[0].Type())
	}
	w.win.SetContent(obj.obj)
	return args[1] // return window for chaining
}

// ---------------------------------------------------------------------------
// fyne.settitle  (dyad: win fyne.settitle "new title")
// ---------------------------------------------------------------------------

func vfSetTitle(_ *goal.Context, args []goal.V) goal.V {
	if len(args) != 2 {
		return goal.Panicf("win fyne.settitle s : expected 2 arguments, got %d", len(args))
	}
	w, ok := args[1].BV().(*Win)
	if !ok {
		return goal.Panicf("win fyne.settitle s : expected fyne.window in left arg, got %q", args[1].Type())
	}
	s, ok := args[0].BV().(goal.S)
	if !ok {
		return goal.Panicf("win fyne.settitle s : expected string in right arg, got %q", args[0].Type())
	}
	w.win.SetTitle(string(s))
	return args[1]
}

// ---------------------------------------------------------------------------
// fyne.title  (monad: fyne.title win)
// ---------------------------------------------------------------------------

func vfTitle(_ *goal.Context, args []goal.V) goal.V {
	if len(args) != 1 {
		return goal.Panicf("fyne.title win : expected 1 argument, got %d", len(args))
	}
	w, ok := args[0].BV().(*Win)
	if !ok {
		return goal.Panicf("fyne.title win : expected fyne.window, got %q", args[0].Type())
	}
	return goal.NewS(w.win.Title())
}

// ---------------------------------------------------------------------------
// fyne.resize  (dyad: win fyne.resize (w;h))
// ---------------------------------------------------------------------------

func vfResize(_ *goal.Context, args []goal.V) goal.V {
	if len(args) != 2 {
		return goal.Panicf("win fyne.resize (w;h) : expected 2 arguments, got %d", len(args))
	}
	w, ok := args[1].BV().(*Win)
	if !ok {
		return goal.Panicf("win fyne.resize (w;h) : expected fyne.window in left arg, got %q", args[1].Type())
	}
	// right arg: (w;h) — AV or AF
	wf, hf, ok := toPair(args[0])
	if !ok {
		return goal.Panicf("win fyne.resize (w;h) : expected numeric pair (w;h) in right arg")
	}
	w.win.Resize(fynesdk.NewSize(float32(wf), float32(hf)))
	return args[1]
}

// toPair extracts two floats from an AV of 2 elements or AF of 2 elements.
func toPair(v goal.V) (float64, float64, bool) {
	switch xv := v.BV().(type) {
	case *goal.AV:
		if len(xv.Slice) == 2 {
			a, b := xv.Slice[0], xv.Slice[1]
			return toFloat(a), toFloat(b), true
		}
	case *goal.AF:
		if len(xv.Slice) == 2 {
			return xv.Slice[0], xv.Slice[1], true
		}
	case *goal.AI:
		if len(xv.Slice) == 2 {
			return float64(xv.Slice[0]), float64(xv.Slice[1]), true
		}
	}
	return 0, 0, false
}

func toFloat(v goal.V) float64 {
	if v.IsI() {
		return float64(v.I())
	}
	if v.IsF() {
		return v.F()
	}
	return 0
}

// ---------------------------------------------------------------------------
// Widgets – label, entry, password, multiline, progress, separator, spacer
// ---------------------------------------------------------------------------

// vfLabel creates a Label widget from a string.
// Usage: fyne.label "Hello, World!".
func vfLabel(_ *goal.Context, args []goal.V) goal.V {
	if len(args) != 1 {
		return goal.Panicf("fyne.label s : expected 1 argument, got %d", len(args))
	}
	s, ok := args[0].BV().(goal.S)
	if !ok {
		return goal.Panicf("fyne.label s : expected string, got %q", args[0].Type())
	}
	return newWidget(fynewidget.NewLabel(string(s)))
}

// vfEntry creates an Entry widget. The arg is used as placeholder text.
// Usage: fyne.entry "placeholder"  or  fyne.entry "".
func vfEntry(_ *goal.Context, args []goal.V) goal.V {
	if len(args) > 1 {
		return goal.Panicf("fyne.entry s : too many arguments (%d)", len(args))
	}
	e := fynewidget.NewEntry()
	if len(args) == 1 {
		if s, ok := args[0].BV().(goal.S); ok && s != "" {
			e.SetPlaceHolder(string(s))
		}
	}
	return newWidget(e)
}

// vfPassword creates a password Entry widget.
// Usage: fyne.password "placeholder".
func vfPassword(_ *goal.Context, args []goal.V) goal.V {
	if len(args) > 1 {
		return goal.Panicf("fyne.password s : too many arguments (%d)", len(args))
	}
	e := fynewidget.NewPasswordEntry()
	if len(args) == 1 {
		if s, ok := args[0].BV().(goal.S); ok && s != "" {
			e.SetPlaceHolder(string(s))
		}
	}
	return newWidget(e)
}

// vfMultiline creates a multiline Entry widget.
// Usage: fyne.multiline "placeholder".
func vfMultiline(_ *goal.Context, args []goal.V) goal.V {
	if len(args) > 1 {
		return goal.Panicf("fyne.multiline s : too many arguments (%d)", len(args))
	}
	e := fynewidget.NewMultiLineEntry()
	if len(args) == 1 {
		if s, ok := args[0].BV().(goal.S); ok && s != "" {
			e.SetPlaceHolder(string(s))
		}
	}
	return newWidget(e)
}

// vfProgress creates a ProgressBar at the given value (0.0..1.0).
// Usage: fyne.progress 0.0.
func vfProgress(_ *goal.Context, args []goal.V) goal.V {
	if len(args) > 1 {
		return goal.Panicf("fyne.progress n : too many arguments (%d)", len(args))
	}
	pb := fynewidget.NewProgressBar()
	if len(args) == 1 {
		pb.SetValue(toFloat(args[0]))
	}
	return newWidget(pb)
}

// vfSeparator creates a Separator widget.
func vfSeparator(_ *goal.Context, _ []goal.V) goal.V {
	return newWidget(fynewidget.NewSeparator())
}

// vfSpacer creates a layout.Spacer CanvasObject.
func vfSpacer(_ *goal.Context, _ []goal.V) goal.V {
	return newWidget(fynelayout.NewSpacer())
}

// ---------------------------------------------------------------------------
// fyne.button  (dyad: "label" fyne.button fn)
// ---------------------------------------------------------------------------

func vfButton(ctx *goal.Context, args []goal.V) goal.V {
	switch len(args) {
	case 1:
		// monadic: fyne.button "label" — no callback
		s, ok := args[0].BV().(goal.S)
		if !ok {
			return goal.Panicf("fyne.button s : expected string, got %q", args[0].Type())
		}
		return newWidget(fynewidget.NewButton(string(s), nil))
	case 2:
		// dyadic: "label" fyne.button fn
		// args[0] = fn (right), args[1] = label (left)
		s, ok := args[1].BV().(goal.S)
		if !ok {
			return goal.Panicf("s fyne.button f : expected string in left arg, got %q", args[1].Type())
		}
		fn := args[0]
		if !fn.IsFunction() {
			return goal.Panicf("s fyne.button f : expected function in right arg, got %q", fn.Type())
		}
		btn := fynewidget.NewButton(string(s), func() {
			callFn(ctx, fn, goal.NewI(0))
		})
		return newWidget(btn)
	default:
		return goal.Panicf("fyne.button : too many arguments (%d)", len(args))
	}
}

// ---------------------------------------------------------------------------
// fyne.check  (dyad: "label" fyne.check fn)
// ---------------------------------------------------------------------------

func vfCheck(ctx *goal.Context, args []goal.V) goal.V {
	switch len(args) {
	case 1:
		s, ok := args[0].BV().(goal.S)
		if !ok {
			return goal.Panicf("fyne.check s : expected string, got %q", args[0].Type())
		}
		return newWidget(fynewidget.NewCheck(string(s), nil))
	case 2:
		s, ok := args[1].BV().(goal.S)
		if !ok {
			return goal.Panicf("s fyne.check f : expected string in left arg, got %q", args[1].Type())
		}
		fn := args[0]
		if !fn.IsFunction() {
			return goal.Panicf("s fyne.check f : expected function in right arg, got %q", fn.Type())
		}
		chk := fynewidget.NewCheck(string(s), func(checked bool) {
			v := goal.NewI(0)
			if checked {
				v = goal.NewI(1)
			}
			callFn(ctx, fn, v)
		})
		return newWidget(chk)
	default:
		return goal.Panicf("fyne.check : too many arguments (%d)", len(args))
	}
}

// ---------------------------------------------------------------------------
// fyne.slider  (dyad: (min;max) fyne.slider fn)
// ---------------------------------------------------------------------------

func vfSlider(ctx *goal.Context, args []goal.V) goal.V {
	switch len(args) {
	case 1:
		// monadic: fyne.slider (min;max)
		mn, mx, ok := toPair(args[0])
		if !ok {
			return goal.Panicf("fyne.slider (min;max) : expected numeric pair, got %q", args[0].Type())
		}
		return newWidget(fynewidget.NewSlider(mn, mx))
	case 2:
		// dyadic: (min;max) fyne.slider fn
		// args[0] = fn (right), args[1] = (min;max) (left)
		fn := args[0]
		if !fn.IsFunction() {
			return goal.Panicf("(min;max) fyne.slider f : expected function in right arg, got %q", fn.Type())
		}
		mn, mx, ok := toPair(args[1])
		if !ok {
			return goal.Panicf("(min;max) fyne.slider f : expected numeric pair in left arg, got %q", args[1].Type())
		}
		sl := fynewidget.NewSlider(mn, mx)
		sl.OnChanged = func(v float64) {
			callFn(ctx, fn, goal.NewF(v))
		}
		return newWidget(sl)
	default:
		return goal.Panicf("fyne.slider : too many arguments (%d)", len(args))
	}
}

// ---------------------------------------------------------------------------
// fyne.select  (dyad: options fyne.select fn)
// ---------------------------------------------------------------------------

func vfSelect(ctx *goal.Context, args []goal.V) goal.V {
	switch len(args) {
	case 1:
		// monadic: fyne.select options
		opts, ok := toStringSlice(args[0])
		if !ok {
			return goal.Panicf("fyne.select opts : expected string array, got %q", args[0].Type())
		}
		return newWidget(fynewidget.NewSelect(opts, nil))
	case 2:
		// dyadic: options fyne.select fn
		// args[0] = fn (right), args[1] = options (left)
		fn := args[0]
		if !fn.IsFunction() {
			return goal.Panicf("opts fyne.select f : expected function in right arg, got %q", fn.Type())
		}
		opts, ok := toStringSlice(args[1])
		if !ok {
			return goal.Panicf("opts fyne.select f : expected string array in left arg, got %q", args[1].Type())
		}
		sel := fynewidget.NewSelect(opts, func(s string) {
			callFn(ctx, fn, goal.NewS(s))
		})
		return newWidget(sel)
	default:
		return goal.Panicf("fyne.select : too many arguments (%d)", len(args))
	}
}

// ---------------------------------------------------------------------------
// fyne.text  (monad: fyne.text widget)
// ---------------------------------------------------------------------------

func vfText(_ *goal.Context, args []goal.V) goal.V {
	if len(args) != 1 {
		return goal.Panicf("fyne.text widget : expected 1 argument, got %d", len(args))
	}
	w, ok := args[0].BV().(*Widget)
	if !ok {
		return goal.Panicf("fyne.text widget : expected fyne.widget, got %q", args[0].Type())
	}
	switch obj := w.obj.(type) {
	case *fynewidget.Label:
		return goal.NewS(obj.Text)
	case *fynewidget.Entry:
		return goal.NewS(obj.Text)
	case *fynewidget.Select:
		return goal.NewS(obj.Selected)
	default:
		return goal.Panicf("fyne.text widget : unsupported widget type %T", obj)
	}
}

// ---------------------------------------------------------------------------
// fyne.settext  (dyad: widget fyne.settext "text")
// ---------------------------------------------------------------------------

func vfSetText(_ *goal.Context, args []goal.V) goal.V {
	if len(args) != 2 {
		return goal.Panicf("widget fyne.settext s : expected 2 arguments, got %d", len(args))
	}
	// args[1] = widget (left), args[0] = text (right)
	w, ok := args[1].BV().(*Widget)
	if !ok {
		return goal.Panicf("widget fyne.settext s : expected fyne.widget in left arg, got %q", args[1].Type())
	}
	s, ok := args[0].BV().(goal.S)
	if !ok {
		return goal.Panicf("widget fyne.settext s : expected string in right arg, got %q", args[0].Type())
	}
	switch obj := w.obj.(type) {
	case *fynewidget.Label:
		obj.SetText(string(s))
	case *fynewidget.Entry:
		obj.SetText(string(s))
	case *fynewidget.Button:
		obj.SetText(string(s))
	default:
		return goal.Panicf("widget fyne.settext s : unsupported widget type %T", obj)
	}
	return args[1]
}

// ---------------------------------------------------------------------------
// fyne.value  (monad: fyne.value widget)
// ---------------------------------------------------------------------------

func vfValue(_ *goal.Context, args []goal.V) goal.V {
	if len(args) != 1 {
		return goal.Panicf("fyne.value widget : expected 1 argument, got %d", len(args))
	}
	w, ok := args[0].BV().(*Widget)
	if !ok {
		return goal.Panicf("fyne.value widget : expected fyne.widget, got %q", args[0].Type())
	}
	switch obj := w.obj.(type) {
	case *fynewidget.Slider:
		return goal.NewF(obj.Value)
	case *fynewidget.ProgressBar:
		return goal.NewF(obj.Value)
	case *fynewidget.Check:
		if obj.Checked {
			return goal.NewI(1)
		}
		return goal.NewI(0)
	default:
		return goal.Panicf("fyne.value widget : unsupported widget type %T", obj)
	}
}

// ---------------------------------------------------------------------------
// fyne.setvalue  (dyad: widget fyne.setvalue n)
// ---------------------------------------------------------------------------

func vfSetValue(_ *goal.Context, args []goal.V) goal.V {
	if len(args) != 2 {
		return goal.Panicf("widget fyne.setvalue n : expected 2 arguments, got %d", len(args))
	}
	w, ok := args[1].BV().(*Widget)
	if !ok {
		return goal.Panicf("widget fyne.setvalue n : expected fyne.widget in left arg, got %q", args[1].Type())
	}
	n := toFloat(args[0])
	switch obj := w.obj.(type) {
	case *fynewidget.Slider:
		obj.SetValue(n)
	case *fynewidget.ProgressBar:
		obj.SetValue(n)
	default:
		return goal.Panicf("widget fyne.setvalue n : unsupported widget type %T", obj)
	}
	return args[1]
}

// ---------------------------------------------------------------------------
// fyne.enable / fyne.disable / fyne.show / fyne.hide / fyne.refresh
// ---------------------------------------------------------------------------

type disableable interface {
	Enable()
	Disable()
}

func vfEnable(_ *goal.Context, args []goal.V) goal.V {
	if len(args) != 1 {
		return goal.Panicf("fyne.enable widget : expected 1 argument, got %d", len(args))
	}
	w, ok := args[0].BV().(*Widget)
	if !ok {
		return goal.Panicf("fyne.enable widget : expected fyne.widget, got %q", args[0].Type())
	}
	if d, ok := w.obj.(disableable); ok {
		d.Enable()
	}
	return args[0]
}

func vfDisable(_ *goal.Context, args []goal.V) goal.V {
	if len(args) != 1 {
		return goal.Panicf("fyne.disable widget : expected 1 argument, got %d", len(args))
	}
	w, ok := args[0].BV().(*Widget)
	if !ok {
		return goal.Panicf("fyne.disable widget : expected fyne.widget, got %q", args[0].Type())
	}
	if d, ok := w.obj.(disableable); ok {
		d.Disable()
	}
	return args[0]
}

func vfShow(_ *goal.Context, args []goal.V) goal.V {
	if len(args) != 1 {
		return goal.Panicf("fyne.show widget : expected 1 argument, got %d", len(args))
	}
	w, ok := args[0].BV().(*Widget)
	if !ok {
		return goal.Panicf("fyne.show widget : expected fyne.widget, got %q", args[0].Type())
	}
	w.obj.Show()
	return args[0]
}

func vfHide(_ *goal.Context, args []goal.V) goal.V {
	if len(args) != 1 {
		return goal.Panicf("fyne.hide widget : expected 1 argument, got %d", len(args))
	}
	w, ok := args[0].BV().(*Widget)
	if !ok {
		return goal.Panicf("fyne.hide widget : expected fyne.widget, got %q", args[0].Type())
	}
	w.obj.Hide()
	return args[0]
}

func vfRefresh(_ *goal.Context, args []goal.V) goal.V {
	if len(args) != 1 {
		return goal.Panicf("fyne.refresh widget : expected 1 argument, got %d", len(args))
	}
	w, ok := args[0].BV().(*Widget)
	if !ok {
		return goal.Panicf("fyne.refresh widget : expected fyne.widget, got %q", args[0].Type())
	}
	w.obj.Refresh()
	return args[0]
}

// ---------------------------------------------------------------------------
// Container verbs: vbox, hbox, scroll, padded, center, split
// ---------------------------------------------------------------------------

// vfVBox creates a VBox container from a Goal array of widgets.
// Usage: fyne.vbox (w1;w2;w3).
func vfVBox(_ *goal.Context, args []goal.V) goal.V {
	if len(args) != 1 {
		return goal.Panicf("fyne.vbox widgets : expected 1 argument, got %d", len(args))
	}
	objs, ok := toCanvasObjects(args[0])
	if !ok {
		return goal.Panicf("fyne.vbox widgets : expected array of fyne.widget values, got %q", args[0].Type())
	}
	return newWidget(fynecont.NewVBox(objs...))
}

// vfHBox creates an HBox container from a Goal array of widgets.
// Usage: fyne.hbox (w1;w2;w3).
func vfHBox(_ *goal.Context, args []goal.V) goal.V {
	if len(args) != 1 {
		return goal.Panicf("fyne.hbox widgets : expected 1 argument, got %d", len(args))
	}
	objs, ok := toCanvasObjects(args[0])
	if !ok {
		return goal.Panicf("fyne.hbox widgets : expected array of fyne.widget values, got %q", args[0].Type())
	}
	return newWidget(fynecont.NewHBox(objs...))
}

// vfScroll wraps a widget in a ScrollContainer.
// Usage: fyne.scroll widget.
func vfScroll(_ *goal.Context, args []goal.V) goal.V {
	if len(args) != 1 {
		return goal.Panicf("fyne.scroll widget : expected 1 argument, got %d", len(args))
	}
	w, ok := args[0].BV().(*Widget)
	if !ok {
		return goal.Panicf("fyne.scroll widget : expected fyne.widget, got %q", args[0].Type())
	}
	return newWidget(fynecont.NewScroll(w.obj))
}

// vfPadded wraps a widget in a Padded container.
func vfPadded(_ *goal.Context, args []goal.V) goal.V {
	if len(args) != 1 {
		return goal.Panicf("fyne.padded widget : expected 1 argument, got %d", len(args))
	}
	w, ok := args[0].BV().(*Widget)
	if !ok {
		return goal.Panicf("fyne.padded widget : expected fyne.widget, got %q", args[0].Type())
	}
	return newWidget(fynecont.NewPadded(w.obj))
}

// vfCenter wraps a widget in a Center container.
func vfCenter(_ *goal.Context, args []goal.V) goal.V {
	if len(args) != 1 {
		return goal.Panicf("fyne.center widget : expected 1 argument, got %d", len(args))
	}
	w, ok := args[0].BV().(*Widget)
	if !ok {
		return goal.Panicf("fyne.center widget : expected fyne.widget, got %q", args[0].Type())
	}
	return newWidget(fynecont.NewCenter(w.obj))
}

// vfSplit creates an HSplit or VSplit container.
// Usage: "h" fyne.split (w1;w2)  or  "v" fyne.split (w1;w2).
func vfSplit(_ *goal.Context, args []goal.V) goal.V {
	if len(args) != 2 {
		return goal.Panicf("dir fyne.split (w1;w2) : expected 2 arguments, got %d", len(args))
	}
	// args[0] = (w1;w2) (right), args[1] = direction (left)
	dir, ok := args[1].BV().(goal.S)
	if !ok {
		return goal.Panicf("dir fyne.split (w1;w2) : expected \"h\" or \"v\" in left arg, got %q", args[1].Type())
	}
	av, ok := args[0].BV().(*goal.AV)
	if !ok || len(av.Slice) != 2 {
		return goal.Panicf("dir fyne.split (w1;w2) : expected 2-element widget array in right arg")
	}
	w1, ok1 := av.Slice[0].BV().(*Widget)
	w2, ok2 := av.Slice[1].BV().(*Widget)
	if !ok1 || !ok2 {
		return goal.Panicf("dir fyne.split (w1;w2) : both elements must be fyne.widget values")
	}
	switch string(dir) {
	case "h":
		return newWidget(fynecont.NewHSplit(w1.obj, w2.obj))
	case "v":
		return newWidget(fynecont.NewVSplit(w1.obj, w2.obj))
	default:
		return goal.Panicf("dir fyne.split (w1;w2) : direction must be \"h\" or \"v\", got %q", string(dir))
	}
}

// ---------------------------------------------------------------------------
// fyne.border  (variadic: fyne.border[top;bottom;left;right;...rest])
// ---------------------------------------------------------------------------

// vfBorder creates a Border container.
// The first 4 args are top, bottom, left, right (use 0 for none).
// Remaining args are the centre content widgets.
// Usage: fyne.border[top;bottom;left;right;w1;w2;...].
func vfBorder(_ *goal.Context, args []goal.V) goal.V {
	// args are in stack order: last element is first positional arg
	// fyne.border[top;bottom;left;right;...]
	// → args = [...; right; left; bottom; top]  (reversed)
	if len(args) < 4 {
		return goal.Panicf("fyne.border[top;bottom;left;right;...] : expected at least 4 arguments, got %d", len(args))
	}
	n := len(args)
	top := toCanvasObject(args[n-1])
	bottom := toCanvasObject(args[n-2])
	left := toCanvasObject(args[n-3])
	right := toCanvasObject(args[n-4])

	rest := make([]fynesdk.CanvasObject, 0, n-4)
	for i := n - 5; i >= 0; i-- {
		obj := toCanvasObject(args[i])
		if obj != nil {
			rest = append(rest, obj)
		}
	}
	return newWidget(fynecont.NewBorder(top, bottom, left, right, rest...))
}

// ---------------------------------------------------------------------------
// fyne.tabs  (variadic: fyne.tabs[("label";content);...])
// ---------------------------------------------------------------------------

// vfTabs creates an AppTabs container.
// Each arg is a 2-element AV: (label-string; widget).
// Usage: fyne.tabs[("Tab1";w1);("Tab2";w2)].
func vfTabs(_ *goal.Context, args []goal.V) goal.V {
	if len(args) == 0 {
		return goal.Panicf("fyne.tabs : expected at least 1 argument")
	}
	items := make([]*fynecont.TabItem, 0, len(args))
	// args in stack order: first tab is at args[len-1]
	for i := len(args) - 1; i >= 0; i-- {
		item, err := toTabItem(args[i])
		if err != nil {
			return goal.Panicf("fyne.tabs : %v", err)
		}
		items = append(items, item)
	}
	return newWidget(fynecont.NewAppTabs(items...))
}

func toTabItem(v goal.V) (*fynecont.TabItem, error) {
	av, ok := v.BV().(*goal.AV)
	if !ok || len(av.Slice) != 2 {
		return nil, fmt.Errorf("expected (label;widget) pair, got %q", v.Type())
	}
	label, ok := av.Slice[0].BV().(goal.S)
	if !ok {
		return nil, fmt.Errorf("tab label must be a string, got %q", av.Slice[0].Type())
	}
	w, ok := av.Slice[1].BV().(*Widget)
	if !ok {
		return nil, fmt.Errorf("tab content must be a fyne.widget, got %q", av.Slice[1].Type())
	}
	return fynecont.NewTabItem(string(label), w.obj), nil
}

// ---------------------------------------------------------------------------
// fyne.form  (variadic: fyne.form[("label";widget);...])
// ---------------------------------------------------------------------------

// vfForm creates a Form widget from (label;widget) pairs.
// Usage: fyne.form[("Username";e1);("Password";e2)].
func vfForm(_ *goal.Context, args []goal.V) goal.V {
	if len(args) == 0 {
		return goal.Panicf("fyne.form : expected at least 1 argument")
	}
	items := make([]*fynewidget.FormItem, 0, len(args))
	// args in stack order: first item is at args[len-1]
	for i := len(args) - 1; i >= 0; i-- {
		item, err := toFormItem(args[i])
		if err != nil {
			return goal.Panicf("fyne.form : %v", err)
		}
		items = append(items, item)
	}
	return newWidget(fynewidget.NewForm(items...))
}

func toFormItem(v goal.V) (*fynewidget.FormItem, error) {
	av, ok := v.BV().(*goal.AV)
	if !ok || len(av.Slice) != 2 {
		return nil, fmt.Errorf("expected (label;widget) pair, got %q", v.Type())
	}
	label, ok := av.Slice[0].BV().(goal.S)
	if !ok {
		return nil, fmt.Errorf("form label must be a string, got %q", av.Slice[0].Type())
	}
	w, ok := av.Slice[1].BV().(*Widget)
	if !ok {
		return nil, fmt.Errorf("form widget must be a fyne.widget, got %q", av.Slice[1].Type())
	}
	return fynewidget.NewFormItem(string(label), w.obj), nil
}

// ---------------------------------------------------------------------------
// fyne.toolbar  (variadic: fyne.toolbar[action;...])
// fyne.action   (dyadic: "hint" fyne.action icon-resource)
// ---------------------------------------------------------------------------

// vfToolbar creates a Toolbar from ToolbarItem values.
// Usage: fyne.toolbar[fyne.action[icon;fn];fyne.action[icon;fn]].
func vfToolbar(_ *goal.Context, args []goal.V) goal.V {
	items := make([]fynewidget.ToolbarItem, 0, len(args))
	for i := len(args) - 1; i >= 0; i-- {
		ti, ok := args[i].BV().(toolbarItemVal)
		if !ok {
			return goal.Panicf("fyne.toolbar : expected toolbar item (from fyne.action), got %q", args[i].Type())
		}
		items = append(items, ti.item)
	}
	return newWidget(fynewidget.NewToolbar(items...))
}

type toolbarItemVal struct {
	item fynewidget.ToolbarItem
}

func (t toolbarItemVal) Append(_ *goal.Context, dst []byte, _ bool) []byte {
	return append(dst, "fyne.toolbar-item"...)
}
func (t toolbarItemVal) Matches(_ goal.BV) bool { return false }
func (t toolbarItemVal) Type() string           { return "fyne.toolbar-item" }

// vfAction creates a ToolbarAction.
// Usage: fyne.action[icon;fn]   where icon is a named Fyne theme icon or 0 for default
// icon arg: a fyne.widget wrapping a fyne.Resource, or a string icon name, or 0.
func vfAction(ctx *goal.Context, args []goal.V) goal.V {
	if len(args) < 1 || len(args) > 2 {
		return goal.Panicf("fyne.action[icon;fn] : expected 1 or 2 arguments, got %d", len(args))
	}
	// args in stack order: args[len-1]=icon, args[0]=fn (if 2 args)
	var icon fynesdk.Resource
	var fn goal.V
	if len(args) == 2 {
		icon = toResource(args[1]) // icon is first positional arg
		fn = args[0]
	} else {
		icon = toResource(args[0])
	}
	action := fynewidget.NewToolbarAction(icon, func() {
		if fn != (goal.V{}) && fn.IsFunction() {
			callFn(ctx, fn, goal.NewI(0))
		}
	})
	return goal.NewV(toolbarItemVal{action})
}

// toResource converts a Goal value to a fyne.Resource.
// Supports string icon names (mapped to theme icons) and Widget values that
// wrap a Resource. Returns a fallback icon for unrecognised values.
func toResource(v goal.V) fynesdk.Resource {
	if s, ok := v.BV().(goal.S); ok {
		return themeIcon(string(s))
	}
	// Use a generic document icon as a fallback
	return fynetheme.DocumentIcon()
}

// themeIcon returns a Fyne theme icon by name. Unrecognised names fall back to
// the document icon so callers always get a valid Resource.
func themeIcon(name string) fynesdk.Resource { //nolint:gocyclo,cyclop,funlen // exhaustive: covers all Fyne theme icons
	switch name {
	case "account":
		return fynetheme.AccountIcon()
	case "add", "plus":
		return fynetheme.ContentAddIcon()
	case "cancel", "close":
		return fynetheme.CancelIcon()
	case "check", "confirm":
		return fynetheme.ConfirmIcon()
	case "copy":
		return fynetheme.ContentCopyIcon()
	case "cut":
		return fynetheme.ContentCutIcon()
	case "paste":
		return fynetheme.ContentPasteIcon()
	case "delete", "remove", "trash":
		return fynetheme.DeleteIcon()
	case "document":
		return fynetheme.DocumentIcon()
	case "download":
		return fynetheme.DownloadIcon()
	case "edit":
		return fynetheme.DocumentCreateIcon()
	case "error":
		return fynetheme.ErrorIcon()
	case "file", "folder":
		return fynetheme.FolderIcon()
	case "folder-open":
		return fynetheme.FolderOpenIcon()
	case "grid":
		return fynetheme.GridIcon()
	case "help":
		return fynetheme.HelpIcon()
	case "history":
		return fynetheme.HistoryIcon()
	case "home":
		return fynetheme.HomeIcon()
	case "info":
		return fynetheme.InfoIcon()
	case "list":
		return fynetheme.ListIcon()
	case "mail":
		return fynetheme.MailComposeIcon()
	case "media-play":
		return fynetheme.MediaPlayIcon()
	case "media-pause":
		return fynetheme.MediaPauseIcon()
	case "media-stop":
		return fynetheme.MediaStopIcon()
	case "menu":
		return fynetheme.MenuIcon()
	case "navigate-back":
		return fynetheme.NavigateBackIcon()
	case "navigate-next":
		return fynetheme.NavigateNextIcon()
	case "question":
		return fynetheme.QuestionIcon()
	case "refresh":
		return fynetheme.ViewRefreshIcon()
	case "save":
		return fynetheme.DocumentSaveIcon()
	case "search":
		return fynetheme.SearchIcon()
	case "settings":
		return fynetheme.SettingsIcon()
	case "upload":
		return fynetheme.UploadIcon()
	case "view":
		return fynetheme.VisibilityIcon()
	case "warning":
		return fynetheme.WarningIcon()
	default:
		return fynetheme.DocumentIcon()
	}
}

// ---------------------------------------------------------------------------
// fyne.showinfo  (dyad: "title" fyne.showinfo ("msg";win))
// ---------------------------------------------------------------------------

// vfShowInfo shows an information dialog.
// Usage: "Title" fyne.showinfo ("message";win)
// Or bracket: fyne.showinfo["Title";"Message";win].
func vfShowInfo(_ *goal.Context, args []goal.V) goal.V {
	switch len(args) {
	case 3:
		// bracket: fyne.showinfo["title";"msg";win]
		// args[2]=title, args[1]=msg, args[0]=win  (stack order, last arg first)
		title, tOk := args[2].BV().(goal.S)
		msg, mOk := args[1].BV().(goal.S)
		win, wOk := args[0].BV().(*Win)
		if !tOk || !mOk || !wOk {
			return goal.Panicf("fyne.showinfo[title;msg;win] : type error")
		}
		fynedialog.ShowInformation(string(title), string(msg), win.win)
		return goal.NewI(0)
	case 2:
		// dyad: "title" fyne.showinfo ("msg";win)
		// args[1]=title (left), args[0]=("msg";win) (right)
		title, ok := args[1].BV().(goal.S)
		if !ok {
			return goal.Panicf("t fyne.showinfo (msg;win) : expected string title in left arg")
		}
		av, ok := args[0].BV().(*goal.AV)
		if !ok || len(av.Slice) != 2 {
			return goal.Panicf("t fyne.showinfo (msg;win) : expected (msg;win) pair in right arg")
		}
		msg, ok1 := av.Slice[0].BV().(goal.S)
		win, ok2 := av.Slice[1].BV().(*Win)
		if !ok1 || !ok2 {
			return goal.Panicf("t fyne.showinfo (msg;win) : pair must be (string;fyne.window), got (%s;%s)",
				av.Slice[0].Type(), av.Slice[1].Type())
		}
		fynedialog.ShowInformation(string(title), string(msg), win.win)
		return goal.NewI(0)
	default:
		return goal.Panicf("fyne.showinfo : expected 2 or 3 arguments, got %d", len(args))
	}
}

// ---------------------------------------------------------------------------
// fyne.showerr  (dyad: "title" fyne.showerr ("msg";win))
// ---------------------------------------------------------------------------

func vfShowErr(_ *goal.Context, args []goal.V) goal.V {
	switch len(args) {
	case 3:
		title, tOk := args[2].BV().(goal.S)
		msg, mOk := args[1].BV().(goal.S)
		win, wOk := args[0].BV().(*Win)
		if !tOk || !mOk || !wOk {
			return goal.Panicf("fyne.showerr[title;msg;win] : type error")
		}
		fynedialog.ShowError(fmt.Errorf("%s: %s", string(title), string(msg)), win.win)
		return goal.NewI(0)
	case 2:
		title, ok := args[1].BV().(goal.S)
		if !ok {
			return goal.Panicf("t fyne.showerr (msg;win) : expected string title in left arg")
		}
		av, ok := args[0].BV().(*goal.AV)
		if !ok || len(av.Slice) != 2 {
			return goal.Panicf("t fyne.showerr (msg;win) : expected (msg;win) pair in right arg")
		}
		msg, ok1 := av.Slice[0].BV().(goal.S)
		win, ok2 := av.Slice[1].BV().(*Win)
		if !ok1 || !ok2 {
			return goal.Panicf("t fyne.showerr (msg;win) : pair must be (string;fyne.window)")
		}
		fynedialog.ShowError(fmt.Errorf("%s: %s", string(title), string(msg)), win.win)
		return goal.NewI(0)
	default:
		return goal.Panicf("fyne.showerr : expected 2 or 3 arguments, got %d", len(args))
	}
}

// ---------------------------------------------------------------------------
// fyne.confirm  (dyad: "title" fyne.confirm ("msg";fn;win))
// ---------------------------------------------------------------------------

func vfConfirm(ctx *goal.Context, args []goal.V) goal.V {
	switch len(args) {
	case 4:
		// bracket: fyne.confirm["title";"msg";fn;win]
		// args[3]=title, args[2]=msg, args[1]=fn, args[0]=win
		title, tOk := args[3].BV().(goal.S)
		msg, mOk := args[2].BV().(goal.S)
		fn := args[1]
		win, wOk := args[0].BV().(*Win)
		if !tOk || !mOk || !fn.IsFunction() || !wOk {
			return goal.Panicf("fyne.confirm[title;msg;fn;win] : type error")
		}
		fynedialog.ShowConfirm(string(title), string(msg), func(confirmed bool) {
			v := goal.NewI(0)
			if confirmed {
				v = goal.NewI(1)
			}
			callFn(ctx, fn, v)
		}, win.win)
		return goal.NewI(0)
	case 2:
		// dyad: "title" fyne.confirm ("msg";fn;win)
		title, ok := args[1].BV().(goal.S)
		if !ok {
			return goal.Panicf("t fyne.confirm (msg;fn;win) : expected string title in left arg")
		}
		av, ok := args[0].BV().(*goal.AV)
		if !ok || len(av.Slice) != 3 {
			return goal.Panicf("t fyne.confirm (msg;fn;win) : expected (msg;fn;win) triple in right arg")
		}
		msg, ok1 := av.Slice[0].BV().(goal.S)
		fn := av.Slice[1]
		win, ok3 := av.Slice[2].BV().(*Win)
		if !ok1 || !fn.IsFunction() || !ok3 {
			return goal.Panicf("t fyne.confirm (msg;fn;win) : triple must be (string;function;fyne.window)")
		}
		fynedialog.ShowConfirm(string(title), string(msg), func(confirmed bool) {
			v := goal.NewI(0)
			if confirmed {
				v = goal.NewI(1)
			}
			callFn(ctx, fn, v)
		}, win.win)
		return goal.NewI(0)
	default:
		return goal.Panicf("fyne.confirm : expected 2 or 4 arguments, got %d", len(args))
	}
}

// ---------------------------------------------------------------------------
// fyne.spinner  (monad: fyne.spinner 0)
// ---------------------------------------------------------------------------

// vfSpinner creates an infinite progress bar (animated activity indicator).
// The animation starts automatically when the widget is shown.
// Usage: fyne.spinner 0
func vfSpinner(_ *goal.Context, args []goal.V) goal.V {
	if len(args) > 1 {
		return goal.Panicf("fyne.spinner : too many arguments (%d)", len(args))
	}
	return newWidget(fynewidget.NewProgressBarInfinite())
}

// ---------------------------------------------------------------------------
// fyne.async  (variadic)
// ---------------------------------------------------------------------------

// vfAsync runs a Goal function asynchronously in a background goroutine.
//
// One-argument form:
//
//	fyne.async f
//
// Runs f in a goroutine. f is called with 0i. Use fyne.do inside f to update
// the UI safely from the goroutine.
//
// Four-argument form:
//
//	fyne.async[workfn; onsuccess; ontimeout; secs]
//
//   - workfn    runs in a goroutine, called with 0i; its return value is the result.
//   - onsuccess is called on the Fyne main thread with the result when workfn finishes.
//   - ontimeout is called on the Fyne main thread with 0i if secs elapse before
//     workfn finishes, or if workfn returns a Goal panic (error).
//   - secs      timeout duration in seconds (float).
//
// Note: Goal's evaluation context is not goroutine-safe. This verb is designed
// for the common pattern where workfn performs I/O and returns a result, and the
// main thread is otherwise idle (waiting for Fyne events) during the fetch.
// Avoid pressing buttons while a load is in progress.
func vfAsync(ctx *goal.Context, args []goal.V) goal.V { //nolint:cyclop // switch on arg count
	switch len(args) {
	case 1:
		// fyne.async f — simple fire-and-forget goroutine
		fn := args[0]
		if !fn.IsFunction() {
			return goal.Panicf("fyne.async f : expected function, got %q", fn.Type())
		}
		go func() { callFn(ctx, fn, goal.NewI(0)) }()
		return goal.NewI(0)

	case 4:
		// fyne.async[workfn; onsuccess; ontimeout; secs]
		// args in Goal stack order (last positional arg first):
		//   args[0]=secs, args[1]=ontimeout, args[2]=onsuccess, args[3]=workfn
		workfn := args[3]
		onsuccess := args[2]
		ontimeout := args[1]
		secs := toFloat(args[0])
		if !workfn.IsFunction() {
			return goal.Panicf("fyne.async[workfn;...]: workfn must be a function, got %q", workfn.Type())
		}
		if !onsuccess.IsFunction() {
			return goal.Panicf("fyne.async[...;onsuccess;...]: onsuccess must be a function, got %q", onsuccess.Type())
		}
		if !ontimeout.IsFunction() {
			return goal.Panicf("fyne.async[...;ontimeout;...]: ontimeout must be a function, got %q", ontimeout.Type())
		}
		// Buffered channel of size 1 so the work goroutine never blocks on send
		// even if the timeout has already fired.
		done := make(chan goal.V, 1)
		// Work goroutine: evaluates workfn and sends result (or panic) to done.
		go func() {
			var result goal.V
			func() {
				defer func() {
					if r := recover(); r != nil {
						result = goal.Panicf("fyne.async: goroutine panic: %v", r)
					}
				}()
				result = workfn.ApplyAt(ctx, goal.NewI(0))
			}()
			done <- result
		}()
		// Coordinator goroutine: waits for work or timeout, then dispatches
		// the appropriate callback on the Fyne main thread.
		go func() {
			timer := time.NewTimer(time.Duration(float64(time.Second) * secs))
			defer timer.Stop()
			select {
			case result := <-done:
				if result.IsPanic() {
					fmt.Fprintln(os.Stderr, "fyne.async workfn error:", result.Sprint(ctx, true))
					fynesdk.Do(func() { callFn(ctx, ontimeout, goal.NewI(0)) })
				} else {
					fynesdk.Do(func() { callFn(ctx, onsuccess, result) })
				}
			case <-timer.C:
				fynesdk.Do(func() { callFn(ctx, ontimeout, goal.NewI(0)) })
			}
		}()
		return goal.NewI(0)

	default:
		return goal.Panicf("fyne.async : expected 1 or 4 arguments, got %d", len(args))
	}
}

// ---------------------------------------------------------------------------
// fyne.do  (monad: fyne.do fn)
// ---------------------------------------------------------------------------

// vfDo schedules a Goal function to run on the Fyne main event thread.
// This is useful when updating the UI from inside a goroutine.
// Usage: fyne.do {[_] fyne.settext lbl "updated"}.
func vfDo(ctx *goal.Context, args []goal.V) goal.V {
	if len(args) != 1 {
		return goal.Panicf("fyne.do f : expected 1 argument, got %d", len(args))
	}
	fn := args[0]
	if !fn.IsFunction() {
		return goal.Panicf("fyne.do f : expected function, got %q", fn.Type())
	}
	fynesdk.Do(func() {
		callFn(ctx, fn, goal.NewI(0))
	})
	return goal.NewI(0)
}

// ---------------------------------------------------------------------------
// fyne.table  (monad: fyne.table d)
// ---------------------------------------------------------------------------

// vfTable creates a sortable Table widget from a Goal dict (e.g. an sql.q
// result). The dict keys must be a string array (AS) of column names; the
// dict values must be an AV of per-column arrays.
//
// Clicking a column header sorts by that column (click again to reverse).
// Integers and floats sort numerically; everything else sorts lexicographically.
// NULL (NaN) values always sort last regardless of direction.
//
// Usage: tbl: fyne.table (db sql.q "SELECT ...").
func vfTable(ctx *goal.Context, args []goal.V) goal.V { //nolint:gocognit,gocyclo,cyclop,funlen,lll // complex sortable table
	if len(args) != 1 {
		return goal.Panicf("fyne.table d : expected 1 argument, got %d", len(args))
	}
	d, ok := args[0].BV().(*goal.D)
	if !ok {
		return goal.Panicf("fyne.table d : expected dict, got %q", args[0].Type())
	}

	// Column names
	keysAS, ok := d.KeyArray().(*goal.AS)
	if !ok {
		return goal.Panicf("fyne.table d : dict keys must be a string array (AS)")
	}
	colNames := keysAS.Slice
	numCols := len(colNames)

	// Column data arrays
	valsAV, ok := d.ValueArray().(*goal.AV)
	if !ok {
		return goal.Panicf("fyne.table d : dict values must be AV")
	}

	// Row count from first column (0 when table is empty)
	numRows := 0
	if numCols > 0 {
		if arr, ok2 := valsAV.Slice[0].BV().(goal.Array); ok2 {
			numRows = arr.Len()
		}
	}

	// Build display string matrix [row][col] and raw value matrix for
	// type-aware sorting (numeric vs lexicographic).
	cells := make([][]string, numRows)
	raw := make([][]goal.V, numRows)
	for r := range numRows {
		cells[r] = make([]string, numCols)
		raw[r] = make([]goal.V, numCols)
		for c := range numCols {
			colArr := valsAV.Slice[c].BV().(goal.Array) //nolint:errcheck,lll // type is guaranteed: valsAV is built from sql.q columnar result
			v := colArr.At(r)
			raw[r][c] = v
			switch {
			case v.IsF() && math.IsNaN(v.F()):
				cells[r][c] = "NULL"
			default:
				if s, ok3 := v.BV().(goal.S); ok3 {
					cells[r][c] = string(s)
				} else {
					cells[r][c] = v.Sprint(ctx, true)
				}
			}
		}
	}

	// Auto-size each column from the longest header or cell string.
	const (
		chWidth  = 8.5 // dp per character at default font size
		padding  = 16  // dp total horizontal padding per cell
		minWidth = 48  // dp floor
		maxWidth = 260 // dp ceiling
	)
	colWidths := make([]float32, numCols)
	for c, name := range colNames {
		maxLen := len(name) + 4 // +4 for the sort arrow (" ↑ ") headroom
		for r := range numRows {
			if l := len(cells[r][c]); l > maxLen {
				maxLen = l
			}
		}
		w := float32(maxLen)*chWidth + padding
		if w < minWidth {
			w = minWidth
		}
		if w > maxWidth {
			w = maxWidth
		}
		colWidths[c] = w
	}

	// ---------------------------------------------------------------------------
	// Sort state — mutated only from the Fyne main thread (button callbacks).
	// ---------------------------------------------------------------------------
	perm := make([]int, numRows) // perm[displayRow] = sourceRow
	for i := range perm {
		perm[i] = i
	}
	sortCol := -1 // -1 = unsorted
	sortAsc := true

	// isNull reports whether a raw value is a null (NaN sentinel).
	isNull := func(v goal.V) bool {
		return v.IsF() && math.IsNaN(v.F())
	}

	// sortByCol re-sorts perm by column c, toggling direction if c == sortCol.
	var tbl *fynewidget.Table
	sortByCol := func(c int) {
		if sortCol == c {
			sortAsc = !sortAsc
		} else {
			sortCol = c
			sortAsc = true
		}
		asc := sortAsc
		sort.SliceStable(perm, func(i, j int) bool {
			ri, rj := perm[i], perm[j]
			a, b := raw[ri][c], raw[rj][c]
			// NULLs always last
			na, nb := isNull(a), isNull(b)
			if na != nb {
				return !na
			}
			if na && nb {
				return false
			}
			// Numeric comparison
			if (a.IsI() || a.IsF()) && (b.IsI() || b.IsF()) {
				af, bf := toFloat(a), toFloat(b)
				if asc {
					return af < bf
				}
				return af > bf
			}
			// Lexicographic fallback (uses display strings)
			sa, sb := cells[ri][c], cells[rj][c]
			if asc {
				return sa < sb
			}
			return sa > sb
		})
		tbl.Refresh()
	}

	// headerLabel builds the column header text with a sort indicator.
	headerLabel := func(c int) string {
		if c != sortCol {
			return colNames[c]
		}
		if sortAsc {
			return colNames[c] + " ↑"
		}
		return colNames[c] + " ↓"
	}

	// Build table with sticky column headers.
	tbl = fynewidget.NewTableWithHeaders(
		func() (int, int) { return numRows, numCols },
		func() fynesdk.CanvasObject { return fynewidget.NewLabel("") },
		func(id fynewidget.TableCellID, obj fynesdk.CanvasObject) {
			lbl := obj.(*fynewidget.Label) //nolint:errcheck // type is guaranteed: CreateFunc returns a *Label
			if id.Row >= 0 && id.Row < numRows && id.Col >= 0 && id.Col < numCols {
				lbl.SetText(cells[perm[id.Row]][id.Col])
			}
		},
	)
	// Disable the row-number gutter — only column headers are needed.
	tbl.ShowHeaderColumn = false

	// Column headers are buttons so they are tappable for sorting.
	tbl.CreateHeader = func() fynesdk.CanvasObject {
		return fynewidget.NewButton("", nil)
	}
	tbl.UpdateHeader = func(id fynewidget.TableCellID, obj fynesdk.CanvasObject) {
		if id.Row == -1 && id.Col >= 0 && id.Col < numCols {
			c := id.Col
			btn := obj.(*fynewidget.Button) //nolint:errcheck // type is guaranteed: CreateHeader returns a *Button
			btn.SetText(headerLabel(c))
			btn.OnTapped = func() { sortByCol(c) }
			btn.Refresh()
		}
	}

	// Apply the computed column widths.
	for c, w := range colWidths {
		tbl.SetColumnWidth(c, w)
	}

	return newWidget(tbl)
}
