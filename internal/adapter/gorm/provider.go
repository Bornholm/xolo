package gorm

import (
	"time"

	"github.com/bornholm/xolo/internal/core/model"
)

type Provider struct {
	ID        string `gorm:"primaryKey;autoIncrement:false"`
	CreatedAt time.Time
	UpdatedAt time.Time
	OrgID     string `gorm:"index;not null"`
	Name      string `gorm:"not null"`
	Type      string `gorm:"not null"`
	BaseURL   string
	APIKey    string `gorm:"not null"` // AES-GCM encrypted hex
	Active    int    `gorm:"default:1"`
	Currency  string `gorm:"default:'USD'"`
	CloudTier int    `gorm:"default:0"`
	RetryConfig     JSONColumn[model.RetryConfig]     `gorm:"type:text"`
	RateLimitConfig JSONColumn[model.RateLimitConfig] `gorm:"type:text"`

	LLMModels []*LLMModel `gorm:"foreignKey:ProviderID;constraint:OnDelete:CASCADE"`
}

type wrappedProvider struct {
	p *Provider
}

func (w *wrappedProvider) ID() model.ProviderID  { return model.ProviderID(w.p.ID) }
func (w *wrappedProvider) OrgID() model.OrgID    { return model.OrgID(w.p.OrgID) }
func (w *wrappedProvider) Name() string          { return w.p.Name }
func (w *wrappedProvider) Type() string          { return w.p.Type }
func (w *wrappedProvider) BaseURL() string       { return w.p.BaseURL }
func (w *wrappedProvider) APIKey() string        { return w.p.APIKey }
func (w *wrappedProvider) Active() bool          { return w.p.Active != 0 }
func (w *wrappedProvider) Currency() string      { return w.p.Currency }
func (w *wrappedProvider) CloudTier() int        { return w.p.CloudTier }
func (w *wrappedProvider) CreatedAt() time.Time                 { return w.p.CreatedAt }
func (w *wrappedProvider) UpdatedAt() time.Time                 { return w.p.UpdatedAt }
func (w *wrappedProvider) RetryConfig() *model.RetryConfig         { return w.p.RetryConfig.Val }
func (w *wrappedProvider) RateLimitConfig() *model.RateLimitConfig { return w.p.RateLimitConfig.Val }

var _ model.Provider = &wrappedProvider{}

func fromProvider(p model.Provider) *Provider {
	return &Provider{
		ID:              string(p.ID()),
		OrgID:           string(p.OrgID()),
		Name:            p.Name(),
		Type:            p.Type(),
		BaseURL:         p.BaseURL(),
		APIKey:          p.APIKey(),
		Active:          boolToInt(p.Active()),
		Currency:        p.Currency(),
		CloudTier:       p.CloudTier(),
		RetryConfig:     JSONColumn[model.RetryConfig]{Val: p.RetryConfig()},
		RateLimitConfig: JSONColumn[model.RateLimitConfig]{Val: p.RateLimitConfig()},
	}
}
