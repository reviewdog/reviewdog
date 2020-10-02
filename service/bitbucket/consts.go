package bitbucket

const (
	annotationTypeCodeSmell     = "CODE_SMELL"
	annotationTypeVulnerability = "VULNERABILITY"
	annotationTypeBug           = "BUG"

	// list possible, but not used for now annotation results
	// annotationResultPassed  = "PASSED"
	// annotationResultFailed  = "FAILED"
	// annotationResultSkipped = "SKIPPED"
	// annotationResultIgnored = "IGNORED"
	// annotationResultPending = "PENDING"

	// list of possible, but not used for now
	// report data types
	// reportDataTypeBool       = "BOOLEAN"
	// reportDataTypeDate       = "DATE"
	// reportDataTypeDuration   = "DURATION"
	// reportDataTypeLink       = "LINK"
	// reportDataTypeNumber     = "NUMBER"
	// reportDataTypePercentage = "PERCENTAGE"
	// reportDataTypeText       = "TEXT"

	annotationSeverityHigh     = "HIGH"
	annotationSeverityMedium   = "MEDIUM"
	annotationSeverityLow      = "LOW"
	annotationSeverityCritical = "CRITICAL"

	reportTypeSecurity = "SECURITY"
	reportTypeCoverage = "COVERAGE"
	reportTypeTest     = "TEST"
	reportTypeBug      = "BUG"

	reportResultPassed  = "PASSED"
	reportResultFailed  = "FAILED"
	reportResultPending = "PENDING"
)
