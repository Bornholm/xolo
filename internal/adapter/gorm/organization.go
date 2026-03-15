package gorm

import (
	"time"

	"github.com/bornholm/xolo/internal/core/model"
)

type Organization struct {
	ID          string `gorm:"primaryKey;autoIncrement:false"`
	CreatedAt   time.Time
	UpdatedAt   time.Time
	Slug        string `gorm:"uniqueIndex;not null"`
	Name        string `gorm:"not null"`
	Description string
	Active      int    `gorm:"default:1"`
	Currency    string `gorm:"not null;default:'USD'"`
	ShareQuotaEqually int `gorm:"column:share_quota_equally;default:0"`

	Memberships []*Membership `gorm:"foreignKey:OrgID;constraint:OnDelete:CASCADE"`
	Providers   []*Provider   `gorm:"foreignKey:OrgID;constraint:OnDelete:CASCADE"`
	LLMModels   []*LLMModel   `gorm:"foreignKey:OrgID;constraint:OnDelete:CASCADE"`
}

type wrappedOrganization struct {
	o *Organization
}

func (w *wrappedOrganization) ID() model.OrgID          { return model.OrgID(w.o.ID) }
func (w *wrappedOrganization) Slug() string              { return w.o.Slug }
func (w *wrappedOrganization) Name() string              { return w.o.Name }
func (w *wrappedOrganization) Description() string       { return w.o.Description }
func (w *wrappedOrganization) Active() bool              { return w.o.Active != 0 }
func (w *wrappedOrganization) Currency() string          { return w.o.Currency }
func (w *wrappedOrganization) CreatedAt() time.Time      { return w.o.CreatedAt }
func (w *wrappedOrganization) UpdatedAt() time.Time      { return w.o.UpdatedAt }
func (w *wrappedOrganization) ShareQuotaEqually() bool   { return w.o.ShareQuotaEqually != 0 }

var _ model.Organization = &wrappedOrganization{}

func boolToInt(b bool) int {
	if b {
		return 1
	}
	return 0
}

func fromOrganization(org model.Organization) *Organization {
	currency := org.Currency()
	if currency == "" {
		currency = model.DefaultCurrency
	}
	return &Organization{
		ID:          string(org.ID()),
		Slug:        org.Slug(),
		Name:        org.Name(),
		Description: org.Description(),
		Active:            boolToInt(org.Active()),
		Currency:    currency,
		ShareQuotaEqually: boolToInt(org.ShareQuotaEqually()),
	}
}
