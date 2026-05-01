package rules

import "testing"

func TestListRulesSorted(t *testing.T) {
	list := List()
	if len(list) == 0 {
		t.Fatal("expected rules")
	}
	for i := 1; i < len(list); i++ {
		if list[i-1].ID > list[i].ID {
			t.Fatalf("rules are not sorted: %s > %s", list[i-1].ID, list[i].ID)
		}
	}
}

func TestFindRule(t *testing.T) {
	rule, ok := Find("mcp_secret_header")
	if !ok {
		t.Fatal("expected mcp_secret_header")
	}
	if rule.Title == "" || rule.Recommendation == "" {
		t.Fatalf("expected useful rule metadata: %#v", rule)
	}
}

func TestFindAmbiguousPrefix(t *testing.T) {
	if _, ok := Find("mcp_secret"); ok {
		t.Fatal("expected ambiguous prefix to fail")
	}
}
