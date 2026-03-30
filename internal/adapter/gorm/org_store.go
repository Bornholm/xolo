package gorm

import (
	"context"

	"github.com/bornholm/xolo/internal/core/model"
	"github.com/bornholm/xolo/internal/core/port"
	"github.com/ncruces/go-sqlite3"
	"github.com/pkg/errors"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// CreateOrg implements port.OrgStore.
func (s *Store) CreateOrg(ctx context.Context, org model.Organization) error {
	return s.withRetry(ctx, true, func(ctx context.Context, db *gorm.DB) error {
		return errors.WithStack(db.Create(fromOrganization(org)).Error)
	}, sqlite3.BUSY, sqlite3.LOCKED)
}

// GetOrgByID implements port.OrgStore.
func (s *Store) GetOrgByID(ctx context.Context, id model.OrgID) (model.Organization, error) {
	var org Organization
	err := s.withRetry(ctx, false, func(ctx context.Context, db *gorm.DB) error {
		if err := db.First(&org, "id = ?", string(id)).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return errors.WithStack(port.ErrNotFound)
			}
			return errors.WithStack(err)
		}
		return nil
	}, sqlite3.BUSY, sqlite3.LOCKED)
	if err != nil {
		return nil, err
	}
	return &wrappedOrganization{&org}, nil
}

// GetOrgBySlug implements port.OrgStore.
func (s *Store) GetOrgBySlug(ctx context.Context, slug string) (model.Organization, error) {
	var org Organization
	err := s.withRetry(ctx, false, func(ctx context.Context, db *gorm.DB) error {
		if err := db.First(&org, "slug = ?", slug).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return errors.WithStack(port.ErrNotFound)
			}
			return errors.WithStack(err)
		}
		return nil
	}, sqlite3.BUSY, sqlite3.LOCKED)
	if err != nil {
		return nil, err
	}
	return &wrappedOrganization{&org}, nil
}

// ListOrgs implements port.OrgStore.
func (s *Store) ListOrgs(ctx context.Context, opts port.ListOrgsOptions) ([]model.Organization, int64, error) {
	var orgs []*Organization
	var total int64

	err := s.withRetry(ctx, false, func(ctx context.Context, db *gorm.DB) error {
		query := db.Model(&Organization{})

		if err := query.Count(&total).Error; err != nil {
			return errors.WithStack(err)
		}

		if opts.Limit != nil {
			query = query.Limit(*opts.Limit)
		}
		if opts.Page != nil && opts.Limit != nil {
			query = query.Offset(*opts.Page * *opts.Limit)
		}

		return errors.WithStack(query.Order("name ASC").Find(&orgs).Error)
	}, sqlite3.BUSY, sqlite3.LOCKED)
	if err != nil {
		return nil, 0, err
	}

	result := make([]model.Organization, 0, len(orgs))
	for _, o := range orgs {
		result = append(result, &wrappedOrganization{o})
	}
	return result, total, nil
}

// SaveOrg implements port.OrgStore.
func (s *Store) SaveOrg(ctx context.Context, org model.Organization) error {
	return s.withRetry(ctx, true, func(ctx context.Context, db *gorm.DB) error {
		return errors.WithStack(db.Clauses(clause.OnConflict{
			Columns:   []clause.Column{{Name: "id"}},
			UpdateAll: true,
		}).Create(fromOrganization(org)).Error)
	}, sqlite3.BUSY, sqlite3.LOCKED)
}

// DeleteOrg implements port.OrgStore.
func (s *Store) DeleteOrg(ctx context.Context, id model.OrgID) error {
	return s.withRetry(ctx, true, func(ctx context.Context, db *gorm.DB) error {
		result := db.Delete(&Organization{}, "id = ?", string(id))
		if result.Error != nil {
			return errors.WithStack(result.Error)
		}
		if result.RowsAffected == 0 {
			return errors.WithStack(port.ErrNotFound)
		}
		return nil
	}, sqlite3.BUSY, sqlite3.LOCKED)
}

// AddMember implements port.OrgStore.
func (s *Store) AddMember(ctx context.Context, membership model.Membership) error {
	return s.withRetry(ctx, true, func(ctx context.Context, db *gorm.DB) error {
		return errors.WithStack(db.Create(fromMembership(membership)).Error)
	}, sqlite3.BUSY, sqlite3.LOCKED)
}

// RemoveMember implements port.OrgStore.
func (s *Store) RemoveMember(ctx context.Context, id model.MembershipID) error {
	return s.withRetry(ctx, true, func(ctx context.Context, db *gorm.DB) error {
		result := db.Delete(&Membership{}, "id = ?", string(id))
		if result.Error != nil {
			return errors.WithStack(result.Error)
		}
		if result.RowsAffected == 0 {
			return errors.WithStack(port.ErrNotFound)
		}
		return nil
	}, sqlite3.BUSY, sqlite3.LOCKED)
}

// GetMembership implements port.OrgStore.
func (s *Store) GetMembership(ctx context.Context, id model.MembershipID) (model.Membership, error) {
	var m Membership
	err := s.withRetry(ctx, false, func(ctx context.Context, db *gorm.DB) error {
		if err := db.Preload("User").Preload("Org").First(&m, "id = ?", string(id)).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return errors.WithStack(port.ErrNotFound)
			}
			return errors.WithStack(err)
		}
		return nil
	}, sqlite3.BUSY, sqlite3.LOCKED)
	if err != nil {
		return nil, err
	}
	return &wrappedMembership{&m}, nil
}

// GetUserOrgMembership implements port.OrgStore.
func (s *Store) GetUserOrgMembership(ctx context.Context, userID model.UserID, orgID model.OrgID) (model.Membership, error) {
	var m Membership
	err := s.withRetry(ctx, false, func(ctx context.Context, db *gorm.DB) error {
		if err := db.Preload("User").Preload("Org").
			Where("user_id = ? AND org_id = ?", string(userID), string(orgID)).
			First(&m).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return errors.WithStack(port.ErrNotFound)
			}
			return errors.WithStack(err)
		}
		return nil
	}, sqlite3.BUSY, sqlite3.LOCKED)
	if err != nil {
		return nil, err
	}
	return &wrappedMembership{&m}, nil
}

// ListOrgMembers implements port.OrgStore.
func (s *Store) ListOrgMembers(ctx context.Context, orgID model.OrgID, opts port.ListOrgMembersOptions) ([]model.Membership, int64, error) {
	var members []*Membership
	var total int64

	err := s.withRetry(ctx, false, func(ctx context.Context, db *gorm.DB) error {
		base := db.Model(&Membership{}).Where("org_id = ?", string(orgID))

		if err := base.Count(&total).Error; err != nil {
			return errors.WithStack(err)
		}

		query := db.Preload("User").Preload("Org").Where("org_id = ?", string(orgID))

		if opts.Page != nil && opts.Limit != nil {
			query = query.Offset(*opts.Page * *opts.Limit)
		}
		if opts.Limit != nil {
			query = query.Limit(*opts.Limit)
		}

		return errors.WithStack(query.Find(&members).Error)
	}, sqlite3.BUSY, sqlite3.LOCKED)
	if err != nil {
		return nil, 0, err
	}

	result := make([]model.Membership, 0, len(members))
	for _, m := range members {
		result = append(result, &wrappedMembership{m})
	}
	return result, total, nil
}

// GetUserMemberships implements port.OrgStore.
func (s *Store) GetUserMemberships(ctx context.Context, userID model.UserID) ([]model.Membership, error) {
	var members []*Membership
	err := s.withRetry(ctx, false, func(ctx context.Context, db *gorm.DB) error {
		return errors.WithStack(db.Preload("Org").
			Where("user_id = ?", string(userID)).
			Find(&members).Error)
	}, sqlite3.BUSY, sqlite3.LOCKED)
	if err != nil {
		return nil, err
	}
	result := make([]model.Membership, 0, len(members))
	for _, m := range members {
		result = append(result, &wrappedMembership{m})
	}
	return result, nil
}

// IsMember implements port.OrgStore.
func (s *Store) IsMember(ctx context.Context, userID model.UserID, orgID model.OrgID) (bool, error) {
	var count int64
	err := s.withRetry(ctx, false, func(ctx context.Context, db *gorm.DB) error {
		return errors.WithStack(db.Model(&Membership{}).
			Where("user_id = ? AND org_id = ?", string(userID), string(orgID)).
			Count(&count).Error)
	}, sqlite3.BUSY, sqlite3.LOCKED)
	if err != nil {
		return false, err
	}
	return count > 0, nil
}

var _ port.OrgStore = &Store{}
