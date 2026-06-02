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