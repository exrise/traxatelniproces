package main

import (
	"fmt"
	"sort"

	"svinovoyna/internal/ai"
	"svinovoyna/internal/balance"
)

// staticReport prints raw efficiency numbers straight from the config.
func staticReport(c *balance.Config) {
	fmt.Printf("== Статический отчёт: %s ==\n\n", c.Name)
	fmt.Println("Оружие (урон за применение; без учёта попаданий)")
	fmt.Printf("%-18s %7s %6s %5s %7s %8s\n", "оружие", "урон", "радиус", "залп", "ammo", "цена")
	for _, w := range c.Weapons {
		n := float64(max(1, w.Count))
		fmt.Printf("%-18s %7.0f %6.0f %5.0f %7d %8d\n", w.Name, w.Damage*n, w.Radius, n, w.Ammo, w.Cost)
	}
	fmt.Println("\nСтруктуры: цена / прочность, HP за $")
	type row struct {
		name  string
		hpPer float64
	}
	var rows []row
	for _, s := range c.Structs {
		if s.Cost > 0 {
			rows = append(rows, row{s.Name, s.HP / float64(s.Cost)})
		}
	}
	sort.Slice(rows, func(i, j int) bool { return rows[i].hpPer > rows[j].hpPer })
	for _, r := range rows {
		fmt.Printf("  %-22s %.2f HP/$\n", r.name, r.hpPer)
	}
}

type agg struct {
	n                    int
	shots, unit, str, hq float64
	kills, intercepted   float64
}

// weaponReport aggregates WStat across matches.
func weaponReport(c *balance.Config, results []ai.MatchResult) {
	total := map[string]*agg{}
	for _, r := range results {
		for id, s := range r.Stats {
			a := total[id]
			if a == nil {
				a = &agg{}
				total[id] = a
			}
			a.shots += float64(s.Shots)
			a.unit += s.Unit
			a.str += s.Struct
			a.hq += s.HQ
			a.kills += float64(s.Kills)
			a.intercepted += float64(s.Intercepted)
		}
	}
	var ids []string
	for id := range total {
		ids = append(ids, id)
	}
	sort.Strings(ids)
	fmt.Println("\nОружие в боях ботов (на выстрел):")
	fmt.Printf("%-18s %7s %8s %8s %8s %7s %9s\n", "оружие", "выстр.", "юниты", "постр.", "штаб", "убийств", "сбито ПВО")
	for _, id := range ids {
		a := total[id]
		w := c.W(id)
		name := id
		if w != nil {
			name = w.Name
		}
		if a.shots == 0 {
			continue
		}
		fmt.Printf("%-18s %7.0f %8.1f %8.1f %8.1f %7.2f %8.0f%%\n", name, a.shots, a.unit/a.shots, a.str/a.shots, a.hq/a.shots, a.kills/a.shots, a.intercepted/a.shots*100)
	}
}
