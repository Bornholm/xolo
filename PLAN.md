# Xolo — MVP Implementation Plan

## Vision

Xolo is a self-hosted, multi-tenant LLM gateway. It sits between end-users and LLM providers (OpenAI, Mistral, OpenRouter, vLLM-compatible endpoints…), enforcing authentication, routing, quota and usage tracking — all managed through a web UI.

The proxy surface is the OpenAI-compatible API exposed by `github.com/bornholm/genai/proxy`, already wired in the backbone. Everything else is built around it.

---

## Tenancy Model

```
Organization  (tenant root — owns providers, models, quotas)
  └── Team     (sub-group — members inherit org providers, can have sub-quotas)
        └── User (member of one or more orgs/teams)
```

Roles per membership:
- **org:owner** — full control, can manage other admins
- **org:admin** — providers, models, teams, quotas, members
- **team:admin** — team members, team quotas
- **member** — use the proxy within their quota

A user can belong to multiple organizations with different roles. The global `admin` role (already in the backbone) is the super-admin with cross-org access.

---

## Domain Models (new, additions to existing `core/model/`)

```
Organization
  ID, Slug, Name, Description, Active, CreatedAt, UpdatedAt

Team
  ID, OrgID, Name, Description, CreatedAt, UpdatedAt

Membership      (user ↔ org or user ↔ team)
  ID, UserID, OrgID, TeamID (nullable), Role, CreatedAt

Provider        (LLM backend defined by an org admin)
  ID, OrgID, Name, Type (openai|mistral|openrouter|yzma), BaseURL, APIKey (encrypted), Active, CreatedAt, UpdatedAt

Model           (proxy route: what users call it → what the provider sees)
  ID, ProviderID, OrgID, ProxyName, RealModel, Description, Enabled,
  PromptCostPer1KTokens int64,      // microcents (1 microcent = $0.000001)
  CompletionCostPer1KTokens int64,  // microcents
  CreatedAt, UpdatedAt

Quota           (applies to org, team, or user — polymorphic)
  ID, Scope (org|team|user), ScopeID,
  DailyBudget   *int64,   // microcents; nil = unlimited
  MonthlyBudget *int64,   // microcents; nil = unlimited
  YearlyBudget  *int64,   // microcents; nil = unlimited
  CreatedAt, UpdatedAt

UsageRecord     (one row per proxy call)
  ID, UserID, OrgID, TeamID, ProviderID, ModelID, ProxyModelName,
  AuthTokenID string,  // which API key made the call — nullable (web session calls have none)
  PromptTokens, CompletionTokens, TotalTokens, RequestType,
  Cost int64,          // microcents, frozen at recording time from model pricing
  CreatedAt
```

`APIKey` is stored AES-GCM encrypted (key from env `XOLO_SECRET_KEY`). It is never returned in plaintext after creation.

### Why microcents?

Storing monetary values as `int64` microcents (1/100 of a cent, i.e. $0.000001) avoids floating-point precision issues entirely, supports sub-cent per-token prices, and fits comfortably in 64 bits for any realistic usage volume. The UI divides by 1,000,000 to display dollars/euros.

Cost is computed at recording time as:
```
cost = (promptTokens * promptCostPer1K / 1000) + (completionTokens * completionCostPer1K / 1000)
```
Freezing the computed cost in `UsageRecord` means historical spend is unaffected if an org admin later updates a model's pricing.

---

## Proxy Integration Architecture

The genai proxy's `UserID` comes from the `Authorization: Bearer <token>` header — which maps directly to Xolo's existing `AuthToken` model. The full request flow is:

```
Client → POST /v1/chat/completions
  → authn middleware (token → User)           [existing]
  → XoloAuthExtractor (User → ProxyRequest.UserID = user.ID)
  → QuotaEnforcer hook   (check user/team/org quota)
  → OrgModelRouter hook  (resolve ProxyName → Provider + real model, scoped to user's org)
  → LLM provider call
  → UsageTracker hook    (record tokens to DB with org_id, team_id)
```

Three custom hooks are needed, all implementing genai's `proxy.Hook` interfaces:

| Hook | Interface | Role |
|------|-----------|------|
| `OrgModelRouter` | `ModelResolverHook` + `ModelListerHook` | Looks up active Provider+Model for the requesting user's org; instantiates `llm.Client` via `provider.Create()` |
| `XoloQuotaEnforcer` | `PreRequestHook` | Resolves effective budget quota (user → team → org, most restrictive wins); compares `SUM(cost)` for the current period against the limit |
| `XoloUsageTracker` | `PostResponseHook` | Computes call cost from model pricing, records `UsageRecord` to gorm with org/team context |

The `OrgModelRouter` must cache instantiated `llm.Client`s (invalidated when a provider or model is updated) to avoid re-creating TLS connections on every request.

---

## Phase 1 — Organization & Team Management

**Goal:** super-admins and org-owners can create and manage the tenant hierarchy.

### Data layer
- `internal/core/model/` — `Organization`, `Team`, `Membership`
- `internal/core/port/` — `OrgStore` interface:
  ```go
  CreateOrg(ctx, org) error
  GetOrgByID(ctx, id) (Organization, error)
  ListOrgs(ctx, opts) ([]Organization, int64, error)
  SaveOrg(ctx, org) error
  DeleteOrg(ctx, id) error
  // teams
  CreateTeam(ctx, team) error
  GetTeamByID(ctx, id) (Team, error)
  ListTeams(ctx, orgID, opts) ([]Team, int64, error)
  SaveTeam(ctx, team) error
  DeleteTeam(ctx, id) error
  // memberships
  AddMember(ctx, membership) error
  RemoveMember(ctx, membershipID) error
  ListMembers(ctx, orgID, teamID, opts) ([]Membership, int64, error)
  GetUserMemberships(ctx, userID) ([]Membership, error)
  ```
- `internal/adapter/gorm/` — gorm models + migration for org, team, membership tables
- Cache wrapper for org store (`internal/adapter/cache/`)

### HTTP layer — super-admin section (`/admin/`)
New pages, all using templui components exclusively:
- `GET /admin/orgs` — org list (Table, Badge for active/inactive, Button)
- `GET /admin/orgs/new` — create org (Card, Input, Button)
- `POST /admin/orgs` — create org
- `GET /admin/orgs/{id}` — org detail (tabs: Overview, Teams, Members)
- `GET /admin/orgs/{id}/edit` — edit org
- `POST /admin/orgs/{id}/edit`
- `DELETE /admin/orgs/{id}`

### HTTP layer — org-admin section (`/orgs/{orgSlug}/admin/`)
New handler mounted under authn + `authz.HasOrgRole(orgSlug, "org:admin")`:
- `GET /` — org dashboard (placeholder for Phase 4)
- `GET /teams` — team list
- `GET /teams/new` + `POST /teams`
- `GET /teams/{id}` + `GET /teams/{id}/edit` + `POST /teams/{id}/edit`
- `DELETE /teams/{id}`
- `GET /members` — members list with role badge
- `POST /members` — invite (add existing user by email)
- `DELETE /members/{id}`
- Team-level member management: `GET /teams/{id}/members`, `POST`, `DELETE`

### UI components used
`Card`, `Table`, `Badge`, `Button`, `Input`, `Dialog` (confirm delete), `Tabs`, `Breadcrumb`, `Select` (role picker), `Separator`, `Avatar` (user initials).

---

## Phase 2 — Provider & Model Definition

**Goal:** org admins define LLM backends and expose named models to their members.

### Data layer
- `internal/core/model/` — `Provider`, `Model`
- `internal/core/port/` — `ProviderStore` interface:
  ```go
  CreateProvider(ctx, p) error
  GetProviderByID(ctx, id) (Provider, error)
  ListProviders(ctx, orgID, opts) ([]Provider, int64, error)
  SaveProvider(ctx, p) error
  DeleteProvider(ctx, id) error
  // models
  CreateModel(ctx, m) error
  GetModelByID(ctx, id) (Model, error)
  ListModels(ctx, orgID, opts) ([]Model, int64, error)
  ListModelsForUser(ctx, userID) ([]Model, error)   // resolves via user's org memberships
  SaveModel(ctx, m) error
  DeleteModel(ctx, id) error
  ```
- `internal/adapter/gorm/` — provider + model gorm models
- `internal/crypto/` — AES-GCM helpers for API key encryption/decryption

### Proxy hook — `OrgModelRouter`
`internal/adapter/proxy/org_model_router.go`
```go
type OrgModelRouter struct {
  providerStore port.ProviderStore
  userStore     port.UserStore
  priority      int
}

// ResolveModel: reads orgID from req.Metadata["orgID"], looks up ProxyName in that org's Models,
//              decrypts API key, creates llm.Client via provider.Create() on every request (no cache for MVP)
// ListModels:  returns all enabled models for the org identified by req.Metadata["orgID"]
```
Registered with `proxy.WithHook(router)` in `setup/http_server.go`. The `provider.Create()` call from `github.com/bornholm/genai/llm/provider` instantiates the backend using the stored `Type`, `BaseURL`, and decrypted `APIKey`. Import the providers you want to support:
```go
_ "github.com/bornholm/genai/llm/provider/openai"
_ "github.com/bornholm/genai/llm/provider/mistral"
_ "github.com/bornholm/genai/llm/provider/openrouter"
```

> **MVP note:** no client cache is implemented. A new `llm.Client` is instantiated per request. Add an LRU cache (keyed on `modelID`) post-MVP once load warrants it — the hook's internal structure is already isolated to make that swap trivial.

### HTTP layer — provider management under `/orgs/{orgSlug}/admin/`
- `GET /providers` — list (Table with type Badge, active Toggle)
- `GET /providers/new` + `POST /providers`
  - Form fields: Name, Type (Select: openai/mistral/openrouter/yzma), Base URL, API Key (Input type=password), Active (Switch)
  - API key shown masked after save; "Reveal" requires re-authentication
- `GET /providers/{id}/edit` + `POST /providers/{id}/edit`
- `POST /providers/{id}/test` — **test connection**: makes a cheap probe call (list models or a minimal completion) using the stored credentials; returns success/failure inline without leaving the page (htmx swap). Shown as a "Test" button on both the create form and the edit page. This is a fast feedback loop to catch bad API keys or wrong base URLs before users encounter errors.
- `DELETE /providers/{id}`
- `GET /providers/{id}/models` — model list for this provider
- `GET /providers/{id}/models/new` + `POST /providers/{id}/models`
  - Form fields: Proxy Name (what users pass as `model`), Real Model name, Description, Enabled (Switch)
  - Pricing fields: Prompt cost per 1K tokens (Input, displayed in $/€ per 1K, stored as microcents), Completion cost per 1K tokens
  - Helper text links to the provider's public pricing page
- `GET /providers/{id}/models/{modelID}/edit` + `POST` + `DELETE`

### UI components used
`Card`, `Table`, `Badge`, `Button`, `Input`, `Select`, `Switch`, `Dialog`, `Tooltip` (show what "proxy name" means), `Code` (example API call showing the proxy name), `Separator`.

---

## Phase 3 — User API Key Management

**Goal:** users can create and revoke their own API keys to call the proxy, set a TTL on them, and see per-key usage.

### Model changes

**`AuthToken`** (core model `internal/core/model/user.go` + gorm model `internal/adapter/gorm/user.go`):

```
AuthToken
  existing: ID, OwnerID, Label, Value, CreatedAt, UpdatedAt
  new:      ExpiresAt *time.Time   // nil = never expires
            OrgID     string       // which org this key is scoped to
```

`ExpiresAt` and `OrgID` are added to both the `model.AuthToken` interface and `BaseAuthToken`, and to the gorm `AuthToken` struct.

**Why `OrgID` on `AuthToken` is required:** a user who belongs to multiple orgs would otherwise create ambiguity in `OrgModelRouter` — it cannot know which org's providers, models, and quotas to apply from `userID` alone. Scoping the key to an org resolves routing deterministically. The token creation form includes an **Organization** select field.

**Expiry enforcement in `FindAuthToken`** (gorm user store):
```go
// after loading the token from DB:
if t.ExpiresAt != nil && time.Now().After(*t.ExpiresAt) {
    return nil, errors.WithStack(port.ErrNotFound)  // treat expired as absent
}
```
Expired tokens are invisible to the proxy without needing a background cleanup job. A separate scheduled cleanup can periodically `DELETE FROM auth_tokens WHERE expires_at < now()` to keep the table lean, but it is not required for correctness.

### Proxy auth extractor
`internal/adapter/proxy/auth_extractor.go`

Wraps the raw bearer token lookup to return the user ID **and** stash the resolved token ID in the request metadata, so the usage tracker can link records to the specific key:

```go
func XoloAuthExtractor(userStore port.UserStore) proxy.AuthExtractor {
    return func(r *http.Request) (string, error) {
        // 1. extract raw Bearer token from Authorization header
        // 2. userStore.FindAuthToken(ctx, raw) — returns ErrNotFound if missing or expired
        // 3. stash authToken.ID() in r's context or return it via req.Metadata in a PreRequestHook
        //    → use a context key so XoloUsageTracker can retrieve it
        // 4. return authToken.Owner().ID() as the UserID
    }
}
```

Because `proxy.AuthExtractor` only returns a `userID` string, the token ID and org ID are carried via a request-scoped context value set by a thin `PreRequestHook` (priority 0, runs first) that reads the raw header and stores `authTokenID` and `orgID` in `req.Metadata`.

This is passed as `proxy.WithAuthExtractor(...)` in `setup/http_server.go`.

### HTTP layer — profile section (`/profile/`)

- `GET /profile/tokens` — token list with inline usage summary per key:

  | Name | Organization | Expires | Requests (30 d) | Spend (30 d) | Status | Actions |
  |------|-------------|---------|-----------------|--------------|--------|---------|
  | prod-key | Acme Corp | 2025-12-31 | 142 | $1.84 | Active | Revoke |
  | test-key | Beta Inc | Never | 8 | $0.02 | Active | Revoke |
  | old-key | Acme Corp | 2024-01-01 | — | — | Expired | Delete |

  The "Requests" and "Spend" columns come from `UsageStore.AggregateUsage` filtered by `AuthTokenID` for the last 30 days. Expired tokens show a muted row with a strikethrough badge. Tokens are grouped or filtered by org when the user has multiple memberships.

- `POST /profile/tokens` — create token form:
  - **Organization** (Select — user's org memberships; required)
  - **Name** (Input, required)
  - **Expires** (DatePicker, optional — clearing it means "never")
  - On success: show the raw token value **once** in an `Alert` with a `CopyButton`. It cannot be retrieved again.

- `DELETE /profile/tokens/{id}` — revoke (Dialog confirmation)

- `GET /profile/tokens/{id}/usage` — per-key usage detail page:
  - Period selector (Select: 7d / 30d / 90d / all time)
  - Summary cards: total requests, total spend, avg cost/request
  - Chart: spend per day (line Chart)
  - Table: individual calls (model, prompt tokens, completion tokens, cost, timestamp)

### UsageStore filter addition

`UsageFilter` (Phase 4) gains `AuthTokenID *string` so all aggregation queries can be scoped to a single key.

### UI components used
`Card`, `Table`, `Button`, `Input`, `DatePicker`, `Alert` (one-time token reveal with `CopyButton`), `Badge` (Active / Expiring Soon / Expired), `Dialog` (confirm revoke), `Separator`, `Chart`, `Select` (period), `Progress`.

---

## Phase 4 — Usage Tracking & Dashboards

**Goal:** every proxy call is recorded; usage is visible at user, team, and org level.

### Data layer
- `internal/core/port/` — `UsageStore` interface (persisted, multi-tenant):
  ```go
  Record(ctx, record UsageRecord) error
  QueryUsage(ctx, filter UsageFilter) ([]UsageRecord, error)
  AggregateUsage(ctx, filter UsageFilter, groupBy []string) ([]UsageAggregate, error)
  // UsageFilter: UserID, OrgID, TeamID, ModelID, AuthTokenID, Since, Until, RequestType
  //   AuthTokenID *string — when set, scopes results to a single API key
  // UsageAggregate: GroupKey map[string]string, TotalTokens, TotalRequests,
  //                 PromptTokens, CompletionTokens, TotalCost int64 (microcents)
  ```
- `internal/adapter/gorm/usage_store.go` — gorm implementation (raw SQL aggregates: `SUM(cost)`, `SUM(prompt_tokens)`, etc.)

### Proxy hook — `XoloUsageTracker`
`internal/adapter/proxy/usage_tracker.go`

Implements `proxy.PostResponseHook`. At call time, it:
1. Resolves `req.UserID` → user's org and team memberships
2. Looks up the matched `Model` record (passed via `req.Metadata["modelID"]` set by `OrgModelRouter`)
3. Reads `req.Metadata["authTokenID"]` (set by the auth extractor pre-hook) — may be empty for web-session requests
4. Computes `cost = (promptTokens * model.PromptCostPer1KTokens / 1000) + (completionTokens * model.CompletionCostPer1KTokens / 1000)`
5. Writes a `UsageRecord` with all foreign keys, the frozen cost, and `AuthTokenID` (nullable)

### HTTP layer — dashboards

**User dashboard** `GET /profile/usage`
- Period selector (Select: today/7d/30d/this month/this year)
- Summary cards: spend this period (formatted currency), requests count, budget remaining
- Progress bar: spend vs. effective budget (daily / monthly / yearly, whichever applies)
- Chart: spend per day (Chart component, line)
- Table: recent calls (model, prompt tokens, completion tokens, cost, timestamp)

**Team dashboard** `GET /orgs/{orgSlug}/teams/{teamID}/usage`
- Same structure, aggregated for all team members
- Table breakdown per user (Avatar + name, spend, request count)

**Org dashboard** `GET /orgs/{orgSlug}/admin/usage`
- Summary cards: total spend, total requests, active users, active models
- Chart: spend per day grouped by model (stacked bar)
- Table: breakdown per team and per user (spend, requests)
- Table: breakdown per model/provider (spend, tokens, avg cost/request)

### UI components used
`Card` (metric summary), `Chart` (line/bar for time-series, stacked bar for model breakdown), `Table`, `Select` (period), `Badge`, `Tabs` (switch between by-user / by-model / by-team views), `Separator`, `Avatar`.

---

## Phase 5 — Quota Management

**Goal:** org admins set token and request limits; the proxy enforces them with appropriate HTTP 429 responses.

### Data layer
- `internal/core/port/` — `QuotaStore`:
  ```go
  SetQuota(ctx, quota Quota) error
  GetQuota(ctx, scope, scopeID) (Quota, error)
  ResolveEffectiveQuota(ctx, userID) (Quota, error)  // merges user → team → org, most restrictive non-nil wins
  ```
- `internal/adapter/gorm/quota_store.go`

### Quota resolution rules

> **MVP scope:** only org-level and user-level quotas are enforced. Team-level quotas are deferred — org + user coverage addresses the majority of real-world use cases, and team quota enforcement adds a cross-membership join on every proxy request. Team quotas can be added post-MVP by extending the `Quota` model and `ResolveEffectiveQuota`.

`ResolveEffectiveQuota` compares non-nil budget values across user and org — taking the **minimum** at each period granularity independently:

```
effectiveDaily   = min(user.DailyBudget,   org.DailyBudget)
effectiveMonthly = min(user.MonthlyBudget, org.MonthlyBudget)
effectiveYearly  = min(user.YearlyBudget,  org.YearlyBudget)
```

A nil value means "no limit at this level" — it only constrains when explicitly set. This lets an org set a monthly org-wide cap without needing to touch every user.

### Proxy hook — `XoloQuotaEnforcer`
`internal/adapter/proxy/quota_enforcer.go`

Implements `proxy.PreRequestHook`. For each active period (day/month/year):
1. Gets effective budget via `QuotaStore.ResolveEffectiveQuota(userID)`
2. Queries `UsageStore.AggregateUsage` with `Since = startOf(period)` → `SUM(cost)`
3. If `spentMicrocents >= budgetMicrocents` → returns a `429` `HookResult`

The error message includes which period and scope triggered the limit (e.g. `"monthly team budget exceeded: $48.32 / $50.00"`).

### HTTP layer — quota configuration

**Under `/orgs/{orgSlug}/admin/`:**
- `GET /quota` — org budget form: three optional inputs (daily / monthly / yearly), displayed in currency ($/€), stored as microcents
- `POST /quota`
- `GET /members/{id}/quota` + `POST /members/{id}/quota`

Team-level quota routes (`/teams/{id}/quota`) are deferred to post-MVP.

**Under `/profile/`:**
- `GET /profile/quota` — read-only: effective budget per period + current spend + Progress bars

### UI components used
`Card`, `Input` (currency amount, step=0.01), `Button`, `Progress` (spend vs. budget per period), `Separator`, `Alert` (warn at 80% consumption), `Badge` (shows which level set the effective limit: org/team/user).

---

## Phase 6 — Organization Invitations

**Goal:** org admins can invite new users to join their organization; invited users see pending invitations and can accept or decline them.

### Domain model — `InviteToken`

```
InviteToken
  ID              string           // xid, used as the URL token
  OrgID           string
  TeamID          *string          // nil = org-level only; set = also adds to that team
  Role            string           // membership role granted on accept (e.g. "member")
  InviteeEmail    *string          // nil = open link (anyone); set = targeted to that email
  ExpiresAt       *time.Time       // nil = never expires
  MaxUses         *int             // nil = unlimited (typical for open links)
  UsesCount       int              // incremented on each accept
  CreatedByUserID string
  RevokedAt       *time.Time       // nil = still active; set = revoked by admin
  CreatedAt       time.Time
```

Two invite modes:
- **Open link** (`InviteeEmail = nil`): any authenticated user who visits `/join/{token}` can accept. Useful for sharing an org link with a group.
- **Targeted invite** (`InviteeEmail = email`): only the user whose OIDC email matches can accept. The invitation appears on that user's pending invitations page without needing the link.

### Port — `InviteStore`

```go
CreateInvite(ctx, invite InviteToken) error
GetInviteByID(ctx, id string) (InviteToken, error)
ListInvites(ctx, orgID string, opts ListOptions) ([]InviteToken, error)
RevokeInvite(ctx, id string) error
// For pending invitations page:
ListPendingInvitesForEmail(ctx, email string) ([]InviteToken, error)  // non-expired, non-revoked, targeted at email, user not already a member
IncrementInviteUses(ctx, id string) error
```

### Invite validity rules (enforced at accept time)

An invite is valid if **all** of the following hold:
1. `RevokedAt == nil`
2. `ExpiresAt == nil || time.Now().Before(*ExpiresAt)`
3. `MaxUses == nil || UsesCount < *MaxUses`
4. The accepting user is not already a member of the org (idempotent: return success without error)
5. For targeted invites: the accepting user's email matches `InviteeEmail`

### OIDC redirect — `nextURL` session key

The OIDC callback currently hardcodes `http.Redirect(w, r, "/", ...)`. Add support for a post-login redirect stored in the session:

- Any handler that requires authentication and wants to redirect back after login stores the destination in the session under the key `"nextURL"`, then redirects to `/auth/oidc/login`.
- In `handleProviderCallback` (`internal/http/middleware/authn/oidc/provider.go`), after `storeSessionUser`, check the session for `"nextURL"`. If present, clear it from the session and redirect there; otherwise fall back to `"/"`.

This mechanism is used by the join flow and is available to any future handler.

### HTTP flow — open link

```
GET  /join/{token}                 public — validate token, show "Join [Org]" page (Accept / Sign in to accept)
POST /join/{token}/accept          requires auth — validate, create Membership, redirect to org dashboard
```

1. `GET /join/{token}`:
   - Validate token (load, check revoked/expired/maxUses). Show an error page if invalid.
   - If token is valid, render a public page: org name, invited-by user name, role, expiry.
   - If the user is already authenticated, show **Accept** and **Decline** buttons (`POST /join/{token}/accept` / `POST /join/{token}/decline`).
   - If not authenticated, show a **"Sign in to join"** button that stores `nextURL = /join/{token}` in the session and redirects to OIDC login.

2. `POST /join/{token}/accept` (requires auth):
   - Re-validate token.
   - If targeted (`InviteeEmail != nil`), assert `currentUser.Email == *InviteeEmail`.
   - Create `Membership` (org-level; if `TeamID != nil`, also create team membership).
   - `IncrementInviteUses`.
   - Redirect to `/orgs/{orgSlug}/`.

3. `POST /join/{token}/decline` (requires auth):
   - For open links: simply redirect away (no state change — the link remains valid for others).
   - For targeted links: mark the invite as declined by the user (optional MVP simplification: just redirect away without recording decline).

### HTTP layer — admin invite management (`/orgs/{orgSlug}/admin/invites`)

- `GET /invites` — table of all invites: target email (or "Open link"), role, team, uses, expires, status (Active / Expired / Revoked). Actions: Copy link, Revoke.
- `GET /invites/new` + `POST /invites` — create invite form:
  - **Target email** (Input, optional — leave blank for open link)
  - **Role** (Select: member / team:admin / org:admin)
  - **Team** (Select, optional — org-level if blank)
  - **Expires** (DatePicker, optional)
  - **Max uses** (Input number, optional — default unlimited; set to 1 for single-use targeted invites)
  - On success: show the generated join URL in an `Alert` with a `CopyButton`.
- `DELETE /invites/{id}` — revoke (Dialog confirmation)

### HTTP layer — pending invitations page (`/profile/invitations`)

Visible to any authenticated user with an active account. Shows targeted invites where `InviteeEmail` matches the user's email and the user is not yet a member:

| Organization | Role | Team | Invited by | Expires | Actions |
|--------------|------|------|------------|---------|---------|
| Acme Corp | member | — | alice@acme.com | 2026-04-01 | Accept · Decline |
| Beta Inc | org:admin | Engineering | bob@beta.com | Never | Accept · Decline |

- `POST /profile/invitations/{inviteID}/accept` — same validation logic as the join flow; on success redirect to the org dashboard.
- `POST /profile/invitations/{inviteID}/decline` — marks invite as declined for this user (or just redirects without state — MVP simplification).

A simple **"Invitations"** link in the profile navigation is sufficient for MVP. A notification badge showing the pending count is deferred — it requires an extra DB query on every page load for the layout.

### File structure additions

```
internal/
  core/
    model/
      invite.go
    port/
      invite_store.go
  adapter/
    gorm/
      invite.go           ← gorm model
      invite_store.go
  http/
    handler/
      webui/
        join/             ← public join flow (/join/{token})
          handler.go
          component/
        org/
          invites.go      ← admin invite management
        profile/
          invitations.go  ← pending invitations page
  setup/
    invite_store.go
```

### UI components used
`Card`, `Table`, `Badge` (Active / Expired / Revoked), `Button`, `Input`, `DatePicker`, `Select`, `Alert` (generated link with `CopyButton`), `Dialog` (confirm revoke), `Separator`, `Avatar`.

### Build Order Summary addition

| Phase | Deliverable | Unblocks |
|-------|-------------|----------|
| 6 | InviteToken domain + join flow + admin UI + pending invitations page | User self-onboarding |

---

## Cross-cutting Concerns

### Database — SQLite and PostgreSQL first-class support

Both backends are supported from day one. The backend is selected automatically from the DSN prefix; no explicit type configuration is needed.

#### Driver detection (`internal/setup/gorm_database.go`)

```go
func dialectorFromDSN(dsn string) (gorm.Dialector, dbKind) {
    if strings.HasPrefix(dsn, "postgres://") ||
       strings.HasPrefix(dsn, "postgresql://") ||
       strings.HasPrefix(dsn, "host=") {
        return postgres.Open(dsn), dbKindPostgres
    }
    return gormlite.Open(dsn), dbKindSQLite
}
```

`dbKind` is an internal enum used to branch on dialect-specific setup steps below.

#### SQLite-specific setup (only when `dbKindSQLite`)

```go
internalDB.SetMaxOpenConns(1)   // WAL mode + single writer avoids BUSY
db.Exec("PRAGMA journal_mode=wal; PRAGMA foreign_keys=on; PRAGMA busy_timeout=5000")
```

#### PostgreSQL-specific setup (only when `dbKindPostgres`)

```go
internalDB.SetMaxOpenConns(25)
internalDB.SetMaxIdleConns(10)
internalDB.SetConnMaxLifetime(5 * time.Minute)
// foreign keys are enforced by default; no extra PRAGMA
```

#### Database-agnostic retry predicate (`internal/adapter/gorm/retry.go`)

The current `withRetry` signature takes `...sqlite3.ErrorCode`, which is SQLite-specific. Replace it with a `func(error) bool` retryability predicate injected at store construction:

```go
type Store struct {
    getDatabase func(ctx context.Context) (*gorm.DB, error)
    isRetryable func(error) bool
}
```

Each dialect provides its own predicate:

```go
// SQLite (ncruces/go-sqlite3)
func sqliteRetryable(err error) bool {
    var e *sqlite3.Error
    return errors.As(err, &e) &&
        (e.Code() == sqlite3.BUSY || e.Code() == sqlite3.LOCKED)
}

// PostgreSQL (jackc/pgx via gorm.io/driver/postgres)
func postgresRetryable(err error) bool {
    var e *pgconn.PgError
    return errors.As(err, &e) &&
        (e.Code == "40001" || e.Code == "40P01") // serialization failure, deadlock
}
```

`NewStore` accepts the predicate; `getGormStoreFromConfig` passes the right one based on `dbKind`. The rest of the store code is unchanged.

#### Portable gorm model guidelines

To keep models compatible with both backends without raw SQL:

| Avoid | Use instead |
|-------|-------------|
| `gorm:"type:text"` SQLite implicit | `gorm:"type:varchar(255)"` — portable |
| `int` primary keys (SQLite `INTEGER`) | `string` PKs (xid — already used) |
| SQLite `AUTOINCREMENT` | xid-generated string IDs (already used) |
| `gorm:"type:blob"` | `gorm:"type:bytea"` for PG / `gorm:"type:blob"` for SQLite → use `gorm:"serializer:bytes"` |
| `time.Time` without timezone | Use `gorm:"type:timestamptz"` on PG fields, `time.Time` maps to `DATETIME` on SQLite |
| Raw `db.Exec("PRAGMA …")` | Always guard with `dbKind` check |

All new gorm models added in Phases 1–5 must follow these rules.

#### Migrations

`AutoMigrate` is used for the MVP. Known limitations:
- SQLite: cannot drop or rename columns — `AutoMigrate` is additive only. Acceptable for MVP since the schema is being defined now.
- PostgreSQL: `AutoMigrate` creates tables and adds columns but does not drop or alter existing ones either — same additive-only behaviour.

For post-MVP production deployments, replace `AutoMigrate` with [goose](https://github.com/pressly/goose) or [golang-migrate](https://github.com/golang-migrate/migrate) with explicit versioned SQL files. The `createGetDatabase` helper in `internal/adapter/gorm/database.go` is the single place to make that swap.

#### Dependencies to add (`go.mod`)

```
gorm.io/driver/postgres            # PostgreSQL dialector (uses pgx v5 under the hood)
github.com/jackc/pgconn            # for pgconn.PgError in retry predicate
```

SQLite remains `ncruces/go-sqlite3` + `gormlite` (CGO-free, WASM-based — keeps the binary portable).

#### Environment variable examples

```bash
# SQLite (default, zero-config)
XOLO_STORAGE_DATABASE_DSN=data.sqlite

# PostgreSQL
XOLO_STORAGE_DATABASE_DSN=postgres://xolo:secret@localhost:5432/xolo?sslmode=disable
```

### Encryption
`XOLO_SECRET_KEY` (32-byte hex, required) used for AES-GCM encryption of provider API keys. Implement in `internal/crypto/` alongside the existing `rand.go`.

### Authorization middleware additions
`internal/http/middleware/authz/` needs new assertion functions:
- `HasOrgRole(orgSlug, role)` — checks user's `Membership`
- `HasTeamRole(orgSlug, teamID, role)` — checks team membership
These compose with the existing `authz.Middleware(onForbidden, ...)` pattern.

### Navigation structure
The app layout sidebar (`AppLayoutVModel.NavigationItems`) adapts based on the user's memberships. A user who is an org admin in one org and a plain member in another sees different nav items per org context. The active org context is carried in the URL path (`/orgs/{orgSlug}/`).

### Org context switcher
A user in multiple orgs needs a way to switch between them. Implement as a `Select` (or `DropdownMenu`) in the top navigation bar displaying the current org name, with a list of the user's other org memberships. Selecting an org navigates to `/orgs/{slug}/`. This component is rendered in the shared app layout and populated from the user's `Membership` list loaded on each request.

### Bootstrap / first-run
The existing backbone supports email-based role assignment via environment variables. Document the minimal setup to get a working instance:

```bash
# Grant super-admin to the first user who logs in via OIDC
XOLO_HTTP_AUTH_ADMIN_EMAILS=you@example.com

# OIDC provider (example: Gitea)
XOLO_HTTP_AUTH_OIDC_PROVIDERS=gitea
XOLO_HTTP_AUTH_OIDC_GITEA_URL=https://gitea.example.com
XOLO_HTTP_AUTH_OIDC_GITEA_CLIENT_ID=...
XOLO_HTTP_AUTH_OIDC_GITEA_CLIENT_SECRET=...

# Required for provider API key encryption
XOLO_SECRET_KEY=<32-byte hex>
```

First-run flow: super-admin logs in → creates an org at `/admin/orgs/new` → creates providers and models → generates an invite link → shares it with users.

### Empty states
Every list page must handle the zero-item case with a helpful empty state (using templui `EmptyState` or equivalent `Card` with description + CTA):

| Page | Empty state message | CTA |
|------|---------------------|-----|
| `/admin/orgs` | "No organizations yet." | "Create organization" |
| `/orgs/{slug}/admin/providers` | "No providers configured." | "Add provider" |
| `/orgs/{slug}/admin/members` | "No members yet." | "Create invite link" |
| `/profile/tokens` | "No API keys yet." | "Create key" |
| `/profile/invitations` | "No pending invitations." | — |
| `/profile/usage` | "No usage recorded yet." | — |

A user who authenticates but has no org membership at all sees a dedicated **"No organization"** page (not a 404): _"You are not a member of any organization. Check your [invitations](/profile/invitations) or contact an administrator."_

### Proxy error UX
The proxy returns standard HTTP errors. Map them to user-visible messages for API consumers:

| Scenario | HTTP status | `error.message` in OpenAI-compatible response |
|----------|-------------|-----------------------------------------------|
| No API key / invalid key | 401 | `"Invalid or missing API key"` |
| Expired API key | 401 | `"API key has expired"` |
| Daily budget exceeded | 429 | `"Daily budget exceeded: $X.XX / $Y.YY"` |
| Monthly budget exceeded | 429 | `"Monthly budget exceeded: $X.XX / $Y.YY"` |
| Yearly budget exceeded | 429 | `"Yearly budget exceeded: $X.XX / $Y.YY"` |
| Model not found in org | 404 | `"Model 'gpt-4o' not available"` |
| Provider unreachable | 502 | `"Provider error: <upstream message>"` |

All error responses follow the OpenAI error envelope so clients using OpenAI-compatible SDKs parse them correctly:
```json
{"error": {"message": "...", "type": "...", "code": "..."}}
```

### Setup wiring (`internal/setup/http_server.go`)
After all phases, the setup adds:
```go
proxy.NewServer(
  // XoloAuthExtractor resolves the bearer token → userID and stashes authTokenID + orgID in req.Metadata
  proxy.WithAuthExtractor(proxyAdapter.XoloAuthExtractor(userStore)),
  // OrgModelRouter reads orgID from req.Metadata, resolves ProxyName → Provider + real model
  proxy.WithHook(proxyAdapter.NewOrgModelRouter(providerStore, ...)),
  // XoloQuotaEnforcer checks org + user budgets; returns OpenAI-compatible 429 on breach
  proxy.WithHook(proxyAdapter.NewXoloQuotaEnforcer(quotaStore, usageStore, ...)),
  // XoloUsageTracker records cost-frozen UsageRecord with org/user/token attribution
  proxy.WithHook(proxyAdapter.NewXoloUsageTracker(usageStore, ...)),
)
```

---

## File Structure (additions to existing backbone)

```
internal/
  core/
    model/
      organization.go
      team.go
      membership.go
      provider.go          ← LLM provider definition
      model.go             ← proxy model route
      quota.go
      usage.go
    port/
      org_store.go
      provider_store.go
      quota_store.go
      usage_store.go
      invite_store.go
  adapter/
    gorm/
      organization.go      ← gorm model
      team.go
      membership.go
      provider.go          ← encrypted api_key field
      model.go
      quota.go
      usage.go
      invite.go
      org_store.go
      provider_store.go
      quota_store.go
      usage_store.go
      invite_store.go
      retry.go             ← sqliteRetryable / postgresRetryable predicates
    cache/
      org_store.go
    proxy/
      auth_extractor.go    ← XoloAuthExtractor
      org_model_router.go  ← ModelResolverHook + ModelListerHook
      quota_enforcer.go    ← PreRequestHook
      usage_tracker.go     ← PostResponseHook
  crypto/
    aes.go                 ← AES-GCM encrypt/decrypt for API keys
  http/
    handler/
      webui/
        admin/             ← extend: add orgs management
        org/               ← new: org-admin section (/orgs/{slug}/admin/)
          handler.go
          dashboard.go
          teams.go
          members.go
          providers.go
          models.go
          quotas.go
          usage.go
          component/       ← all .templ files, templui only
        profile/           ← extend: tokens tab, usage tab, quota tab, invitations tab
        join/              ← public join flow (/join/{token})
  setup/
    org_store.go
    provider_store.go
    quota_store.go
    usage_store.go
    invite_store.go
    proxy.go               ← assembles proxy.Server with all hooks
    gorm_database.go       ← extend: dialectorFromDSN, per-dialect pool + PRAGMA setup
```

---

## Build Order Summary

| Phase | Deliverable | Unblocks |
|-------|-------------|----------|
| 1 | Org/Team/Membership domain + UI | Phases 2–6 (all scoped to org) |
| 2 | Provider/Model domain + OrgModelRouter | Working proxy calls |
| 3 | API key UI + XoloAuthExtractor | Proxy auth, usage attribution |
| 4 | UsageRecord + tracker hook + dashboards | Quota enforcement |
| 5 | Quota domain + enforcer hook + UI | Full quota enforcement |
| 6 | InviteToken domain + join flow + admin UI + pending invitations page | User self-onboarding |
