// bench runs N Battlesnake games between two snake URLs and reports stats.
//
// Usage:
//
//	go run ./cmd/bench [flags]
//
// Flags:
//
//	-games     number of games to play (default 10)
//	-workers   parallel games (default 4)
//	-a-name    name for snake A (default "A")
//	-a-url     URL for snake A (default http://localhost:8080)
//	-b-name    name for snake B (default "B")
//	-b-url     URL for snake B (default http://localhost:8080)
//	-width     board width  (default 11)
//	-height    board height (default 11)
//	-timeout   move timeout ms (default 200)
//	-save FILE write per-game JSONL records to FILE (optional)
//
// JSONL record fields:
//
//	n       game number
//	winner  "A", "B", or "draw"
//	turns   total turns played
//	seed    engine seed — replay with: go tool battlesnake play --seed <seed>
package main

import (
	"bufio"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"os/exec"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
)

type result struct {
	n      int
	winner string // "A", "B", or "draw"
	turns  int
	seed   int64
}

// gameRecord is the JSONL schema written to -save file.
type gameRecord struct {
	N      int    `json:"n"`
	Winner string `json:"winner"`
	Turns  int    `json:"turns"`
	Seed   int64  `json:"seed"`
}

var (
	reWinner = regexp.MustCompile(`Game completed after (\d+) turns\. (.+) was the winner\.`)
	reDraw   = regexp.MustCompile(`Game completed after (\d+) turns\.`)
	reSeed   = regexp.MustCompile(`Seed: (-?\d+)`)
)

func runGame(n int, aName, aURL, bName, bURL string, width, height, timeout int) result {
	cmd := exec.Command("go", "tool", "battlesnake", "play",
		"--name", aName, "--url", aURL,
		"--name", bName, "--url", bURL,
		"-W", strconv.Itoa(width),
		"-H", strconv.Itoa(height),
		"--timeout", strconv.Itoa(timeout),
	)

	out, _ := cmd.CombinedOutput()

	r := result{n: n, winner: "draw"}
	for _, line := range strings.Split(string(out), "\n") {
		if m := reSeed.FindStringSubmatch(line); m != nil {
			r.seed, _ = strconv.ParseInt(m[1], 10, 64)
		}
		if m := reWinner.FindStringSubmatch(line); m != nil {
			r.turns, _ = strconv.Atoi(m[1])
			switch m[2] {
			case aName:
				r.winner = "A"
			case bName:
				r.winner = "B"
			}
		} else if m := reDraw.FindStringSubmatch(line); m != nil {
			r.turns, _ = strconv.Atoi(m[1])
		}
	}
	return r
}

func main() {
	games := flag.Int("games", 10, "number of games")
	workers := flag.Int("workers", 4, "parallel games")
	aName := flag.String("a-name", "A", "name for snake A")
	aURL := flag.String("a-url", "http://localhost:8080", "URL for snake A")
	bName := flag.String("b-name", "B", "name for snake B")
	bURL := flag.String("b-url", "http://localhost:8080", "URL for snake B")
	width := flag.Int("width", 11, "board width")
	height := flag.Int("height", 11, "board height")
	timeout := flag.Int("timeout", 200, "move timeout ms")
	saveFile := flag.String("save", "", "write per-game JSONL to this file")
	flag.Parse()

	fmt.Printf("Running %d games (%d workers): %s vs %s\n", *games, *workers, *aName, *bName)

	var logFile *os.File
	var logMu sync.Mutex
	if *saveFile != "" {
		var err error
		logFile, err = os.Create(*saveFile)
		if err != nil {
			fmt.Fprintf(os.Stderr, "cannot open save file: %v\n", err)
			os.Exit(1)
		}
		defer logFile.Close()
	}

	var (
		winsA atomic.Int64
		winsB atomic.Int64
		draws atomic.Int64
		total atomic.Int64
		minT  atomic.Int64
		maxT  atomic.Int64
	)
	minT.Store(1 << 62)

	work := make(chan int, *games)
	for i := 0; i < *games; i++ {
		work <- i + 1
	}
	close(work)

	var wg sync.WaitGroup
	for w := 0; w < *workers; w++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for n := range work {
				r := runGame(n, *aName, *aURL, *bName, *bURL, *width, *height, *timeout)
				switch r.winner {
				case "A":
					winsA.Add(1)
				case "B":
					winsB.Add(1)
				default:
					draws.Add(1)
				}
				total.Add(int64(r.turns))
				if int64(r.turns) < minT.Load() {
					minT.Store(int64(r.turns))
				}
				if int64(r.turns) > maxT.Load() {
					maxT.Store(int64(r.turns))
				}

				if logFile != nil {
					rec, _ := json.Marshal(gameRecord{N: r.n, Winner: r.winner, Turns: r.turns, Seed: r.seed})
					logMu.Lock()
					logFile.Write(rec)
					logFile.WriteString("\n")
					logMu.Unlock()
				}

				fmt.Fprint(os.Stdout, ".")
			}
		}()
	}
	wg.Wait()
	fmt.Println()

	n := int64(*games)
	wa := winsA.Load()
	wb := winsB.Load()
	dr := draws.Load()
	avg := total.Load() / n

	bw := bufio.NewWriter(os.Stdout)
	fmt.Fprintln(bw, strings.Repeat("─", 50))
	fmt.Fprintf(bw, "%-28s %4d wins  (%5.1f%%)\n", *aName+" (A)", wa, pct(wa, n))
	fmt.Fprintf(bw, "%-28s %4d wins  (%5.1f%%)\n", *bName+" (B)", wb, pct(wb, n))
	fmt.Fprintf(bw, "%-28s %4d        (%5.1f%%)\n", "Draws", dr, pct(dr, n))
	fmt.Fprintln(bw, strings.Repeat("─", 50))
	fmt.Fprintf(bw, "Turns — avg: %d  min: %d  max: %d\n", avg, minT.Load(), maxT.Load())
	fmt.Fprintln(bw, strings.Repeat("─", 50))
	if *saveFile != "" {
		fmt.Fprintf(bw, "Records saved to: %s\n", *saveFile)
	}
	bw.Flush()
}

func pct(v, total int64) float64 {
	if total == 0 {
		return 0
	}
	return float64(v) / float64(total) * 100
}
