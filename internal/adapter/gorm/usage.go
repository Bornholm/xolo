package gorm

import (
	"time"

	"github.com/bornholm/xolo/internal/core/model"
)

type UsageRecord struct {
	ID                string `gorm:"primaryKey;autoIncrement:false"`
	CreatedAt         time.Time
	UserID            string `gorm:"index;not null"`
	OrgID             string `gorm:"index;not null"`
	ProviderID        string `gorm:"index;not null"`
	ModelID           string `gorm:"index;not null"`
	ProxyModelName    string `gorm:"not null"`
	ResolvedModelName string `gorm:""`      // actual model used when virtual model was resolved
	AuthTokenID       string `gorm:"index"` // empty = web session
	PromptTokens      int
	CompletionTokens  int
	TotalTokens       int
	Cost              int64  // microcents, frozen at recording time
	Currency          string // frozen from provider
}

type wrappedUsageRecord struct {
	r *UsageRecord
}

func (w *wrappedUsageRecord) ID() model.UsageRecordID      { return model.UsageRecordID(w.r.ID) }
func (w *wrappedUsageRecord) UserID() model.UserID         { return model.UserID(w.r.UserID) }
func (w *wrappedUsageRecord) OrgID() model.OrgID           { return model.OrgID(w.r.OrgID) }
func (w *wrappedUsageRecord) ProviderID() model.ProviderID { return model.ProviderID(w.r.ProviderID) }
func (w *wrappedUsageRecord) ModelID() model.LLMModelID    { return model.LLMModelID(w.r.ModelID) }
func (w *wrappedUsageRecord) ProxyModelName() string       { return w.r.ProxyModelName }
func (w *wrappedUsageRecord) AuthTokenID() string          { return w.r.AuthTokenID }
func (w *wrappedUsageRecord) PromptTokens() int            { return w.r.PromptTokens }
func (w *wrappedUsageRecord) CompletionTokens() int        { return w.r.CompletionTokens }
func (w *wrappedUsageRecord) TotalTokens() int             { return w.r.TotalTokens }
func (w *wrappedUsageRecord) Cost() int64                  { return w.r.Cost }
func (w *wrappedUsageRecord) Currency() string             { return w.r.Currency }
func (w *wrappedUsageRecord) ResolvedModelName() string    { return w.r.ResolvedModelName }
func (w *wrappedUsageRecord) CreatedAt() time.Time         { return w.r.CreatedAt }

var _ model.UsageRecord = &wrappedUsageRecord{}

func fromUsageRecord(r model.UsageRecord) *UsageRecord {
	return &UsageRecord{
		ID:                string(r.ID()),
		UserID:            string(r.UserID()),
		OrgID:             string(r.OrgID()),
		ProviderID:        string(r.ProviderID()),
		ModelID:           string(r.ModelID()),
		ProxyModelName:    r.ProxyModelName(),
		ResolvedModelName: r.ResolvedModelName(),
		AuthTokenID:       r.AuthTokenID(),
		PromptTokens:      r.PromptTokens(),
		CompletionTokens:  r.CompletionTokens(),
		TotalTokens:       r.TotalTokens(),
		Cost:              r.Cost(),
		Currency:          r.Currency(),
	}
}
