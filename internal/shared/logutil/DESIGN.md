# Log value conversion design

The shared log value converter turns arbitrary Go values into OpenTelemetry
attribute values. It is implemented in `convert.go.tmpl` and generated into the
`otellogr`, `otellogrus`, `otelslog`, and `otelzap` bridges.

This document records the invariants behind its cycle and traversal-depth
handling. Preserve these invariants when changing the converter or its generated
copies.

## Goals

- Terminate when maps, slices, arrays, pointers, interfaces, or values traversed
  by `fmt` contain cycles, even when a repeated identity is extremely deep.
- Preserve as much of the original value as possible by replacing a recursive
  edge with `"<cycle>"` and an over-limit edge with `"<depth-limit>"`.
- Preserve the existing output for acyclic inputs within the traversal safety
  bound, including shared directed acyclic graphs and values with formatting
  methods.
- Avoid new heap allocations on common conversion paths.
- Keep behavior consistent across all four bridges generated from the shared
  template.

## Cycle identities and path-local tracking

Conversion tracks identities on the active traversal path, rather than keeping
a global set of every value seen. A map identity is `(type, address)`. A slice
identity is `(type, address, length)`. Including the type and slice length avoids
false positives for overlapping slices and for the same address viewed through
different types. The final pointer before an array is also tracked with a
`(type, address)` identity while that array is converted.

An identity is removed when its branch returns. Consequently, a map or slice
shared by sibling branches is converted normally on each branch instead of
being mistaken for a cycle.

The tracker stores its first eight active identities in an inline array in the
root conversion stack frame. A path with more than eight active identities gets
a call-local, open-addressed overflow table. The table is not pooled, so an
unusually deep input cannot leave a large table retained in process-wide state.
Identities use `reflect.Value.UnsafePointer`; this keeps the referenced objects
visible to the garbage collector and avoids the escape behavior of converting
their addresses to `uintptr`.

Empty maps and slices return before identity tracking. This both preserves their
existing conversion and avoids ambiguous identities for zero-length slices.

## Traversal-depth safety bound

Identity tracking alone does not bound stack use: a cycle can contain many
distinct containers before repeating, and arrays and interfaces add recursive
edges without adding identities. Normal conversion and the formatting preflight
therefore stop recursive traversal at the package-internal `maxTraversalDepth`,
whose production default is 1000. This is an implementation-safety bound, not a
user-configurable or public attribute-value limit.
It is a variable only so package tests can exercise the boundary with small
values; production code does not modify it.

Normal conversion counts active recursive conversions separately from active
container identities. Scalar fast paths and nil, empty, and statically
non-recursive containers continue converting normally at the boundary.
Pointer-only chains are iterative and do not consume one unit per pointer. An
edge that would enter another potentially recursive container at the bound
becomes `"<depth-limit>"`.

The formatting preflight counts every value edge that `fmt` would follow,
including array and interface edges, and its preliminary type walk has the same
bound. If a recursively traversable aggregate remains at the bound, the
preflight reports depth exhaustion, so the formatted struct or map key is
replaced as a unit with `"<depth-limit>"` and `fmt` is not called on the
over-limit graph. Terminal methods, scalars, and nil or empty values still
complete normally.

The distinct markers preserve the reason for replacement: `"<cycle>"` means an
identity repeated on the active path, while `"<depth-limit>"` means the safety
bound was exhausted before the value was proven safe. Exceptionally deep
acyclic input can therefore be truncated. This trade-off keeps stack use bounded
while leaving ordinary and moderately deep values unchanged.

An identity already active exactly at the depth boundary is still reported as
`"<cycle>"`, because recognizing that repetition requires no further descent.

The preflight stops at the first cycle or exhausted edge. This prevents a
depth-limited branch from causing unbounded work across the remaining branches
of a compact shared graph. For maps that contain both conditions, the diagnostic
marker can follow map iteration order; either result prevents the unsafe call to
`fmt`. A cycle hidden past an exhausted edge remains unproven and uses
`"<depth-limit>"`.

## Pointer chains

Pointer-only chains are unwrapped iteratively and use Brent's cycle-detection
algorithm. This provides O(1) auxiliary memory, avoids recursion through the
pointer chain, and does not add one tracker entry per pointer. The final pointer
before an array remains active in the general tracker while the array is
converted, which also detects pointer/array cycles.

## Formatting boundary

Structs and non-string map keys preserve their existing `%+v` representation.
Calling `fmt` directly is unsafe for graphs such as
`map -> struct -> same map`, because the cycle leaves the converter's normal
traversal and recurses inside `fmt`.

Before calling `fmt`, the converter therefore performs a separate safety
preflight that mirrors the relevant `fmt` traversal rules. Nested pointers are
terminal addresses, while implementations of `fmt.Formatter`, `fmt.Stringer`,
and `error` are terminal method calls. Mirroring those rules avoids rejecting
values that `fmt` handles without recursion, including types with promoted
formatting methods.

For maps, slices, and arrays, only the repeated or over-limit edge is replaced,
using the corresponding marker. When a cycle or depth exhaustion is found
inside a value formatted by `fmt`, the formatted value is replaced as a unit
because the converter cannot resume partway through `fmt` while preserving its
output rules.

## Hot-path strategy

The exact-type switch for common scalar values remains directly in every
conversion entry point. Keeping it there is intentional: moving it into a
non-inlined helper caused measurable regressions.

Non-empty maps, slices, and arrays whose static element type cannot recurse use
the original direct conversion loops and do not construct the general
conversion tracker. Non-string map keys retain their independent formatting
preflight. Potentially recursive roots enter a separate stack frame, so the
inline identity array is not charged to scalar or statically non-recursive
paths. Paired benchmarks must continue to check both time and allocation counts
when this structure changes.

There is no opt-out option. Statically non-recursive conversion paths avoid the
general tracker, and the formatting preflight returns early when a type cannot
lead `fmt` into a cycle or reach the safety bound. The mechanism does not add
allocations to the measured hot paths. An opt-out would add configuration to
four bridges while restoring an unrecoverable stack-overflow failure mode.

## Trade-offs and rejected alternatives

### Configurable maximum depth

[PR #9277](https://github.com/open-telemetry/opentelemetry-go-contrib/pull/9277)
explored a user-facing maximum traversal depth. A configurable limit introduces
API, counting, and replacement semantics that should follow the discussion in
[opentelemetry-specification#5186](https://github.com/open-telemetry/opentelemetry-specification/issues/5186).
The package-internal bound here is only a last-resort stack-safety guard. It
defaults to 1000 in production and is deliberately not exposed as policy or
presented as a general attribute-value size limit.

### Global seen set

A global set is simpler, but incorrectly marks a shared map or slice used by
sibling branches as a cycle. Tracking is deliberately path-local.

### Always using a Go map or a pooled map

Always using a map adds an allocation to ordinary recursively typed but acyclic
values. A pool adds an operation on those paths and can retain the largest table
reached by an adversarial input. Feedback on
[PR #9398](https://github.com/open-telemetry/opentelemetry-go-contrib/pull/9398)
motivated the inline tracker with call-local overflow storage.

### Recursive pointer ancestry

Retaining every pointer adds per-level work and storage, and recursive traversal
can itself exhaust the stack. Iterative Brent detection keeps pointer chains
constant-space.

### Explicit iterative aggregate traversal

Explicit traversal stacks for normal conversion and the formatting preflight
would remove their dependence on recursive Go frames and could preserve
arbitrarily deep acyclic input. Normal conversion would also need to retain
partially built map and slice results in those frames. That substantially
increases code and state-management complexity on a hot path. The
package-internal safety bound addresses both stack-overflow cases reported in
[PR #9542's normal conversion review](https://github.com/open-telemetry/opentelemetry-go-contrib/pull/9542#discussion_r3990566698)
and
[formatting preflight review](https://github.com/open-telemetry/opentelemetry-go-contrib/pull/9542#discussion_r3969904141)
without that machinery.

### `encoding/json` fallback

JSON encoding detects cycles, but changes the converter's established textual
representation and its method and formatting behavior.

### Replacing the entire root

Replacing the entire converted root would discard useful non-cyclic data. The
converter replaces only the traversal edge at which it detects a repeated
identity or exhausts the depth bound, using the marker for that reason. The
exception is a formatted value, which is replaced as a unit because preserving
`%+v` while safely resuming inside `fmt` would require reimplementing its output
rules.
