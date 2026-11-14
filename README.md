# risk-catalogger

A just-for-fun tool for viewing incident risks, risk factors and their relative ranking against a reliability
target, after Google SRE's risk-catalog spreadsheet. It ranks each risk by its *bad minutes per year* and
compares them with an error budget.

Ideally this should serve as a tool to capture past incidents, their ETT{D,R,F} and % of failures so as to
enable core platform engineers to make informed reliability decisions, and ideally to improve their platform
reliability score.

## Usage

    riskcatalog [-f file] [-markdown]

| Flag | Default | What it does |
|---|---|---|
| `-f file` | `incidents.yml` | The risk catalog to read (format below). |
| `-markdown` | off | Print a Markdown report to stdout and exit, instead of opening the terminal UI. |

From a checkout, `make run` builds the tool and opens the catalog in this directory; pass flags with `ARGS`:

    make run
    make run ARGS="-f ~/my-catalog.yml"
    make -s run ARGS="-f ~/my-catalog.yml -markdown" > risks.md

### The terminal UI

`h`/`l` or the left and right arrows switch tabs, `j`/`k` or the up and down arrows scroll the Risk List, and
`q` quits. It needs a terminal of at least 120 by 40.

- **Risk Catalog**: the incidents with their ETTD, ETTR, impact, ETTF, incidents per year (IP/Y) and bad
  minutes per year (BM/Y).
- **Risk Factors**: the risk factors and the bad minutes per year each adds.
- **Risk Stack Rank**: every incident and risk factor, ranked by bad minutes per year and coloured by verdict:
  green fits (at most half the single-risk threshold), yellow is cautionary (over half of it), red is too big
  (over it). The bars show, for the targets 99.99, 99.95, 99.9 and 99.5, how much of that target's yearly
  budget the catalog uses, from risks under (green) and over (red) that target's single-risk threshold. A bar
  over 100% is drawn full, in proportion; the table under the bars has the uncapped figures.
- **Risk List**: the incidents' descriptions.
- **Error Budget**: the target, the budget per year and per 4 weeks, the single-risk threshold, the bad minutes
  of all incidents and risk factors together, and what's left.

### The Markdown report

`-markdown` prints the Error Budget tab's lines as a list, the stack rank as a table (description, ETTD, ETTR,
ETTF, impact, incidents per year, bad minutes per year, % of the budget, verdict; a risk factor shows a dash
from ETTD to incidents per year) and the risk factors as a table. Its sections are level-2 headings (`## Error budget`,
`## Stack rank`, `## Risk factors`), so it can be embedded under a page's own title.

## The catalog file

```yaml
error-budget:
  availability-target: 99.5               # percent
  single-risk-acceptable-threshold: 0.25  # fraction of the yearly budget one risk may use
incidents:
  - description: A restart drops every request for 20 seconds
    estimated-time-to-detect: 0           # minutes
    estimated-time-to-repair: 0.33        # minutes: 20 seconds
    estimated-time-to-fail: 7.5           # days between occurrences
    percentage-impact: 1                  # fraction of users or requests affected, 0 to 1
risks:
  - description: Detection takes 5 minutes longer (noisy alerting)
    increased-ettd-minutes: 5             # minutes added to every incident's ETTD
  - description: Repairs take 2.5 minutes longer (no playbooks)
    increased-ettr-minutes: 2.5           # minutes added to every incident's ETTR
  - description: Every incident happens 10% more often (no postmortem follow-up)
    increased-ettf-percentage: 0.1        # fraction: 0.1 is 10% more often
```

| Key | Unit |
|---|---|
| `error-budget.availability-target` | percent, more than 0 and less than 100 (99.5 is 99.5%) |
| `error-budget.single-risk-acceptable-threshold` | fraction of the yearly budget, more than 0 and at most 1 |
| `incidents[].estimated-time-to-detect` (ETTD) | minutes |
| `incidents[].estimated-time-to-repair` (ETTR) | minutes |
| `incidents[].estimated-time-to-fail` (ETTF) | days between occurrences, more than 0 |
| `incidents[].percentage-impact` | fraction from 0 to 1 (0.2 is 20%) |
| `risks[].increased-ettd-minutes`, `risks[].increased-ettr-minutes` | minutes |
| `risks[].increased-ettf-percentage` | fraction (0.1 is 10% more often) |

Minutes and days can have decimals (`0.33` is 20 seconds); whole numbers work too. A risk factor needs only the
keys it uses. The tool shows inputs with at most two decimals.

The arithmetic:

- An incident's bad minutes per year are (ETTD + ETTR) × impact × 365.25 / ETTF.
- A risk factor's are (1 + ETTF increase) × (its ETTD and ETTR minutes) × Σ(impact × incidents per year), plus
  the ETTF increase × the incidents' total bad minutes per year.
- The error budget is (1 − target / 100) × 1440 × 365.25 minutes per year (× 28 days for 4 weeks), and the
  single-risk threshold is that × `single-risk-acceptable-threshold`.

## Building

`make build` cross-compiles `bin/riskcatalog-darwin-arm64`, `bin/riskcatalog-darwin-amd64` and
`bin/riskcatalog-linux-amd64`; `make test` runs `go test ./...`; `make clean` removes what `build` and `run` made.

It needs Go 1.19 or newer, but on a recent macOS use a current Go: binaries from an old one (1.19, for example)
don't start there (`dyld: missing LC_UUID load command`). Without a local Go, Docker's image works:

    docker run --rm -v "$PWD":/src -w /src golang:1.25.4 go test ./...
    docker run --rm -v "$PWD":/src -w /src -e GOOS=darwin -e GOARCH=arm64 golang:1.25.4 go build -o riskcatalog ./cmd
