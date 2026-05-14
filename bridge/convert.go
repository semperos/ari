package bridge

import (
	"fmt"

	goal "codeberg.org/anaseto/goal"
	"github.com/nooga/let-go/pkg/vm"
)

// GoalToLG converts a Goal value into a let-go vm.Value.
//
// Primitive mappings:
//
//	goal int        -> vm.Int
//	goal float      -> vm.Float
//	goal string (S) -> vm.String
//	goal AB / AI    -> PersistentVector of vm.Int
//	goal AF         -> PersistentVector of vm.Float
//	goal AS         -> PersistentVector of vm.String
//	goal AV         -> PersistentVector of recursively-converted values
//	goal dict (D)   -> PersistentHashMap of recursively-converted kv
//	goal function   -> *vm.NativeFn wrapping a goal.V.Apply call
//
// Anything else is exposed to let-go as an opaque GoalValue handle that
// callers can pass back into Goal via LGToGoal.
func (b *Bridge) GoalToLG(v goal.V) vm.Value {
	switch {
	case v.IsI():
		return vm.Int(v.I())
	case v.IsF():
		return vm.Float(v.F())
	case v.IsFunction():
		return b.wrapGoalFn("", v)
	case v.IsBV():
		switch bv := v.BV().(type) {
		case goal.S:
			return vm.String(string(bv))
		case *goal.AB:
			out := make([]vm.Value, bv.Len())
			for i := 0; i < bv.Len(); i++ {
				out[i] = vm.Int(int(bv.At(i).I()))
			}
			return vm.NewPersistentVector(out)
		case *goal.AI:
			out := make([]vm.Value, bv.Len())
			for i := 0; i < bv.Len(); i++ {
				out[i] = vm.Int(int(bv.At(i).I()))
			}
			return vm.NewPersistentVector(out)
		case *goal.AF:
			out := make([]vm.Value, bv.Len())
			for i := 0; i < bv.Len(); i++ {
				out[i] = vm.Float(bv.At(i).F())
			}
			return vm.NewPersistentVector(out)
		case *goal.AS:
			out := make([]vm.Value, bv.Len())
			for i := 0; i < bv.Len(); i++ {
				out[i] = vm.String(string(bv.At(i).BV().(goal.S)))
			}
			return vm.NewPersistentVector(out)
		case *goal.AV:
			out := make([]vm.Value, bv.Len())
			for i := 0; i < bv.Len(); i++ {
				out[i] = b.GoalToLG(bv.At(i))
			}
			return vm.NewPersistentVector(out)
		case *goal.D:
			keys := bv.Keys()
			vals := bv.Values()
			n := arrayLen(keys)
			kvs := make([]vm.Value, 0, 2*n)
			for i := 0; i < n; i++ {
				kvs = append(kvs, b.GoalToLG(arrayAt(keys, i)))
				kvs = append(kvs, b.GoalToLG(arrayAt(vals, i)))
			}
			return vm.NewPersistentMap(kvs)
		}
	}
	// Fallback: opaque handle.
	return &GoalValue{ctx: b.Goal, v: v}
}

// LGToGoal converts a let-go value into a Goal V.
func (b *Bridge) LGToGoal(x vm.Value) (goal.V, error) {
	if x == nil {
		return goal.NewGap(), nil
	}
	switch xv := x.(type) {
	case vm.Int:
		return goal.NewI(int64(xv)), nil
	case vm.Float:
		return goal.NewF(float64(xv)), nil
	case vm.String:
		return goal.NewS(string(xv)), nil
	case vm.Boolean:
		if bool(xv) {
			return goal.NewI(1), nil
		}
		return goal.NewI(0), nil
	case vm.Symbol:
		return goal.NewS(string(xv)), nil
	case vm.Keyword:
		return goal.NewS(string(xv)), nil
	case *GoalValue:
		return xv.v, nil
	}
	if x == vm.NIL {
		return goal.NewGap(), nil
	}
	// Try generic seq conversion via Type tags.
	switch x.Type() {
	case vm.PersistentVectorType, vm.ArrayVectorType:
		// Sequence-y: iterate and build AV.
		seq, ok := x.(vm.Seq)
		if !ok {
			return goal.NewGap(), fmt.Errorf("bridge: cannot convert %T as seq", x)
		}
		out := []goal.V{}
		for seq != nil && seq != vm.EmptyList {
			gv, err := b.LGToGoal(seq.First())
			if err != nil {
				return goal.NewGap(), err
			}
			out = append(out, gv)
			seq = seq.Next()
		}
		return goal.NewAV(out), nil
	}
	return goal.NewGap(), fmt.Errorf("bridge: unsupported let-go value type %s", x.Type().Name())
}

// arrayLen / arrayAt are tiny shims around the Array interface so we can
// walk an unknown specialized array kind without a type switch at the
// call site.
func arrayLen(v goal.V) int {
	if !v.IsBV() {
		return 0
	}
	if a, ok := v.BV().(interface{ Len() int }); ok {
		return a.Len()
	}
	return 0
}

func arrayAt(v goal.V, i int) goal.V {
	if !v.IsBV() {
		return goal.NewGap()
	}
	if a, ok := v.BV().(interface{ At(int) goal.V }); ok {
		return a.At(i)
	}
	return goal.NewGap()
}
