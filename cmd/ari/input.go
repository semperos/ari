package main

import (
	"bufio"
	"strings"
)

func matchesSystemCommand(s string) bool {
	return strings.HasPrefix(s, ")")
}

// scanner represents the state of a readline scanner for the Goal REPL. It
// handles multi-line expressions.
type scanner struct {
	r      *bufio.Reader
	depth  []byte // (){}[] depth stack
	state  scanState
	done   bool
	escape bool
}

type scanState int

const (
	scanNormal scanState = iota
	scanComment
	scanCommentBlock
	scanString
	scanQuote
	scanRawQuote
)

const delimchars = ":+-*%!&|=~,^#_?@/`"

// readLine reads until the first end of line that also ends a Goal expression.
//
// Adapated from Goal's implementation.
//
//nolint:cyclop,funlen,gocognit,gocyclo // Vendored code
func (sc *scanner) readLine() (string, error) {
	*sc = scanner{r: sc.r, depth: sc.depth[:0]}
	sb := strings.Builder{}
	var qr byte = '/'
	nl := true       // at newline
	cs := true       // at possible start of comment
	cbdelim := false // after possible comment block start/end delimiter
	for {
		c, err := sc.r.ReadByte()
		if err != nil {
			return sb.String(), err
		}
		switch c {
		case '\r':
			continue
		default:
			sb.WriteByte(c)
		}
		switch sc.state {
		case scanNormal:
			switch c {
			case '\n':
				if len(sc.depth) == 0 || sc.done {
					return sb.String(), nil
				}
				cs = true
			case ' ', '\t':
				cs = true
			case '"':
				sc.state = scanString
				cs = false
			case '{', '(', '[':
				sc.depth = append(sc.depth, c)
				cs = true
			case '}', ')', ']':
				if len(sc.depth) > 0 && sc.depth[len(sc.depth)-1] == opening(c) {
					sc.depth = sc.depth[:len(sc.depth)-1]
				} else {
					// error, so return on next \n
					sc.done = true
				}
				cs = false
			default:
				if strings.IndexByte(delimchars, c) != -1 {
					acc := sb.String()
					switch {
					case strings.HasSuffix(acc[:len(acc)-1], "rx"):
						qr = c
						sc.state = scanQuote
					case strings.HasSuffix(acc[:len(acc)-1], "rq"):
						qr = c
						sc.state = scanRawQuote
					case strings.HasSuffix(acc[:len(acc)-1], "qq"):
						qr = c
						sc.state = scanQuote
					default:
						if c == '/' && cs {
							sc.state = scanComment
							cbdelim = nl
						}
					}
				}
				cs = false
			}
		case scanComment:
			if c == '\n' {
				//nolint:gocritic // vendored code
				if cbdelim {
					sc.state = scanCommentBlock
				} else if len(sc.depth) == 0 || sc.done {
					return sb.String(), nil
				} else {
					cs = true
					sc.state = scanNormal
				}
			}
			cbdelim = false
		case scanCommentBlock:
			if cbdelim && c == '\n' {
				if len(sc.depth) == 0 || sc.done {
					return sb.String(), nil
				}
				cs = true
				sc.state = scanNormal
			} else {
				cbdelim = nl && c == '\\'
			}
		case scanQuote:
			switch c {
			case '\\':
				sc.escape = !sc.escape
			case qr:
				if !sc.escape {
					sc.state = scanNormal
				}
				sc.escape = false
			default:
				sc.escape = false
			}
		case scanString:
			switch c {
			case '\\':
				sc.escape = !sc.escape
			case '"':
				if !sc.escape {
					sc.state = scanNormal
				}
				sc.escape = false
			default:
				sc.escape = false
			}
		case scanRawQuote:
			if c == qr {
				//nolint:govet // vendored code
				c, err := sc.r.ReadByte()
				if err != nil {
					return sb.String(), err
				}
				if c == qr {
					sb.WriteByte(c)
				} else {
					//nolint:errcheck // Goal impl says cannot error
					sc.r.UnreadByte() // cannot error
					sc.state = scanNormal
				}
			}
		}
		nl = c == '\n'
	}
}

// opening returns matching opening delimiter for a given closing delimiter.
//
// Adapted from Goal's implementation.
func opening(r byte) byte {
	switch r {
	case ')':
		return '('
	case ']':
		return '['
	case '}':
		return '{'
	default:
		return r
	}
}
