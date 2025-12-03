# ObjectWeaver Performance Guide

## The Key Formula

```
Max Throughput = Worker Pool Size ÷ (LLM Rounds × LLM Latency)
```

**Default Config**: 10,000 workers

---

## Theoretical Limits by LLM Latency

| LLM Latency | 1-Round Schema | 2-Round Schema | 4-Round Schema |
|-------------|----------------|----------------|----------------|
| **100ms**   | 100,000 req/s  | 50,000 req/s   | 25,000 req/s   |
| **500ms**   | 20,000 req/s   | 10,000 req/s   | 5,000 req/s    |
| **1s**      | 10,000 req/s   | 5,000 req/s    | 2,500 req/s    |
| **2s**      | 5,000 req/s    | 2,500 req/s    | 1,250 req/s    |
| **5s**      | 2,000 req/s    | 1,000 req/s    | 500 req/s      |

---

## Schema Complexity = LLM Rounds

| Schema Type | LLM Rounds | Why |
|-------------|------------|-----|
| **Minimal** | 1 | Single flat object |
| **Simple** | 1 | Flat object with multiple fields |
| **Nested** | 1-2 | Nested objects may add rounds |
| **Complex (with arrays)** | 2+ | Arrays need: 1) size determination + 2) item generation |

---

## Test Results Summary

### With 5s LLM (Realistic Mock)
| Schema | Target | Achieved | Efficiency | Bottleneck |
|--------|--------|----------|------------|------------|
| minimal | 2000 | 726 | 36% | LLM latency (max ~2000) |
| simple | 2000 | 427 | 21% | LLM latency |
| nested | 2000 | 180 | 9% | Multi-round processing |
| complex | 2000 | 94 | 5% | Array sequential rounds |

### With 100ms LLM (Fast Test)
| Schema | Target | Achieved | Efficiency | Notes |
|--------|--------|----------|------------|-------|
| minimal | 5,000 | 4,055 | 81% | Near theoretical max |
| minimal | 10,000 | 10,000 | 100% | ** Full target sustained** |

---

## What This Means

1. **ObjectWeaver is NOT the bottleneck** — it can handle 10,000+ req/s
2. **LLM latency dominates performance** — faster LLM = proportionally higher throughput
3. **Schema complexity matters** — arrays/nested objects add LLM rounds

---

## Quick Reference: Sizing Your Deployment

**To achieve X req/s with your LLM latency:**

```
Required Workers = Target_RPS × LLM_Rounds × LLM_Latency × 1.2 (safety margin)
```

**Examples:**
- 1,000 req/s with 500ms LLM, simple schema: `1000 × 1 × 0.5 × 1.2 = 600 workers`
- 5,000 req/s with 200ms LLM, complex schema: `5000 × 2 × 0.2 × 1.2 = 2,400 workers`
- 10,000 req/s with 100ms LLM, minimal schema: `10000 × 1 × 0.1 × 1.2 = 1,200 workers`

---

## Running Performance Tests

```bash
# Test with realistic 5s LLM latency
make integration-test-matrix-2000

# Test true server capacity (100ms LLM)
make fast-llm-test-5k
make fast-llm-test-10k
```
