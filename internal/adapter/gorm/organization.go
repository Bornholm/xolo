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
	Active      bool   `gorm:"default:true"`
	Currency    string `gorm:"not null;default:'USD'"`

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
func (w *wrappedOrganization) Active() bool              { return w.o.Active }
func (w *wrappedOrganization) Currency() string          { return w.o.Currency }
func (w *wrappedOrganization) CreatedAt() time.Time      { return w.o.CreatedAt }
func (w *wrappedOrganization) UpdatedAt() time.Time      { return w.o.UpdatedAt }

var _ model.Organization = &wrappedOrganization{}

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
		Active:      org.Active(),
		Currency:    currency,
	}
}
