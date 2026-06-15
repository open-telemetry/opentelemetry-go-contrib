# go-jsonschema always generates patternProperties as
# map[string]interface{}, for more specific types, they must
# be replaced here
s+type Headers.*+type Headers map[string]string+g
/^type ExperimentalResourceDetector struct {/a\
\	// Enable the AWS ECS resource detector.\
\	// If omitted, ignore.\
\	//\
\	AWSECS ExperimentalAWSECSResourceDetector `json:"aws.ecs,omitempty,omitzero" yaml:"aws.ecs,omitempty" mapstructure:"aws.ecs,omitempty"`\

/^type ExperimentalServiceResourceDetector map\[string\]interface{}$/i\
type ExperimentalAWSECSResourceDetector map[string]interface{}
