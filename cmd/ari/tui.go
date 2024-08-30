package main

import (
	"fmt"

	"codeberg.org/anaseto/goal"
	"github.com/charmbracelet/lipgloss"
)

type TuiStyle struct {
	style lipgloss.Style
}

// LessT implements goal.BV.
func (tuiStyle *TuiStyle) LessT(y goal.BV) bool {
	// Goal falls back to ordering by type name,
	// and there is no other reasonable way to order
	// these HTTPClient structs.
	return tuiStyle.Type() < y.Type()
}

// Matches implements goal.BV.
func (tuiStyle *TuiStyle) Matches(_ goal.BV) bool {
	// lipgloss.Style has no public fields
	return false
}

// Type implements goal.BV.
func (tuiStyle *TuiStyle) Type() string {
	return "tui.TuiStyle"
}

// Append implements goal.BV.
func (tuiStyle *TuiStyle) Append(_ *goal.Context, dst []byte, _ bool) []byte {
	// Go prints nil as `<nil>` so following suit.
	return append(dst, fmt.Sprintf("<%v %#v>", tuiStyle.Type(), tuiStyle.style)...)
}

// TuiColor is a Goal value that abstracts over lipgloss color types. Currently only lipgloss.Color is supported,
// but this allows adding support for lipgloss.AdaptiveColor later without needing to change calling code.
type TuiColor struct {
	color lipgloss.Color
}

// LessT implements goal.BV.
func (tuiColor *TuiColor) LessT(y goal.BV) bool {
	// Goal falls back to ordering by type name,
	// and there is no other reasonable way to order
	// these HTTPClient structs.
	return tuiColor.Type() < y.Type()
}

// Matches implements goal.BV.
func (tuiColor *TuiColor) Matches(_ goal.BV) bool {
	// TODO This is just a string alias
	return false
}

// Type implements goal.BV.
func (tuiColor *TuiColor) Type() string {
	return "tui.TuiColor"
}

// Append implements goal.BV.
func (tuiColor *TuiColor) Append(_ *goal.Context, dst []byte, _ bool) []byte {
	// Go prints nil as `<nil>` so following suit.
	return append(dst, fmt.Sprintf("<%v %#v>", tuiColor.Type(), tuiColor.color)...)
}

const (
	monadic = 1
	dyadic  = 2
	triadic = 3
)

func VFTuiColor(_ *goal.Context, args []goal.V) goal.V {
	x := args[len(args)-1]
	colorS, ok := x.BV().(goal.S)
	switch len(args) {
	case monadic:
		if !ok {
			return panicType("tui.color s", "s", x)
		}
		color := lipgloss.Color(string(colorS))
		tuiColor := TuiColor{color}
		return goal.NewV(&tuiColor)
	default:
		return goal.Panicf("tui.color : too many arguments (%d), expects 1 argument", len(args))
	}
}

const quadrilateral = 4

// Implements tui.style monad.
//
//nolint:cyclop,funlen,gocognit,gocyclo // These dictionary translators are best kept together
func VFTuiStyle(_ *goal.Context, args []goal.V) goal.V {
	x := args[len(args)-1]
	styleD, okD := x.BV().(*goal.D)
	switch len(args) {
	case monadic:
		style := lipgloss.NewStyle()
		if !okD {
			return panicType("tui.style d", "d", x)
		}
		// TODO START HERE Process dictionary entries as in augmentRequestWithOptions
		styleKeys := styleD.KeyArray()
		styleValues := styleD.ValueArray()
		switch kas := styleKeys.(type) {
		case (*goal.AS):
			for i, k := range kas.Slice {
				value := styleValues.At(i)
				switch k {
				case "Align":
					switch {
					case value.IsF():
						style = style.Align(lipgloss.Position(value.F()))
					case value.IsI():
						style = style.Align(lipgloss.Position(value.I()))
					default:
						s, ok := value.BV().(goal.S)
						if !ok {
							return goal.NewPanicError(fmt.Errorf(`field Align supports either floats or one of "t", "r", "b", "l", or "c", `+
								"but received a %v: %v", value.Type(), value))
						}
						switch string(s) {
						case "t":
							style = style.Align(lipgloss.Top)
						case "r":
							style = style.Align(lipgloss.Right)
						case "b":
							style = style.Align(lipgloss.Bottom)
						case "l":
							style = style.Align(lipgloss.Left)
						case "c":
							style = style.Align(lipgloss.Center)
						default:
							return goal.NewPanicError(fmt.Errorf(`field Align supports either floats or one of "t", "r", "b", "l", or "c", `+
								"but received: %v", s))
						}
					}
				case "Background":
					tuiColor, ok := value.BV().(*TuiColor)
					if !ok {
						return goal.NewPanicError(fmt.Errorf("field Background must be a tui.color value, "+
							"but received a %v: %v", value.Type(), value))
					}
					style = style.Background(tuiColor.color)
				case "Blink":
					if value.IsTrue() {
						style = style.Blink(true)
					}
				case "Bold":
					if value.IsTrue() {
						style = style.Bold(true)
					}
				case "Border":
					//nolint:nestif // keep Border handling in one place
					if value.IsI() {
						style = style.Border(lipgloss.NormalBorder())
					} else {
						switch v := value.BV().(type) {
						case (goal.S):
							switch v {
							case "double":
								style = style.Border(lipgloss.DoubleBorder())
							case "hidden":
								style = style.Border(lipgloss.HiddenBorder())
							case "normal":
								style = style.Border(lipgloss.NormalBorder())
							case "rounded":
								style = style.Border(lipgloss.RoundedBorder())
							case "thick":
								style = style.Border(lipgloss.ThickBorder())
							default:
								return goal.NewPanicError(fmt.Errorf("field Border supports one of "+
									`"double", "hidden", "normal", "rounded", or "thick" border names, but `+
									"received %v", v))
							}
						case (*goal.AV):
							//nolint: mnd // Argument is a 5-element list of: border-type, top, right, bottom, left
							if v.Len() != 5 {
								return goal.NewPanicError(fmt.Errorf("field Border expects either a 0 or 1, or "+
									`a list of where the first item is one of "double", "hidden", "normal", "rounded", or "thick" `+
									"and the remainder are 0 or 1 for top, right, bottom, and left borders. "+
									"Received a list with %d items: %v", v.Len(), v))
							}
							borderS := v.Slice[0]
							s, ok := borderS.BV().(goal.S)
							if !ok {
								return goal.NewPanicError(fmt.Errorf("field Border expects either a 0 or 1, or "+
									`a list of where the first item is one of "double", "hidden", "normal", "rounded", or "thick" `+
									"and the remainder are 0 or 1 for top, right, bottom, and left borders. "+
									"Instead of s, received a %v: %v", borderS.Type(), borderS))
							}
							borderSidesAB := v.Slice[1:]
							bools := make([]bool, 0, v.Len()-1)
							for i, v := range borderSidesAB {
								if v.IsI() {
									if v.I() == 0 {
										bools = append(bools, false)
									} else {
										bools = append(bools, true)
									}
								} else {
									return goal.NewPanicError(fmt.Errorf("field Border expects either a 0 or 1, or "+
										`a list of where the first item is one of "double", "hidden", "normal", "rounded", or "thick" `+
										"and the remainder are 0 or 1 for top, right, bottom, and left borders. "+
										"For the number in position %d, received a %v: %v", i, v.Type(), v))
								}
							}
							switch s {
							case "double":
								style = style.Border(lipgloss.DoubleBorder(), bools...)
							case "hidden":
								style = style.Border(lipgloss.HiddenBorder(), bools...)
							case "normal":
								style = style.Border(lipgloss.NormalBorder(), bools...)
							case "rounded":
								style = style.Border(lipgloss.RoundedBorder(), bools...)
							case "thick":
								style = style.Border(lipgloss.ThickBorder(), bools...)
							default:
								return goal.NewPanicError(fmt.Errorf("field Border supports one of "+
									`"double", "hidden", "normal", "rounded", or "thick" border names, but `+
									"received %v", s))
							}
						default:
							return goal.NewPanicError(fmt.Errorf("field Border expects either a 0 or 1, or "+
								`a list of where the first item is one of "double", "hidden", "normal", "rounded", or "thick" `+
								"and the remainder are 0 or 1 for top, right, bottom, and left borders. "+
								"Received a %v: %v", value.Type(), value))
						}
					}
				case "BorderBackground":
					tuiColor, ok := value.BV().(*TuiColor)
					if !ok {
						return goal.NewPanicError(fmt.Errorf("field BorderBackground must be a tui.color value, "+
							"but received a %v: %v", value.Type(), value))
					}
					style = style.BorderBackground(tuiColor.color)
				case "BorderForeground":
					tuiColor, ok := value.BV().(*TuiColor)
					if !ok {
						return goal.NewPanicError(fmt.Errorf("field BorderForeground must be a tui.color value, "+
							"but received a %v: %v", value.Type(), value))
					}
					style = style.BorderForeground(tuiColor.color)

				case "Faint":
					if value.IsTrue() {
						style = style.Faint(true)
					}
				case "Foreground":
					tuiColor, ok := value.BV().(*TuiColor)
					if !ok {
						return goal.NewPanicError(fmt.Errorf("field Foreground must be a tui.color value, "+
							"but received a %v: %v", value.Type(), value))
					}
					style = style.Foreground(tuiColor.color)
				case "Height":
					switch {
					case value.IsI():
						style = style.Height(int(value.I()))
					default:
						return goal.NewPanicError(fmt.Errorf("field Height expects an integer, "+
							"but received a %v: %v", value.Type(), value))
					}
				case "Italic":
					if value.IsTrue() {
						style = style.Italic(true)
					}
				case "Margin":
					marginArgs := make([]int, 0, quadrilateral)
					switch v := value.BV().(type) {
					case *goal.AB:
						if v.Len() > quadrilateral {
							return goal.NewPanicError(fmt.Errorf("field Margin must be an array of 4 integers, "+
								"but received %v: %v", v.Len(), value))
						}
						for _, x := range v.Slice {
							marginArgs = append(marginArgs, int(x))
						}
					case *goal.AI:
						if v.Len() > quadrilateral {
							return goal.NewPanicError(fmt.Errorf("field Margin must be an array of 4 integers, "+
								"but received %v: %v", v.Len(), value))
						}
						for _, x := range v.Slice {
							marginArgs = append(marginArgs, int(x))
						}
					default:
						return goal.NewPanicError(fmt.Errorf("field Margin must be an array of 4 numbers, "+
							"but received a %v: %v", value.Type(), value))
					}
					style = style.Margin(marginArgs...)
				case "Padding":
					paddingArgs := make([]int, 0, quadrilateral)
					switch v := value.BV().(type) {
					case *goal.AB:
						if v.Len() > quadrilateral {
							return goal.NewPanicError(fmt.Errorf("field Padding must be an array of 4 integers, "+
								"but received %v: %v", v.Len(), value))
						}
						for _, x := range v.Slice {
							paddingArgs = append(paddingArgs, int(x))
						}
					case *goal.AI:
						if v.Len() > quadrilateral {
							return goal.NewPanicError(fmt.Errorf("field Padding must be an array of 4 integers, "+
								"but received %v: %v", v.Len(), value))
						}
						for _, x := range v.Slice {
							paddingArgs = append(paddingArgs, int(x))
						}
					default:
						return goal.NewPanicError(fmt.Errorf("field Padding must be an array of 4 numbers, "+
							"but received a %v: %v", value.Type(), value))
					}
					style = style.Padding(paddingArgs...)
				case "Reverse":
					if value.IsTrue() {
						style = style.Reverse(true)
					}
				case "Strikethrough":
					if value.IsTrue() {
						style = style.Strikethrough(true)
					}
				case "Underline":
					if value.IsTrue() {
						style = style.Underline(true)
					}
				case "Width":
					switch {
					case value.IsI():
						style = style.Width(int(value.I()))
					default:
						return goal.NewPanicError(fmt.Errorf("field Width expects an integer, "+
							"but received a %v: %v", value.Type(), value))
					}
				}
			}
		default:
			return goal.NewPanicError(fmt.Errorf("tui.style expects a Goal dictionary with string keys, but received a %v: %v",
				kas.Type(),
				kas))
		}
		tuiStyle := TuiStyle{style}
		return goal.NewV(&tuiStyle)
	default:
		return goal.Panicf("tui.style : too many arguments (%d), expects 1 argument", len(args))
	}
}

func VFTuiRender(_ *goal.Context, args []goal.V) goal.V {
	x := args[len(args)-1]
	tuiStyle, ok := x.BV().(*TuiStyle)
	switch len(args) {
	case dyadic:
		if !ok {
			return panicType("tui.render style s", "style", x)
		}
		y := args[0]
		s, ok := y.BV().(goal.S)
		if !ok {
			return panicType("tui.render style s", "s", y)
		}
		rendered := tuiStyle.style.Render(string(s))
		return goal.NewS(rendered)
	default:
		return goal.Panicf("tui.color : too many arguments (%d), expects 1 argument", len(args))
	}
}

// Copied from Goal's implementation, a panic value for type mismatches.
func panicType(op, sym string, x goal.V) goal.V {
	return goal.Panicf("%s : bad type %q in %s", op, x.Type(), sym)
}
