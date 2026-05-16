# Device Selection Service

A multi-criteria optimization service for selecting smart-home devices from a marketplace catalog, given user requirements and a budget. Produces a Pareto-optimal set of configurations trading off device quality, ecosystem complexity, and number of hubs needed.

---

## Problem Statement

Given:
- A **catalog** of devices (each with type, brand, model, price, quality score, and compatibility metadata)
- A **request** specifying main ecosystem, budget, device requirements (type + count + optional filters), and include/exclude ecosystem lists

Find the **non-dominated set of solutions** (a Pareto front) under three objectives:

| Objective | Direction | Definition |
|---|---|---|
| `avg_quality` | maximize | mean of per-requirement quality averages plus hub device qualities |
| `num_ecosystems` | minimize | distinct smart-home ecosystems used across all solution devices |
| `num_hubs` | minimize | number of physical hub devices in the solution |

`total_cost` is a **hard constraint**, not an objective. Solutions over budget are infeasible; solutions with identical (quality, ecos, hubs) tuples are deduplicated keeping the first.

Each device contributes via a `ConnectionPlan` describing how it reaches the main ecosystem — directly (if `direct_compat` lists the main ecosystem) or via a bridge (cloud or matter, for example). Devices using hub-required protocols (zigbee, matter) require a hub in their ecosystems

---

## Solution Approach

Four algorithms are implemented:

| Algorithm | Purpose | Complexity |
|---|---|---|
| **enum_repair** | Production heuristic | O(2^E_b · R · N) where E_b = bridge ecosystems, R = requirements, N = devices |
| **brute_force** | Exact ground truth (small instances) | O(N^(C+E_b)) where C = total devices placed |
| **greedy_cheapest** | Baseline: lowest-price device per requirement | O(2^E_b · R · N) |
| **greedy_quality** | Baseline: highest-quality device per requirement | O(2^E_b · R · N) |

### enum_repair (heuristic)

For each subset of (hub-types, bridge-ecosystems), picks the best device per requirement that fits the chosen routing topology, then maintains a Pareto archive of non-dominated solutions across all subsets. Limitation: picks one (device model × count) per requirement — homogeneous within requirement. Misses heterogeneous-within-requirement optima where, for example, 1× mid + 1× premium gives strictly higher average quality than 2× mid when 2× premium overshoots budget.

### brute_force

Enumerates `combinations_with_replacement` of devices per requirement, takes Cartesian product across requirements, evaluates every (device-combo, hub-set) pair. Exact but exponential. Used as ground truth on small instances; infeasible on the real catalog.

### greedy_cheapest / greedy_quality

Same outer structure as `enum_repair` (enumerate 2^E_b ecosystem subsets) but without hub-type enumeration. Within each subset, picks one device per requirement using a selector (min-price or max-quality), then picks the matching cheapest/best hub for ecosystems that need one. Produces a Pareto front via the ecosystem-subset sweep. Expected to be strictly dominated by `enum_repair` on any test with a non-trivial Pareto front.

---

## Architecture

```
data pipeline (medallion: bronze / silver / gold)   →   device catalog   →   selection service
  extract listings from marketplace APIs                                       (Temporal worker,
  LLM enrichment for compatibility metadata                                    iot_opt.proto contract)
```

Evaluation runs the same selection logic outside the Temporal worker for batch metric collection.

---

## CLI

```bash
# Start the Temporal worker; selection requests come via iot_opt.proto.
device-selection run --config config.toml

# Run all test cases in evaluation/, write per-test JSONs + summary.csv to evaluation/results/
device-selection evaluate --config config.toml
```

---

## Test Catalogs

| Catalog | Devices | Ecosystems | Purpose |
|---|---|---|---|
| `catalog_synth_001` | 5 | yandex | sanity check, wifi-only |
| `catalog_synth_002` | 10 | yandex | hub vs no-hub trade-off (wifi vs zigbee) |
| `catalog_synth_003` | 12 | yandex, aqara | cloud bridge test |
| `catalog_synth_004` | 15 | yandex | tight-budget mix |
| `catalog_synth_005` | 4 | yandex | minimal catalog, constructed heuristic gap |
| `catalog_multieco` | 35 | yandex, aqara, tuya | realistic 3-ecosystem medium catalog |
| `catalog_multihub` | 40 | yandex, aqara, tuya | extends `multieco` with ultra-quality (q≥0.97) devices |
| `catalog_forced_multihub` | 24 | yandex, aqara, tuya | sensors aqara-only, plugs tuya-only — forces H≥2 |
| `catalog_natural_5pt` | 23 | yandex, aqara, tuya | designed for 5-point natural front: yandex best lamps, aqara best sensors, tuya best plugs |
| `catalog_synth_matter_bridge` | 7 | yandex, aqara | matter bridge test, dual matter-hub requirement |
| `real_catalog` | ~700 | several | production catalog extracted from marketplace listings |

---

## Test Cases

| ID | Catalog | Requirements | Budget | BF? | What it tests |
|---|---|---|---|---|---|
| synth_001 | synth_001 | 2× lamp | 12k | ✓ | Sanity baseline (all algos agree) |
| synth_002 | synth_002 | 2L + 2S | 5500 | ✓ | Hub vs no-hub Pareto trade-off |
| synth_003 | synth_003 | 2L + 2S | 4500 | ✓ | Cloud bridge to non-main ecosystem |
| synth_004 | synth_004 | 2L + 2S + 2P | 2500 | ✓ | Tight budget, small feasible region |
| synth_005 | synth_005 | 2L + 2S | 4000 | ✓ | **Constructed heuristic gap** (heterogeneous-within-req) |
| synth_006 | multieco | 2L + 2S + 2P | 12k | ✓ | Multi-ecosystem medium, BF timing baseline |
| synth_007 | multieco | 3L + 3S + 3P | 18k | ✗ | Heuristic stability at count=3 (BF intractable) |
| synth_008 | multieco | 2L + 2S + 2P, include yandex+aqara | 10k | ✓ | `include_ecosystems` filter behaviour |
| synth_009 | multieco | 2L + 2S | 2900 | ✓ | **Natural heuristic gap** under tight budget |
| synth_010 | multihub | 2L + 2S + 2P | 15k | ✓ | Multi-hub catalog timing test |
| synth_010v2 | forced_multihub | 2L + 2S + 2P | 12k | ✓ | Forced multi-hub topology (H≥2 mandatory) |
| synth_011 | multihub | 2L + 2S + 2P | 6000 | ✓ | **BF finds extra point under tight budget** |
| synth_013 | multieco | 5 reqs + brand filters | 25k | ✗ | Many requirements + filter combinations |
| synth_014 | natural_5pt | 2L + 2S + 2P | 15k | ✓ | **5-point natural front** spanning H=0..H=3 |
| synth_015 | matter_bridge | 2 lamps | 7000 | ✓ | Matter bridge requires matter-capable hub on both sides |
| real_001 | real | 3L(E27) + 2M + 3Leak | 10k | ✗ | Starter kit, real catalog |
| real_002 | real | 3L(E27) + 2M + 5Leak + 3DW + 1Lock | 25k | ✗ | Full home (reference scenario) |
| real_003 | real | same as real_002 | 12k | ✗ | Tight budget on real catalog |
| real_004 | real | 3Cam + 3M + 5DW + 1Lock | 20k | ✗ | Security-focused setup |

---

## Results

`#S` = Pareto points found; `HV` = hypervolume (higher = better); `IGD+` = modified inverted generational distance (0 = matches reference, lower = better; ignores reference points dominated by the algorithm's front).

Reference front is `brute_force` when run, otherwise the union of all heuristic+baseline fronts (`best_known_union`).

### Synthetic tests

| Test | enum #S / HV / IGD+ | greedy_cheapest #S / HV / IGD+ | greedy_quality #S / HV / IGD+ | brute_force #S / HV / runtime |
|---|---|---|---|---|
| synth_001 | 1 / 1.064 / 0.000 | 1 / 0.854 / 0.190 | 1 / 1.064 / 0.000 | 1 / 1.064 / <0.01s |
| synth_002 | 2 / 1.048 / 0.000 | 1 / 0.044 / 0.572 | 1 / 0.051 / 0.500 | 2 / 1.048 / <0.01s |
| synth_003 | 2 / 1.063 / **0.013** | 1 / 0.992 / 0.088 | 1 / 0.303 / 0.354 | 2 / 1.083 / 0.006s |
| synth_004 | 2 / 0.939 / **0.013** | 1 / 0.043 / 0.533 | 0 / — / ∞ | 2 / 0.965 / 0.015s |
| synth_005 | 1 / 1.047 / **0.025** | 1 / 0.992 / 0.075 | 0 / — / ∞ | 1 / 1.075 / <0.01s |
| synth_006 | 3 / 1.086 / 0.000 | 1 / 0.677 / 0.173 | 2 / 0.744 / 0.111 | 3 / 1.086 / **27.1s** |
| synth_007 | 3 / 1.086 / 0.000 | 1 / 0.677 / 0.173 | 2 / 0.744 / 0.111 | (skipped) |
| synth_008 | 3 / 1.086 / 0.000 | 1 / 0.677 / 0.173 | 2 / 0.744 / 0.111 | 3 / 1.086 / 5.6s |
| synth_009 | 3 / 0.965 / **0.019** | 1 / 0.342 / 0.342 | 0 / — / ∞ | 3 / 0.978 / 0.094s |
| synth_010 | 3 / 1.096 / 0.000 | 1 / 0.963 / 0.120 | 2 / 0.757 / 0.114 | 3 / 1.096 / **54.7s** |
| synth_010v2 | 1 / 0.019 / 0.000 | 1 / 0.016 / 0.160 | 1 / 0.002 / 0.333 | 1 / 0.019 / 2.8s |
| synth_011 | 2 / 1.081 / **0.001** | 1 / 0.963 / 0.107 | 0 / — / ∞ | 3 / 1.083 / 36.4s |
| synth_013 | 1 / 0.035 / 0.000 | 1 / 0.017 / 0.351 | 1 / 0.035 / 0.000 | (skipped) |
| synth_014 | **5** / 1.089 / 0.000 | 2 / 0.976 / 0.085 | 3 / 0.752 / 0.067 | **5** / 1.089 / 1.0s |
| synth_015 | 3 / 0.940 / 0.000 | 2 / 0.938 / 0.042 | 2 / 0.937 / 0.030 | 3 / 0.940 / <0.01s |

Bold IGD+ values mark tests where `enum_repair` finds a strictly worse front than brute force — the **heuristic gap**.

### Real catalog tests

| Test | enum #S / HV | greedy_cheapest #S / HV / IGD+ | greedy_quality #S / HV |
|---|---|---|---|
| real_001 (starter kit) | 1 / 0.647 | 1 / 0.379 / 0.315 | 0 / — |
| real_002 (full home) | 3 / 0.670 | 1 / 0.339 / 0.358 | 0 / — |
| real_003 (tight budget) | 1 / 0.466 | 1 / 0.339 / 0.149 | 0 / — |
| real_004 (security) | 1 / 0.476 | 1 / 0.347 / 0.151 | 0 / — |

`greedy_quality` returns no solutions on tight-budget tests: it picks the most expensive device per requirement first, exceeding the budget before all requirements are filled. This is expected baseline behaviour.

---

## Findings

**Heuristic optimality.** On 8 of the 13 brute-force-comparable tests `enum_repair` reproduces the brute-force Pareto front exactly. On 5 tests (synth_003, synth_004, synth_005, synth_009, synth_011) it shows the heuristic gap, with IGD+ between 0.001 and 0.025 — small in absolute terms. Every observed gap stems from the homogeneous-within-requirement limitation, which the brute-force solver can exploit by mixing device models within a single requirement.

**Greedy baselines.** Both greedy variants are strictly dominated by `enum_repair` on every test with more than one Pareto point. Where the optimum collapses to a single configuration (synth_001, synth_013), `greedy_quality` coincidentally finds it because the best-quality pick matches the global optimum when there is no trade-off. `greedy_quality` returns zero solutions on all tight-budget tests (synth_004, synth_005, synth_009, synth_011, real_001–real_004): picking the most expensive device per requirement exceeds the budget before completion.

**Runtime.** `enum_repair` completes in under 20 ms on every test, including real-catalog instances (~700 devices). Brute force is feasible only on small synthetic instances — 27s at the medium scale (synth_006), 55s at the multi-hub scale (synth_010), and 111s on the dropped count=3 variant. The heuristic-vs-brute-force runtime ratio at synth_006 is roughly 18,000×.

**Real catalog.** The heuristic produces 1–3 Pareto points per real-catalog request in around 10 ms. Quality values are lower than synthetic-catalog results (0.47–0.74 vs 0.91–0.95) because the real catalog's quality distribution is wider and many devices score below 0.5 after the Bayesian-rating floor (`quality = max(0, bayesian_rating − 4.0)`).
