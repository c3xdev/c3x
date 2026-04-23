package engine

type UsageVariableType int

const (
	Int64 UsageVariableType = iota
	String
	Float64
	StringArray
	SubResourceUsage
	KeyValueMap
)

type ConsumptionField struct {
	Key          string
	DefaultValue interface{}
	Value        interface{}
	ValueType    UsageVariableType
	Description  string
}
