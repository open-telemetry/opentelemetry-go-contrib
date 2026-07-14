# go-jsonschema always generates patternProperties as
# map[string]interface{}, for more specific types, they must
# be replaced here
s+type Headers.*+type Headers map[string]string+g
/^type ExperimentalResourceDetector struct {/a\
\	// Enable the GCP resource detector.\
\	// If omitted, ignore.\
\	//\
\	GCP ExperimentalGCPResourceDetector `json:"gcp,omitempty,omitzero" yaml:"gcp,omitempty" mapstructure:"gcp,omitempty"`\
\	// Enable the AWS EKS resource detector.\
\	// If omitted, ignore.\
\	//\
\	AWSEKS ExperimentalAWSEKSResourceDetector `json:"aws.eks,omitempty,omitzero" yaml:"aws.eks,omitempty" mapstructure:"aws.eks,omitempty"`\

/^type ExperimentalServiceResourceDetector map\[string\]interface{}$/i\
type ExperimentalGCPResourceDetector map[string]interface{}\
\
type ExperimentalAWSEKSResourceDetector map[string]interface{}\
