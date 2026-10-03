package sim

import "testing"

// Walk a unit all the way across the gap terrain and report where it gets stuck.
func TestWalkAcrossTerrain(t *testing.T) {
	for seed := uint64(1); seed <= 8; seed++ {
		w := newTest(2, seed)
		p := w.Players[0]
		x := float64(p.ZoneX0+p.ZoneX1) / 2
		w.PlaceUnit(0, "assault", x+100, 100)
		w.PlaceUnit(1, "assault", float64(w.Players[1].ZoneX0+w.Players[1].ZoneX1)/2, 100)
		w.SetReady(0, true)
		w.SetReady(1, true)
		w.Step()
		for w.Cur != 0 {
			w.Apply(Command{Player: w.Cur, Type: CmdEndTurn})
			run(w, 1.5)
		}
		u := w.UnitsOf(0)[0]
		w.Apply(Command{Player: 0, Type: CmdSelectUnit, ID: u.ID})
		dir := 1
		if w.Players[1].ZoneX0 < p.ZoneX0 {
			dir = -1
		}
		w.Apply(Command{Player: 0, Type: CmdWalk, Dir: dir})
		start := u.Pos.X
		lastX, still := u.Pos.X, 0
		for i := 0; i < 60*40 && w.Cur == 0; i++ {
			w.TurnTimer = 100
			w.Step()
			if i%30 == 0 {
				if u.Pos.X == lastX {
					still++
				} else {
					still = 0
				}
				lastX = u.Pos.X
				if still >= 1 && u.Ground {
					// try jumping to get over it
					w.Apply(Command{Player: 0, Type: CmdJump})
				}
				if still >= 10 && false {
					t.Logf("seed %d: unit stuck at %.0f,%.0f ground=%v bodyFree=%v onGround=%v", seed, u.Pos.X, u.Pos.Y, u.Ground, w.unitBodyFree(u.Pos.X, u.Pos.Y), w.unitOnGround(u.Pos.X, u.Pos.Y))
					break
				}
			}
		}
		if walked := (u.Pos.X - start) * float64(dir); walked < 1000 {
			t.Fatalf("seed %d: unit could only walk %.0f px (got stuck at x=%.0f)", seed, walked, u.Pos.X)
		}
	}
}
