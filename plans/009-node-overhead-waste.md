# Node Overhead Waste for Standard GKE Workloads

## Goal
For standard workloads, track unallocated node capacity as "Waste". When a node
is not fully allocated (total pod requests < node capacity), the excess cost
is already distributed proportionally among pods. We now label that overhead
portion separately and count it toward wasted cost.

## Calculation
For each resource (CPU, memory) on a node:
- `allocation_ratio = min(total_requests / node_capacity, 1.0)`
- `overhead_fraction = 1 - allocation_ratio`
- Per pod: `overhead_cost = pod_proportional_cost × overhead_fraction`

Equivalently: `overhead = proportional_cost - (pod_request / node_capacity × node_cost)`

When the node is overcommitted (`total_requests > node_capacity`), overhead is 0.

## Changes

### 1. PodCost struct (`internal/cost/calculator.go`)
- Add `OverheadCostPerHour float64` — node overhead portion of this pod's hourly cost

### 2. StandardCalculator (`internal/cost/standard.go`)
- Compute overhead per pod based on allocation ratio

### 3. AggregatedCost (`internal/cost/aggregator.go`)
- Add `NodeOverheadCostPerHour float64`
- Include node overhead in `WastedCostPerHour`

### 4. Aggregator
- Sum `OverheadCostPerHour` from PodCosts into `NodeOverheadCostPerHour`
- Add to `WastedCostPerHour` (total waste = utilization waste + overhead waste)

### 5. BigQuery schema (`internal/bigquery/schema.go`)
- Add `node_overhead_cost` FLOAT64 NULLABLE field to CostSnapshot and TableSchema

### 6. Parquet (`internal/parquet/writer.go`)
- Add `NodeOverheadCost` field to Row, SnapshotToRow, RowToSnapshot

### 7. Record command (`cmd/record.go`)
- Populate `NodeOverheadCost` in `aggregatedToSnapshot`

### 8. TUI
- Node overhead is included in `WastedCostPerHour` automatically (no TUI changes needed beyond what aggregator provides)
- `groupByTeam` already sums `WastedCostPerHour`

### 9. Tests
- Update standard calculator tests to verify overhead calculation
- Add aggregator tests for overhead summation
- Update BigQuery/parquet round-trip tests if any
