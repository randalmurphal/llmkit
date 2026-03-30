package codexconfig

import (
	"strings"
	"testing"
)

func TestPrefixRuleRender(t *testing.T) {
	rule := PrefixRule{
		Pattern:       LiteralPattern("gh", "pr", "view"),
		Decision:      RulePrompt,
		Justification: "Viewing PRs is allowed with approval",
		Match:         []string{"gh pr view 123"},
		NotMatch:      []string{"gh pr --repo openai/codex view 123"},
	}

	rendered := rule.Render()
	if !strings.Contains(rendered, `pattern = ["gh", "pr", "view"]`) {
		t.Fatalf("missing pattern in rendered rule:\n%s", rendered)
	}
	if !strings.Contains(rendered, `decision = "prompt"`) {
		t.Fatalf("missing decision in rendered rule:\n%s", rendered)
	}
}

func TestManagedRuleUpsertAndRemove(t *testing.T) {
	file := &RuleFile{}
	UpsertManagedPrefixRule(file, "demo", PrefixRule{
		Pattern:  LiteralPattern("gh", "pr", "view"),
		Decision: RulePrompt,
	})
	if !strings.Contains(file.Content, "# BEGIN llmkit:demo") {
		t.Fatalf("missing managed block markers:\n%s", file.Content)
	}

	UpsertManagedPrefixRule(file, "demo", PrefixRule{
		Pattern:  LiteralPattern("git", "status"),
		Decision: RuleAllow,
	})
	if strings.Count(file.Content, "# BEGIN llmkit:demo") != 1 {
		t.Fatalf("expected managed block replacement, got:\n%s", file.Content)
	}
	if !strings.Contains(file.Content, `pattern = ["git", "status"]`) {
		t.Fatalf("expected updated pattern, got:\n%s", file.Content)
	}

	if !RemoveManagedRule(file, "demo") {
		t.Fatal("expected managed rule removal")
	}
	if strings.Contains(file.Content, "# BEGIN llmkit:demo") {
		t.Fatalf("managed rule still present:\n%s", file.Content)
	}
}
