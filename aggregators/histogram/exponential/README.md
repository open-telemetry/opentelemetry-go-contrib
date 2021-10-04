# OTEP 149 Base-2 Exponential Histogram prototypes

## Logarithm histogram mapper

The file `logarithm.go` demonstrates an implementation of the base-2
exponential histogram based on the built-in logarithm function.

## LookupTable histogram mapper

The file `lookuptable.go` demonstrates an implementation of the OTEP
base-2 exponential histogram based on a precomputed table of
constants.

### Generate a constants table

The constants table for the LookupTable histogram mapper is not
checked-in. The constants table can be used by any scale of histogram
less than or equal to the maximum scale computed.  To generate a
constants table, run:

```
go run ./generate MAXSCALE
```

For some value of `MAXSCALE`.  Note that the generated table will
contain 2**MAXSCALE entries, where `**` represents exponentiation
(i.e., two to the power of MAXSCALE).  Practical limits start to apply
around `MAXSCALE=16`, where this program takes weeks of CPU time to run.

## Acknowledgements

[Yuke Zhuge](https://github.com/yzhuge) and [Otmar Ertl](https://github.com/oertl) 
are the primary authors of these prototypes.  See
[NrSketch](https://github.com/newrelic-experimental/newrelic-sketch-java/blob/1ce245713603d61ba3a4510f6df930a5479cd3f6/src/main/java/com/newrelic/nrsketch/indexer/LogIndexer.java)
and [DynaHist](https://github.com/dynatrace-oss/dynahist/blob/9a6003fd0f661a9ef9dfcced0b428a01e303805e/src/main/java/com/dynatrace/dynahist/layout/OpenTelemetryExponentialBucketsLayout.java) repositories
for more detail.
