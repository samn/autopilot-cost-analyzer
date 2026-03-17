# Plan 001: Initial Implementation

## Summary

Build the gke-cost-analyzer CLI that monitors GKE Autopilot pod costs
(CPU + Memory) and can display them in a terminal table (`watch`) or write
periodic snapshots to BigQuery (`record`). Prices are fetched from the Cloud
Billing Catalog API and cached locally.

## Design Decisions (from spec clarification)

| Decision | Choice |
|---|---|
| Pricing source | Cloud Billing Catalog API (`cloudbilling.googleapis.com`) |
| Resource types | CPU + Memory requests only |
| Cluster discovery | Current kubeconfig context (single cluster) |
| BigQuery write mode | Continuous daemon, periodic snapshots (default 5 min) |
| Label config | CLI flags (`--team-label`, `--workload-label`, `--subtype-label`) |
| Watch output | Aggregated by configured label hierarchy |
| SPOT detection | Auto-detect from pod spec (`cloud.google.com/gke-spot=true` or `compute-class=autopilot-spot`) |
| BigQuery partitioning | Day-partitioned, designed for hourly/daily roll-up |

## Package Structure

```
gke-cost-analyzer/
├── main.go
├── cmd/
│   ├── root.go          # root command, global flags
│   ├── watch.go         # `watch` subcommand
│   ├── record.go        # `record` subcommand (BigQuery writer)
│   └── setup.go         # `setup` subcommand (create BQ dataset/tables)
├── internal/
│   ├── kube/
│   │   ├── pods.go      # Pod listing & watching from k8s API
│   │   └── pods_test.go
│   ├── pricing/
│   │   ├── catalog.go   # Cloud Billing Catalog API client
│   │   ├── cache.go     # Price caching in ~/.cache/gke-cost-analyzer/
│   │   ├── types.go     # Price types (region, tier, resource → price)
│   │   └── *_test.go
│   ├── cost/
│   │   ├── calculator.go # Cost calculation: requests × duration × price
│   │   ├── aggregator.go # Aggregate costs by label hierarchy
│   │   └── *_test.go
│   └── bigquery/
│       ├── schema.go    # Table schema definition
│       ├── writer.go    # Write cost snapshots to BigQuery
│       ├── setup.go     # Create dataset & tables
│       └── *_test.go
```

## Phases

### Phase 1: Core — Pricing & Cost Calculation

Build the foundational packages that everything else depends on.

#### 1a. Pricing types & cache (`internal/pricing/`)

- Define `Price` struct: region, resource type (cpu/memory), tier (on-demand/spot), unit price (per hour)
- Implement file-based cache in `~/.cache/gke-cost-analyzer/prices.json`
- Cache includes TTL (e.g. 24 hours) so prices aren't re-fetched every run
- **Tests**: cache read/write/expiry with temp directories

#### 1b. Cloud Billing Catalog API client (`internal/pricing/catalog.go`)

- Fetch Autopilot SKUs from the Cloud Billing Catalog API
  - Service: `services/6F81-5844-456A` (Compute Engine)
  - Filter for Autopilot Pod-level SKUs (CPU and Memory, both on-demand and Spot)
  - Parse tiered pricing from the SKU response
- Extract per-region, per-tier (on-demand vs spot) prices for CPU (per vCPU-hour)
  and memory (per GB-hour)
- Use Google Cloud Go SDK (`cloud.google.com/go/billing`)
- **Tests**: parse real API response fixtures, verify price extraction

#### 1c. Kubernetes pod data (`internal/kube/`)

- List pods in the current cluster (all namespaces or configurable)
- Extract from each pod:
  - Name, namespace, labels
  - CPU and memory requests (sum across containers)
  - Start time (from `status.startTime`)
  - SPOT detection: check `nodeSelector["cloud.google.com/gke-spot"]` or
    `nodeSelector["cloud.google.com/compute-class"] == "autopilot-spot"`
  - Pod phase (Running, Succeeded, Failed)
- Return a `PodInfo` struct with these fields
- Use `client-go` with kubeconfig from default loader
- **Tests**: unit tests with fake/mock k8s clientset

#### 1d. Cost calculator (`internal/cost/calculator.go`)

- Given a `PodInfo` and a price lookup, compute cost:
  - `cpu_cost = cpu_requests_vcpu × duration_hours × cpu_price_per_vcpu_hour`
  - `mem_cost = mem_requests_gb × duration_hours × mem_price_per_gb_hour`
  - `total = cpu_cost + mem_cost`
  - Use SPOT prices when pod is detected as SPOT
- Duration = time since `startTime` for running pods
- **Tests**: verify calculations with known inputs, edge cases (zero requests, just started, etc.)

#### 1e. Cost aggregator (`internal/cost/aggregator.go`)

- Group pod costs by configured label hierarchy: team → workload → subtype
- Produce a tree structure: `map[team]map[workload]map[subtype]AggregatedCost`
- `AggregatedCost`: total cost, cpu cost, memory cost, pod count, total cpu requests, total mem requests
- **Tests**: aggregation with various label combinations, missing labels

### Phase 2: CLI Commands

#### 2a. `watch` command (`cmd/watch.go`)

- Periodically (default 10s, configurable) fetch pods, calculate costs, aggregate, display table
- Table columns: Team | Workload | Subtype | Pods | CPU Req | Mem Req | $/hr | Spot
- Uses a terminal table library (e.g. `tablewriter` or simple formatted output)
- Clear and redraw on each cycle
- Global flags (defined on root): `--team-label`, `--workload-label`, `--subtype-label`, `--namespace` (default all)
- **Tests**: verify table formatting with mock data

#### 2b. `record` command (`cmd/record.go`)

- Run as a daemon, periodically (default 5 min, configurable via `--interval`) snapshot pod costs
- Write aggregated cost records to BigQuery
- Flags: `--project`, `--dataset` (default `autopilot_costs`), `--table` (default `cost_snapshots`), `--interval`
- On each tick:
  1. List all pods, compute costs for the snapshot window
  2. Write one row per (team, workload, subtype) combination
- **Tests**: verify record construction with mock data

#### 2c. `setup` command (`cmd/setup.go`)

- Create BigQuery dataset and table if they don't exist
- Flags: `--project`, `--dataset`, `--table`
- Print confirmation of what was created
- **Tests**: verify setup logic with mock BQ client

### Phase 3: BigQuery Integration

#### 3a. Table schema (`internal/bigquery/schema.go`)

BigQuery table schema for cost snapshots:

| Column | Type | Description |
|---|---|---|
| `timestamp` | TIMESTAMP | Snapshot time |
| `project_id` | STRING | GCP project ID |
| `region` | STRING | Cluster region |
| `cluster_name` | STRING | GKE cluster name |
| `namespace` | STRING | Kubernetes namespace |
| `team` | STRING | Team label value |
| `workload` | STRING | Workload label value |
| `subtype` | STRING | Subtype label value (nullable) |
| `pod_count` | INT64 | Number of pods in this group |
| `cpu_request_vcpu` | FLOAT64 | Total vCPU requests |
| `memory_request_gb` | FLOAT64 | Total memory requests (GB) |
| `cpu_cost` | FLOAT64 | CPU cost for this window ($) |
| `memory_cost` | FLOAT64 | Memory cost for this window ($) |
| `total_cost` | FLOAT64 | Total cost for this window ($) |
| `is_spot` | BOOL | Whether these pods are SPOT |
| `interval_seconds` | INT64 | Snapshot interval in seconds |

- Table partitioned by `timestamp` (DAY granularity)
- Clustered by `team`, `workload` for efficient queries

#### 3b. Writer (`internal/bigquery/writer.go`)

- Batch insert rows using the BigQuery streaming insert API
- Handle errors gracefully (log and continue)
- **Tests**: verify row construction, use mock BQ client

#### 3c. Setup (`internal/bigquery/setup.go`)

- Create dataset (if not exists) in specified project/region
- Create table (if not exists) with the schema above
- **Tests**: mock BQ admin client

### Phase 4: Polish & Documentation

- Update README.md with usage instructions
- Update CHANGELOG.md
- Verify all linting, formatting, tests pass
- Verify `prek` pre-commit hooks pass
- End-to-end manual testing instructions

## Implementation Order

```
Phase 1a (pricing types + cache)
    ↓
Phase 1b (billing catalog API client)
    ↓
Phase 1c (kube pod listing)     Phase 3a (BQ schema)
    ↓                               ↓
Phase 1d (cost calculator)      Phase 3b (BQ writer)
    ↓                               ↓
Phase 1e (cost aggregator)      Phase 3c (BQ setup)
    ↓                               ↓
Phase 2a (watch cmd)            Phase 2b (record cmd)
    ↓                               ↓
Phase 2c (setup cmd)
    ↓
Phase 4 (polish)
```

Phases 1c and 3a can be done in parallel since they're independent.
The kube and BigQuery sides converge at the `record` command.

## Key Dependencies (Go modules)

- `github.com/spf13/cobra` — CLI framework (already in go.mod)
- `k8s.io/client-go` — Kubernetes API client
- `cloud.google.com/go/billing` — Cloud Billing Catalog API
- `cloud.google.com/go/bigquery` — BigQuery client
- `github.com/olekukonez/tablewriter` or similar — terminal table formatting

## Example Queries (BigQuery)

```sql
-- Total cost by team for today
SELECT team, SUM(total_cost) as cost
FROM `project.autopilot_costs.cost_snapshots`
WHERE DATE(timestamp) = CURRENT_DATE()
GROUP BY team;

-- Hourly cost breakdown for a workload
SELECT
  TIMESTAMP_TRUNC(timestamp, HOUR) as hour,
  workload,
  SUM(total_cost) as cost
FROM `project.autopilot_costs.cost_snapshots`
WHERE DATE(timestamp) = CURRENT_DATE()
  AND team = 'my-team'
GROUP BY hour, workload
ORDER BY hour;

-- Daily cost trend for past week
SELECT
  DATE(timestamp) as day,
  team,
  SUM(total_cost) as cost,
  SUM(CASE WHEN is_spot THEN total_cost ELSE 0 END) as spot_cost
FROM `project.autopilot_costs.cost_snapshots`
WHERE timestamp >= TIMESTAMP_SUB(CURRENT_TIMESTAMP(), INTERVAL 7 DAY)
GROUP BY day, team
ORDER BY day;
```
