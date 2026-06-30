package gorm

import (
	"time"

	"github.com/bornholm/xolo/internal/core/model"
)

type UsageRecord struct {
	ID                string `gorm:"primaryKey;autoIncrement:false"`
	CreatedAt         time.Time
	UserID            string `gorm:"index"`
	ApplicationID     string `gorm:"index"`
	OrgID             string `gorm:"index;not null"`
	ProviderID        string `gorm:"index;not null"`
	ModelID           string `gorm:"index;not null"`
	ProxyModelName    string `gorm:"not null"`
	ResolvedModelName string `gorm:""`      // actual model used when virtual model was resolved
	AuthTokenID       string `gorm:"index"` // empty = web session
	PromptTokens      int
	CachedTokens      int
	CompletionTokens  int
	TotalTokens       int
	Cost              int64  // microcents, frozen at recording time (converted to org currency)
	Currency          string // frozen from provider
	CostSource        string // "provider" or "computed", see model.CostSource
	PlanCovered       int    `gorm:"index;default:0"` // 1 if served by a subscription provider
	ProviderCost      int64  // equivalent PAYG cost in provider currency (microcents), for plan value budgets
}

type wrappedUsageRecord struct {
	r *UsageRecord
}

func (w *wrappedUsageRecord) ID() model.UsageRecordID { return model.UsageRecordID(w.r.ID) }
func (w *wrappedUsageRecord) UserID() model.UserID    { return model.UserID(w.r.UserID) }
func (w *wrappedUsageRecord) ApplicationID() model.ApplicationID {
	return model.ApplicationID(w.r.ApplicationID)
}
func (w *wrappedUsageRecord) OrgID() model.OrgID           { return model.OrgID(w.r.OrgID) }
func (w *wrappedUsageRecord) ProviderID() model.ProviderID { return model.ProviderID(w.r.ProviderID) }
func (w *wrappedUsageRecord) ModelID() model.LLMModelID    { return model.LLMModelID(w.r.ModelID) }
func (w *wrappedUsageRecord) ProxyModelName() string       { return w.r.ProxyModelName }
func (w *wrappedUsageRecord) AuthTokenID() string          { return w.r.AuthTokenID }
func (w *wrappedUsageRecord) PromptTokens() int            { return w.r.PromptTokens }
func (w *wrappedUsageRecord) CachedTokens() int            { return w.r.CachedTokens }
func (w *wrappedUsageRecord) CompletionTokens() int        { return w.r.CompletionTokens }
func (w *wrappedUsageRecord) TotalTokens() int             { return w.r.TotalTokens }
func (w *wrappedUsageRecord) Cost() int64                  { return w.r.Cost }
func (w *wrappedUsageRecord) Currency() string             { return w.r.Currency }
func (w *wrappedUsageRecord) CostSource() model.CostSource { return model.CostSource(w.r.CostSource) }
func (w *wrappedUsageRecord) ResolvedModelName() string    { return w.r.ResolvedModelName }
func (w *wrappedUsageRecord) CreatedAt() time.Time         { return w.r.CreatedAt }
func (w *wrappedUsageRecord) PlanCovered() bool            { return w.r.PlanCovered != 0 }
func (w *wrappedUsageRecord) ProviderCost() int64          { return w.r.ProviderCost }

var _ model.UsageRecord = &wrappedUsageRecord{}

func fromUsageRecord(r model.UsageRecord) *UsageRecord {
	return &UsageRecord{
		ID:                string(r.ID()),
		UserID:            string(r.UserID()),
		ApplicationID:     string(r.ApplicationID()),
		OrgID:             string(r.OrgID()),
		ProviderID:        string(r.ProviderID()),
		ModelID:           string(r.ModelID()),
		ProxyModelName:    r.ProxyModelName(),
		ResolvedModelName: r.ResolvedModelName(),
		AuthTokenID:       r.AuthTokenID(),
		PromptTokens:      r.PromptTokens(),
		CachedTokens:      r.CachedTokens(),
		CompletionTokens:  r.CompletionTokens(),
		TotalTokens:       r.TotalTokens(),
		Cost:              r.Cost(),
		Currency:          r.Currency(),
		CostSource:        string(r.CostSource()),
		PlanCovered:       boolToInt(r.PlanCovered()),
		ProviderCost:      r.ProviderCost(),
	}
}
