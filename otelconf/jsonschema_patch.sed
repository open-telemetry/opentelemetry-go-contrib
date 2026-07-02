# go-jsonschema always generates patternProperties as
# map[string]interface{}, for more specific types, they must
# be replaced here
s+type Headers.*+type Headers map[string]string+g
/^type ExperimentalResourceDetector struct {/a\
\	// Enable the GCP resource detector.\
\	// If omitted, ignore.\
\	//\
\	GCP ExperimentalGCPResourceDetector `json:"gcp,omitempty,omitzero" yaml:"gcp,omitempty" mapstructure:"gcp,omitempty"`\

/^type ExperimentalServiceResourceDetector map\[string\]interface{}$/i\
type ExperimentalGCPResourceDetector map[string]interface{}\
