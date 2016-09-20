package sum

import (
	"math"
	"math/big"
	"testing"
)

const eps = 1e-80 * 9.87654321

const N = 100000

func TestCancellationBF(t *testing.T) {
	a := bfAdder{}
	for _, x := range []float64{eps, 1000, 1000, 1000, 1000, 1000, -5000} {
		a.Add(big.NewFloat(x))
	}
	diff := big.NewFloat(eps)
	diff.Sub(diff, a.BigVal())
	if diff.Abs(diff).Cmp(big.NewFloat(eps/1000)) != -1 {
		t.Fatalf("exptected %s and %f to be close", a.BigVal().String(), eps)
	}
}

func TestCancellationDumb(t *testing.T) {
	a := Dumb{}
	for _, x := range []float64{eps, 1000, 1000, 1000, 1000, 1000, -5000} {
		a.Add(x)
	}
	if math.Abs(a.Val()-eps)*1000 < eps {
		t.Fatalf("not exptected %s and %s to be close", big.NewFloat(a.Val()).String(), big.NewFloat(eps).String())
	}
}

func TestCancellationKahan(t *testing.T) {
	a := Kahan{}
	for _, x := range []float64{eps, 1000, 1000, 1000, 1000, 1000, -5000} {
		a.Add(x)
	}
	if math.Abs(a.Val()-eps)*1000 < eps {
		t.Fatalf("not exptected %s and %s to be close", big.NewFloat(a.Val()).String(), big.NewFloat(eps).String())
	}
}

func TestNeg(t *testing.T) {
	a := Sum{}
	for _, x := range []float64{-1} {
		a.Add(x)
	}
	if math.Abs(a.Val()+1)*1000 > 1e-16 {
		t.Fatalf("exptected %s and %s to be close", big.NewFloat(a.Val()).String(), "-1")
	}
}

func TestSubnormals(t *testing.T) {
	a := Sum{}
	a.Add(2e100)
	for i := 0; i < 100; i++ {
		a.Add(math.SmallestNonzeroFloat64)
	}
	a.Add(-1e100)
	a.Add(-1e100)
	if math.Abs(math.SmallestNonzeroFloat64*100-a.Val()) >= math.SmallestNonzeroFloat64 {
		t.Fatalf("exptected %s and %g to be close", big.NewFloat(a.Val()).String(), math.SmallestNonzeroFloat64*100)
	}
}

func TestCancellationSum(t *testing.T) {
	a := Sum{}
	for _, x := range []float64{eps, 1000, 1000, 1000, 1000, 1000, -5000} {
		a.Add(x)
	}
	if math.Abs(a.Val()-eps)*1000 > eps {
		t.Fatalf("exptected %s and %s to be close", big.NewFloat(a.Val()).String(), big.NewFloat(eps).String())
	}
}

func TestCancellationBig(t *testing.T) {
	a := Big{}
	for _, x := range []float64{eps, 1000, 1000, 1000, 1000, 1000, -5000} {
		a.Add(x)
	}
	// big.Floats do not fandle cacnellation nicely.
	// So it the result is 0, and not eps.
	if math.Abs(a.Val()-0)*1000 > eps {
		t.Fatalf("exptected %s and %s to be close", big.NewFloat(a.Val()).String(), big.NewFloat(eps).String())
	}
}

func TestSumKahan(t *testing.T) {
	a := &Kahan{}
	a.Add(17)

	for i := 0; i < N; i++ {
		a.Add(eps)
	}
	a.Add(-17)
	if math.Abs(a.Val()) > math.SmallestNonzeroFloat64 {
		t.Fatalf("exptected %s to be zero", big.NewFloat(a.Val()).String())
	}
}

func TestSumKahan2(t *testing.T) {
	a := &Kahan{}
	a.Add(1)

	for i := 0; i < N*100; i++ {
		a.Add(1e-18)
	}
	a.Add(-1)
	if math.Abs(a.Val()-1e-18*N*100) > 1e-18 {
		t.Fatalf("exptected %s and %s to be close", big.NewFloat(a.Val()).String(), big.NewFloat(-1e-18*N*100).String())
	}
}

func TestSum(t *testing.T) {
	a := &Sum{}
	a.Add(17)

	for i := 0; i < N; i++ {
		a.Add(eps)
	}
	a.Add(-17)
	if math.Abs(a.Val()-eps*N)*1000 > eps {
		t.Fatalf("exptected %s and %s to be close", big.NewFloat(a.Val()).String(), big.NewFloat(eps*N).String())
	}
}

func TestSumInfs(t *testing.T) {
	plusInf := func(v float64) {
		if !math.IsInf(v, 1) {
			t.Fatalf("expected +inf, got %f", v)
		}
	}
	minusInf := func(v float64) {
		if !math.IsInf(v, -1) {
			t.Fatalf("expected -inf, got %f", v)
		}
	}
	nan := func(v float64) {
		if !math.IsNaN(v) {
			t.Fatalf("expected nan, got %f", v)
		}
	}
	for _, tc := range []struct {
		in    []float64
		check func(v float64)
	}{
		{
			[]float64{math.Inf(1)},
			plusInf,
		},
		{
			[]float64{math.NaN()},
			nan,
		},
		{
			[]float64{math.Inf(-1)},
			minusInf,
		},
		{
			[]float64{math.Inf(-1), 0},
			minusInf,
		},
		{
			[]float64{math.Inf(1), math.Inf(1), 0},
			plusInf,
		},
		{
			[]float64{math.Inf(1), math.Inf(-1)},
			nan,
		},
		{
			[]float64{math.NaN(), 0, 1, 2, 3, 4},
			nan,
		},
		{
			[]float64{math.NaN(), math.Inf(1)},
			nan,
		},
		{
			[]float64{math.NaN(), math.Inf(-1)},
			nan,
		},
	} {
		var a Sum // Note: Kahan does not handle -Inf + 0 and similar cases.
		for _, x := range tc.in {
			a.Add(x)
		}
		t.Log(tc)
		tc.check(a.Val())
	}
}

func TestSum0(t *testing.T) {
	a := &Sum{}
	a.Add(17)
	for i := 0; i < N; i++ {
		a.Add(0)
	}
	a.Add(-17)
	if math.Abs(a.Val()) > math.SmallestNonzeroFloat64 {
		t.Fatalf("exptected %s to be zero", big.NewFloat(a.Val()).String())
	}
}

func TestSumNeg0(t *testing.T) {
	a := &Sum{}
	a.Add(17)
	for i := 0; i < N; i++ {
		a.Add(math.Copysign(0, -1))
	}
	a.Add(-17)
	if math.Abs(a.Val()) > math.SmallestNonzeroFloat64 {
		t.Fatalf("exptected %s to be zero", big.NewFloat(a.Val()).String())
	}
}

func TestSumBF(t *testing.T) {
	a := bfAdder{}
	a.Add(big.NewFloat(17))

	for i := 0; i < N; i++ {
		a.Add(big.NewFloat(eps))
	}
	a.Add(big.NewFloat(-17))
	diff := big.NewFloat(eps * N)
	diff.Sub(diff, a.BigVal())
	if diff.Abs(diff).Cmp(big.NewFloat(eps/1000)) != -1 {
		t.Fatalf("exptected %s and %f to be close", a.BigVal().String(), eps)
	}
}

func TestSumBig(t *testing.T) {
	a := Big{}
	a.Add(17)

	for i := 0; i < N; i++ {
		a.Add(eps)
	}
	a.Add(-17)
	if math.Abs(a.Val()) > math.SmallestNonzeroFloat64 {
		t.Fatalf("exptected %s to be zero", big.NewFloat(a.Val()).String())
	}
}

func BenchmarkBig(b *testing.B) {
	b.SetBytes(8)
	a := Big{}
	a.Add(17)

	for i := 0; i < b.N; i++ {
		a.Add(eps)
	}
	a.Add(-17)
}

func BenchmarkBF(b *testing.B) {
	b.SetBytes(8)
	a := bfAdder{}
	a.Add(big.NewFloat(17))

	fe := big.NewFloat(1e-10)
	for i := 0; i < b.N; i++ {
		a.Add(fe)
	}
	a.Add(big.NewFloat(-17))
}

func BenchmarkSum(b *testing.B) {
	b.SetBytes(8)
	a := Sum{}
	a.Add(17)
	for i := 0; i < b.N; i++ {
		a.Add(-1e-10)
	}
	a.Add(-17)
}

var da Dumb

func BenchmarkDumb(b *testing.B) {
	b.SetBytes(8)
	da = Dumb{}
	da.Add(17)
	for i := 0; i < b.N; i++ {
		da.Add(-1e-10)
	}
	da.Add(-17)
}

var a float64

func BenchmarkDirect(b *testing.B) {
	b.SetBytes(8)
	a = 17.0
	for i := 0; i < b.N; i++ {
		a += -1e-10
	}
	a -= 17
}

func BenchmarkKahan(b *testing.B) {
	b.SetBytes(8)
	a := Kahan{}
	a.Add(17)
	for i := 0; i < b.N; i++ {
		a.Add(-1e-10)
	}
	a.Add(-17)
}

// Big adds numbers as big.Floats.
type Big struct {
	s *big.Float
}

func (b *Big) Add(v float64) {
	if b.s == nil {
		b.s = big.NewFloat(0)
	}
	b.s.Add(b.s, big.NewFloat(v))
}

func (b Big) Val() float64 {
	f, _ := b.s.Float64()
	return f
}

// Dumb adds numbers as float64s.
type Dumb struct {
	float64
}

func (d *Dumb) Add(v float64) {
	d.float64 += v
}

func (d Dumb) Val() float64 {
	return d.float64
}
