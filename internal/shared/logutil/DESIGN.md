# Log value conversion design

The shared log value converter turns arbitrary Go values into OpenTelemetry
attribute values. It is implemented in `convert.go.tmpl` and generated into the
`otellogr`, `otellogrus`, `otelslog`, and `otelzap` bridges.

This document records the invariants behind its cycle and bounded-traversal
handling. Preserve these invariants when changing the converter or its generated
copies.

## Goals

- Terminate when maps, slices, arrays, pointers, interfaces, or values traversed
  by `fmt` contain cycles, even when a repeated identity is extremely deep.
- Preserve as much of the original value as possible by replacing a recursive
  edge with `"<cycle>"`, an over-depth edge with `"<depth-limit>"`, and an edge
  that would exceed the total expansion budget with `"<work-limit>"`.
- Preserve the existing output for acyclic inputs within the traversal depth
  and work bounds, including shared directed acyclic graphs and values with
  formatting methods.
- Avoid new heap allocations on scalar and direct conversion paths, and keep
  the reflection allocations required by bounded formatting scans minimal.
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

## Traversal safety bounds

Identity tracking alone does not bound stack use: a cycle can contain many
distinct containers before repeating, and arrays and interfaces add recursive
edges without adding identities. Path-local tracking and a depth bound also do
not bound total work: sibling edges can repeatedly expand the same shared suffix
of a compact branching graph. Normal conversion and the formatting preflight
therefore enforce two package-internal bounds:

- `maxTraversalDepth` stops a recursive path after 1,000 value edges.
- `maxTraversalWork` stops a top-level conversion after 10,000 units covering
  non-empty aggregates, their child edges, variable-size output and lookup
  payloads, and large by-value reflection copies.

These are implementation-safety bounds, not user-configurable or public
attribute-value limits. They are variables only so package tests can exercise
the boundaries with small inputs; production code does not modify them. The
larger work bound leaves room for broad ordinary values while capping the
modeled output, copy, lookup, and traversal amplification of a compact shared
graph. Reserving child edges before allocating result storage prevents a single
wide or zero-sized-element aggregate from bypassing the bound.

Normal conversion counts active recursive conversions separately from active
container identities. Scalar fast paths and nil and empty containers continue
converting normally at the depth boundary.
Pointer-only chains are iterative and do not consume recursive depth. Hops in a
multi-hop chain or a chain ending in an aggregate consume work; a single direct
pointer-to-terminal fast path remains uncharged. An edge that would enter
another potentially recursive container at the depth bound becomes
`"<depth-limit>"`.

Before expanding any non-empty map, slice, or array, normal conversion reserves
one unit for the aggregate and one for each direct output slot. The reservation
happens before allocating that result's storage, including on the optimized
statically non-recursive paths. The counter is monotonic for the entire
top-level conversion: popping an active identity does not refund work, and
nested struct or map-key formatting uses the same counter. Consequently,
neither a wide child nor a shared suffix revisited by siblings can amplify
output without consuming the root-owned budget. An aggregate that cannot
reserve all its slots becomes `"<work-limit>"` as a unit.

Scalar, nil, empty, and already materialized terminal values still convert after
the work budget is exhausted. Statically non-recursive containers retain their
direct loops and avoid the identity tracker, but their output slots and nested
formatting participate in the shared budget.

Reflection sometimes has to copy a by-value array or struct before traversal can
inspect it. The converter reserves one work unit per machine word before such a
copy, including map key snapshots, map value lookups, addressable array
snapshots, and reflected structs passed to `fmt`. This prevents repeated aliases
of one compact map or pointer from multiplying a large fixed-size copy. Map keys
are still snapshotted before formatting begins, preserving the previous behavior
when a formatting method mutates the source map. Addressable arrays are copied
only after the reservation succeeds, preserving array value semantics without
copying a value that is already known to exceed the bound.

Other variable-size operations use the same machine-word approximation. Nested
`[]byte` values reserve their cloned payload, and formatting reserves ordinary
string payloads, struct field names emitted by `%+v`, diagnostic type names, and
direct string keys of multi-entry maps retained for final output sorting.
Standalone terminal roots remain unbudgeted because their one-time cost is
proportional to the supplied value rather than multiplied by graph traversal.

Normal map conversion snapshots the key set before formatting, as it did before
cycle detection, and then uses `MapIndex` to retrieve each snapshotted value.
Before that lookup it accounts for hashing the key's raw comparable
representation. Regular-memory arrays and structs reserve their flat byte size;
non-regular arrays, structs, interfaces, and strings use a bounded value walk,
including dynamic values stored in interface keys. Formatting methods do not
short-circuit this accounting because Go hashes the raw key before invoking any
method for its textual representation. If raw hashing reaches a traversal bound,
the lookup is skipped and its value is replaced by the corresponding marker.

The formatting preflight's depth count includes every value edge that `fmt`
would follow, including array and interface edges. Before scanning an
aggregate, it reserves one work unit for the aggregate and its direct child
edges; maps reserve both key and value edges. A shallow runtime-shape check
selects an identity-free reservation walk when the immediate child kinds cannot
revisit an aggregate. Other values use the identity-aware value walk. Neither
path recursively inspects static types that the runtime value does not reach,
so an empty or nil branch with an unusually deep element type cannot multiply
preflight work when it is shared.

Each formatting check shares the top-level conversion's monotonic work counter
and gets a fresh identity tracker when its runtime shape requires one. If a
recursively traversable aggregate remains at the depth bound, or an aggregate
cannot reserve its work, the
formatted struct or map key is replaced as a unit and `fmt` is not called on the
unsafe graph. Formatting methods remain trusted terminals, but any by-value
reflection materialization needed before dispatching them must first fit the
shared work budget. The method body, receiver ABI work, and returned output are
trusted user code and are not budgeted. Scalars and nil or empty values still
complete normally.

Classifying whether a raw map key uses Go's flat regular-memory hash is the only
recursive type-only analysis. Every type inspection consumes the same
root-owned work budget as value traversal. A separate 1,000-step guard limits
one classification attempt; reaching it falls back to the bounded structural
hash walk rather than producing a marker by itself. This keeps both a single
deep type and repeated aliases of a wide dynamic key type from escaping the
top-level work bound.

The distinct markers preserve the reason for replacement: `"<cycle>"` means an
identity repeated on the active path, `"<depth-limit>"` means the recursive-path
bound was exhausted, and `"<work-limit>"` means the traversal or reflection-copy
work budget was exhausted before the value was proven safe. Exceptionally deep,
broad, or repeatedly shared acyclic input can therefore be truncated. This
trade-off keeps stack use and graph amplification bounded while leaving ordinary
values unchanged.

An identity already active exactly at either boundary is still reported as
`"<cycle>"`, because recognizing that repetition requires no further descent.
When both non-cycle bounds are known to apply to a value edge, depth takes
precedence over work. Work needed before an edge exists—for example, to snapshot
a map's keys or materialize and hash a key for lookup—can instead exhaust first.

The preflight stops at the first cycle or exhausted edge. Its monotonic work
counter prevents safe sibling branches before that edge from causing exponential
preflight work. For maps that contain more than one condition, the diagnostic
marker can follow map iteration order; every unsafe result prevents the call to
`fmt`. A cycle hidden past an exhausted edge remains unproven and uses the
applicable limit marker.

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
formatting methods. User method execution is deliberately a trust boundary: it
can perform arbitrary work or return arbitrarily large output, so the converter
only accounts for its own reflection materialization before method dispatch.

Before formatting individual keys, `fmt` sorts multi-entry maps by comparing
their raw keys. An exceptionally deep struct, array, or interface key can
therefore recurse inside `internal/fmtsort` beyond the preflight's method
boundary even when the key's formatting method would be terminal. This is a
known limitation. Mirroring the private comparator would couple the converter
to implementation details that can change between supported Go versions, so
this preflight does not attempt to bound that raw-key sort.

The public reflection API also exposes a map's logical length but not its
retained bucket capacity. Iterating a very sparse map can therefore scan more
buckets than its reserved logical-entry work implies. The work bound limits how
many times such a shared map can be expanded, but cannot strictly bound the cost
of each runtime bucket scan without depending on private runtime layout.

For maps, slices, and arrays, only the repeated or over-limit edge is replaced,
using the corresponding marker. When a cycle, depth exhaustion, or work
exhaustion is found inside a value formatted by `fmt`, the formatted value is
replaced as a unit because the converter cannot resume partway through `fmt`
while preserving its output rules.

## Hot-path strategy

The exact-type switch for common scalar values remains directly in every
conversion entry point. Keeping it there is intentional: moving it into a
non-inlined helper caused measurable regressions.

Non-empty maps, slices, and arrays whose static element type cannot recurse use
direct conversion loops and do not construct the general conversion tracker;
they only maintain the small work counter. Non-string map keys retain a fresh
formatting preflight, with identity tracking when needed, while sharing that
top-level counter. Potentially
recursive roots enter a separate stack frame, so the inline identity array is
not charged to scalar or statically non-recursive paths. The root-owned work
state does not escape. Paired benchmarks must continue to check both time and
allocation counts when this structure changes.

There is no opt-out option. Statically non-recursive conversion paths avoid the
general tracker, and formatting values whose shallow runtime shape cannot cycle
use the lighter work-reservation scan instead of identity tracking. Unusually
deep acyclic formatting paths can cross the tracker's eight-identity inline
capacity and allocate its call-local overflow table; measured common paths keep
their direct-conversion allocation counts. A formatting scan that inspects map
keys through public reflection may allocate a bounded key snapshot. An opt-out
would add configuration to four bridges while restoring an unrecoverable
stack-overflow failure mode.

## Trade-offs and rejected alternatives

### Configurable traversal limits

[PR #9277](https://github.com/open-telemetry/opentelemetry-go-contrib/pull/9277)
explored a user-facing maximum traversal depth. Configurable limits introduce
API, counting, and replacement semantics that should follow the discussion in
[opentelemetry-specification#5186](https://github.com/open-telemetry/opentelemetry-specification/issues/5186).
The package-internal bounds here are last-resort stack- and amplification-safety
guards. They are deliberately not exposed as policy or presented as general
attribute-value size limits.

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
identity or exhausts a traversal bound, using the marker for that reason. The
exception is a formatted value, which is replaced as a unit because preserving
`%+v` while safely resuming inside `fmt` would require reimplementing its output
rules.
