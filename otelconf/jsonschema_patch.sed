# go-jsonschema always generates patternProperties as
# map[string]interface{}, for more specific types, they must
# be replaced here
s+type Headers.*+type Headers map[string]string+g

# go-jsonschema emits `AdditionalProperties interface{}` for the
# `,remain` field, but mapstructure's `,remain` decoder only accepts
# map types and hard-fails on a bare interface{}. Rewrite to
# map[string]any, and hide the field from json/yaml so round-tripping
# still only emits the named keys (#8842).
s+AdditionalProperties interface{} `mapstructure:",remain"`+AdditionalProperties map[string]any `mapstructure:",remain" json:"-" yaml:"-"`+g
/^type ExperimentalResourceDetector struct {/a\
\	// Enable the AWS EC2 resource detector.\
\	// If omitted, ignore.\
\	//\
\	AWSEC2 ExperimentalAWSEC2ResourceDetector `json:"aws.ec2,omitempty,omitzero" yaml:"aws.ec2,omitempty" mapstructure:"aws.ec2,omitempty"`\
\
\	// Enable the GCP resource detector.\
\	// If omitted, ignore.\
\	//\
\	GCP ExperimentalGCPResourceDetector `json:"gcp,omitempty,omitzero" yaml:"gcp,omitempty" mapstructure:"gcp,omitempty"`\
\
\	// Enable the AWS ECS resource detector.\
\	// If omitted, ignore.\
\	//\
\	AWSECS ExperimentalAWSECSResourceDetector `json:"aws.ecs,omitempty,omitzero" yaml:"aws.ecs,omitempty" mapstructure:"aws.ecs,omitempty"`\
\
\	// Enable the AWS EKS resource detector.\
\	// If omitted, ignore.\
\	//\
\	AWSEKS ExperimentalAWSEKSResourceDetector `json:"aws.eks,omitempty,omitzero" yaml:"aws.eks,omitempty" mapstructure:"aws.eks,omitempty"`\
\
\	// Enable the Azure VM resource detector.\
\	// If omitted, ignore.\
\	//\
\	AzureVM ExperimentalAzureVMResourceDetector `json:"azure.vm,omitempty,omitzero" yaml:"azure.vm,omitempty" mapstructure:"azure.vm,omitempty"`\

/^type ExperimentalServiceResourceDetector map\[string\]interface{}$/i\
type ExperimentalAWSEC2ResourceDetector map[string]interface{}\
\
type ExperimentalGCPResourceDetector map[string]interface{}\
\
type ExperimentalAWSECSResourceDetector map[string]interface{}\
\
type ExperimentalAWSEKSResourceDetector map[string]interface{}\
\
type ExperimentalAzureVMResourceDetector map[string]interface{}
