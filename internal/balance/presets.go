package balance

// Default returns the "Стандарт" preset — the reference balance.
func Default() *Config {
	c := &Config{
		Name:              "Стандарт",
		TurnMode:          TurnClassic,
		StartMoney:        2000,
		BuildTimeFirst:    150,
		BuildTime:         90,
		RoundsPerBattle:   6,
		AutoRounds:        true,
		TurnTime:          45,
		RetreatTime:       8,
		PlanTime:          30,
		MaxHQHP:           750,
		SuddenDeathBattle: 4,
		SuddenDeathDmg:    55,

		BaseIncome:     900,
		DmgMoneyUnit:   1.0,
		DmgMoneyStruct: 0.35,
		KillBonus:      80,
		StructKillPct:  0.10,
		CaptureIncome:  100,
		SellRefund:     0.5,
		RepairCostPct:  0.5,
		LeaderMult:     1.3,
		UnderdogMult:   0.7,
		CatchUpPct:     0.25,
		CatchUpCap:     600,
		TwoHandDmgMul:  0.75,
		FriendlyFire:   true,
		MaxUnits:       8,
		GravityPx:      620,
		WindMax:        120,
		FallDamage:     0.35,
		BuildHeal:      35,
		UnitSpeed:      70,

		Weapons: []Weapon{
			{ID: "ak", Name: "АК-74", Kind: KindBurst, Damage: 8, Count: 8, Spread: 0.035, Range: 720},
			{ID: "grenade", Name: "РГД-5", Kind: KindShell, Damage: 50, Radius: 46, Speed: 760, Gravity: 1, Fuse: 3, Crater: 0.8, Class: ClassNone},
			{ID: "saiga", Name: "Сайга-12", Kind: KindPellets, Damage: 17, Count: 6, Spread: 0.24, Range: 280, Falloff: 0.8},
			{ID: "svd", Name: "СВД", Kind: KindShot, Damage: 75, Count: 1, Spread: 0.004, Range: 1500},
			{ID: "rpg", Name: "РПГ-7", Kind: KindShell, Damage: 72, Radius: 42, Speed: 950, Gravity: 0.35, WindK: 0.2, Pierce: 0.6, BlockMul: 1.5, Crater: 0.9},
			{ID: "mortar", Name: "Миномёт 82 мм", Kind: KindShell, Damage: 70, Radius: 62, Speed: 1050, Gravity: 1, WindK: 0.7, Crater: 1.0},
			{ID: "fpv", Name: "FPV-дрон", Kind: KindDrone, Damage: 72, Radius: 36, Speed: 330, Flight: 9, Class: ClassDrone, Crater: 0.8},
			{ID: "makarov", Name: "ПМ", Kind: KindBurst, Damage: 12, Count: 3, Spread: 0.03, Range: 420},
			{ID: "tm62", Name: "Мина ТМ-62", Kind: KindMine, Damage: 85, Radius: 46, Ammo: 2, Crater: 0.7},
			{ID: "repair", Name: "Ремонт", Kind: KindRepair, Damage: 40, Radius: 56, Ammo: 4},

			{ID: "dshk", Name: "ДШК", Kind: KindBurst, Damage: 12, Count: 14, Spread: 0.03, Range: 1050, BlockMul: 1.0, Ammo: 5},
			{ID: "d30", Name: "Д-30", Kind: KindShell, Damage: 72, Radius: 58, Speed: 1300, Gravity: 1, WindK: 0.5, Crater: 1.1, BlockMul: 1.2, Ammo: 3},
			{ID: "grad", Name: "БМ-21 «Град»", Kind: KindSalvo, Damage: 28, Radius: 36, Count: 10, Spread: 0.06, Speed: 1150, Gravity: 1, WindK: 0.5, Class: ClassRocket, Crater: 0.8, Ammo: 2},
			{ID: "kornet", Name: "ПТРК «Корнет»", Kind: KindGuided, Damage: 130, Radius: 30, Speed: 520, Flight: 14, Turn: 2.4, Pierce: 0.9, BlockMul: 1.6, Class: ClassNone, Crater: 0.6, Ammo: 4},
			{ID: "zu23gun", Name: "ЗУ-23-2 (огонь)", Kind: KindBurst, Damage: 6, Count: 22, Spread: 0.03, Range: 820, BlockMul: 0.7, Ammo: 4},
			{ID: "iskander", Name: "ПУ «Искандер»", Kind: KindBallis, Damage: 150, Radius: 82, Count: 1, Speed: 900, Class: ClassBallis, Crater: 1.2, BlockMul: 1.3, Ammo: 1},
			{ID: "oreshnik", Name: "«Орешник»", Kind: KindMIRV, Damage: 125, Radius: 74, Count: 6, Spread: 150, Speed: 1100, Class: ClassOreshnik, Crater: 1.3, BlockMul: 1.3, Ammo: 1},

			{ID: "fab", Name: "Авиаудар ФАБ-500", Kind: KindAirstrike, Damage: 90, Radius: 66, Count: 3, Spread: 90, Speed: 520, Gravity: 2, Class: ClassAir, Crater: 1.1, BlockMul: 1.2, Cost: 160},
			{ID: "kab", Name: "КАБ-1500", Kind: KindAirstrike, Damage: 230, Radius: 100, Count: 1, Spread: 10, Speed: 520, Gravity: 2, Class: ClassAir, Crater: 1.4, BlockMul: 1.4, Cost: 330},
			{ID: "geran", Name: "«Герань»", Kind: KindGeran, Damage: 165, Radius: 68, Speed: 170, Flight: 14, Class: ClassDrone, Crater: 1.0, Cost: 140},
		},

		Units: []UnitDef{
			{ID: "assault", Name: "Штурмовик", Cost: 130, HP: 100, Weapons: []string{"ak", "grenade"}, Desc: "АК-74 и гранаты. Универсал."},
			{ID: "shotgun", Name: "Дробовик", Cost: 110, HP: 110, Weapons: []string{"saiga", "grenade"}, Desc: "Сайга-12. Страшен вблизи."},
			{ID: "sniper", Name: "Снайпер", Cost: 180, HP: 80, Weapons: []string{"svd"}, Desc: "СВД. Один точный выстрел."},
			{ID: "rpg", Name: "Гранатомётчик", Cost: 180, HP: 100, Weapons: []string{"rpg", "makarov"}, Desc: "РПГ-7. Ломает укрепления."},
			{ID: "mortar", Name: "Миномётчик", Cost: 200, HP: 90, Weapons: []string{"mortar", "makarov"}, Desc: "Миномёт. Навесом через стены."},
			{ID: "dronner", Name: "Оператор БПЛА", Cost: 220, HP: 80, Weapons: []string{"fpv", "makarov"}, Max: 2, Desc: "FPV-дрон, управляемый полёт."},
			{ID: "spotter", Name: "Наводчик", Cost: 160, HP: 90, Weapons: []string{"makarov"}, Max: 2, Desc: "Точное наведение авиаударов."},
			{ID: "engineer", Name: "Инженер", Cost: 140, HP: 100, Weapons: []string{"makarov", "tm62", "repair"}, Max: 2, Desc: "Мины и ремонт укреплений."},
		},

		Structs: []StructDef{
			{ID: "hq", Name: "Штаб", Kind: SHQ, Cost: 0, W: 4, H: 3, HP: 750, Armor: 0.3, Blast: 0.35, Desc: "Потеряешь штаб — проиграл."},
			{ID: "sandbag", Name: "Мешки с песком", Kind: SBlock, Cost: 10, W: 2, H: 1, HP: 55, Armor: 0.5, Blast: 0.6, Desc: "Гасят взрывы."},
			{ID: "concrete", Name: "Бетонный блок", Kind: SBlock, Cost: 26, W: 2, H: 1, HP: 150, Armor: 0.6, Blast: 0.35, Desc: "Много HP."},
			{ID: "armor", Name: "Бронеплита", Kind: SBlock, Cost: 40, W: 1, H: 2, HP: 210, Armor: 0.9, Blast: 0.3, Desc: "Держит пули; слаба к кумулятиву."},
			{ID: "window", Name: "Бойница", Kind: SWindow, Cost: 32, W: 2, H: 1, HP: 95, Armor: 0.6, Blast: 0.3, Desc: "Чужие пули и снаряды не пролетают, а твои — свободно."},
			{ID: "ezh", Name: "Ёж", Kind: SBlock, Cost: 10, W: 2, H: 1, HP: 90, Armor: 0.4, Blast: 0.3, Desc: "Дешёвая преграда."},
			{ID: "bunker", Name: "Бункер", Kind: SBunker, Cost: 150, W: 4, H: 3, HP: 380, Armor: 0.7, Blast: 0.55, Desc: "Заходи внутрь: пули и осколки не достанут, взрывы слабее."},
			{ID: "net", Name: "Антидроновая сетка", Kind: SNet, Cost: 25, W: 3, H: 1, HP: 25, Armor: 0, Blast: 0, Desc: "Дроны подрываются на сетке."},

			{ID: "dshk", Name: "Гнездо ДШК", Kind: SWeapon, Weapon: "dshk", Cost: 120, W: 2, H: 2, HP: 90, Armor: 0.3, Blast: 0.2, Max: 3, Desc: "Пулемётная очередь."},
			{ID: "d30", Name: "Гаубица Д-30", Kind: SWeapon, Weapon: "d30", Cost: 300, W: 3, H: 2, HP: 110, Armor: 0.3, Blast: 0.2, Max: 3, Desc: "Дальняя артиллерия."},
			{ID: "grad", Name: "РСЗО «Град»", Kind: SWeapon, Weapon: "grad", Cost: 450, W: 3, H: 2, HP: 100, Armor: 0.2, Blast: 0.2, Max: 2, Desc: "Залп 12 ракет."},
			{ID: "kornet", Name: "ПТРК «Корнет»", Kind: SWeapon, Weapon: "kornet", Cost: 350, W: 2, H: 2, HP: 90, Armor: 0.3, Blast: 0.2, Max: 2, Desc: "ПТУР: пуск, потом наводишь мышью и подрываешь кликом. Пробивает броню."},
			{ID: "iskander", Name: "ПУ «Искандер»", Kind: SWeapon, Weapon: "iskander", Cost: 800, W: 3, H: 2, HP: 120, Armor: 0.3, Blast: 0.2, Max: 1, Tier: 2, Desc: "Баллистическая ракета сверху."},
			{ID: "oreshnik", Name: "«Орешник»", Kind: SWeapon, Weapon: "oreshnik", Cost: 1600, W: 4, H: 2, HP: 140, Armor: 0.3, Blast: 0.2, Max: 1, Tier: 2, Desc: "6 боевых блоков. Раз за бой."},

			{ID: "zu23", Name: "ЗУ-23-2", Kind: SAA, Weapon: "zu23gun", Cost: 140, W: 2, H: 2, HP: 80, Armor: 0.3, Blast: 0.2, Max: 4, AARange: 380, AAAmmo: 8,
				AAHit: map[AAClass]float64{ClassDrone: 0.8, ClassAir: 0.4, ClassRocket: 0.25}, Desc: "Сама бьёт по самолётам и дронам; можно стрелять и по земле."},
			{ID: "pantsir", Name: "Панцирь-С1", Kind: SAA, Cost: 420, W: 3, H: 3, HP: 150, Armor: 0.4, Blast: 0.3, Max: 3, Tier: 1, AARange: 520, AAAmmo: 10,
				AAHit: map[AAClass]float64{ClassDrone: 0.85, ClassAir: 0.62, ClassRocket: 0.55, ClassOreshnik: 0.05}, Desc: "Универсальное ПВО."},
			{ID: "s400", Name: "С-400", Kind: SAA, Cost: 850, W: 4, H: 3, HP: 200, Armor: 0.4, Blast: 0.3, Max: 1, Tier: 2, AARange: 950, AAAmmo: 6,
				AAHit: map[AAClass]float64{ClassDrone: 0.25, ClassAir: 0.85, ClassRocket: 0.65, ClassBallis: 0.6, ClassOreshnik: 0.2}, Desc: "Дальнее ПВО против авиации и ракет."},
			{ID: "reb", Name: "РЭБ", Kind: SJammer, Cost: 220, W: 2, H: 2, HP: 70, Armor: 0.2, Blast: 0.2, Max: 2, AARange: 450, AAAmmo: 99,
				AAHit: map[AAClass]float64{ClassDrone: 0.55}, Desc: "Глушит дроны в радиусе."},

			{ID: "oil", Name: "Нефтевышка", Kind: SEco, Cost: 200, W: 2, H: 3, HP: 80, Armor: 0.1, Blast: 0.1, Max: 3, Income: 60, Desc: "+60$ каждый твой ход."},
			{ID: "farm", Name: "Свиноферма", Kind: SEco, Cost: 120, W: 3, H: 2, HP: 70, Armor: 0.1, Blast: 0.1, Max: 4, Income: 35, Desc: "+35$ каждый твой ход."},
		},
	}
	c.Index()
	return c
}

// Presets returns all built-in presets (first one is the default).
func Presets() []*Config {
	std := Default()

	trench := Default()
	trench.Name = "Окопная война"
	trench.RoundsPerBattle = 8
	for i := range trench.Structs {
		s := &trench.Structs[i]
		if s.Kind == SBlock || s.Kind == SBunker {
			s.Cost = s.Cost * 3 / 4
			s.HP *= 1.15
		}
	}
	for i := range trench.Weapons {
		w := &trench.Weapons[i]
		if w.Kind == KindBallis || w.Kind == KindMIRV || w.Kind == KindAirstrike {
			w.Cost = w.Cost * 5 / 4
		}
	}
	trench.StartMoney = 2200

	rockets := Default()
	rockets.Name = "Ракетный ад"
	rockets.StartMoney = 3600
	rockets.BaseIncome = 1400
	rockets.RoundsPerBattle = 5
	rockets.SuddenDeathBattle = 3
	rockets.SuddenDeathDmg = 70
	rockets.MaxHQHP = 650
	for i := range rockets.Structs {
		if rockets.Structs[i].Kind == SHQ {
			rockets.Structs[i].HP = 650
		}
	}
	for i := range rockets.Structs {
		s := &rockets.Structs[i]
		if s.Kind == SAA {
			for k, v := range s.AAHit {
				s.AAHit[k] = min(0.95, v*1.05)
			}
			s.AAAmmo += 2
		}
		if s.Tier > 1 {
			s.Tier = 1
		}
	}
	for i := range rockets.Weapons {
		w := &rockets.Weapons[i]
		if w.Kind == KindGrad() || w.Kind == KindBallis || w.Kind == KindMIRV || w.Kind == KindMissile {
			w.Damage *= 1.25
		}
	}

	infantry := Default()
	infantry.Name = "Пехота"
	for i := range infantry.Units {
		infantry.Units[i].Cost = infantry.Units[i].Cost * 3 / 4
	}
	infantry.MaxUnits = 10
	infantry.StartMoney = 1800
	for i := range infantry.Structs {
		if infantry.Structs[i].ID == "oreshnik" {
			infantry.Structs[i].Tier = 99 // disabled in this preset
		}
	}

	blitz := Default()
	blitz.Name = "Блиц"
	blitz.StartMoney = 1300
	blitz.BuildTimeFirst = 75
	blitz.BuildTime = 45
	blitz.TurnTime = 25
	blitz.RoundsPerBattle = 4
	blitz.BaseIncome = 700

	list := []*Config{std, trench, rockets, infantry, blitz}
	for _, p := range list {
		p.Index()
	}
	return list
}

// KindGrad is a tiny helper kept for readability in presets.
func KindGrad() WeaponKind { return KindSalvo }
