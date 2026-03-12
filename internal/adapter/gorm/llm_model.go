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
	Enabled                   bool  `gorm:"default:true"`
	PromptCostPer1KTokens     int64 `gorm:"default:0"`
	CompletionCostPer1KTokens int64 `gorm:"default:0"`
	ContextWindow             int64 `gorm:"default:0"`
	OutputWindow              int64 `gorm:"default:0"`
	CapTools                  bool  `gorm:"default:false"`
	CapVision                 bool  `gorm:"default:false"`
	CapReasoning              bool  `gorm:"default:false"`
	CapAudio                  bool  `gorm:"default:false"`
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
func (w *wrappedLLMModel) Enabled() bool                      { return w.m.Enabled }
func (w *wrappedLLMModel) PromptCostPer1KTokens() int64       { return w.m.PromptCostPer1KTokens }
func (w *wrappedLLMModel) CompletionCostPer1KTokens() int64   { return w.m.CompletionCostPer1KTokens }
func (w *wrappedLLMModel) ContextWindow() int64               { return w.m.ContextWindow }
func (w *wrappedLLMModel) OutputWindow() int64                { return w.m.OutputWindow }
func (w *wrappedLLMModel) Capabilities() model.ModelCapabilities {
	return model.ModelCapabilities{
		Tools:     w.m.CapTools,
		Vision:    w.m.CapVision,
		Reasoning: w.m.CapReasoning,
		Audio:     w.m.CapAudio,
	}
}
func (w *wrappedLLMModel) CreatedAt() time.Time { return w.m.CreatedAt }
func (w *wrappedLLMModel) UpdatedAt() time.Time { return w.m.UpdatedAt }

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
		Enabled:                   m.Enabled(),
		PromptCostPer1KTokens:     m.PromptCostPer1KTokens(),
		CompletionCostPer1KTokens: m.CompletionCostPer1KTokens(),
		ContextWindow:             m.ContextWindow(),
		OutputWindow:              m.OutputWindow(),
		CapTools:                  caps.Tools,
		CapVision:                 caps.Vision,
		CapReasoning:              caps.Reasoning,
		CapAudio:                  caps.Audio,
	}
}
