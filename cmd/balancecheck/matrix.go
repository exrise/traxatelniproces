package main

import (
	"fmt"
	"runtime"
	"sync"

	"svinovoyna/internal/ai"
	"svinovoyna/internal/balance"
)

// matrixReport plays every pair of strategies (both seat orders) and prints the win-rate matrix.
func matrixReport(cfg *balance.Config, perPair int, skill, maxMin float64) {
	n := int(ai.NumStyles)
	type job struct{ a, b, k int }
	var jobs []job
	for a := 0; a < n; a++ {
		for b := 0; b < n; b++ {
			if a == b {
				continue
			}
			for k := 0; k < perPair; k++ {
				jobs = append(jobs, job{a, b, k})
			}
		}
	}
	wins := make([][]int, n)
	total := make([][]int, n)
	for i := range wins {
		wins[i] = make([]int, n)
		total[i] = make([]int, n)
	}
	var mu sync.Mutex
	var wg sync.WaitGroup
	sem := make(chan struct{}, runtime.NumCPU())
	for _, j := range jobs {
		wg.Add(1)
		sem <- struct{}{}
		go func(j job) {
			defer wg.Done()
			r := ai.RunMatch(cfg, uint64(1000+j.k), []ai.Style{ai.Style(j.a), ai.Style(j.b)}, skill, maxMin*60)
			mu.Lock()
			total[j.a][j.b]++
			total[j.b][j.a]++
			if r.Winner == 0 {
				wins[j.a][j.b]++
			} else if r.Winner == 1 {
				wins[j.b][j.a]++
			}
			mu.Unlock()
			<-sem
		}(j)
	}
	wg.Wait()
	fmt.Printf("\nМатрица побед (строка против столбца, %% побед; ничьи не считаются победами):\n%-18s", "")
	for b := 0; b < n; b++ {
		fmt.Printf("%9.8s", ai.Style(b).String())
	}
	fmt.Printf("%9s\n", "итого")
	for a := 0; a < n; a++ {
		fmt.Printf("%-18s", ai.Style(a))
		sw, st := 0, 0
		for b := 0; b < n; b++ {
			if a == b {
				fmt.Printf("%9s", "-")
				continue
			}
			p := 0.0
			if total[a][b] > 0 {
				p = float64(wins[a][b]) / float64(total[a][b]) * 100
			}
			fmt.Printf("%8.0f%%", p)
			sw += wins[a][b]
			st += total[a][b]
		}
		fmt.Printf("%8.0f%%\n", float64(sw)/float64(max(1, st))*100)
	}
}
