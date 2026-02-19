package prer

// Exported aliases for testing internal types and
// functions from prer_test package.

// CqueryResult is an alias for cqueryResult.
type CqueryResult = cqueryResult

// ConfiguredTarget is an alias for configuredTarget.
type ConfiguredTarget = configuredTarget

// QueryTarget is an alias for queryTarget.
type QueryTarget = queryTarget

// QueryRule is an alias for queryRule.
type QueryRule = queryRule

// QueryAttribute is an alias for queryAttribute.
type QueryAttribute = queryAttribute

// BuildDepsQueryForTest exposes buildDepsQuery.
var BuildDepsQueryForTest = buildDepsQuery

// GetStampContextForTest exposes getStampContext.
var GetStampContextForTest = getStampContext

// StampFileForTest exposes stampFile.
var StampFileForTest = stampFile

// GroupByTrainForTest exposes groupByTrain.
var GroupByTrainForTest = groupByTrain

// HasDeletedTargetsForTest exposes hasDeletedTargets.
var HasDeletedTargetsForTest = hasDeletedTargets

// CollectAllTargetsForTest exposes collectAllTargets.
var CollectAllTargetsForTest = collectAllTargets

// ExtractTargetNamesForTest exposes
// extractTargetNames.
var ExtractTargetNamesForTest = extractTargetNames
