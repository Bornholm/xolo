package profile

import (
	"errors"
	"log/slog"
	"net/http"

	"github.com/a-h/templ"
	"github.com/bornholm/go-x/slogx"
	"github.com/bornholm/xolo/internal/core/model"
	"github.com/bornholm/xolo/internal/core/port"
	httpCtx "github.com/bornholm/xolo/internal/http/context"
	common "github.com/bornholm/xolo/internal/http/handler/webui/common/component"
	"github.com/bornholm/xolo/internal/http/handler/webui/profile/component"
)

func (h *Handler) getPersonalModelsPage(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	user := httpCtx.User(ctx)

	vms, err := h.personalVMStore.ListPersonalVirtualModels(ctx, user.ID())
	if err != nil {
		slog.ErrorContext(ctx, "could not list personal virtual models", slogx.Error(err))
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}

	baseURL := httpCtx.BaseURL(ctx)

	vmodel := component.PersonalModelsPageVModel{
		VirtualModels: vms,
		BaseURL:       baseURL.String(),
		Success:       r.URL.Query().Get("success"),
		Error:         r.URL.Query().Get("error"),
		AppLayoutVModel: common.AppLayoutVModel{
			User:         user,
			SelectedItem: "personal-models",
			HomeLink:     "/usage",
			Breadcrumbs: []common.BreadcrumbItem{
				{Label: "Espace personnel", Href: "/usage"},
				{Label: "Mes modèles", Href: ""},
			},
			NavigationItems: func(vmodel common.AppLayoutVModel) templ.Component {
				return common.AppNavigationItems(vmodel)
			},
			FooterItems: func(vmodel common.AppLayoutVModel) templ.Component {
				return common.AppFooterItems(vmodel)
			},
		},
	}

	templ.Handler(component.PersonalModelsPage(vmodel)).ServeHTTP(w, r)
}

func (h *Handler) getNewPersonalModelPage(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	user := httpCtx.User(ctx)

	vmodel := component.PersonalModelFormVModel{
		IsNew: true,
		AppLayoutVModel: common.AppLayoutVModel{
			User:         user,
			SelectedItem: "personal-models",
			HomeLink:     "/usage",
			Breadcrumbs: []common.BreadcrumbItem{
				{Label: "Espace personnel", Href: "/usage"},
				{Label: "Mes modèles", Href: "/profile/personal-models"},
				{Label: "Nouveau", Href: ""},
			},
			NavigationItems: func(vmodel common.AppLayoutVModel) templ.Component {
				return common.AppNavigationItems(vmodel)
			},
			FooterItems: func(vmodel common.AppLayoutVModel) templ.Component {
				return common.AppFooterItems(vmodel)
			},
		},
	}

	templ.Handler(component.PersonalModelFormPage(vmodel)).ServeHTTP(w, r)
}

func (h *Handler) createPersonalModel(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	user := httpCtx.User(ctx)

	if err := r.ParseForm(); err != nil {
		http.Error(w, "Invalid form", http.StatusBadRequest)
		return
	}

	name := r.FormValue("name")
	description := r.FormValue("description")

	if name == "" {
		http.Redirect(w, r, "/profile/personal-models?error=create_failed", http.StatusSeeOther)
		return
	}

	existing, err := h.personalVMStore.GetPersonalVirtualModelByName(ctx, user.ID(), name)
	if err == nil && existing != nil {
		http.Redirect(w, r, "/profile/personal-models?error=exists", http.StatusSeeOther)
		return
	}

	vm := model.NewPersonalVirtualModel(user.ID(), name, description)
	if err := h.personalVMStore.CreatePersonalVirtualModel(ctx, vm); err != nil {
		slog.ErrorContext(ctx, "could not create personal virtual model", slogx.Error(err))
		http.Redirect(w, r, "/profile/personal-models?error=create_failed", http.StatusSeeOther)
		return
	}

	http.Redirect(w, r, "/profile/personal-models?success=created", http.StatusSeeOther)
}

func (h *Handler) getEditPersonalModelPage(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	user := httpCtx.User(ctx)
	vmID := r.PathValue("vmID")

	vm, err := h.personalVMStore.GetPersonalVirtualModelByID(ctx, model.PersonalVirtualModelID(vmID))
	if err != nil {
		if errors.Is(err, port.ErrNotFound) {
			http.NotFound(w, r)
			return
		}
		slog.ErrorContext(ctx, "could not get personal virtual model", slogx.Error(err))
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}

	if vm.UserID() != user.ID() {
		http.NotFound(w, r)
		return
	}

	vmodel := component.PersonalModelFormVModel{
		VirtualModel: vm,
		IsNew:        false,
		Name:         vm.Name(),
		Description:  vm.Description(),
		AppLayoutVModel: common.AppLayoutVModel{
			User:         user,
			SelectedItem: "personal-models",
			HomeLink:     "/usage",
			Breadcrumbs: []common.BreadcrumbItem{
				{Label: "Espace personnel", Href: "/usage"},
				{Label: "Mes modèles", Href: "/profile/personal-models"},
				{Label: vm.Name(), Href: ""},
			},
			NavigationItems: func(vmodel common.AppLayoutVModel) templ.Component {
				return common.AppNavigationItems(vmodel)
			},
			FooterItems: func(vmodel common.AppLayoutVModel) templ.Component {
				return common.AppFooterItems(vmodel)
			},
		},
	}

	templ.Handler(component.PersonalModelFormPage(vmodel)).ServeHTTP(w, r)
}

func (h *Handler) updatePersonalModel(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	user := httpCtx.User(ctx)
	vmID := r.PathValue("vmID")

	vm, err := h.personalVMStore.GetPersonalVirtualModelByID(ctx, model.PersonalVirtualModelID(vmID))
	if err != nil {
		if errors.Is(err, port.ErrNotFound) {
			http.NotFound(w, r)
			return
		}
		slog.ErrorContext(ctx, "could not get personal virtual model", slogx.Error(err))
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}

	if vm.UserID() != user.ID() {
		http.NotFound(w, r)
		return
	}

	if err := r.ParseForm(); err != nil {
		http.Error(w, "Invalid form", http.StatusBadRequest)
		return
	}

	description := r.FormValue("description")

	v, ok := vm.(*model.BasePersonalVirtualModel)
	if !ok {
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}
	v.SetDescription(description)

	if err := h.personalVMStore.SavePersonalVirtualModel(ctx, vm); err != nil {
		slog.ErrorContext(ctx, "could not save personal virtual model", slogx.Error(err))
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}

	http.Redirect(w, r, "/profile/personal-models?success=updated", http.StatusSeeOther)
}

func (h *Handler) deletePersonalModel(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	user := httpCtx.User(ctx)
	vmID := r.PathValue("vmID")

	vm, err := h.personalVMStore.GetPersonalVirtualModelByID(ctx, model.PersonalVirtualModelID(vmID))
	if err != nil {
		if errors.Is(err, port.ErrNotFound) {
			http.NotFound(w, r)
			return
		}
		slog.ErrorContext(ctx, "could not get personal virtual model", slogx.Error(err))
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}

	if vm.UserID() != user.ID() {
		http.NotFound(w, r)
		return
	}

	if err := h.personalVMStore.DeletePersonalVirtualModel(ctx, model.PersonalVirtualModelID(vmID)); err != nil {
		slog.ErrorContext(ctx, "could not delete personal virtual model", slogx.Error(err))
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}

	http.Redirect(w, r, "/profile/personal-models?success=deleted", http.StatusSeeOther)
}

func (h *Handler) getPersonalPipelineEditorPage(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	user := httpCtx.User(ctx)
	vmID := r.PathValue("vmID")

	vm, err := h.personalVMStore.GetPersonalVirtualModelByID(ctx, model.PersonalVirtualModelID(vmID))
	if err != nil {
		if errors.Is(err, port.ErrNotFound) {
			http.NotFound(w, r)
			return
		}
		slog.ErrorContext(ctx, "could not get personal virtual model", slogx.Error(err))
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}

	if vm.UserID() != user.ID() {
		http.NotFound(w, r)
		return
	}

	baseURL := httpCtx.BaseURL(ctx)

	vmodel := component.PersonalModelEditorVModel{
		VM:      vm,
		APIBase: baseURL.String(),
		AppLayoutVModel: common.AppLayoutVModel{
			User:         user,
			SelectedItem: "personal-models",
			HomeLink:     "/usage",
			FullBleed:    true,
			Breadcrumbs: []common.BreadcrumbItem{
				{Label: "Espace personnel", Href: "/usage"},
				{Label: "Mes modèles", Href: "/profile/personal-models"},
				{Label: vm.Name(), Href: ""},
			},
			NavigationItems: func(vmodel common.AppLayoutVModel) templ.Component {
				return common.AppNavigationItems(vmodel)
			},
			FooterItems: func(vmodel common.AppLayoutVModel) templ.Component {
				return common.AppFooterItems(vmodel)
			},
		},
	}

	templ.Handler(component.PersonalModelEditorPage(vmodel)).ServeHTTP(w, r)
}
