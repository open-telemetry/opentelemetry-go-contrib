# go-jsonschema always generates patternProperties as
# map[string]interface{}, for more specific types, they must
# be replaced here
s+type Headers.*+type Headers map[string]string+g
/^type ExperimentalResourceDetector struct {/a\
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
type ExperimentalAWSEKSResourceDetector map[string]interface{}\
\
type ExperimentalAzureVMResourceDetector map[string]interface{}
