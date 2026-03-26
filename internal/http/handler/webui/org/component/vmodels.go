package component

import (
	"time"

	"github.com/bornholm/xolo/internal/core/model"
	"github.com/bornholm/xolo/internal/core/port"
	common "github.com/bornholm/xolo/internal/http/handler/webui/common/component"
	proto "github.com/bornholm/xolo/pkg/pluginsdk/proto"
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
	// Aperçu quota partagé (nil si non applicable)
	MemberCount         int
	SharedDailyBudget   *int64 // microcents, nil si org n'a pas ce budget
	SharedMonthlyBudget *int64
	SharedYearlyBudget  *int64
}

type OrgDisplayUsageRecord struct {
	Record           model.UsageRecord
	DisplayModelName string // display name: "virtual -> resolved" or just the model name
	DisplayCost      int64
	DisplayCurrency  string
	Converted        bool
	EnergyWh         float64 // midpoint estimation in Wh (0 = unknown)
	EnergyLowWh      float64
	EnergyHighWh     float64
	CO2GramsMid      float64 // gCO₂ at world-average carbon intensity
	CO2GramsMin      float64 // gCO₂ at France nuclear intensity (best case)
	CO2GramsMax      float64 // gCO₂ at coal plant intensity (worst case)
}

type ChartDataPoint struct {
	Label string
	Value float64 // currency units (NOT microcents)
}

type OrgUsagePageVModel struct {
	common.AppLayoutVModel
	Org        model.Organization
	Aggregate  *port.UsageAggregate
	Records    []OrgDisplayUsageRecord
	Users      map[model.UserID]model.User
	Members    []model.Membership // pour le filtre utilisateur
	UserFilter []string           // IDs sélectionnés pour le filtre
	Since      time.Time
	Range      string // "7d", "30d", "90d", "180d", "365d"
	Page       int
	PageSize   int
	HasNext    bool
	// Chart/quota fields
	OrgQuota      model.Quota // may be nil if no quota defined
	DailyCost     int64       // today's org cost in org currency (microcents)
	MonthlyCost   int64       // this month's org cost in org currency (microcents)
	YearlyCost    int64       // this year's org cost in org currency (microcents)
	Currency      string      // org currency
	ChartPerDay   []ChartDataPoint
	ChartPerModel []ChartDataPoint
	ChartPerUser  []ChartDataPoint
	// Energy estimation
	TotalEnergyWh    float64 // sum of midpoint estimates (0 if all unknown)
	TotalCO2GramsMid float64 // sum of CO₂ midpoints (world average, grams)
}

type PluginsPageVModel struct {
	common.AppLayoutVModel
	Org         model.Organization
	Descriptors []*proto.PluginDescriptor
	Active      map[string]bool
}

type PluginConfigPageVModel struct {
	common.AppLayoutVModel
	Org         model.Organization
	Descriptor  *proto.PluginDescriptor
	Properties  map[string]map[string]any
	Values      map[string]string
	FieldErrors map[string]string
	HasHTTPUI   bool // true → render iframe instead of JSONSchemaForm
}

type VirtualModelsPageVModel struct {
	common.AppLayoutVModel
	Org           model.Organization
	VirtualModels []model.VirtualModel
	BaseURL       string
	Success       string
	Error         string
}

type VirtualModelFormVModel struct {
	common.AppLayoutVModel
	Org          model.Organization
	VirtualModel model.VirtualModel
	IsNew        bool
	Name         string
	Description  string
	Error        string
}
