#!/usr/bin/env bash
# Phase 5 Matching Engine V2 Benchmark Suite
# Runs benchmarks and produces a comparison report.
set -euo pipefail

cd "$(dirname "$0")/../backend"

echo "========================================================"
echo "  FunnyOption Matching Engine V2 — Phase 5 Benchmark"
echo "========================================================"
echo ""
echo "Running benchmarks (3 iterations, with memory)..."
echo ""

RESULTS=$(go test -bench=. -benchmem -count=3 ./internal/matching/engine/ 2>&1)

echo "$RESULTS"
echo ""
echo "========================================================"
echo "  Summary"
echo "========================================================"
echo ""

parse_bench() {
    local name=$1
    local lines
    lines=$(echo "$RESULTS" | grep "^$name" || true)
    if [ -z "$lines" ]; then
        echo "  $name: (not found)"
        return
    fi
    local avg_ns
    avg_ns=$(echo "$lines" | awk '{sum+=$3; n++} END {printf "%.0f", sum/n}')
    local avg_alloc
    avg_alloc=$(echo "$lines" | awk '{sum+=$5; n++} END {printf "%.0f", sum/n}')
    local avg_bytes
    avg_bytes=$(echo "$lines" | awk '{sum+=$7; n++} END {printf "%.0f", sum/n}')
    printf "  %-40s %8s ns/op  %4s allocs  %8s B/op\n" "$name" "$avg_ns" "$avg_alloc" "$avg_bytes"
}

parse_bench "BenchmarkPlaceOrder_EmptyBook"
parse_bench "BenchmarkPlaceOrder_DeepBook"
parse_bench "BenchmarkMatch_CrossSpread-"
parse_bench "BenchmarkDeterministicTradeID"
parse_bench "BenchmarkMatch_CrossSpread_WithEpoch"
parse_bench "BenchmarkAddOrder_Fresh"

echo ""
echo "========================================================"
echo "  Phase 5 Overhead Analysis"
echo "========================================================"
echo ""

CS=$(echo "$RESULTS" | grep "^BenchmarkMatch_CrossSpread-" | awk '{sum+=$3; n++} END {printf "%.0f", sum/n}')
CSE=$(echo "$RESULTS" | grep "^BenchmarkMatch_CrossSpread_WithEpoch" | awk '{sum+=$3; n++} END {printf "%.0f", sum/n}')
TID=$(echo "$RESULTS" | grep "^BenchmarkDeterministicTradeID" | awk '{sum+=$3; n++} END {printf "%.0f", sum/n}')

if [ -n "$CS" ] && [ -n "$CSE" ]; then
    OVERHEAD=$((CSE - CS))
    if [ "$CS" -gt 0 ]; then
        PCT=$(echo "scale=1; $OVERHEAD * 100 / $CS" | bc 2>/dev/null || echo "N/A")
    else
        PCT="N/A"
    fi
    echo "  CrossSpread baseline:          ${CS} ns/op"
    echo "  CrossSpread + Epoch/TradeID:   ${CSE} ns/op"
    echo "  Phase 5 overhead per trade:    ${OVERHEAD} ns  (${PCT}%)"
fi

if [ -n "$TID" ]; then
    echo "  DeterministicTradeID alone:    ${TID} ns/op"
fi

echo ""
echo "All benchmarks complete."
