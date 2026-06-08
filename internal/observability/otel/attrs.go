package otel

import (
	"go.opentelemetry.io/otel/attribute"
)

// Appendix D §5 attribute keys for ASH run execution spans.
const (
	AttrRunID            = "ash.runId"
	AttrTraceID          = "ash.traceId"
	AttrScenarioVersion  = "ash.scenarioVersion"
	AttrScenarioName     = "ash.scenarioName"
	AttrPolicyProfile    = "ash.policyProfile"
	AttrStepID           = "ash.stepId"
	AttrRole             = "ash.role"
	AttrTool             = "ash.tool"
	AttrProvider         = "ash.provider"
	AttrModel            = "ash.model"
	AttrCheckpointID     = "ash.checkpointId"
	AttrGateID           = "ash.gateId"
	AttrSpaceID          = "ash.spaceId"
)

func runAttrs(runID, traceID, scenarioName, scenarioVersion, policyProfile, spaceID string) []attribute.KeyValue {
	return []attribute.KeyValue{
		attribute.String(AttrRunID, runID),
		attribute.String(AttrTraceID, traceID),
		attribute.String(AttrScenarioName, scenarioName),
		attribute.String(AttrScenarioVersion, scenarioVersion),
		attribute.String(AttrPolicyProfile, policyProfile),
		attribute.String(AttrSpaceID, spaceID),
	}
}
