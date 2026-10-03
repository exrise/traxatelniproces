package sim

import "math"

// RNG is a small deterministic splitmix64 generator.
type RNG struct{ S uint64 }

func NewRNG(seed uint64) RNG { return RNG{S: seed*0x9E3779B97F4A7C15 + 0x1234567} }

func (r *RNG) U64() uint64 {
	r.S += 0x9E3779B97F4A7C15
	z := r.S
	z = (z ^ (z >> 30)) * 0xBF58476D1CE4E5B9
	z = (z ^ (z >> 27)) * 0x94D049BB133111EB
	return z ^ (z >> 31)
}

// F returns a float in [0,1).
func (r *RNG) F() float64 { return float64(r.U64()>>11) / (1 << 53) }

// Range returns a float in [a,b).
func (r *RNG) Range(a, b float64) float64 { return a + (b-a)*r.F() }

// Intn returns an int in [0,n).
func (r *RNG) Intn(n int) int {
	if n <= 0 {
		return 0
	}
	return int(r.U64() % uint64(n))
}

// Norm returns a roughly normal value (sum of uniforms), sd ~ 1.
func (r *RNG) Norm() float64 {
	s := 0.0
	for i := 0; i < 4; i++ {
		s += r.F()
	}
	return (s - 2) * math.Sqrt(3)
}
