#!/bin/bash

# Run benchmarks for previous commit
git checkout zapbench
for i in {1..10}; do go test -bench=. -benchmem >> old_bench.txt; done

# Run benchmarks for latest commit
git checkout zappool
for i in {1..10}; do go test -bench=. -benchmem >> new_bench.txt; done
