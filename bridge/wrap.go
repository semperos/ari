package bridge

import (
	"fmt"

	goal "codeberg.org/anaseto/goal"
	"github.com/nooga/let-go/pkg/vm"
)

// GoalValue is an opaque let-go handle for a Goal value the bridge could
// not (or chose not to) map to a native let-go type. It is callable when
// the underlying goal.V is a function value, so:
//
//	(def f (goal/get "myfn"))
//	(f 1 2 3)
//
// works for any Goal callable, even when the bridge has not registered
// the function under a let-go namespace var.
type GoalValue struct {
	ctx *goal.Context
	v   goal.V
}

// V returns the underlying goal value.
func (g *GoalValue) V() goal.V { return g.v }

// Type implements vm.Value.
func (g *GoalValue) Type() vm.ValueType { return goalValueType }

// Unbox implements vm.Value.
func (g *GoalValue) Unbox() interface{} { return g.v }

// String implements vm.Value via Goal's own append-formatter.
func (g *GoalValue) String() string {
	return string(g.v.Append(g.ctx, nil, false))
}

// Hash returns a coarse hash. Goal values do not expose a stable hash, so
// we hash the formatted representation. This is acceptable for our use
// (handles end up in let-go vars, not used as map keys hot-path).
func (g *GoalValue) Hash() uint32 {
	s := g.String()
	var h uint32 = 2166136261
	for i := 0; i < len(s); i++ {
		h ^= uint32(s[i])
		h *= 16777619
	}
	return h
}

// --- ValueType plumbing ---

type theGoalValueType struct{}

func (t *theGoalValueType) String() string     { return t.Name() }
func (t *theGoalValueType) Type() vm.ValueType { return vm.TypeType }
func (t *theGoalValueType) Unbox() interface{} { return nil }
func (t *theGoalValueType) Name() string       { return "ari/GoalValue" }
func (t *theGoalValueType) Box(bare interface{}) (vm.Value, error) {
	gv, ok := bare.(*GoalValue)
	if !ok {
		return vm.NIL, fmt.Errorf("ari/GoalValue: cannot box %T", bare)
	}
	return gv, nil
}

var goalValueType = &theGoalValueType{}

// wrapGoalFn wraps a Goal function value as a let-go NativeFn. Calls into
// it convert arguments LG->Goal via LGToGoal and the result Goal->LG via
// GoalToLG. Panics from goal.V.Apply (which signals errors that way) are
// converted to Go errors.
func (b *Bridge) wrapGoalFn(name string, fn goal.V) vm.Value {
	proxy := func(args []vm.Value) (ret vm.Value, err error) {
		defer func() {
			if r := recover(); r != nil {
				err = fmt.Errorf("goal panic in %q: %v", name, r)
				ret = vm.NIL
			}
		}()
		// goal.V.Apply expects args in *stack order* (right-to-left).
		// Reverse so callers can write natural left-to-right invocation.
		gargs := make([]goal.V, len(args))
		for i, a := range args {
			ga, cerr := b.LGToGoal(a)
			if cerr != nil {
				return vm.NIL, fmt.Errorf("goal %q: arg %d: %w", name, i, cerr)
			}
			gargs[len(args)-1-i] = ga
		}
		res := fn.Apply(b.Goal, gargs...)
		if res.IsError() {
			return vm.NIL, fmt.Errorf("goal %q: %s", name, res.Error().Msg(b.Goal))
		}
		return b.GoalToLG(res), nil
	}
	nf, _ := vm.NativeFnType.Wrap(proxy)
	if nfp, ok := nf.(*vm.NativeFn); ok && name != "" {
		nfp.SetName(name)
	}
	return nf
}
