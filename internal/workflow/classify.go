package workflow

import (
	"fmt"
	"regexp"
	"strings"
)

type Classification struct {
	Decision    string `json:"decision"`
	Profile     string `json:"profile"`
	Reason      string `json:"reason"`
	NextCommand string `json:"next_command,omitempty"`
}

var (
	modelPattern = regexp.MustCompile(`\b(architecture|architect|model|design|tradeoff|trade-off|decision|proposal|plan)\b`)
	loopPattern  = regexp.MustCompile(`\b(large|long-running|multi-step|migration|migrate|cross-cutting|refactor|workflow|harness|subagent|parallel)\b`)
	workPattern  = regexp.MustCompile(`\b(add|build|change|fix|debug|implement|integrate|update|create|remove|replace|test|validate|ship|wire|support)\b`)
	skipPattern  = regexp.MustCompile(`\b(explain|summarize|what is|show|list|read|where is|which file)\b`)
)

func Classify(request string) Classification {
	return ClassifyWithCommand(request, "kkt")
}

func ClassifyWithCommand(request, command string) Classification {
	normalized := strings.ToLower(strings.TrimSpace(request))
	command = strings.TrimSpace(command)
	if command == "" {
		command = "kkt"
	}
	profile := "daily"
	if loopPattern.MatchString(normalized) {
		profile = "loop"
	}
	if modelPattern.MatchString(normalized) && !workPattern.MatchString(normalized) {
		profile = "model"
	}

	decision := "invoke"
	reason := "request appears to involve non-trivial coding or planning work"
	if skipPattern.MatchString(normalized) && !workPattern.MatchString(normalized) && !loopPattern.MatchString(normalized) {
		decision = "skip"
		reason = "request appears informational and does not need KKT workflow state"
	}

	result := Classification{
		Decision: decision,
		Profile:  profile,
		Reason:   reason,
	}
	if decision == "invoke" {
		result.NextCommand = fmt.Sprintf("%s start --profile %s %q", command, profile, request)
	}
	return result
}
