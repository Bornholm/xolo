package model

import (
	"time"

	"github.com/rs/xid"
)

type UsageRecordID string

func NewUsageRecordID() UsageRecordID {
	return UsageRecordID(xid.New().String())
}

// UsageRecord captures one proxy call with cost frozen at recording time.
type UsageRecord interface {
	WithID[UsageRecordID]

	UserID() UserID
	OrgID() OrgID
	ProviderID() ProviderID
	ModelID() LLMModelID
	ProxyModelName() string
	AuthTokenID() string // empty = web session call
	PromptTokens() int
	CompletionTokens() int
	TotalTokens() int
	Cost() int64      // microcents, frozen at recording time
	Currency() string // currency code, e.g. USD, EUR — frozen from provider at recording time
	// ResolvedModelName is the actual model used when a virtual model was resolved.
	// Empty if the requested model was not a virtual model.
	ResolvedModelName() string
	CreatedAt() time.Time
}

type BaseUsageRecord struct {
	id                UsageRecordID
	userID            UserID
	orgID             OrgID
	providerID        ProviderID
	modelID           LLMModelID
	proxyModelName    string
	resolvedModelName string
	authTokenID       string
	promptTokens      int
	completionTokens  int
	totalTokens       int
	cost              int64
	currency          string
	createdAt         time.Time
}

func (r *BaseUsageRecord) ID() UsageRecordID         { return r.id }
func (r *BaseUsageRecord) UserID() UserID            { return r.userID }
func (r *BaseUsageRecord) OrgID() OrgID              { return r.orgID }
func (r *BaseUsageRecord) ProviderID() ProviderID    { return r.providerID }
func (r *BaseUsageRecord) ModelID() LLMModelID       { return r.modelID }
func (r *BaseUsageRecord) ProxyModelName() string    { return r.proxyModelName }
func (r *BaseUsageRecord) AuthTokenID() string       { return r.authTokenID }
func (r *BaseUsageRecord) PromptTokens() int         { return r.promptTokens }
func (r *BaseUsageRecord) CompletionTokens() int     { return r.completionTokens }
func (r *BaseUsageRecord) TotalTokens() int          { return r.totalTokens }
func (r *BaseUsageRecord) Cost() int64               { return r.cost }
func (r *BaseUsageRecord) Currency() string          { return r.currency }
func (r *BaseUsageRecord) ResolvedModelName() string { return r.resolvedModelName }
func (r *BaseUsageRecord) CreatedAt() time.Time      { return r.createdAt }

func (r *BaseUsageRecord) SetResolvedModelName(v string) { r.resolvedModelName = v }

var _ UsageRecord = &BaseUsageRecord{}

func NewUsageRecord(
	userID UserID, orgID OrgID, providerID ProviderID, modelID LLMModelID,
	proxyModelName, authTokenID string,
	promptTokens, completionTokens int,
	cost int64,
	currency string,
	resolvedModelName string,
) *BaseUsageRecord {
	total := promptTokens + completionTokens
	return &BaseUsageRecord{
		id:                NewUsageRecordID(),
		userID:            userID,
		orgID:             orgID,
		providerID:        providerID,
		modelID:           modelID,
		proxyModelName:    proxyModelName,
		resolvedModelName: resolvedModelName,
		authTokenID:       authTokenID,
		promptTokens:      promptTokens,
		completionTokens:  completionTokens,
		totalTokens:       total,
		cost:              cost,
		currency:          currency,
		createdAt:         time.Now(),
	}
}
