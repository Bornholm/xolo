package component

import (
	"strings"
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
	Org          model.Organization
	Members      []model.Membership
	Success      string
	CurrentPage  int
	PageSize     int
	TotalMembers int
}

type ProvidersPageVModel struct {
	common.AppLayoutVModel
	Org         model.Organization
	Providers   []model.Provider
	ModelCounts map[model.ProviderID]int
	Success     string
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
	Org         model.Organization
	Membership  model.Membership
	ScopeType   string
	ScopeID     string
	Quota       model.Quota
	Success     string
	DailyCost   int64 // current period spend in org currency (microcents)
	MonthlyCost int64
	YearlyCost  int64
}

type InvitesPageVModel struct {
	common.AppLayoutVModel
	Org       model.Organization
	Invites   []model.InviteToken
	BaseURL   string
	Success   string
	NewURL    string
	RoleNames map[string]string // role ID → display name
}

type InviteFormVModel struct {
	common.AppLayoutVModel
	Org      model.Organization
	OrgRoles []model.Role
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
	// Rétention des événements (ring buffer).
	EventsMaxOverride *int // nil = utilise la valeur par défaut globale
	EventsDefault     int  // valeur par défaut globale
	EventsGlobalCap   int  // plafond global (borne supérieure)
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

type ChartDataPoint = common.ChartDataPoint

// ChartShare represents a value's share of a total, as a percentage, with
// an associated chart color (cycling through the design system's palette).
type ChartShare = common.ChartShare

// SubscriptionConstraintUsage holds runtime consumption data for one plan constraint.
type SubscriptionConstraintUsage struct {
	Constraint model.PlanConstraint
	// rolling_window fields
	TokensUsed  int64
	ValueUsed   int64     // microcents, provider currency
	WindowStart time.Time // start of the current window (anchored or sliding)
	// OldestUsage is the creation time of the oldest usage record still counted in
	// the window (zero when the window is empty). For sliding windows, usage frees up
	// progressively from OldestUsage + Duration onwards.
	OldestUsage time.Time
	// Anchored is true when the window is a fixed (tumbling) window aligned on a manual
	// anchor, matching the upstream provider's real reset schedule.
	Anchored bool
	// ResetAt is the instant the current fixed window resets (zero for sliding windows).
	ResetAt time.Time
	// concurrency fields
	InFlight  int
	Exhausted bool
}

// SubscriptionProviderUsage aggregates plan usage for one subscription provider.
type SubscriptionProviderUsage struct {
	Provider    model.Provider
	Plan        model.SubscriptionPlan
	Constraints []SubscriptionConstraintUsage
	// PerUser is true when the figures represent the viewer's personal fair-share
	// (usage and budgets divided by the org member count) rather than the org-wide total.
	PerUser bool
}

type OrgUsagePageVModel struct {
	common.AppLayoutVModel
	Org             model.Organization
	Aggregate       *port.UsageAggregate
	Records         []OrgDisplayUsageRecord
	Users           map[model.UserID]model.User
	Members         []model.Membership // pour le filtre utilisateur
	OwnerFilter     []string           // IDs sélectionnés (users + applications)
	Applications    []model.Application
	ApplicationsMap map[model.ApplicationID]model.Application
	Since           time.Time
	Range           string // "7d", "30d", "90d", "180d", "365d"
	Page            int
	PageSize        int
	HasNext         bool
	// Subscription providers plan consumption
	SubscriptionProviders []SubscriptionProviderUsage
	// Chart/quota fields
	OrgQuota            model.Quota // may be nil if no quota defined
	DailyCost           int64       // today's org cost in org currency (microcents)
	MonthlyCost         int64       // this month's org cost in org currency (microcents)
	YearlyCost          int64       // this year's org cost in org currency (microcents)
	Currency            string      // org currency
	ChartPerDay         []ChartDataPoint
	ChartSharesPerModel []ChartShare
	ChartPerUser        []ChartDataPoint
	ChartPerProvider    []ChartDataPoint
	// Consommation (en tokens) des requêtes couvertes par un abonnement, par utilisateur.
	// Séparé des graphiques de coût car le forfait est un coût fixe, pas marginal.
	ChartSubTokensPerUser []ChartDataPoint
	// Energy estimation
	TotalEnergyWh    float64 // sum of midpoint estimates (0 if all unknown)
	TotalCO2GramsMid float64 // sum of CO₂ midpoints (world average, grams)
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

type RolesPageVModel struct {
	common.AppLayoutVModel
	Org     model.Organization
	Roles   []model.Role
	Success string
}

// RoleModelOption is a selectable model (LLM or virtual) for role model grants.
type RoleModelOption struct {
	ID    string
	Kind  string // rbac.ModelKindLLM | rbac.ModelKindVirtual
	Label string
}

type RoleFormVModel struct {
	common.AppLayoutVModel
	Org          model.Organization
	Role         model.Role
	IsNew        bool
	Error        string
	Selected     map[string]bool // permission code -> checked
	ModelOptions []RoleModelOption
	SelectedMode map[string]bool // "kind\x00id" -> checked
}

// roleLabel returns the display label for an organization role (legacy builtin strings).
func roleLabel(role string) string {
	switch role {
	case model.RoleOrgOwner:
		return "Propriétaire"
	case model.RoleOrgAdmin:
		return "Administrateur"
	default:
		return "Utilisateur"
	}
}

// roleLabelFromMap resolves a role display name from the preloaded RoleNames map,
// falling back to roleLabel for legacy builtin strings.
func roleLabelFromMap(names map[string]string, roleID string) string {
	if names != nil {
		if name, ok := names[roleID]; ok {
			return name
		}
	}
	return roleLabel(roleID)
}

// BuiltinRoleLabel returns the French label for a builtin role kind.
func BuiltinRoleLabel(kind string) string {
	switch kind {
	case model.BuiltinKindOwner:
		return "Propriétaire"
	case model.BuiltinKindAdmin:
		return "Administrateur"
	default:
		return "Utilisateur"
	}
}

// initials returns the 1-2 letter initials used as an avatar fallback.
func initials(name string) string {
	fields := strings.FieldsFunc(name, func(r rune) bool {
		switch r {
		case ' ', '@', '.', '_', '-':
			return true
		default:
			return false
		}
	})
	if len(fields) == 0 {
		return "?"
	}
	if len(fields) == 1 {
		if len(fields[0]) >= 2 {
			return strings.ToUpper(fields[0][:2])
		}
		return strings.ToUpper(fields[0])
	}
	first := fields[0][:1]
	last := fields[len(fields)-1][:1]
	return strings.ToUpper(first + last)
}
