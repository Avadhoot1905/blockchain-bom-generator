package graph

import (
	"testing"
)

func TestAddNode(t *testing.T) {
	g := New()

	n := &Node{ID: "A", Type: "contract"}
	if err := g.AddNode(n); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if err := g.AddNode(n); err == nil {
		t.Fatal("expected error for duplicate node")
	}
}

func TestAddEdge(t *testing.T) {
	g := New()
	g.UpsertNode(&Node{ID: "A", Type: "contract"})
	g.UpsertNode(&Node{ID: "B", Type: "library"})

	if err := g.AddEdge(Edge{From: "A", To: "B", Relationship: "imports"}); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if err := g.AddEdge(Edge{From: "A", To: "missing"}); err == nil {
		t.Fatal("expected error for missing target node")
	}
}

func TestTopologicalSort(t *testing.T) {
	g := New()
	for _, id := range []string{"A", "B", "C"} {
		g.UpsertNode(&Node{ID: id, Type: "contract"})
	}
	// A → B → C
	_ = g.AddEdge(Edge{From: "A", To: "B", Relationship: "imports"})
	_ = g.AddEdge(Edge{From: "B", To: "C", Relationship: "imports"})

	sorted, err := g.TopologicalSort()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(sorted) != 3 {
		t.Fatalf("expected 3 nodes, got %d", len(sorted))
	}
	// Topological sort puts sources first: A → B → C means A first, C last.
	pos := make(map[string]int, 3)
	for i, n := range sorted {
		pos[n.ID] = i
	}
	if pos["A"] >= pos["B"] || pos["B"] >= pos["C"] {
		t.Errorf("unexpected order: A=%d B=%d C=%d (want A < B < C)", pos["A"], pos["B"], pos["C"])
	}
}

func TestTopologicalSortCycle(t *testing.T) {
	g := New()
	g.UpsertNode(&Node{ID: "X"})
	g.UpsertNode(&Node{ID: "Y"})
	_ = g.AddEdge(Edge{From: "X", To: "Y"})
	_ = g.AddEdge(Edge{From: "Y", To: "X"})

	if _, err := g.TopologicalSort(); err == nil {
		t.Fatal("expected cycle error")
	}
}

func TestDependenciesOf(t *testing.T) {
	g := New()
	g.UpsertNode(&Node{ID: "Token"})
	g.UpsertNode(&Node{ID: "ERC20"})
	g.UpsertNode(&Node{ID: "Ownable"})
	_ = g.AddEdge(Edge{From: "Token", To: "ERC20", Relationship: "inherits"})
	_ = g.AddEdge(Edge{From: "Token", To: "Ownable", Relationship: "inherits"})

	deps := g.DependenciesOf("Token")
	if len(deps) != 2 {
		t.Fatalf("expected 2 deps, got %d", len(deps))
	}
}

func TestDependentsOf(t *testing.T) {
	g := New()
	g.UpsertNode(&Node{ID: "ERC20"})
	g.UpsertNode(&Node{ID: "TokenA"})
	g.UpsertNode(&Node{ID: "TokenB"})
	_ = g.AddEdge(Edge{From: "TokenA", To: "ERC20", Relationship: "inherits"})
	_ = g.AddEdge(Edge{From: "TokenB", To: "ERC20", Relationship: "inherits"})

	deps := g.DependentsOf("ERC20")
	if len(deps) != 2 {
		t.Fatalf("expected 2 dependents, got %d", len(deps))
	}
}

func TestFindNode(t *testing.T) {
	g := New()
	g.UpsertNode(&Node{ID: "X", Type: "contract"})

	if n, ok := g.FindNode("X"); !ok || n.Type != "contract" {
		t.Fatal("node not found or wrong type")
	}
	if _, ok := g.FindNode("missing"); ok {
		t.Fatal("expected missing node not found")
	}
}
