package gorm

import (
	"time"

	"github.com/bornholm/xolo/internal/core/model"
)

type LLMModel struct {
	ID                        string `gorm:"primaryKey;autoIncrement:false"`
	CreatedAt                 time.Time
	UpdatedAt                 time.Time
	ProviderID                string `gorm:"index;not null"`
	OrgID                     string `gorm:"uniqueIndex:idx_org_proxy;not null"`
	ProxyName                 string `gorm:"uniqueIndex:idx_org_proxy;not null"`
	RealModel                 string `gorm:"not null"`
	Description               string
	Enabled                   int   `gorm:"default:1"`
	PromptCostPer1KTokens     int64 `gorm:"default:0"`
	CompletionCostPer1KTokens int64 `gorm:"default:0"`
	ContextWindow             int64 `gorm:"default:0"`
	OutputWindow              int64 `gorm:"default:0"`
	CapTools                  int   `gorm:"default:0"`
	CapVision                 int   `gorm:"default:0"`
	CapReasoning              int   `gorm:"default:0"`
	CapAudio                  int   `gorm:"default:0"`
	TokenLimitConfig JSONColumn[model.TokenLimitConfig] `gorm:"type:text"`
}

type wrappedLLMModel struct {
	m *LLMModel
}

func (w *wrappedLLMModel) ID() model.LLMModelID               { return model.LLMModelID(w.m.ID) }
func (w *wrappedLLMModel) ProviderID() model.ProviderID       { return model.ProviderID(w.m.ProviderID) }
func (w *wrappedLLMModel) OrgID() model.OrgID                 { return model.OrgID(w.m.OrgID) }
func (w *wrappedLLMModel) ProxyName() string                  { return w.m.ProxyName }
func (w *wrappedLLMModel) RealModel() string                  { return w.m.RealModel }
func (w *wrappedLLMModel) Description() string                { return w.m.Description }
func (w *wrappedLLMModel) Enabled() bool                      { return w.m.Enabled != 0 }
func (w *wrappedLLMModel) PromptCostPer1KTokens() int64       { return w.m.PromptCostPer1KTokens }
func (w *wrappedLLMModel) CompletionCostPer1KTokens() int64   { return w.m.CompletionCostPer1KTokens }
func (w *wrappedLLMModel) ContextWindow() int64               { return w.m.ContextWindow }
func (w *wrappedLLMModel) OutputWindow() int64                { return w.m.OutputWindow }
func (w *wrappedLLMModel) Capabilities() model.ModelCapabilities {
	return model.ModelCapabilities{
		Tools:     w.m.CapTools != 0,
		Vision:    w.m.CapVision != 0,
		Reasoning: w.m.CapReasoning != 0,
		Audio:     w.m.CapAudio != 0,
	}
}
func (w *wrappedLLMModel) CreatedAt() time.Time { return w.m.CreatedAt }
func (w *wrappedLLMModel) UpdatedAt() time.Time { return w.m.UpdatedAt }
func (w *wrappedLLMModel) TokenLimitConfig() *model.TokenLimitConfig { return w.m.TokenLimitConfig.Val }

var _ model.LLMModel = &wrappedLLMModel{}

func fromLLMModel(m model.LLMModel) *LLMModel {
	caps := m.Capabilities()
	return &LLMModel{
		ID:                        string(m.ID()),
		ProviderID:                string(m.ProviderID()),
		OrgID:                     string(m.OrgID()),
		ProxyName:                 m.ProxyName(),
		RealModel:                 m.RealModel(),
		Description:               m.Description(),
		Enabled:                   boolToInt(m.Enabled()),
		PromptCostPer1KTokens:     m.PromptCostPer1KTokens(),
		CompletionCostPer1KTokens: m.CompletionCostPer1KTokens(),
		ContextWindow:             m.ContextWindow(),
		OutputWindow:              m.OutputWindow(),
		CapTools:                  boolToInt(caps.Tools),
		CapVision:                 boolToInt(caps.Vision),
		CapReasoning:              boolToInt(caps.Reasoning),
		CapAudio:                  boolToInt(caps.Audio),
		TokenLimitConfig: JSONColumn[model.TokenLimitConfig]{Val: m.TokenLimitConfig()},
	}
}
