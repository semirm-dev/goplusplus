# C1 · Measure, tune, rewrite — a hands-on lab

A small HTTP service that builds a CSV report, so you can practice the whole loop on
something real: **measure** where memory goes, **tune** the GC, and only then
**rewrite** code.

```
report.go       ReportNaive (how it usually starts) and ReportTuned (the rewrite)
report_test.go  a test that both produce the same bytes, plus two benchmarks
main.go         the service: /report on :8080, pprof on localhost:6060
```

All output below is from a real run on Go 1.27. Your numbers will be different,
what matters is how to read them.

---

## 1 · Measure in a benchmark

```bash
go test -run '^$' -bench . -benchmem -memprofile mem.out
```

- `-run '^$'` skips the normal tests (no test name matches an empty string)
- `-bench .` runs every benchmark (`.` is a regex that matches any name)
- `-benchmem` adds the memory columns
- `-memprofile mem.out` saves *where* allocations happen, for step 2

```
BenchmarkReportNaive-32     163   7619017 ns/op   29655409 B/op   7606 allocs/op
BenchmarkReportTuned-32   23653     50429 ns/op       5720 B/op     13 allocs/op
```

How to read a line:

| column | meaning |
|---|---|
| `-32` | GOMAXPROCS the benchmark ran with |
| `163` | how many times the loop ran (Go picks this) |
| `ns/op` | time per call |
| `B/op` | bytes allocated per call |
| `allocs/op` | heap allocations per call |

**What to look for:** `allocs/op` first. Every allocation is work for the GC later.
Here the naive report makes about 7600 allocations for one CSV. That's the number to
explain.

> Benchmark trap: the result has to go somewhere, or the compiler may remove the
> call. Use a sink of the **same type** (`var sinkString string`). A sink of type
> `any` boxes the result and adds an allocation that isn't yours.

---

## 2 · Find where they come from

### In the terminal

```bash
go tool pprof -alloc_space -top mem.out
```

```
      flat  flat%   sum%        cum   cum%
 4529.82MB 96.76% 96.76%  4546.35MB 97.11%  ...ReportNaive
   82.77MB  1.77% 98.52%   128.80MB  2.75%  ...ReportTuned
```

- **flat**: allocated *in this function's own code*
- **cum** (cumulative): this function *plus everything it calls*
- big `flat` = the function itself allocates; big `cum` but small `flat` = something it
  calls does

Then go to the exact line:

```bash
go tool pprof -alloc_space -list ReportNaive mem.out
```

```
    4.13GB     4.15GB     28:	out += fmt.Sprintf("%d,%s,%.2f\n", o.ID, o.Customer, o.Amount)
         .          .     29:	totals[o.Customer] += o.Amount
  297.82MB   299.32MB     33:	out += fmt.Sprintf("total,%s,%.2f\n", c, totals[c])
```

Line 28 is almost everything. `out +=` copies the whole string so far on every line,
and `Sprintf` allocates a new string each time.

### In the browser (much nicer)

```bash
go tool pprof -http=:8081 mem.out
```

This opens a web UI. The two menus at the top are what you need:

**VIEW**
- **Top**: the same table as `-top`, sortable
- **Graph**: boxes and arrows, who calls whom. Bigger, redder box = more memory
- **Flame Graph**: the most useful one. Each bar is a function, **width = memory**.
  Bars on top are called by bars below. Look for the widest bars that are *your* code
- **Source**: your code with memory per line, like `-list`
- **Peek**: callers and callees of one function

**SAMPLE**
- `alloc_space`: total bytes allocated (what creates garbage)
- `alloc_objects`: number of allocations (many small ones also hurt)
- `inuse_space` / `inuse_objects`: what is still alive right now (what holds memory)

Click a function in any view to focus on it, `Esc` or the search box to reset.

**What to look for:**
- your own functions with a big `flat`
- `fmt.Sprintf`, `strings` building, `runtime.concatstrings`: string building in a loop
- `runtime.growslice`: an `append` without enough capacity
- `runtime.mapassign` / map growth: a map without a size hint
- anything from `sync.Pool.New`: the pool keeps creating new objects (see step 4)

---

## 3 · Measure a running service

The same profiles, from the real program under load.

```bash
go run . -load
```

`-load` starts a few clients that keep calling `/report`. The service imports
`net/http/pprof`, which adds `/debug/pprof/` on `localhost:6060`. Open
<http://localhost:6060/debug/pprof/> to see every profile it offers (`allocs`, `heap`,
`goroutine`, `profile` for CPU, ...).

Allocations **during the next 5 seconds**:

```bash
go tool pprof -http=:8081 -sample_index=alloc_space "http://localhost:6060/debug/pprof/allocs?seconds=5"
```

> Without `-sample_index=alloc_space` this profile opens on `inuse_space`, which shows
> almost nothing. Or switch it in the SAMPLE menu.

```
Duration: 5s, Total samples = 57.77GB
      flat  flat%        cum   cum%
   57.61GB 99.71%    57.69GB 99.87%  main.ReportNaive
```

What is alive right now:

```bash
go tool pprof -http=:8081 http://localhost:6060/debug/pprof/heap
```

**Never expose `/debug/pprof/` publicly.** Here it listens on `localhost` only, on a
port the app doesn't use. In Kubernetes, reach it with `kubectl port-forward`.

---

## 4 · Tune: watch the GC with gctrace

`GODEBUG=gctrace=1` prints one line per GC:

```bash
GODEBUG=gctrace=1 go run . -load
```

```
gc 4576 @5.002s 16%: 0.29+0.19+0.10 ms clock, ..., 18->24->6 MB, 21 MB goal, ..., 32 P
```

| part | meaning |
|---|---|
| `gc 4576` | the 4576th GC since start |
| `@5.002s` | time since the program started |
| `16%` | share of CPU spent in GC since start |
| `18->24->6 MB` | heap when this GC started -> when it ended -> **still alive after it** |
| `21 MB goal` | heap size that triggers the next GC |

The last number of `18->24->6` is the live heap. With `GOGC=100` the next goal is about
double that. Change it and compare (run each for a few seconds):

```bash
GOGC=100 GODEBUG=gctrace=1 go run . -load
GOGC=400 GODEBUG=gctrace=1 go run . -load
GOGC=400 GOMEMLIMIT=64MiB GODEBUG=gctrace=1 go run . -load
```

Naive report, 5 seconds each:

| setting | GCs | GC CPU | heap goal |
|---|---|---|---|
| `GOGC=100` | ~4500 | 16% | 21 MB |
| `GOGC=400` | ~2360 | 9% | 69 MB |
| `GOGC=400 GOMEMLIMIT=64MiB` | ~2760 | 11% | 48 MB |

Higher `GOGC`: fewer GCs, more memory. `GOMEMLIMIT` pulls the goal back down when memory
gets close to the limit. That's the whole trade-off, with no code change.

**Compare per request, not per run.** The service logs
`served N requests in the last 5s`. A faster version serves more requests, so it can run
*more* GCs in total and still be much better. Divide GCs by requests.

---

## 5 · Rewrite, then measure again

Tuning helped, but the naive report still makes ~7600 allocations per request. The
profile pointed at line 28, so that's what `ReportTuned` rewrites: `strconv.Append*` into
one buffer, a map size hint, and a pooled buffer per request in `main.go`.

```bash
GODEBUG=gctrace=1 go run . -load -tuned
GOGC=400 GODEBUG=gctrace=1 go run . -load -tuned
```

6 seconds each:

| version | requests / 5s | GCs | GC CPU |
|---|---|---|---|
| naive, `GOGC=100` | ~1 600 | ~5 200 | 16% |
| tuned, `GOGC=100` | ~48 000 | ~3 300 | 13% |
| tuned, `GOGC=400` | ~88 000 | ~130 | 0% |

Tuned at `GOGC=100` still runs many GCs. Profile it and the top allocator is
`main.init.func1`, which is `bufPool.New`:

```
 1045.15MB 57.62%  main.init.func1
```

The live heap is tiny (`4->4->2 MB`), so the GC runs very often, and **every GC empties
the `sync.Pool`**, so the pool keeps allocating new 64 KB buffers. With `GOGC=400` the
pool survives between GCs and `main.init.func1` drops to a few percent. Measure, tune,
rewrite, then measure and tune again.

---

## Cheat sheet

```bash
# benchmark + profile
go test -run '^$' -bench . -benchmem -memprofile mem.out
go tool pprof -http=:8081 mem.out

# running service (needs: import _ "net/http/pprof" and an HTTP server)
go tool pprof -http=:8081 -sample_index=alloc_space "http://localhost:6060/debug/pprof/allocs?seconds=30"
go tool pprof -http=:8081 http://localhost:6060/debug/pprof/heap

# GC
GODEBUG=gctrace=1 GOGC=200 GOMEMLIMIT=7GiB ./app

# compare two benchmark runs (go install golang.org/x/perf/cmd/benchstat@latest)
go test -run '^$' -bench . -benchmem -count 10 > old.txt
#   ...change the code...
go test -run '^$' -bench . -benchmem -count 10 > new.txt
benchstat old.txt new.txt
```

Files created while you practice: `mem.out`, `c1-measure-tune-rewrite.test`, and
profiles that `go tool pprof` saves under `~/pprof/`. Delete them when you're done.
