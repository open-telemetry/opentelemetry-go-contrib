# go-jsonschema always generates patternProperties as
# map[string]interface{}, for more specific types, they must
# be replaced here
s+type Headers.*+type Headers map[string]string+g
/^type ExperimentalResourceDetector struct {/a\
\	// Enable the AWS EC2 resource detector.\
\	// If omitted, ignore.\
\	//\
\	AWSEC2 ExperimentalAWSEC2ResourceDetector `json:"aws.ec2,omitempty,omitzero" yaml:"aws.ec2,omitempty" mapstructure:"aws.ec2,omitempty"`\

/^type ExperimentalServiceResourceDetector map\[string\]interface{}$/i\
type ExperimentalAWSEC2ResourceDetector map[string]interface{}\
