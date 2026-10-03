package sim

// WStat accumulates how a weapon performed (used by the balance tools).
type WStat struct {
	Shots       int
	Unit        float64 // damage dealt to enemy units
	Struct      float64 // damage dealt to enemy structures (without HQ)
	HQ          float64 // damage dealt to enemy HQs
	Kills       int
	Intercepted int
}

func (w *World) stat(id string) *WStat {
	if w.Stats == nil {
		w.Stats = map[string]*WStat{}
	}
	s := w.Stats[id]
	if s == nil {
		s = &WStat{}
		w.Stats[id] = s
	}
	return s
}
