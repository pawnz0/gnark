package nonnative

import (
	"fmt"
	"math/big"

	"github.com/consensys/gnark-crypto/ecc"
	"github.com/consensys/gnark/backend"
	"github.com/consensys/gnark/backend/hint"
	"github.com/consensys/gnark/frontend"
)

func NewAPI(native frontend.API, params *Params) frontend.API {
	return &fakeAPI{
		api:    native,
		params: params,
	}
}

type fakeAPI struct {
	// api is the native API
	api    frontend.API
	params *Params
}

func (f *fakeAPI) varToElement(in frontend.Variable) Element {
	var e Element
	switch vv := in.(type) {
	case Element:
		e = vv
	case *Element:
		e = *vv
	case *big.Int:
		e = f.params.ConstantFromBigOrPanic(vv)
	case big.Int:
		e = f.params.ConstantFromBigOrPanic(&vv)
	case int:
		e = f.params.ConstantFromBigOrPanic(big.NewInt(int64(vv)))
	default:
		panic(fmt.Sprintf("can not cast %T to Element", in))
	}
	if !f.params.isEqual(e.params) {
		panic("incompatible Element parameters")
	}
	return e
}

func (f *fakeAPI) varsToElements(in ...frontend.Variable) []Element {
	var els []Element
	for i := range in {
		switch v := in[i].(type) {
		case []frontend.Variable:
			subels := f.varsToElements(v...)
			els = append(els, subels...)
		case frontend.Variable:
			els = append(els, f.varToElement(v))
		}
	}
	return els
}

func (f *fakeAPI) Add(i1 frontend.Variable, i2 frontend.Variable, in ...frontend.Variable) frontend.Variable {
	els := f.varsToElements(i1, i2, in)
	res := f.params.Element(f.api)
	res.Add(els[0], els[1])
	for i := 2; i < len(els); i++ {
		res.Add(res, els[i])
	}
	return res
}

func (f *fakeAPI) Neg(i1 frontend.Variable) frontend.Variable {
	el := f.varToElement(i1)
	res := f.params.Element(f.api)
	res.Negate(el)
	return res
}

func (f *fakeAPI) Sub(i1 frontend.Variable, i2 frontend.Variable, in ...frontend.Variable) frontend.Variable {
	els := f.varsToElements(i1, i2, in)
	sub := f.params.Element(f.api)
	sub.Set(els[1])
	for i := 2; i < len(els); i++ {
		sub.Add(sub, els[i])
	}
	res := f.params.Element(f.api)
	res.Sub(els[0], sub)
	return res
}

func (f *fakeAPI) Mul(i1 frontend.Variable, i2 frontend.Variable, in ...frontend.Variable) frontend.Variable {
	els := f.varsToElements(i1, i2, in)
	res := f.params.Element(f.api)
	res.Mul(els[0], els[1])
	for i := 2; i < len(els); i++ {
		res.Mul(res, els[i])
	}
	return res
}

func (f *fakeAPI) DivUnchecked(i1 frontend.Variable, i2 frontend.Variable) frontend.Variable {
	// TODO: implement unchecked div?
	return f.Div(i1, i2)
}

func (f *fakeAPI) Div(i1 frontend.Variable, i2 frontend.Variable) frontend.Variable {
	els := f.varsToElements(i1, i2)
	res := f.params.Element(f.api)
	res.Div(els[0], els[1])
	return res
}

func (f *fakeAPI) Inverse(i1 frontend.Variable) frontend.Variable {
	el := f.varToElement(i1)
	res := f.params.Element(f.api)
	res.Inverse(el)
	return res
}

func (f *fakeAPI) ToBinary(i1 frontend.Variable, n ...int) []frontend.Variable {
	el := f.varToElement(i1)
	res := f.params.Element(f.api)
	res.Reduce(*el)
	out := res.ToBits()
	switch len(n) {
	case 0:
	case 1:
		out = out[:n[0]]
	default:
		panic("only single vararg permitted to ToBinary")
	}
	return out
}

func (f *fakeAPI) FromBinary(b ...frontend.Variable) frontend.Variable {
	res := f.params.Element(f.api)
	res.FromBits(b)
	return &res
}

func (f *fakeAPI) Xor(a frontend.Variable, b frontend.Variable) frontend.Variable {
	return f.api.Xor(a, b)
}

func (f *fakeAPI) Or(a frontend.Variable, b frontend.Variable) frontend.Variable {
	return f.api.Or(a, b)
}

func (f *fakeAPI) And(a frontend.Variable, b frontend.Variable) frontend.Variable {
	return f.api.And(a, b)
}

func (f *fakeAPI) Select(b frontend.Variable, i1 frontend.Variable, i2 frontend.Variable) frontend.Variable {
	els := f.varsToElements(i1, i2)
	res := f.params.Element(f.api)
	if els[0].overflow == els[1].overflow && len(els[0].Limbs) == len(els[1].Limbs) {
		res.Select(b, *els[0], *els[1])
		return &res
	}
	s1 := els[0]
	s2 := els[1]
	if s1.overflow != 0 || len(s1.Limbs) != int(f.params.nbLimbs) {
		v := f.params.Element(f.api)
		v.Reduce(*s1)
		s1 = &v
	}
	if s2.overflow != 0 || len(s2.Limbs) != int(f.params.nbLimbs) {
		v := f.params.Element(f.api)
		v.Reduce(*s2)
		s2 = &v
	}
	res.Select(b, *s1, *s2)
	return &res
}

func (f *fakeAPI) Lookup2(b0 frontend.Variable, b1 frontend.Variable, i0 frontend.Variable, i1 frontend.Variable, i2 frontend.Variable, i3 frontend.Variable) frontend.Variable {
	els := f.varsToElements(i0, i1, i2, i3)
	res := f.params.Element(f.api)
	res.Lookup2(b0, b1, els[0], els[1], els[2], els[3])
	return res
}

func (f *fakeAPI) IsZero(i1 frontend.Variable) frontend.Variable {
	panic("not implemented") // TODO: Implement
}

func (f *fakeAPI) Cmp(i1 frontend.Variable, i2 frontend.Variable) frontend.Variable {
	panic("not implemented") // TODO: Implement
}

func (f *fakeAPI) AssertIsEqual(i1 frontend.Variable, i2 frontend.Variable) {
	els := f.varsToElements(i1, i2)
	tmp := f.params.Element(f.api)
	tmp.Set(els[0])
	tmp.AssertIsEqual(els[1])
}

func (f *fakeAPI) AssertIsDifferent(i1 frontend.Variable, i2 frontend.Variable) {
	panic("not implemented") // TODO: Implement
}

func (f *fakeAPI) AssertIsBoolean(i1 frontend.Variable) {
	switch vv := i1.(type) {
	case Element:
		v := f.params.Element(f.api)
		v.Reduce(vv)
		f.api.AssertIsBoolean(v.Limbs[0])
		for i := 1; i < len(v.Limbs); i++ {
			f.api.AssertIsEqual(v.Limbs[i], 0)
		}
	case *Element:
		v := f.params.Element(f.api)
		v.Reduce(*vv)
		f.api.AssertIsBoolean(v.Limbs[0])
		for i := 1; i < len(v.Limbs); i++ {
			f.api.AssertIsEqual(v.Limbs[i], 0)
		}
	default:
		f.api.AssertIsBoolean(vv)
	}
}

func (f *fakeAPI) AssertIsLessOrEqual(v frontend.Variable, bound frontend.Variable) {
	panic("not implemented") // TODO: Implement
}

func (f *fakeAPI) Println(a ...frontend.Variable) {
	els := f.varsToElements(a)
	for i := range els {
		f.api.Println(els[i].Limbs...)
	}
}

func (f *fakeAPI) Compiler() frontend.Compiler {
	return f.api.Compiler()
}

// Deprecated APIs

func (f *fakeAPI) NewHint(hf hint.Function, nbOutputs int, inputs ...frontend.Variable) ([]frontend.Variable, error) {
	// this is a trick to allow calling hint functions using non-native
	// elements. We use the fact that the hints take as inputs *big.Int values.
	// Instead of supplying hf to the solver for calling, we wrap it with
	// another function (implementing hint.Function), which takes as inputs the
	// "expanded" version of inputs (where instead of Element values we provide
	// as inputs the limbs of every Element) and returns nbLimbs*nbOutputs
	// number of outputs (i.e. the limbs of non-native Element values). The
	// wrapper then recomposes and decomposes the *big.Int values at runtime and
	// provides them as input to the initially provided hint function.
	var expandedInputs []frontend.Variable
	type typedInput struct {
		pos       int
		nbLimbs   int
		isElement bool
	}
	typedInputs := make([]typedInput, len(inputs))
	for i := range inputs {
		switch vv := inputs[i].(type) {
		case Element:
			expandedInputs = append(expandedInputs, vv.Limbs...)
			typedInputs[i] = typedInput{
				pos:       len(expandedInputs) - len(vv.Limbs),
				nbLimbs:   len(vv.Limbs),
				isElement: true,
			}
		case *Element:
			expandedInputs = append(expandedInputs, vv.Limbs...)
			typedInputs[i] = typedInput{
				pos:       len(expandedInputs) - len(vv.Limbs),
				nbLimbs:   len(vv.Limbs),
				isElement: true,
			}
		default:
			expandedInputs = append(expandedInputs, inputs[i])
			typedInputs[i] = typedInput{
				pos:       len(expandedInputs) - 1,
				nbLimbs:   1,
				isElement: false,
			}
		}
	}
	nbNativeOutputs := nbOutputs * int(f.params.nbLimbs)
	wrappedHint := func(curveID ecc.ID, expandedHintInputs []*big.Int, expandedHintOutputs []*big.Int) error {
		hintInputs := make([]*big.Int, len(inputs))
		hintOutputs := make([]*big.Int, nbOutputs)
		for i, ti := range typedInputs {
			hintInputs[i] = new(big.Int)
			if ti.isElement {
				if err := recompose(expandedHintInputs[ti.pos:ti.pos+ti.nbLimbs], f.params.nbBits, hintInputs[i]); err != nil {
					return fmt.Errorf("recompose: %w", err)
				}
			} else {
				hintInputs[i].Set(expandedHintInputs[ti.pos])
			}
		}
		for i := range hintOutputs {
			hintOutputs[i] = new(big.Int)
		}
		if err := hf(curveID, hintInputs, hintOutputs); err != nil {
			return fmt.Errorf("call hint: %w", err)
		}
		for i := range hintOutputs {
			if err := decompose(hintOutputs[i], f.params.nbBits, expandedHintOutputs[i*int(f.params.nbLimbs):(i+1)*int(f.params.nbLimbs)]); err != nil {
				return fmt.Errorf("decompose: %w", err)
			}
		}
		return nil
	}
	hintRet, err := f.api.Compiler().NewHint(wrappedHint, nbNativeOutputs, expandedInputs...)
	if err != nil {
		return nil, fmt.Errorf("NewHint: %w", err)
	}
	ret := make([]frontend.Variable, nbOutputs)
	for i := 0; i < nbOutputs; i++ {
		el := f.params.Element(f.api)
		el.Limbs = hintRet[i*int(f.params.nbLimbs) : (i+1)*int(f.params.nbLimbs)]
		ret[i] = &el
	}
	return ret, nil
}

func (f *fakeAPI) Tag(name string) frontend.Tag {
	return f.api.Compiler().Tag(name)
}

func (f *fakeAPI) AddCounter(from frontend.Tag, to frontend.Tag) {
	f.api.Compiler().AddCounter(from, to)
}

func (f *fakeAPI) ConstantValue(v frontend.Variable) (*big.Int, bool) {
	panic("deprecated, use Compiler().ConstantValue()")
}

func (f *fakeAPI) Curve() ecc.ID {
	panic("using emulated API")
}

func (f *fakeAPI) Backend() backend.ID {
	return f.api.Compiler().Backend()
}
