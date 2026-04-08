#!/bin/sed -f

# Rename package
s+^package x+package otelconf+g

# Remove experimental const definitions
/^const Experimental/d

# Remove single-line experimental type definitions (aliases to basic types)
/^type Experimental.*\*int$/d
/^type Experimental.*\*float64$/d
/^type Experimental.*\*string$/d
/^type Experimental.*\*bool$/d
/^type Experimental.*string$/d
/^type Experimental.*map\[string\]interface{}$/d
/^type Experimental.*map\[string\]map\[string\]interface{}$/d

# Remove array type definitions
/^type Experimental.*\[\]Experimental/d

# Remove multi-line experimental struct type definitions using address range
/^type Experimental.*struct {$/,/^}$/d

# Remove struct fields that reference Experimental types (including Development suffix fields)
/^	.*Development.*\*Experimental/d
/^	[A-Z][A-Za-z0-9_]* \*Experimental/d
/^	[A-Z][A-Za-z0-9_]* Experimental[A-Z]/d
/^	[A-Z][A-Za-z0-9_]* \[\]Experimental/d

# Remove comment lines that reference experimental types
/^[[:space:]]*\/\/ .*Experimental/d
/^[[:space:]]*\/\/ A rule for Experimental/d

# Remove comment blocks before experimental Development fields
# Pattern: multi-line comment block ending with empty comment line, followed by Development field
/^	\/\/ Configure exporter to be OTLP with file transport\.$/,/^	\/\/$/d
/^	\/\/ Configure loggers\.$/,/^	\/\/$/d
/^	\/\/ Configure meters\.$/,/^	\/\/$/d
/^	\/\/ Configure instrumentation\.$/,/^	\/\/$/d
/^	\/\/ Configure exporter to be prometheus\.$/,/^	\/\/$/d
/^	\/\/ Configure resource detection\.$/,/^	\/\/$/d
/^	\/\/ Configure sampler to be composite\.$/,/^	\/\/$/d
/^	\/\/ Configure sampler to be jaeger_remote\.$/,/^	\/\/$/d
/^	\/\/ Configure sampler to be probability\.$/,/^	\/\/$/d
/^	\/\/ Configure tracers\.$/,/^	\/\/$/d

# Remove orphaned top-level comment blocks (from deleted type definitions)
# Probability sampler ratio comments
/^\/\/ Configure ratio\.$/,/^$/d

# Rule-based sampler comments
/^\/\/ match conditions - the sampler will be applied/,/^$/d
/^\/\/ The rules for the sampler, matched in order\./,/^$/d

# Jaeger remote sampler comments
/^\/\/ Configure the polling interval.*to fetch from the remote$/,/^$/d

# Logger config comments
/^\/\/ Configure if the logger is enabled or not\.$/,/^$/d
/^\/\/ Configure trace based filtering\.$/,/^$/d

# OTLP file exporter comments  
/^\/\/ Configure output stream\.$/,/^$/d

# Prometheus exporter comments
/^\/\/ Configure host\.$/,/^$/d
/^\/\/ Configure port\.$/,/^$/d
/^\/\/ Configure Prometheus Exporter to produce metrics without a scope info metric\.$/,/^$/d
/^\/\/ Configure Prometheus Exporter to produce metrics without a target info metric$/,/^$/d
