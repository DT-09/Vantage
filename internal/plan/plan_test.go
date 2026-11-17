package plan

import "testing"

func TestBuildStableLevels(t *testing.T) {
	w := Workflow{Tasks: []Task{{ID: "c", Kind: "x", Needs: []string{"a", "b"}}, {ID: "b", Kind: "x"}, {ID: "a", Kind: "x"}}}
	p, err := Build(w)
	if err != nil {
		t.Fatal(err)
	}
	if len(p.Levels) != 2 || p.Levels[0][0].ID != "a" || p.Levels[0][1].ID != "b" {
		t.Fatalf("unexpected plan: %#v", p.Levels)
	}
}
func TestRejectCycle(t *testing.T) {
	_, err := Build(Workflow{Tasks: []Task{{ID: "a", Kind: "x", Needs: []string{"b"}}, {ID: "b", Kind: "x", Needs: []string{"a"}}}})
	if err == nil {
		t.Fatal("expected cycle error")
	}
}
func TestRejectUnknownDependency(t *testing.T) {
	_, err := Build(Workflow{Tasks: []Task{{ID: "a", Kind: "x", Needs: []string{"missing"}}}})
	if err == nil {
		t.Fatal("expected unknown dependency error")
	}
}
