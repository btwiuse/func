package graph_test

type mockDef struct {
	Value string
}

func (mockDef) Type() string { return "mock" }
