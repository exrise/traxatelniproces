package sim

import (
	"math"
)

// Terrain material values stored in the mask.
const (
	Air  = 0
	Dirt = 1
	Rock = 2
)

// Terrain is a destructible pixel map.
type Terrain struct {
	W, H int
	Mask []uint8
	// Dirty rectangles since the UI last looked (x0,y0,x1,y1 exclusive).
	Dirty [][4]int
	// Craters lists every carve operation so network clients can replay them.
	Craters []Crater
}

// Crater is a recorded terrain carve.
type Crater struct {
	X, Y, R int
}

// NewTerrain allocates an empty terrain.
func NewTerrain(w, h int) *Terrain {
	return &Terrain{W: w, H: h, Mask: make([]uint8, w*h)}
}

// At returns the material at pixel (x,y); outside the map: sides are air, below is rock.
func (t *Terrain) At(x, y int) uint8 {
	if x < 0 || x >= t.W || y < 0 {
		return Air
	}
	if y >= t.H {
		return Rock
	}
	return t.Mask[y*t.W+x]
}

// Solid reports whether the pixel blocks movement.
func (t *Terrain) Solid(x, y int) bool { return t.At(x, y) != Air }

// SurfaceY returns the topmost solid pixel y in column x at or below fromY (or H if none).
func (t *Terrain) SurfaceY(x, fromY int) int {
	if x < 0 || x >= t.W {
		return t.H
	}
	if fromY < 0 {
		fromY = 0
	}
	for y := fromY; y < t.H; y++ {
		if t.Mask[y*t.W+x] != Air {
			return y
		}
	}
	return t.H
}

// Carve removes a disc. Rock is only carved with a reduced radius.
func (t *Terrain) Carve(cx, cy, r int, record bool) {
	if r < 1 {
		return
	}
	rr := r * r
	rockR := int(float64(r) * 0.4)
	rockRR := rockR * rockR
	x0, x1 := max(0, cx-r), min(t.W-1, cx+r)
	y0, y1 := max(0, cy-r), min(t.H-1, cy+r)
	changed := false
	for y := y0; y <= y1; y++ {
		for x := x0; x <= x1; x++ {
			dx, dy := x-cx, y-cy
			d := dx*dx + dy*dy
			if d > rr {
				continue
			}
			i := y*t.W + x
			switch t.Mask[i] {
			case Dirt:
				t.Mask[i] = Air
				changed = true
			case Rock:
				if d <= rockRR {
					t.Mask[i] = Air
					changed = true
				}
			}
		}
	}
	if changed {
		t.Dirty = append(t.Dirty, [4]int{x0 - 1, y0 - 1, x1 + 2, y1 + 2})
		if record {
			t.Craters = append(t.Craters, Crater{cx, cy, r})
		}
	}
}

// Fill adds a disc of dirt (used by engineers / debris).
func (t *Terrain) Fill(cx, cy, r int) {
	rr := r * r
	for y := max(0, cy-r); y <= min(t.H-1, cy+r); y++ {
		for x := max(0, cx-r); x <= min(t.W-1, cx+r); x++ {
			dx, dy := x-cx, y-cy
			if dx*dx+dy*dy <= rr && t.Mask[y*t.W+x] == Air {
				t.Mask[y*t.W+x] = Dirt
			}
		}
	}
	t.Dirty = append(t.Dirty, [4]int{cx - r - 1, cy - r - 1, cx + r + 2, cy + r + 2})
}

// Normal estimates the surface normal around (x,y) by sampling the mask.
func (t *Terrain) Normal(x, y float64) Vec {
	var nx, ny float64
	for dy := -4; dy <= 4; dy += 2 {
		for dx := -4; dx <= 4; dx += 2 {
			if t.Solid(int(x)+dx, int(y)+dy) {
				nx -= float64(dx)
				ny -= float64(dy)
			}
		}
	}
	l := math.Hypot(nx, ny)
	if l < 0.001 {
		return Vec{0, -1}
	}
	return Vec{nx / l, ny / l}
}

// ---- generation ---------------------------------------------------------

func smooth(t float64) float64 { t = math.Max(0, math.Min(1, t)); return t * t * (3 - 2*t) }

// MapWidth returns the world width for n players.
func MapWidth(n int) int { return MapMargin*2 + ZoneW + ZoneStep*(n-1) }

// ZoneRange returns the build zone [x0,x1) of zone index i for n players.
func ZoneRange(n, i int) (int, int) {
	x0 := MapMargin + ZoneStep*i
	return x0, x0 + ZoneW
}

// PlateauY is the flat ground level of all build zones.
const PlateauY = 656 // multiple of Cell so structures sit flush

// GenerateTerrain builds the landscape for n zones and returns capture point positions.
func GenerateTerrain(n int, seed uint64) (*Terrain, []Vec) {
	rng := NewRNG(seed)
	W := MapWidth(n)
	t := NewTerrain(W, MapH)

	// random smooth components (shared "wavelength" bands)
	type wave struct{ amp, freq, ph float64 }
	var waves []wave
	for _, b := range []struct{ amp, freq float64 }{{110, 1 / 520.0}, {55, 1 / 230.0}, {24, 1 / 95.0}, {9, 1 / 40.0}} {
		waves = append(waves, wave{b.amp * rng.Range(0.7, 1.15), b.freq, rng.Range(0, 2*math.Pi)})
	}
	height := make([]float64, W)
	for x := 0; x < W; x++ {
		h := float64(PlateauY)
		for _, w := range waves {
			h += w.amp * math.Sin(float64(x)*w.freq*2*math.Pi+w.ph)
		}
		// keep every build zone flat and level
		blend := 0.0
		for i := 0; i < n; i++ {
			z0, z1 := ZoneRange(n, i)
			d := 0.0
			if x < z0 {
				d = float64(z0 - x)
			} else if x >= z1 {
				d = float64(x - z1 + 1)
			}
			b := 1 - smooth(d/150)
			if b > blend {
				blend = b
			}
		}
		h = h*(1-blend) + PlateauY*blend
		// outer walls rise at the map edges
		if x < MapMargin-120 {
			h -= float64(MapMargin-120-x) * 1.3
		}
		if x > W-MapMargin+120 {
			h -= float64(x-(W-MapMargin+120)) * 1.3
		}
		height[x] = h
	}
	// rock blobs and a rock layer
	type blob struct{ x, y, r float64 }
	var blobs []blob
	for i := 0; i < W/70; i++ {
		blobs = append(blobs, blob{rng.Range(0, float64(W)), rng.Range(PlateauY+30, MapH-100), rng.Range(18, 60)})
	}
	for x := 0; x < W; x++ {
		top := int(height[x])
		if top < 0 {
			top = 0
		}
		rockDepth := 150 + 40*math.Sin(float64(x)*0.013+1.7)
		for y := top; y < MapH; y++ {
			m := uint8(Dirt)
			if float64(y-top) > rockDepth {
				m = Rock
			}
			t.Mask[y*W+x] = m
		}
	}
	for _, b := range blobs {
		for y := int(b.y - b.r); y <= int(b.y+b.r); y++ {
			for x := int(b.x - b.r); x <= int(b.x+b.r); x++ {
				if x < 0 || x >= W || y < 0 || y >= MapH {
					continue
				}
				dx, dy := float64(x)-b.x, float64(y)-b.y
				if dx*dx+dy*dy <= b.r*b.r && t.Mask[y*W+x] == Dirt {
					t.Mask[y*W+x] = Rock
				}
			}
		}
	}
	// small caves under the gaps give some tactical variety
	for i := 0; i < n-1; i++ {
		z1 := MapMargin + ZoneStep*i + ZoneW
		mid := z1 + (ZoneStep-ZoneW)/2
		cx := mid + int(rng.Range(100, 200))*(1-2*rng.Intn(2))
		cy := t.SurfaceY(cx, 0) + int(rng.Range(40, 75))
		t.Carve(cx, cy, int(rng.Range(24, 36)), false)
	}
	// capture points in the middle of every gap, standing on the ground
	var pts []Vec
	for i := 0; i < n-1; i++ {
		z1 := MapMargin + ZoneStep*i + ZoneW
		cx := z1 + (ZoneStep-ZoneW)/2
		sy := t.SurfaceY(cx, 0)
		pts = append(pts, Vec{float64(cx), float64(sy)})
	}
	t.Dirty = nil
	t.Craters = nil
	return t, pts
}
