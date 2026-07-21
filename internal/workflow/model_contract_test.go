package workflow

import (
	"bytes"
	"os"
	"strings"
	"testing"
)

func TestOptimizationContractAcceptsCompressedShape(t *testing.T) {
	content := `## Objective Function
Fix the workflow with the smallest safe change.

## Decision Variables and Affected Surfaces
- Decision variables: workflow trigger and version setup.
- Affected surfaces: the publish workflow.

## Constraint Functions
- Hard: preserve the release tag and OIDC publishing.
- Soft: minimize workflow complexity.

## Candidate Feasibility
- Feasible: pin the supported npm version.
- Rejected: retag the existing release.

## Selected Optimum
Pin the supported npm version and add a recovery trigger.

## Binding Constraints
The existing release event has already fired.

## Validation Plan and Certificate
Run actionlint and verify the published registry version.
`
	if issues := optimizationContractIssues(content); len(issues) != 0 {
		t.Fatalf("compressed contract rejected: %v", issues)
	}
}

func TestOptimizationContractAcceptsDeepShape(t *testing.T) {
	content := `## Objective Function
Choose the safest implementation.

## Known Constraints
- Explicit: preserve the public contract.

## Decision Variables
- implementation shape

## Affected Surfaces
- internal workflow package

## Constraint Functions
- Hard: preserve behavior.
- Soft: minimize blast radius.

## Candidate Feasibility
- Feasible: bounded implementation.
- Rejected: broad rewrite.

## Selected Plan
Use the bounded implementation.

## Binding Constraints
The public contract is binding.

## Validation Plan and Proof
Run the test suite.

## Execution Implications
Implement only the selected plan.

## Guardrail Variables
Allowed paths are modeled explicitly.

## Analysis Extensions
None.

## Residual Risk
Validation must pass.
`
	if issues := optimizationContractIssues(content); len(issues) != 0 {
		t.Fatalf("deep contract rejected: %v", issues)
	}
}

func TestOptimizationContractRejectsChecklist(t *testing.T) {
	issues := optimizationContractIssues("Implement the feature. Run tests.")
	if len(issues) != len(optimizationContractFields) {
		t.Fatalf("checklist issues = %v, want one issue per required field", issues)
	}
	if !strings.Contains(strings.Join(issues, "; "), "candidate_feasibility") {
		t.Fatalf("checklist rejection omitted feasibility: %v", issues)
	}
}

func TestRunRejectsChecklistModel(t *testing.T) {
	root := t.TempDir()
	previous, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		if err := os.Chdir(previous); err != nil {
			t.Fatal(err)
		}
	}()
	if err := os.Chdir(root); err != nil {
		t.Fatal(err)
	}
	if err := Run([]string{"start", "model", "reject", "checklist"}, &bytes.Buffer{}, &bytes.Buffer{}); err != nil {
		t.Fatal(err)
	}
	err = Run([]string{"model", "Implement the feature and run tests."}, &bytes.Buffer{}, &bytes.Buffer{})
	if err == nil || !strings.Contains(err.Error(), "constrained-optimization contract") {
		t.Fatalf("checklist model should be rejected by CLI, got %v", err)
	}
}

func TestOptimizationContractRequiresHardAndSoftConstraints(t *testing.T) {
	content := `## Objective Function
Choose the best feasible change.
## Decision Variables and Affected Surfaces
Choose the implementation surface.
## Constraint Functions
- Hard: preserve behavior.
## Candidate Feasibility
- Feasible: bounded change.
## Selected Optimum
Choose the bounded change.
## Binding Constraints
Preserve behavior.
## Validation Plan and Certificate
Run tests.
`
	issues := optimizationContractIssues(content)
	joined := strings.Join(issues, "; ")
	if !strings.Contains(joined, "soft constraints") {
		t.Fatalf("missing soft constraint was not rejected: %v", issues)
	}
}
