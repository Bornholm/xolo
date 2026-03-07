package model

import (
	"time"

	"github.com/rs/xid"
)

type ProviderID string

func NewProviderID() ProviderID {
	return ProviderID(xid.New().String())
}

type Provider interface {
	WithID[ProviderID]

	OrgID() OrgID
	Name() string
	Type() string   // openai | mistral | openrouter | yzma
	BaseURL() string
	APIKey() string // encrypted at rest, decrypted on use
	Active() bool
	Currency() string
	CreatedAt() time.Time
	UpdatedAt() time.Time
}

type BaseProvider struct {
	id        ProviderID
	orgID     OrgID
	name      string
	pType     string
	baseURL   string
	apiKey    string
	active    bool
	currency  string
	createdAt time.Time
	updatedAt time.Time
}

func (p *BaseProvider) ID() ProviderID       { return p.id }
func (p *BaseProvider) OrgID() OrgID         { return p.orgID }
func (p *BaseProvider) Name() string         { return p.name }
func (p *BaseProvider) Type() string         { return p.pType }
func (p *BaseProvider) BaseURL() string      { return p.baseURL }
func (p *BaseProvider) APIKey() string       { return p.apiKey }
func (p *BaseProvider) Active() bool         { return p.active }
func (p *BaseProvider) Currency() string     { return p.currency }
func (p *BaseProvider) CreatedAt() time.Time { return p.createdAt }
func (p *BaseProvider) UpdatedAt() time.Time { return p.updatedAt }

var _ Provider = &BaseProvider{}

func NewProvider(orgID OrgID, name, pType, baseURL, apiKey, currency string) *BaseProvider {
	return &BaseProvider{
		id:        NewProviderID(),
		orgID:     orgID,
		name:      name,
		pType:     pType,
		baseURL:   baseURL,
		apiKey:    apiKey,
		active:    true,
		currency:  currency,
		createdAt: time.Now(),
		updatedAt: time.Now(),
	}
}
