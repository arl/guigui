package dock

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestRootSnapshotRoundTrip(t *testing.T) {
	first := Group("first", &Panel{Title: "First"})
	second := Group("second", &Panel{Title: "Second"})
	absent := Group("absent", &Panel{Title: "Absent"})
	root, err := NewRoot(Split(Horizontal, 0.4, first, second), absent)
	if err != nil {
		t.Fatal(err)
	}

	data, err := json.Marshal(root)
	if err != nil {
		t.Fatal(err)
	}
	var snapshot rootSnapshot
	if err := json.Unmarshal(data, &snapshot); err != nil {
		t.Fatal(err)
	}
	if snapshot.Version != 1 || snapshot.Tree == nil || snapshot.Tree.Split == nil {
		t.Fatalf("unexpected snapshot: %#v", snapshot)
	}
	if len(snapshot.Tree.Split.First.Group.Nodes) != 1 || snapshot.Tree.Split.First.Group.Nodes[0] != "first" {
		t.Fatalf("first node was not serialized: %#v", snapshot.Tree.Split.First.Group)
	}
	if _, _, _, err := root.layoutFromSnapshot(&snapshot); err != nil {
		t.Fatalf("round trip validation failed: %v", err)
	}
}

func TestRootSnapshotRejectsUnknownAndDuplicateNodes(t *testing.T) {
	first := Group("first", &Panel{Title: "First"})
	second := Group("second", &Panel{Title: "Second"})
	root, err := NewRoot(Split(Horizontal, 0.4, first, second))
	if err != nil {
		t.Fatal(err)
	}

	unknown := &rootSnapshot{
		Version: 1,
		Tree:    &snapshotNode{Group: &snapshotTabGroup{Nodes: []string{"missing"}, Selected: 0}},
	}
	if _, _, _, err := root.layoutFromSnapshot(unknown); err == nil || !strings.Contains(err.Error(), "unknown node ID") {
		t.Fatalf("unknown node error = %v, want unknown node ID", err)
	}

	duplicate := &rootSnapshot{
		Version: 1,
		Tree:    &snapshotNode{Group: &snapshotTabGroup{Nodes: []string{"first", "first"}, Selected: 0}},
	}
	if _, _, _, err := root.layoutFromSnapshot(duplicate); err == nil || !strings.Contains(err.Error(), "appears more than once") {
		t.Fatalf("duplicate node error = %v, want duplicate node", err)
	}
}

func TestRootApplyJSONIsAtomic(t *testing.T) {
	first := Group("first", &Panel{Title: "First"})
	second := Group("second", &Panel{Title: "Second"})
	root, err := NewRoot(Split(Horizontal, 0.4, first, second))
	if err != nil {
		t.Fatal(err)
	}

	valid, err := json.Marshal(root)
	if err != nil {
		t.Fatal(err)
	}
	if err := root.ApplyJSON(valid); err != nil {
		t.Fatalf("ApplyJSON(valid): %v", err)
	}
	before, err := json.Marshal(root)
	if err != nil {
		t.Fatal(err)
	}
	invalid, err := json.Marshal(rootSnapshot{
		Version: 1,
		Tree:    &snapshotNode{Group: &snapshotTabGroup{Nodes: []string{"missing"}, Selected: 0}},
	})
	if err != nil {
		t.Fatal(err)
	}
	if err := root.ApplyJSON(invalid); err == nil {
		t.Fatal("ApplyJSON(invalid) succeeded")
	}
	after, err := json.Marshal(root)
	if err != nil {
		t.Fatal(err)
	}
	if string(after) != string(before) {
		t.Fatalf("invalid restore mutated layout:\n got %s\nwant %s", after, before)
	}
}
