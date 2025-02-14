// Types in this file are copied from:
// https://github.com/aws/aws-sdk-go/blob/v1.55.6/service/xray/api.go

package sampling

import "time"

type SamplingStatisticsDocument struct {
	_ struct{} `type:"structure"`

	// The number of requests recorded with borrowed reservoir quota.
	BorrowCount *int64 `type:"integer"`

	// A unique identifier for the service in hexadecimal.
	//
	// ClientID is a required field
	ClientID *string `min:"24" type:"string" required:"true"`

	// The number of requests that matched the rule.
	//
	// RequestCount is a required field
	RequestCount *int64 `type:"integer" required:"true"`

	// The name of the sampling rule.
	//
	// RuleName is a required field
	RuleName *string `min:"1" type:"string" required:"true"`

	// The number of requests recorded.
	//
	// SampledCount is a required field
	SampledCount *int64 `type:"integer" required:"true"`

	// The current time.
	//
	// Timestamp is a required field
	Timestamp *time.Time `type:"timestamp" required:"true"`
}

// A sampling rule that services use to decide whether to instrument a request.
// Rule fields can match properties of the service, or properties of a request.
// The service can ignore rules that don't match its properties.
type SamplingRule struct {
	_ struct{} `type:"structure"`

	// Matches attributes derived from the request.
	Attributes map[string]*string `type:"map"`

	// The percentage of matching requests to instrument, after the reservoir is
	// exhausted.
	//
	// FixedRate is a required field
	FixedRate *float64 `type:"double" required:"true"`

	// Matches the HTTP method of a request.
	//
	// HTTPMethod is a required field
	HTTPMethod *string `type:"string" required:"true"`

	// Matches the hostname from a request URL.
	//
	// Host is a required field
	Host *string `type:"string" required:"true"`

	// The priority of the sampling rule.
	//
	// Priority is a required field
	Priority *int64 `min:"1" type:"integer" required:"true"`

	// A fixed number of matching requests to instrument per second, prior to applying
	// the fixed rate. The reservoir is not used directly by services, but applies
	// to all services using the rule collectively.
	//
	// ReservoirSize is a required field
	ReservoirSize *int64 `type:"integer" required:"true"`

	// Matches the ARN of the Amazon Web Services resource on which the service
	// runs.
	//
	// ResourceARN is a required field
	ResourceARN *string `type:"string" required:"true"`

	// The ARN of the sampling rule. Specify a rule by either name or ARN, but not
	// both.
	RuleARN *string `type:"string"`

	// The name of the sampling rule. Specify a rule by either name or ARN, but
	// not both.
	RuleName *string `min:"1" type:"string"`

	// Matches the name that the service uses to identify itself in segments.
	//
	// ServiceName is a required field
	ServiceName *string `type:"string" required:"true"`

	// Matches the origin that the service uses to identify its type in segments.
	//
	// ServiceType is a required field
	ServiceType *string `type:"string" required:"true"`

	// Matches the path from a request URL.
	//
	// URLPath is a required field
	URLPath *string `type:"string" required:"true"`

	// The version of the sampling rule format (1).
	//
	// Version is a required field
	Version *int64 `min:"1" type:"integer" required:"true"`
}

// A SamplingRule (https://docs.aws.amazon.com/xray/latest/api/API_SamplingRule.html)
// and its metadata.
type SamplingRuleRecord struct {
	_ struct{} `type:"structure"`

	// When the rule was created.
	CreatedAt *time.Time `type:"timestamp"`

	// When the rule was last modified.
	ModifiedAt *time.Time `type:"timestamp"`

	// The sampling rule.
	SamplingRule *SamplingRule `type:"structure"`
}

type GetSamplingTargetsOutput struct {
	_ struct{} `type:"structure"`

	// The last time a user changed the sampling rule configuration. If the sampling
	// rule configuration changed since the service last retrieved it, the service
	// should call GetSamplingRules (https://docs.aws.amazon.com/xray/latest/api/API_GetSamplingRules.html)
	// to get the latest version.
	LastRuleModification *time.Time `type:"timestamp"`

	// Updated rules that the service should use to sample requests.
	SamplingTargetDocuments []*SamplingTargetDocument `type:"list"`

	// Information about SamplingStatisticsDocument (https://docs.aws.amazon.com/xray/latest/api/API_SamplingStatisticsDocument.html)
	// that X-Ray could not process.
	UnprocessedStatistics []*UnprocessedStatistics `type:"list"`
}

// Temporary changes to a sampling rule configuration. To meet the global sampling
// target for a rule, X-Ray calculates a new reservoir for each service based
// on the recent sampling results of all services that called GetSamplingTargets
// (https://docs.aws.amazon.com/xray/latest/api/API_GetSamplingTargets.html).
type SamplingTargetDocument struct {
	_ struct{} `type:"structure"`

	// The percentage of matching requests to instrument, after the reservoir is
	// exhausted.
	FixedRate *float64 `type:"double"`

	// The number of seconds for the service to wait before getting sampling targets
	// again.
	Interval *int64 `type:"integer"`

	// The number of requests per second that X-Ray allocated for this service.
	ReservoirQuota *int64 `type:"integer"`

	// When the reservoir quota expires.
	ReservoirQuotaTTL *time.Time `type:"timestamp"`

	// The name of the sampling rule.
	RuleName *string `type:"string"`
}

// Sampling statistics from a call to GetSamplingTargets (https://docs.aws.amazon.com/xray/latest/api/API_GetSamplingTargets.html)
// that X-Ray could not process.
type UnprocessedStatistics struct {
	_ struct{} `type:"structure"`

	// The error code.
	ErrorCode *string `type:"string"`

	// The error message.
	Message *string `type:"string"`

	// The name of the sampling rule.
	RuleName *string `type:"string"`
}

type GetSamplingTargetsInput struct {
	_ struct{} `type:"structure"`

	// Information about rules that the service is using to sample requests.
	//
	// SamplingStatisticsDocuments is a required field
	SamplingStatisticsDocuments []*SamplingStatisticsDocument `type:"list" required:"true"`
}

type GetSamplingRulesInput struct {
	_ struct{} `type:"structure"`

	// Pagination token.
	NextToken *string `type:"string"`
}

type GetSamplingRulesOutput struct {
	_ struct{} `type:"structure"`

	// Pagination token.
	NextToken *string `type:"string"`

	// Rule definitions and metadata.
	SamplingRuleRecords []*SamplingRuleRecord `type:"list"`
}
