package sum

import (
	"math"
	"math/big"
)

const exponentBits = 11
const mantissaBits = 64 - exponentBits - 1 // Not counting the implicit one.
const exponentBias = 1<<(exponentBits-1) - 1

// Sum float64 numbers up, giving the result with best precision possible.
// Handles NaNs and infs properly, deals with catastrophic cancellation.
// With Sum D == sum((A+B..C) + D - (A+B..C)), even if A+B..C overflows float64.
// Addition is commutative and associative (unlike regular float64 addition).
// It is slightly faster than Kahan, with better (best) precision.
// Does not preserve signed zeroes: summing up single (-0) would give +0.
// Does not propagate the exact value of a NaN: if any NaNs were encountered returns math.NaN().
// Size is ~24Kb.
type Sum struct {
	// Sum of full mantissas (including implicit bit when appopriate).
	mantissaLo [1 << exponentBits]uint64 // unsigned, sign is stored in hi.
	mantissaHi [1 << exponentBits]int32  //
	plusInfs   int                       // Number of +infs among summands.
	minusInfs  int                       // Number of -infs among summands.
	nans       int                       // Number of NaNs among sumands.
}

// Add a float64 value to the sum.
func (a *Sum) Add(v float64) {
	b := math.Float64bits(v)
	if b == 0 {
		return
	}
	sign := b >> 63
	b &= ^uint64(1 << 63)
	exp := b >> mantissaBits
	mantissa := b & (1<<mantissaBits - 1)
	mantissa |= 1 << mantissaBits // implicit bit.
	prev := a.mantissaLo[exp]
	if exp != 0 && exp != 1<<exponentBits-1 {
		if sign == 0 {
			new := prev + mantissa
			a.mantissaLo[exp] = new
			if new < prev {
				a.mantissaHi[exp]++
			}
			return
		}
		new := prev - mantissa
		a.mantissaLo[exp] = new
		if a.mantissaLo[exp] > prev {
			a.mantissaHi[exp]--
		}
		return
	}
	// Handle subnormals, signed zeros, infs and nans.
	// Subnormals: exp == 0 && mantissa != 0.
	// Signed zeroes: exp == 0 &&  mantissa == 0.
	// Infs: exp == 2047 == (1<<exponentBits - 1) && mantissa == 0.
	// NaNs: exp == 2047 == (1<<exponentBits - 1) &&  mantissa != 0.
	switch exp {
	case 0:
		mantissa ^= 1 << mantissaBits // Clear the implicit bit.
		if mantissa == 0 {
			// Signed zero does not change the sum.
			return
		}
		// Subnormals are handleed below.
	case 1<<exponentBits - 1:
		mantissa ^= 1 << mantissaBits
		if mantissa == 0 {
			// Infs.
			if sign == 0 {
				a.plusInfs++
				return
			}
			a.minusInfs++
			return
		}
		// NaNs.
		a.nans++
		return
	}
	// Subnormals: add full mantissa.
	// It is slightly faster with code duplicated like this.
	if sign == 0 {
		new := prev + mantissa
		a.mantissaLo[exp] = new
		if new < prev {
			a.mantissaHi[exp]++
		}
		return
	}
	new := prev - mantissa
	a.mantissaLo[exp] = new
	if a.mantissaLo[exp] > prev {
		a.mantissaHi[exp]--
	}
}

// Val returns the current sum as float64.
func (a *Sum) Val() float64 {
	v, nan := a.BigVal()
	if nan {
		return math.NaN()
	}
	f, _ := v.Float64()
	return f
}

// BigVal returns the current sum as (sum *big.Float, isNan bool) pair
func (a *Sum) BigVal() (*big.Float, bool) {
	if a.nans > 0 {
		return nil, true
	}
	// Handle infs.
	if a.minusInfs != 0 {
		if a.plusInfs != 0 {
			// (+Inf) + (-Inf) => NaN.
			return nil, true
		}
		return big.NewFloat(math.Inf(-1)), false
	}
	if a.plusInfs != 0 {
		return big.NewFloat(math.Inf(1)), false
	}
	var q bfAdder
	// end at exponentBits-1 to ignore nans and infs which were handled above.
	for i := 0; i < 1<<exponentBits-1; i++ {
		sign := 1.0
		hi := a.mantissaHi[i]
		lo := a.mantissaLo[i]
		if lo == 0 && hi == 0 {
			continue
		}
		if hi < 0 {
			sign = -1
			hi = -hi
			hi--
			lo = -lo
		}
		exp := uint64(i)
		if exp == 0 {
			exp = 1 // Handling subnormals
		}
		mantissa := lo & (1<<mantissaBits - 1)
		if mantissa != 0 {
			// ints between -2^(mantissaBits+1) and 2^(mantissaBits+1) can be represented as floats.
			u := big.NewFloat(float64(mantissa) * sign)
			u.SetMantExp(u, int(exp)-exponentBias-mantissaBits)
			q.Add(u)
		}

		mantissa = lo >> (mantissaBits)
		mantissa |= uint64(hi) << (64 - mantissaBits)

		if mantissa != 0 {
			u := big.NewFloat(float64(mantissa) * sign)
			u.SetMantExp(u, int(exp)-exponentBias)
			q.Add(u)
		}
	}
	return q.BigVal(), false
}

// Kahan implements a reasonably robust summation algorithm, see
// https://en.wikipedia.org/wiki/Kahan_summation_algorithm
// Note: does not handle infs properly.
type Kahan struct {
	s, c float64
}

// Add v to the sum.
func (k *Kahan) Add(v float64) {
	y := v - k.c
	t := k.s + y
	k.c = (t - k.s) - y
	k.s = t
}

// Val return the current sum.
func (k Kahan) Val() float64 {
	return k.s
}

// bfAdder uses big.Floats and exponent binning.
// Handles cancellation.
type bfAdder struct {
	nonneg []*big.Float // bin == exponent
	neg    []*big.Float // bin == -exponent+1
}

func (b *bfAdder) Add(v *big.Float) {
	exp := v.MantExp(nil)
	p := &b.nonneg
	bin := exp
	if exp < 0 {
		p = &b.neg
		bin = -bin + 1
	}
	for len(*p) < bin+1 {
		*p = append(*p, &big.Float{})
	}
	a := *p
	a[bin].Add(a[bin], v)
	exp1 := a[bin].MantExp(nil)

	if exp1 != exp {
		b.Add(a[bin])
		a[bin].SetFloat64(0)
	}
}

func (b bfAdder) BigVal() *big.Float {
	var sum bigKahan // Using Kahan here is an overkill, but does not hurt.
	for i := range b.nonneg {
		sum.Add(b.nonneg[len(b.nonneg)-i-1])
	}
	for _, x := range b.neg {
		sum.Add(x)
	}

	return sum.BigVal()
}

// bigKahan: kahan using big.Float.
type bigKahan struct {
	s, c big.Float
}

// Add v to the sum.
func (k *bigKahan) Add(v *big.Float) {
	y := &big.Float{}
	y.Sub(v, &k.c)
	t := &big.Float{}
	t.Add(&k.s, y)
	k.c.Sub(t, &k.s)
	k.c.Sub(&k.c, y)
	k.s = *t
}

// Val return the current sum.
func (k bigKahan) BigVal() *big.Float {
	return &k.s
}
