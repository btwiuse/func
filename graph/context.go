package graph

// DecodeContext is the context to use when decoding a raw configuration to a
// graph.
//
// The context provides a way to map static strings to Resource
// implementations.
type DecodeContext struct {
	Resources map[string]Resource
}
