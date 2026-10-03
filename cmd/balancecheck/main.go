// balancecheck runs headless bot matches and static analysis to help tune balance.
package main

import (
	"flag"
	"fmt"
	"math"
	"os"
	"runtime"
	"sort"
	"strconv"
	"strings"
	"sync"

	"svinovoyna/internal/ai"
	"svinovoyna/internal/balance"
)

func main() {
	n := flag.Int("n", 40, "matches per configuration")
	players := flag.Int("players", 2, "players per match (2-4)")
	preset := flag.String("preset", "Стандарт", "preset name")
	skill := flag.Float64("skill", 0.8, "bot skill 0..1")
	maxMin := flag.Float64("max", 60, "max simulated minutes per match")
	hqHP := flag.Float64("hq", 0, "override HQ hit points")
	rounds := flag.Int("rounds", 0, "override rounds per battle")
	seats := flag.String("seats", "", "trace seats as comma separated style numbers, e.g. 4,5")
	matrix := flag.Int("matrix", 0, "play every strategy pair N times per seat order and print the win matrix")
	weapons := flag.Bool("weapons", false, "print per-weapon statistics")
	trace := flag.Bool("trace", false, "print per-battle log of one match")
	static := flag.Bool("static", false, "print static weapon efficiency table and exit")
	flag.Parse()

	var cfg *balance.Config
	for _, p := range balance.Presets() {
		if p.Name == *preset {
			cfg = p
		}
	}
	if cfg == nil {
		fmt.Fprintln(os.Stderr, "unknown preset")
		os.Exit(1)
	}
	if *hqHP > 0 {
		cfg.MaxHQHP = *hqHP
		for i := range cfg.Structs {
			if cfg.Structs[i].Kind == balance.SHQ {
				cfg.Structs[i].HP = *hqHP
			}
		}
	}
	if *rounds > 0 {
		cfg.RoundsPerBattle = *rounds
	}
	if *matrix > 0 {
		matrixReport(cfg, *matrix, *skill, *maxMin)
		return
	}
	if *static {
		staticReport(cfg)
		return
	}
	if *trace {
		st := make([]ai.Style, *players)
		for i := range st {
			st[i] = ai.Style(i % int(ai.NumStyles))
			if parts := strings.Split(*seats, ","); *seats != "" && i < len(parts) {
				n, _ := strconv.Atoi(parts[i])
				st[i] = ai.Style(n)
			}
		}
		r := ai.RunMatchLog(cfg, 1, st, *skill, *maxMin*60, func(f string, a ...any) { fmt.Printf(f+"\n", a...) })
		fmt.Printf("итог: победитель=%d завершено=%v время=%.0f с\n", r.Winner, r.Finished, r.Seconds)
		return
	}
	fmt.Printf("Пресет: %s, игроков: %d, матчей на конфигурацию: %d\n\n", cfg.Name, *players, *n)

	type job struct {
		seed   uint64
		styles []ai.Style
	}
	var jobs []job
	rng := uint64(12345)
	next := func() uint64 { rng = rng*6364136223846793005 + 1442695040888963407; return rng >> 33 }
	for i := 0; i < *n; i++ {
		styles := make([]ai.Style, *players)
		for k := range styles {
			styles[k] = ai.Style(next() % uint64(ai.NumStyles))
		}
		jobs = append(jobs, job{seed: uint64(i + 1), styles: styles})
	}
	results := make([]ai.MatchResult, len(jobs))
	var wg sync.WaitGroup
	sem := make(chan struct{}, runtime.NumCPU())
	for i := range jobs {
		wg.Add(1)
		sem <- struct{}{}
		go func(i int) {
			defer wg.Done()
			results[i] = ai.RunMatch(cfg, jobs[i].seed, jobs[i].styles, *skill, *maxMin*60)
			<-sem
		}(i)
	}
	wg.Wait()

	played := map[ai.Style]int{}
	won := map[ai.Style]int{}
	seatW := make([]int, *players)
	var secs, builds []float64
	fin, draws := 0, 0
	for i, r := range results {
		for _, s := range jobs[i].styles {
			played[s]++
		}
		if r.Finished {
			fin++
		}
		if r.Winner >= 0 {
			won[jobs[i].styles[r.Winner]]++
			seatW[r.Winner]++
		} else {
			draws++
		}
		secs = append(secs, r.Seconds/60)
		builds = append(builds, float64(r.Builds))
	}
	fmt.Printf("Завершено: %d/%d, ничьих/таймаутов: %d\n", fin, len(results), draws)
	fmt.Printf("Длина матча (мин): медиана %.1f, среднее %.1f; циклов стройки: медиана %.0f\n\n", median(secs), mean(secs), median(builds))
	fmt.Println("Стратегия          игр  побед  винрейт")
	for s := ai.Style(0); s < ai.NumStyles; s++ {
		wr := 0.0
		if played[s] > 0 {
			wr = float64(won[s]) / float64(played[s]) * 100
		}
		fmt.Printf("%-18s %4d %6d  %5.1f%%\n", s, played[s], won[s], wr)
	}
	fmt.Println()
	if *weapons {
		weaponReport(cfg, results)
	}
	fmt.Print("Победы по слоту (порядок игроков): ")
	for i, v := range seatW {
		fmt.Printf("[%d]=%d ", i+1, v)
	}
	fmt.Println()
}

func mean(a []float64) float64 {
	s := 0.0
	for _, v := range a {
		s += v
	}
	return s / math.Max(1, float64(len(a)))
}

func median(a []float64) float64 {
	if len(a) == 0 {
		return 0
	}
	b := append([]float64(nil), a...)
	sort.Float64s(b)
	return b[len(b)/2]
}
