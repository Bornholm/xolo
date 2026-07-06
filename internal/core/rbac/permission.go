package rbac

// Permission is a fixed authorization code checked at enforcement points
// (HTTP routes, proxy hooks). Permissions are defined in code — the code is
// the source of truth — and only known codes are ever exposed to the UI.
type Permission string

const (
	// Admin sections (read/write). A GET route maps to ":read", a
	// POST/DELETE route maps to ":write". "write" implies "read".
	PermMembersRead  Permission = "members:read"
	PermMembersWrite Permission = "members:write"

	PermRolesRead  Permission = "roles:read"
	PermRolesWrite Permission = "roles:write"

	// Providers also cover the LLM models declared under /providers/...
	PermProvidersRead  Permission = "providers:read"
	PermProvidersWrite Permission = "providers:write"

	PermVirtualModelsRead  Permission = "virtual-models:read"
	PermVirtualModelsWrite Permission = "virtual-models:write"

	PermMiddlewaresRead  Permission = "middlewares:read"
	PermMiddlewaresWrite Permission = "middlewares:write"

	PermQuotaRead  Permission = "quota:read"
	PermQuotaWrite Permission = "quota:write"

	PermInvitesRead  Permission = "invites:read"
	PermInvitesWrite Permission = "invites:write"

	PermApplicationsRead  Permission = "applications:read"
	PermApplicationsWrite Permission = "applications:write"

	PermSettingsRead  Permission = "settings:read"
	PermSettingsWrite Permission = "settings:write"

	PermUsageRead Permission = "usage:read"

	// Events. Viewing one's own events requires no permission; these gate access
	// to broader event visibility and alert management.
	PermEventsReadAll  Permission = "events:read:all"   // global + other users' events
	PermEventsWrite    Permission = "events:write"      // manage org-wide alerts
	PermEventsAlertsOwn Permission = "events:alerts:own" // create/manage personal alerts on own events

	// Model usage (enforced at the proxy).
	PermModelUseOrg     Permission = "model:use:org"
	PermModelUseVirtual Permission = "model:use:virtual"

	// Personal virtual models.
	PermPersonalVMCreate Permission = "personal-vm:create"
)

// Model grant kinds, used by resource-scoped model access.
const (
	ModelKindLLM     = "llm"
	ModelKindVirtual = "virtual"
)

// writeImplies maps a "write" permission to the "read" permission it grants
// implicitly. Granting write access always grants the matching read access.
var writeImplies = map[Permission]Permission{
	PermMembersWrite:       PermMembersRead,
	PermRolesWrite:         PermRolesRead,
	PermProvidersWrite:     PermProvidersRead,
	PermVirtualModelsWrite: PermVirtualModelsRead,
	PermMiddlewaresWrite:   PermMiddlewaresRead,
	PermQuotaWrite:         PermQuotaRead,
	PermInvitesWrite:       PermInvitesRead,
	PermApplicationsWrite:  PermApplicationsRead,
	PermSettingsWrite:      PermSettingsRead,
	PermEventsWrite:        PermEventsReadAll,
}

// PermissionDef describes a single permission for display in the UI.
type PermissionDef struct {
	Code  Permission
	Label string
}

// PermissionGroup groups permissions by admin section for the UI.
type PermissionGroup struct {
	Section string
	Label   string
	Perms   []PermissionDef
}

// Catalog returns the full, grouped list of assignable permissions. It is the
// single source of truth for the role-editing UI and for IsKnown validation.
func Catalog() []PermissionGroup {
	return []PermissionGroup{
		{
			Section: "members",
			Label:   "Membres",
			Perms: []PermissionDef{
				{PermMembersRead, "Consulter les membres"},
				{PermMembersWrite, "Gérer les membres"},
			},
		},
		{
			Section: "roles",
			Label:   "Rôles & permissions",
			Perms: []PermissionDef{
				{PermRolesRead, "Consulter les rôles"},
				{PermRolesWrite, "Gérer les rôles"},
			},
		},
		{
			Section: "providers",
			Label:   "Fournisseurs & modèles",
			Perms: []PermissionDef{
				{PermProvidersRead, "Consulter les fournisseurs et modèles"},
				{PermProvidersWrite, "Gérer les fournisseurs et modèles"},
			},
		},
		{
			Section: "virtual-models",
			Label:   "Modèles virtuels",
			Perms: []PermissionDef{
				{PermVirtualModelsRead, "Consulter les modèles virtuels"},
				{PermVirtualModelsWrite, "Gérer les modèles virtuels"},
			},
		},
		{
			Section: "middlewares",
			Label:   "Middlewares",
			Perms: []PermissionDef{
				{PermMiddlewaresRead, "Consulter les middlewares"},
				{PermMiddlewaresWrite, "Gérer les middlewares"},
			},
		},
		{
			Section: "quota",
			Label:   "Budget",
			Perms: []PermissionDef{
				{PermQuotaRead, "Consulter le budget"},
				{PermQuotaWrite, "Gérer le budget"},
			},
		},
		{
			Section: "invites",
			Label:   "Invitations",
			Perms: []PermissionDef{
				{PermInvitesRead, "Consulter les invitations"},
				{PermInvitesWrite, "Gérer les invitations"},
			},
		},
		{
			Section: "applications",
			Label:   "Applications",
			Perms: []PermissionDef{
				{PermApplicationsRead, "Consulter les applications"},
				{PermApplicationsWrite, "Gérer les applications"},
			},
		},
		{
			Section: "settings",
			Label:   "Paramètres",
			Perms: []PermissionDef{
				{PermSettingsRead, "Consulter les paramètres"},
				{PermSettingsWrite, "Gérer les paramètres"},
			},
		},
		{
			Section: "usage",
			Label:   "Usage",
			Perms: []PermissionDef{
				{PermUsageRead, "Consulter l'usage"},
			},
		},
		{
			Section: "events",
			Label:   "Événements & alertes",
			Perms: []PermissionDef{
				{PermEventsReadAll, "Consulter les événements globaux et des autres utilisateurs"},
				{PermEventsWrite, "Gérer les alertes de l'organisation"},
				{PermEventsAlertsOwn, "Créer des alertes sur ses propres événements"},
			},
		},
		{
			Section: "models",
			Label:   "Usage des modèles",
			Perms: []PermissionDef{
				{PermModelUseOrg, "Utiliser tous les modèles de l'organisation"},
				{PermModelUseVirtual, "Utiliser tous les modèles virtuels"},
			},
		},
		{
			Section: "personal",
			Label:   "Personnel",
			Perms: []PermissionDef{
				{PermPersonalVMCreate, "Créer des modèles virtuels personnels"},
			},
		},
	}
}

// knownPermissions is built once from the catalog for fast validation.
var knownPermissions = func() map[Permission]struct{} {
	known := map[Permission]struct{}{}
	for _, group := range Catalog() {
		for _, def := range group.Perms {
			known[def.Code] = struct{}{}
		}
	}
	return known
}()

// IsKnown reports whether code corresponds to a declared permission.
func IsKnown(code string) bool {
	_, ok := knownPermissions[Permission(code)]
	return ok
}

// IsAdminPermission reports whether the permission grants access to an
// administration section (as opposed to plain usage permissions). It is used to
// decide whether to surface the organization admin navigation to a member.
func IsAdminPermission(code Permission) bool {
	switch code {
	case PermUsageRead, PermModelUseOrg, PermModelUseVirtual, PermPersonalVMCreate, PermEventsAlertsOwn:
		return false
	}
	return IsKnown(string(code))
}

// ModelGrant authorizes the use of a specific model resource.
type ModelGrant struct {
	ModelID string
	Kind    string // ModelKindLLM | ModelKindVirtual
}

// PermissionSet holds the effective permissions resolved for a user within an
// organization (the union of the permissions of all their roles).
type PermissionSet struct {
	owner  bool
	codes  map[Permission]struct{}
	grants map[string]struct{} // key: kind + "\x00" + modelID
}

// NewPermissionSet builds a permission set from a list of permission codes and
// model grants. Unknown codes are ignored. The "write implies read" rule is
// applied here so callers only ever need to check the granular permission.
func NewPermissionSet(codes []string, grants []ModelGrant) PermissionSet {
	set := PermissionSet{
		codes:  make(map[Permission]struct{}, len(codes)),
		grants: make(map[string]struct{}, len(grants)),
	}
	for _, raw := range codes {
		code := Permission(raw)
		if _, ok := knownPermissions[code]; !ok {
			continue
		}
		set.codes[code] = struct{}{}
		if implied, ok := writeImplies[code]; ok {
			set.codes[implied] = struct{}{}
		}
	}
	for _, grant := range grants {
		set.grants[grantKey(grant.Kind, grant.ModelID)] = struct{}{}
	}
	return set
}

// OwnerPermissionSet returns a set that authorizes everything. It is used for
// the global admin and for members holding the builtin "owner" org role.
func OwnerPermissionSet() PermissionSet {
	return PermissionSet{owner: true}
}

func grantKey(kind, modelID string) string {
	return kind + "\x00" + modelID
}

// IsOwner reports whether the set bypasses all permission checks.
func (s PermissionSet) IsOwner() bool {
	return s.owner
}

// Has reports whether the set grants the given permission.
func (s PermissionSet) Has(perm Permission) bool {
	if s.owner {
		return true
	}
	_, ok := s.codes[perm]
	return ok
}

// HasModelAccess reports whether the set grants explicit access to the given
// model resource (independently of the generic model:use:* permissions).
func (s PermissionSet) HasModelAccess(modelID, kind string) bool {
	if s.owner {
		return true
	}
	_, ok := s.grants[grantKey(kind, modelID)]
	return ok
}

// HasAnyGrant reports whether the set contains at least one model grant of the
// given kind. Pass an empty string to match any kind.
func (s PermissionSet) HasAnyGrant(kind string) bool {
	if s.owner {
		return true
	}
	for key := range s.grants {
		if kind == "" {
			return true
		}
		if len(key) > len(kind) && key[:len(kind)] == kind {
			return true
		}
	}
	return false
}
