package join

import (
	"log/slog"
	"net/http"

	"github.com/a-h/templ"
	"github.com/bornholm/xolo/internal/core/model"
	"github.com/bornholm/xolo/internal/core/port"
	httpCtx "github.com/bornholm/xolo/internal/http/context"
	common "github.com/bornholm/xolo/internal/http/handler/webui/common/component"
	"github.com/bornholm/xolo/internal/http/handler/webui/join/component"
	"github.com/bornholm/go-x/slogx"
	"github.com/pkg/errors"
)

type Handler struct {
	mux         *http.ServeMux
	orgStore    port.OrgStore
	inviteStore port.InviteStore
}

// ServeHTTP implements http.Handler.
func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	h.mux.ServeHTTP(w, r)
}

func NewHandler(orgStore port.OrgStore, inviteStore port.InviteStore) *Handler {
	h := &Handler{
		mux:         http.NewServeMux(),
		orgStore:    orgStore,
		inviteStore: inviteStore,
	}

	h.mux.HandleFunc("GET /{tokenID}", h.getJoinPage)
	h.mux.HandleFunc("POST /{tokenID}", h.acceptInvite)

	return h
}

func (h *Handler) getJoinPage(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	user := httpCtx.User(ctx)
	tokenID := r.PathValue("tokenID")

	invite, err := h.inviteStore.GetInviteByID(ctx, model.InviteTokenID(tokenID))
	if err != nil {
		if errors.Is(err, port.ErrNotFound) {
			h.renderError(w, r, user, "Cette invitation est introuvable ou a expiré.")
			return
		}
		slog.ErrorContext(ctx, "could not get invite", slogx.Error(err))
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}

	if !model.IsInviteValid(invite) {
		h.renderError(w, r, user, "Cette invitation n'est plus valide (expirée ou révoquée).")
		return
	}

	baseURL := httpCtx.BaseURL(ctx)
	loginURL := baseURL.JoinPath("/auth/oidc/login").String()

	vmodel := component.JoinPageVModel{
		Invite:   invite,
		LoginURL: loginURL,
		AppLayoutVModel: common.AppLayoutVModel{
			User: user,
			NavigationItems: func(vmodel common.AppLayoutVModel) templ.Component {
				return common.AppNavigationItems(vmodel)
			},
			FooterItems: func(vmodel common.AppLayoutVModel) templ.Component {
				return common.AppFooterItems(vmodel)
			},
		},
	}

	templ.Handler(component.JoinPage(vmodel)).ServeHTTP(w, r)
}

func (h *Handler) acceptInvite(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	user := httpCtx.User(ctx)
	tokenID := r.PathValue("tokenID")

	if user == nil {
		http.Redirect(w, r, "/auth/oidc/login", http.StatusSeeOther)
		return
	}

	invite, err := h.inviteStore.GetInviteByID(ctx, model.InviteTokenID(tokenID))
	if err != nil {
		if errors.Is(err, port.ErrNotFound) {
			h.renderError(w, r, user, "Cette invitation est introuvable ou a expiré.")
			return
		}
		slog.ErrorContext(ctx, "could not get invite", slogx.Error(err))
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}

	if !model.IsInviteValid(invite) {
		h.renderError(w, r, user, "Cette invitation n'est plus valide (expirée ou révoquée).")
		return
	}

	// Check if targeted invite matches current user's email
	if invite.InviteeEmail() != nil && *invite.InviteeEmail() != user.Email() {
		h.renderError(w, r, user, "Cette invitation n'est pas destinée à votre adresse email.")
		return
	}

	// Check if already a member
	already, err := h.orgStore.IsMember(ctx, user.ID(), invite.OrgID())
	if err != nil {
		slog.ErrorContext(ctx, "could not check membership", slogx.Error(err))
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}

	if !already {
		membership := model.NewMembership(user.ID(), invite.OrgID(), invite.Role())
		if err := h.orgStore.AddMember(ctx, membership); err != nil {
			slog.ErrorContext(ctx, "could not add member", slogx.Error(err))
			http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
			return
		}
	}

	if err := h.inviteStore.IncrementInviteUses(ctx, invite.ID()); err != nil {
		slog.WarnContext(ctx, "could not increment invite uses", slogx.Error(err))
	}

	// Targeted invites are single-use by design: delete them after acceptance.
	if invite.InviteeEmail() != nil {
		if err := h.inviteStore.DeleteInvite(ctx, invite.ID()); err != nil {
			slog.WarnContext(ctx, "could not delete targeted invite after acceptance", slogx.Error(err))
		}
	}

	org, err := h.orgStore.GetOrgByID(ctx, invite.OrgID())
	if err != nil {
		http.Redirect(w, r, "/", http.StatusSeeOther)
		return
	}

	vmodel := component.JoinSuccessVModel{
		OrgName: org.Name(),
		OrgSlug: org.Slug(),
		AppLayoutVModel: common.AppLayoutVModel{
			User: user,
			NavigationItems: func(vmodel common.AppLayoutVModel) templ.Component {
				return common.AppNavigationItems(vmodel)
			},
			FooterItems: func(vmodel common.AppLayoutVModel) templ.Component {
				return common.AppFooterItems(vmodel)
			},
		},
	}

	templ.Handler(component.JoinSuccess(vmodel)).ServeHTTP(w, r)
}

func (h *Handler) renderError(w http.ResponseWriter, r *http.Request, user model.User, msg string) {
	vmodel := component.JoinErrorVModel{
		Message: msg,
		AppLayoutVModel: common.AppLayoutVModel{
			User: user,
			NavigationItems: func(vmodel common.AppLayoutVModel) templ.Component {
				return common.AppNavigationItems(vmodel)
			},
			FooterItems: func(vmodel common.AppLayoutVModel) templ.Component {
				return common.AppFooterItems(vmodel)
			},
		},
	}
	w.WriteHeader(http.StatusBadRequest)
	templ.Handler(component.JoinError(vmodel)).ServeHTTP(w, r)
}

var _ http.Handler = &Handler{}
