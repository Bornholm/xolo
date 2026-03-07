package component

import (
	"time"

	"github.com/bornholm/xolo/internal/core/model"
	"github.com/bornholm/xolo/internal/core/port"
	common "github.com/bornholm/xolo/internal/http/handler/webui/common/component"
)

type OrgDashboardVModel struct {
	common.AppLayoutVModel
	Org       model.Organization
	Members   []model.Membership
	Providers []model.Provider
}

type MembersPageVModel struct {
	common.AppLayoutVModel
	Org     model.Organization
	Members []model.Membership
	Success string
}

type ProvidersPageVModel struct {
	common.AppLayoutVModel
	Org       model.Organization
	Providers []model.Provider
	Success   string
}

type ProviderFormVModel struct {
	common.AppLayoutVModel
	Org      model.Organization
	Provider model.Provider
	IsNew    bool
	Error    string
}

type ModelsPageVModel struct {
	common.AppLayoutVModel
	Org      model.Organization
	Provider model.Provider
	Models   []model.LLMModel
	Success  string
}

type ModelFormVModel struct {
	common.AppLayoutVModel
	Org      model.Organization
	Provider model.Provider
	Model    model.LLMModel
	IsNew    bool
	Error    string
}

type QuotaPageVModel struct {
	common.AppLayoutVModel
	Org        model.Organization
	Membership model.Membership
	ScopeType  string
	ScopeID    string
	Quota      model.Quota
	Success    string
}

type InvitesPageVModel struct {
	common.AppLayoutVModel
	Org     model.Organization
	Invites []model.InviteToken
	BaseURL string
	Success string
	NewURL  string
}

type InviteFormVModel struct {
	common.AppLayoutVModel
	Org model.Organization
}

type OrgSettingsPageVModel struct {
	common.AppLayoutVModel
	Org     model.Organization
	Success string
}

type OrgDisplayUsageRecord struct {
	Record          model.UsageRecord
	DisplayCost     int64
	DisplayCurrency string
	Converted       bool
}

type ChartDataPoint struct {
	Label string
	Value float64 // currency units (NOT microcents)
}

type OrgUsagePageVModel struct {
	common.AppLayoutVModel
	Org       model.Organization
	Aggregate *port.UsageAggregate
	Records   []OrgDisplayUsageRecord
	Users     map[model.UserID]model.User
	Since     time.Time
	Range     string // "7d", "30d", "90d", "180d", "365d"
	Page      int
	PageSize  int
	HasNext   bool
	// Chart/quota fields
	OrgQuota      model.Quota        // may be nil if no quota defined
	DailyCost     int64              // today's org cost in org currency (microcents)
	MonthlyCost   int64              // this month's org cost in org currency (microcents)
	YearlyCost    int64              // this year's org cost in org currency (microcents)
	Currency      string             // org currency
	ChartPerDay   []ChartDataPoint
	ChartPerModel []ChartDataPoint
	ChartPerUser  []ChartDataPoint
}
