# Log Value Conversion Safety

The logging bridges use `convertValue` to translate arbitrary Go values into
OpenTelemetry values. Converter-owned reflection over recursive maps, slices,
arrays, and pointers must not exhaust the goroutine stack, and the exact-value
fast path must remain cheap. The shared implementation therefore uses a small
fail-fast depth bound rather than trying to model an arbitrary Go object
graph.

## Depth contract

`maxConvertDepth` is fixed at 100. Depth is the number of non-terminal values
on the active path that the converter itself descends into:

- a non-empty map, slice, or array counts as one level;
- a non-nil pointer counts as one level;
- a non-nil interface wrapper does not count; and
- a scalar, nil pointer or interface, and empty map, slice, or array does not
  count because conversion does not recurse through it.

The `reflect.Interface` case is unwrapped in the current helper invocation, so
it does not add a recursive stack frame. Interface wrappers also cannot form an
independent recursive chain: any cycle reached through one must contain a map,
slice, or pointer, which is counted normally. The unwrapped concrete value is
charged according to the rules above.

At most 100 counted values may be active. If descending into another counted
value would make the depth 101, conversion fails. A terminal scalar, nil, or
empty collection is still converted when reached at the boundary.

Failure applies to the value of the offending log field as a whole. The
converter immediately unwinds and returns the string value
`"<max-depth-exceeded>"`; it does not retain a partially converted map or
array. No remaining siblings are recursively visited after failure, which
prevents a branching cycle from expanding every branch to the bound. A wide
container can still allocate output capacity based on its length before a
failing child is reached.

## Fast-path invariant

Exact values handled by the type switch, including primitive numbers,
booleans, strings, `time.Duration`, `time.Time`, `[]byte`, `error`, and
`attribute.Value`, return before depth state is created. These cases must keep
their existing conversion and dispatch order, while adding no converter
allocations on the benchmarked exact-value paths. In particular, `error`
handling remains ahead of reflection and pointer unwrapping; the user-provided
`Error` method may itself allocate. Changes to this path require allocation
benchmarks in addition to correctness tests.

The first reflective level remains in `convertValue`, and only descendants use
the status-returning bounded helper. This small amount of source duplication
keeps shallow values from paying an extra non-inlineable helper call. Recursive
calls repeat the scalar switch so their leaves retain the existing direct
conversion path.

The depth state is local to one slow-path conversion. It is not global, pooled,
or shared between log fields or goroutines.

## Trust boundary and residual risks

The bound covers only recursive descent owned by this converter. Existing
calls to `error.Error` and `fmt` retain Go's normal behavior. Implementations of
`fmt.Formatter`, `fmt.Stringer`, and `error`, along with their runtime, side
effects, panics, blocking, allocations, and recursive formatting, are trusted
user code. Struct and non-string map-key formatting is not preflighted. The
[`fmt` documentation](https://github.com/golang/go/blob/8af21751f066eced273ca3ce49506b366847c623/src/fmt/doc.go#L227-L230)
likewise states that it does not protect against every self-referential
formatting pathology.

Unsynchronized concurrent mutation is also outside this contract. In
particular, mutating a map while it is converted is a caller data race. Wide
acyclic collections, large payloads, expensive formatting methods, and
repeated shared acyclic subgraphs may still consume time or memory
proportional to the input. They are separate resource-policy concerns, not
stack-depth concerns.

This design has standard-library precedents for bounded, whole-value failure:
[`log/slog.Value.Resolve`](https://github.com/golang/go/blob/8af21751f066eced273ca3ce49506b366847c623/src/log/slog/value.go#L491-L515)
uses a fixed iteration bound, while
[`encoding/json`](https://github.com/golang/go/blob/8af21751f066eced273ca3ce49506b366847c623/src/encoding/json/encode.go#L201-L204)
rejects a cyclic value instead of returning a partial encoding.

## Relationship to OpenTelemetry attribute limits

This guard is not an implementation of the OpenTelemetry SDK
`AttributeValueDepthLimit`. The
[SDK attribute limit](https://opentelemetry.io/docs/specs/otel/common/#attribute-limits)
defaults to 64, is configurable when implemented, starts an attribute value at
depth 1, increments only through arrays and maps, and replaces each array or
map whose depth exceeds the limit with an empty value while preserving its
ancestors and siblings.

By contrast, the bridge guard runs while converting an arbitrary Go value,
before a finite `attribute.Value` exists and before any SDK limit can apply. It
is hard-coded at 100, counts pointer traversal as well as non-empty arrays and
maps, and replaces the whole offending field value with a diagnostic marker.
They are independent: changing SDK `AttributeValueDepthLimit` does not
configure this converter stack-depth guard.

[AnyValue](https://opentelemetry.io/docs/specs/otel/common/#anyvalue) itself
permits arbitrary array and map depth. The
[mapping from arbitrary data](https://opentelemetry.io/docs/specs/otel/common/attribute-type-mapping/)
is Development-level guidance, and the diagnostic marker is a bridge-local,
lossy fallback rather than standardized OpenTelemetry depth-limit behavior.

## Rejected alternatives

- **Edge-local cycle markers and identity tracking.** Continuing after a
  recursive edge requires path-local map, slice, and pointer identities to
  distinguish cycles from shared acyclic values. It also permits branching
  amplification unless more limits are added. Whole-value failure needs no
  identity table and is consistent with standard-library encoders.
- **Total work, byte, copy, or hashing budgets.** These impose a broader and
  separately observable truncation policy on wide or large acyclic inputs.
  They are not needed to bound the converter's stack.
- **A `sync.Pool` or custom visit table.** The selected design has no
  variable-size traversal state to reuse. Pooling would add lifecycle rules
  and hot-path overhead without solving another requirement.
- **Formatting preflights or detached snapshots.** These duplicate traversal,
  introduce time-of-check/time-of-use and reflection-access problems, and can
  change method dispatch or formatting. They still cannot bound arbitrary
  work inside user methods.
- **A disable option.** Converter-owned recursive traversal should not regain
  its unbounded stack behavior. An option would weaken that guard, add public
  surface to every bridge, and incorrectly conflate it with configurable SDK
  attribute limits.
