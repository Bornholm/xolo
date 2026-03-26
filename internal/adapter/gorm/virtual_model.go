package gorm

import (
	"time"

	"github.com/bornholm/xolo/internal/core/model"
)

type VirtualModel struct {
	ID          string `gorm:"primaryKey;autoIncrement:false"`
	OrgID       string `gorm:"uniqueIndex:idx_org_name;not null"`
	Name        string `gorm:"uniqueIndex:idx_org_name;not null"`
	Description string
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

type wrappedVirtualModel struct {
	m *VirtualModel
}

func (w *wrappedVirtualModel) ID() model.VirtualModelID { return model.VirtualModelID(w.m.ID) }
func (w *wrappedVirtualModel) OrgID() model.OrgID       { return model.OrgID(w.m.OrgID) }
func (w *wrappedVirtualModel) Name() string             { return w.m.Name }
func (w *wrappedVirtualModel) Description() string      { return w.m.Description }
func (w *wrappedVirtualModel) CreatedAt() time.Time     { return w.m.CreatedAt }
func (w *wrappedVirtualModel) UpdatedAt() time.Time     { return w.m.UpdatedAt }

var _ model.VirtualModel = &wrappedVirtualModel{}

func fromVirtualModel(vm model.VirtualModel) *VirtualModel {
	return &VirtualModel{
		ID:          string(vm.ID()),
		OrgID:       string(vm.OrgID()),
		Name:        vm.Name(),
		Description: vm.Description(),
		CreatedAt:   vm.CreatedAt(),
		UpdatedAt:   vm.UpdatedAt(),
	}
}
