package sim

import (
	"math"
	"testing"

	"svinovoyna/internal/balance"
)

func newTest(n int, seed uint64) *World {
	var setups []PlayerSetup
	for i := 0; i < n; i++ {
		setups = append(setups, PlayerSetup{Name: string(rune('A' + i)), Color: i})
	}
	return NewWorld(balance.Default(), seed, setups)
}

func run(w *World, secs float64) {
	for i := 0; i < int(secs*TickRate); i++ {
		w.Step()
	}
}

func TestWorldCreation(t *testing.T) {
	for n := 2; n <= 4; n++ {
		w := newTest(n, 7)
		if w.Phase != PhaseBuild || len(w.Players) != n || len(w.Points) != n-1 {
			t.Fatalf("n=%d: bad world %+v", n, w.StatusLine())
		}
		for _, p := range w.Players {
			if !w.HasHQ(p.ID) {
				t.Fatalf("player %d has no HQ", p.ID)
			}
			hq := w.Structs[p.HQ]
			if !w.supportedAt(hq.CX, hq.CY, 4, 3, hq.ID) {
				t.Fatalf("HQ of %d is floating", p.ID)
			}
		}
	}
}

func TestBuildAndLimits(t *testing.T) {
	w := newTest(2, 1)
	p := w.Players[0]
	hq := w.Structs[p.HQ]
	// place a sandbag right next to the HQ, on the ground
	cx := hq.CX + 6
	cy := hq.CY + 3 - 1
	if err := w.PlaceStruct(0, "sandbag", cx, cy); err != nil {
		t.Fatalf("place sandbag: %v", err)
	}
	if err := w.PlaceStruct(0, "sandbag", cx, cy); err != ErrBlocked {
		t.Fatalf("expected blocked, got %v", err)
	}
	if err := w.PlaceStruct(0, "sandbag", cx, cy-5); err != ErrSupport {
		t.Fatalf("expected no support, got %v", err)
	}
	// out of the zone
	if err := w.PlaceStruct(0, "sandbag", 2, cy); err != ErrZone {
		t.Fatalf("expected zone error, got %v", err)
	}
	// tier lock
	if err := w.PlaceStruct(0, "oreshnik", cx+6, cy-1); err != ErrTier {
		t.Fatalf("expected tier error, got %v", err)
	}
	cx0 := (p.ZoneX0+p.ZoneX1)/2 + 100
	if err := w.PlaceUnit(0, "assault", float64(cx0), 100); err != nil {
		t.Fatalf("place unit: %v", err)
	}
	u := w.Units[len(w.Units)-1]
	if !w.unitOnGround(u.Pos.X, u.Pos.Y) {
		t.Fatalf("unit not on ground: %+v", u.Pos)
	}
}

func startBattleWith(t *testing.T, w *World) {
	for _, p := range w.Players {
		w.SetReady(p.ID, true)
	}
	w.Step()
	if w.Phase != PhaseBattle {
		t.Fatalf("battle did not start: %s", w.StatusLine())
	}
}

func TestBattleShootingAndMoney(t *testing.T) {
	w := newTest(2, 3)
	// each player gets one assault unit, facing each other
	for _, p := range w.Players {
		x := float64(p.ZoneX0+p.ZoneX1) / 2
		if err := w.PlaceUnit(p.ID, "assault", x+120, 100); err != nil {
			t.Fatal(err)
		}
	}
	startBattleWith(t, w)
	cur := w.Cur
	other := 1 - cur
	if w.SelUnit < 0 {
		t.Fatal("no unit selected")
	}
	u := w.Units[w.SelUnit]
	// teleport the enemy next to the shooter to guarantee a hit
	var target *Unit
	for _, e := range w.Units {
		if e.Owner == other {
			target = e
		}
	}
	target.Pos = Vec{u.Pos.X + 90, u.Pos.Y}
	if p, ok := w.dropUnitPos(target.Pos.X, target.Pos.Y-30); ok {
		target.Pos = p
	}
	before := w.Players[cur].Money
	hp := target.HP
	err := w.Apply(Command{Player: cur, Type: CmdFire, Weapon: "ak", Angle: math.Atan2(target.Pos.Y-UnitH/2-(u.Pos.Y-UnitH*0.6), target.Pos.X-u.Pos.X), Power: 1})
	if err != nil {
		t.Fatal(err)
	}
	if target.HP >= hp {
		t.Fatalf("target not damaged (hp %v)", target.HP)
	}
	if w.Players[cur].Money <= before {
		t.Fatalf("no money earned for damage")
	}
	// firing again in the same turn is not allowed
	if err := w.Apply(Command{Player: cur, Type: CmdFire, Weapon: "ak", Angle: 0, Power: 1}); err == nil {
		t.Fatal("second shot allowed")
	}
	// the turn eventually passes to the other player
	run(w, 15)
	if w.Cur != other {
		t.Fatalf("turn did not pass: cur=%d stage=%d", w.Cur, w.Stage)
	}
}

func TestExplosionCarvesTerrain(t *testing.T) {
	w := newTest(2, 5)
	x, y := 1500, w.Terr.SurfaceY(1500, 0)
	if !w.Terr.Solid(x, y+10) {
		t.Fatal("expected solid ground")
	}
	w.Explode(Vec{float64(x), float64(y + 5)}, ExplosionSpec{Radius: 50, Damage: 50, Owner: 0})
	if w.Terr.Solid(x, y+10) {
		t.Fatal("crater not carved")
	}
	if len(w.Terr.Craters) == 0 || len(w.Terr.Dirty) == 0 {
		t.Fatal("crater not recorded")
	}
}

func TestStructureCollapse(t *testing.T) {
	w := newTest(2, 2)
	p := w.Players[0]
	hq := w.Structs[p.HQ]
	cx, cy := hq.CX+6, hq.CY+3-1
	w.PlaceStruct(0, "concrete", cx, cy)
	w.PlaceStruct(0, "concrete", cx, cy-1)
	base := w.StructAtCell(cx, cy)
	top := w.StructAtCell(cx, cy-1)
	if base == nil || top == nil {
		t.Fatal("placement failed")
	}
	w.destroyStruct(base, 1)
	run(w, 2)
	if top.CY != cy {
		t.Fatalf("upper block should fall to row %d, is at %d", cy, top.CY)
	}
}

func TestAntiAirIntercepts(t *testing.T) {
	w := newTest(2, 9)
	// owner 1 builds an S-400 (tier-locked, so cheat the build number)
	w.BuildNo = 2
	p := w.Players[1]
	p.Money = 5000
	hq := w.Structs[p.HQ]
	if err := w.PlaceStruct(1, "s400", hq.CX-6, hq.CY+3-3); err != nil {
		t.Fatal(err)
	}
	startBattleWith(t, w)
	// fire many airstrikes at the S-400's owner and count intercepts
	intercepts := 0
	for i := 0; i < 40; i++ {
		w.Projs = nil
		w.Spawns = nil
		for _, s := range w.Structs {
			if s.Alive && w.Cfg.S(s.Def).Kind == balance.SAA {
				s.AAAmmo = 6
				s.HP, s.MaxHP = 1e9, 1e9
			}
		}
		w.Players[0].Items["fab"] = 1
		w.Cur = 0
		w.TurnTimer = 1e9
		w.launch(FireSpec{Player: 0, Unit: -1, Struct: -1, Weapon: "fab", TargetX: w.StructCenter(hq).X})
		run(w, 8)
		for _, e := range w.DrainEvents() {
			if e.Type == EvIntercept && e.B == 0 {
				intercepts++
			}
		}
	}
	if intercepts < 20 {
		t.Fatalf("S-400 should intercept most planes, got %d/40", intercepts)
	}
	if intercepts > 39 {
		t.Fatalf("S-400 should not be perfect, got %d/40", intercepts)
	}
}

func TestWinCondition(t *testing.T) {
	w := newTest(2, 4)
	startBattleWith(t, w)
	hq := w.Structs[w.Players[1].HQ]
	w.destroyStruct(hq, 0)
	w.endTurn()
	run(w, 5)
	if w.Phase != PhaseOver || w.Winner != 0 {
		t.Fatalf("expected player 0 to win: phase=%d winner=%d", w.Phase, w.Winner)
	}
}

func TestFullCycle(t *testing.T) {
	w := newTest(3, 11)
	for _, p := range w.Players {
		x := float64(p.ZoneX0+p.ZoneX1) / 2
		w.PlaceUnit(p.ID, "mortar", x+100, 100)
	}
	startBattleWith(t, w)
	// everybody just ends the turn: after RoundsPerBattle rounds we are back to building
	for i := 0; i < 60*60*5 && w.Phase == PhaseBattle; i++ {
		if w.Stage == StageActive {
			w.Apply(Command{Player: w.Cur, Type: CmdEndTurn})
		}
		w.Step()
	}
	if w.Phase != PhaseBuild || w.BuildNo != 2 {
		t.Fatalf("expected second build phase, got %s", w.StatusLine())
	}
	if w.Players[0].Money < w.Cfg.BaseIncome {
		t.Fatalf("base income not paid: %d", w.Players[0].Money)
	}
}

// battleWith prepares a 2-player battle where player 0 owns the given structure and units.
func battleWith(t *testing.T, structDef string, units ...string) (*World, *Struct) {
	t.Helper()
	w := newTest(2, 21)
	w.BuildNo = 3
	for _, p := range w.Players {
		p.Money = 20000
	}
	hq := w.Structs[w.Players[0].HQ]
	var st *Struct
	if structDef != "" {
		d := w.Cfg.S(structDef)
		cx := hq.CX + 6
		if err := w.PlaceStruct(0, structDef, cx, w.DropCell(cx, d.W, d.H, hq.CY-6)); err != nil {
			t.Fatal(err)
		}
		st = w.Structs[len(w.Structs)-1]
	}
	for i, u := range units {
		x := float64(w.Players[0].ZoneX0+w.Players[0].ZoneX1)/2 + 80 + float64(i)*30
		if err := w.PlaceUnit(0, u, x, 100); err != nil {
			t.Fatal(err)
		}
	}
	w.SetReady(0, true)
	w.SetReady(1, true)
	w.Step()
	for w.Cur != 0 {
		w.Apply(Command{Player: w.Cur, Type: CmdEndTurn})
		run(w, 1.5)
	}
	return w, st
}

func TestBallisticMissilesHitTarget(t *testing.T) {
	for _, def := range []string{"iskander", "oreshnik"} {
		w, st := battleWith(t, def)
		enemy := w.Structs[w.Players[1].HQ]
		before := enemy.HP
		w.Apply(Command{Player: 0, Type: CmdSelectStruct, ID: st.ID})
		tx := w.StructCenter(enemy).X
		if err := w.Apply(Command{Player: 0, Type: CmdFire, Weapon: w.Cfg.S(def).Weapon, X: tx, Power: 1, Angle: -1.5}); err != nil {
			t.Fatalf("%s: %v", def, err)
		}
		run(w, 12)
		if enemy.HP >= before {
			t.Fatalf("%s did not damage the target HQ (%.0f -> %.0f)", def, before, enemy.HP)
		}
		t.Logf("%s: HQ %.0f -> %.0f", def, before, enemy.HP)
	}
}

func TestDroneSteersAndExplodes(t *testing.T) {
	w, _ := battleWith(t, "", "dronner")
	enemy := w.Structs[w.Players[1].HQ]
	u := w.UnitsOf(0)[0]
	w.Apply(Command{Player: 0, Type: CmdSelectUnit, ID: u.ID})
	before := enemy.HP
	if err := w.Apply(Command{Player: 0, Type: CmdFire, Weapon: "fpv", Angle: -0.9, Power: 1}); err != nil {
		t.Fatal(err)
	}
	for i := 0; i < 60*10; i++ {
		for _, p := range w.Projs {
			if p.Kind == PDrone && p.Alive {
				tp := w.StructCenter(enemy)
				if math.Abs(tp.X-p.Pos.X) > 250 {
					tp.Y -= 700
				}
				d := tp.Sub(p.Pos)
				w.Apply(Command{Player: 0, Type: CmdSteer, Angle: math.Atan2(d.Y, d.X)})
			}
		}
		w.Step()
	}
	if enemy.HP >= before {
		t.Fatalf("drone never hit the enemy HQ")
	}
}

func TestMineAndRepair(t *testing.T) {
	w, _ := battleWith(t, "sandbag", "engineer")
	u := w.UnitsOf(0)[0]
	w.Apply(Command{Player: 0, Type: CmdSelectUnit, ID: u.ID})
	// damage the sandbag, then repair it with the engineer standing next to it
	var bag *Struct
	for _, s := range w.StructsOf(0) {
		if s.Def == "sandbag" {
			bag = s
		}
	}
	bag.HP = 10
	u.Pos = Vec{w.StructCenter(bag).X + 10, w.StructCenter(bag).Y + 8}
	if err := w.Apply(Command{Player: 0, Type: CmdFire, Weapon: "repair"}); err != nil {
		t.Fatal(err)
	}
	if bag.HP <= 10 {
		t.Fatalf("repair did nothing")
	}
}

func TestAirstrikeNeedsStockAndSpotterHelps(t *testing.T) {
	w, _ := battleWith(t, "", "assault")
	if err := w.Apply(Command{Player: 0, Type: CmdFire, Weapon: "kab", X: 1500}); err == nil {
		t.Fatal("airstrike without stock must fail")
	}
	w.Players[0].Items["kab"] = 1
	enemy := w.Structs[w.Players[1].HQ]
	before := enemy.HP
	if err := w.Apply(Command{Player: 0, Type: CmdFire, Weapon: "kab", X: w.StructCenter(enemy).X}); err != nil {
		t.Fatal(err)
	}
	run(w, 10)
	if enemy.HP >= before {
		t.Fatalf("KAB missed the HQ completely (even without a spotter it should be close)")
	}
}

func TestCanRunAfterShooting(t *testing.T) {
	w, _ := battleWith(t, "", "mortar", "assault")
	var u *Unit
	for _, x := range w.UnitsOf(0) {
		if x.Def == "mortar" {
			u = x
		}
	}
	w.Apply(Command{Player: 0, Type: CmdSelectUnit, ID: u.ID})
	if err := w.Apply(Command{Player: 0, Type: CmdFire, Weapon: "mortar", Angle: -1.2, Power: 0.8}); err != nil {
		t.Fatal(err)
	}
	if w.Stage != StageRetreat {
		t.Fatalf("expected retreat stage after shooting, got %d", w.Stage)
	}
	// the shell is still in the air: the retreat clock must not run yet
	t0 := w.RetreatTime
	run(w, 0.5)
	if w.ProjectilesBusy() && w.RetreatTime != t0 {
		t.Fatalf("retreat clock ran while the shell was flying")
	}
	x0 := u.Pos.X
	w.Apply(Command{Player: 0, Type: CmdWalk, Dir: -1})
	run(w, 2)
	if x0-u.Pos.X < 40 {
		t.Fatalf("pig did not run after shooting (moved %.0f px)", x0-u.Pos.X)
	}
	// other units must stay put
	if err := w.Apply(Command{Player: 0, Type: CmdSelectUnit, ID: w.UnitsOf(0)[1].ID}); err == nil {
		t.Fatal("switching units after the shot must be refused in Classic mode")
	}
	// the turn passes on eventually
	run(w, 30)
	if w.Cur == 0 && w.Stage != StageActive {
		t.Fatalf("turn stuck: cur=%d stage=%d", w.Cur, w.Stage)
	}
}
