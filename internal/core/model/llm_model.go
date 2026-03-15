package model

import (
	"time"

	"github.com/rs/xid"
)

type LLMModelID string

func NewLLMModelID() LLMModelID {
	return LLMModelID(xid.New().String())
}

// ModelCapabilities describes the features supported by a model.
type ModelCapabilities struct {
	Tools     bool // function / tool calling
	Vision    bool // image inputs
	Reasoning bool // extended chain-of-thought / reasoning tokens
	Audio     bool // audio inputs or outputs
}

// LLMModel is a proxy route: what users call it → what the provider sees.
type LLMModel interface {
	WithID[LLMModelID]

	ProviderID() ProviderID
	OrgID() OrgID
	ProxyName() string // name the client sends in the API request
	RealModel() string // name forwarded to the upstream provider
	Description() string
	Enabled() bool
	// Costs in microcents per 1K tokens (1 microcent = $0.000001)
	PromptCostPer1KTokens() int64
	CompletionCostPer1KTokens() int64
	// Context limits (0 = unknown / not set)
	ContextWindow() int64 // max input tokens
	OutputWindow() int64  // max output tokens
	// Capabilities
	Capabilities() ModelCapabilities
	CreatedAt() time.Time
	UpdatedAt() time.Time
	TokenLimitConfig() *TokenLimitConfig
}

type BaseLLMModel struct {
	id                        LLMModelID
	providerID                ProviderID
	orgID                     OrgID
	proxyName                 string
	realModel                 string
	description               string
	enabled                   bool
	promptCostPer1KTokens     int64
	completionCostPer1KTokens int64
	contextWindow             int64
	outputWindow              int64
	capabilities              ModelCapabilities
	createdAt                 time.Time
	updatedAt                 time.Time
	tokenLimitConfig          *TokenLimitConfig
}

func (m *BaseLLMModel) ID() LLMModelID                     { return m.id }
func (m *BaseLLMModel) ProviderID() ProviderID             { return m.providerID }
func (m *BaseLLMModel) OrgID() OrgID                       { return m.orgID }
func (m *BaseLLMModel) ProxyName() string                  { return m.proxyName }
func (m *BaseLLMModel) RealModel() string                  { return m.realModel }
func (m *BaseLLMModel) Description() string                { return m.description }
func (m *BaseLLMModel) Enabled() bool                      { return m.enabled }
func (m *BaseLLMModel) PromptCostPer1KTokens() int64       { return m.promptCostPer1KTokens }
func (m *BaseLLMModel) CompletionCostPer1KTokens() int64   { return m.completionCostPer1KTokens }
func (m *BaseLLMModel) ContextWindow() int64               { return m.contextWindow }
func (m *BaseLLMModel) OutputWindow() int64                { return m.outputWindow }
func (m *BaseLLMModel) Capabilities() ModelCapabilities    { return m.capabilities }
func (m *BaseLLMModel) CreatedAt() time.Time               { return m.createdAt }
func (m *BaseLLMModel) UpdatedAt() time.Time               { return m.updatedAt }
func (m *BaseLLMModel) TokenLimitConfig() *TokenLimitConfig     { return m.tokenLimitConfig }

func (m *BaseLLMModel) SetContextWindow(v int64)               { m.contextWindow = v }
func (m *BaseLLMModel) SetOutputWindow(v int64)                { m.outputWindow = v }
func (m *BaseLLMModel) SetCapabilities(c ModelCapabilities)    { m.capabilities = c }
func (m *BaseLLMModel) SetTokenLimitConfig(c *TokenLimitConfig) { m.tokenLimitConfig = c }

var _ LLMModel = &BaseLLMModel{}

func NewLLMModel(providerID ProviderID, orgID OrgID, proxyName, realModel, description string, promptCost, completionCost int64) *BaseLLMModel {
	return &BaseLLMModel{
		id:                        NewLLMModelID(),
		providerID:                providerID,
		orgID:                     orgID,
		proxyName:                 proxyName,
		realModel:                 realModel,
		description:               description,
		enabled:                   true,
		promptCostPer1KTokens:     promptCost,
		completionCostPer1KTokens: completionCost,
		createdAt:                 time.Now(),
		updatedAt:                 time.Now(),
	}
}
