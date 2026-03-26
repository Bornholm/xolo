package org

import (
	"log/slog"
	"net/http"

	"github.com/a-h/templ"
	"github.com/bornholm/xolo/internal/core/model"
	"github.com/bornholm/xolo/internal/core/port"
	httpCtx "github.com/bornholm/xolo/internal/http/context"
	common "github.com/bornholm/xolo/internal/http/handler/webui/common/component"
	"github.com/bornholm/xolo/internal/http/handler/webui/org/component"
	"github.com/pkg/errors"
)

func (h *Handler) getVirtualModelsPage(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	user := httpCtx.User(ctx)
	orgSlug := r.PathValue("orgSlug")
	nav, footer := orgAdminNav(orgSlug)

	org, err := h.orgFromSlug(ctx, orgSlug)
	if err != nil {
		http.Error(w, "Organization not found", http.StatusNotFound)
		return
	}

	vms, err := h.virtualModelStore.ListVirtualModels(ctx, org.ID())
	if err != nil {
		slog.ErrorContext(ctx, "could not list virtual models", slog.Any("error", err))
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}

	baseURL := httpCtx.BaseURL(ctx)

	vmodel := component.VirtualModelsPageVModel{
		Org:           org,
		VirtualModels: vms,
		BaseURL:       baseURL.String(),
		Success:       r.URL.Query().Get("success"),
		Error:         r.URL.Query().Get("error"),
		AppLayoutVModel: common.AppLayoutVModel{
			User:         user,
			SelectedItem: "org-" + orgSlug + "-virtual-models",
			Breadcrumbs: []common.BreadcrumbItem{
				{Label: org.Name(), Href: "/orgs/" + orgSlug + "/usage"},
				{Label: "Modèles virtuels", Href: ""},
			},
			NavigationItems: nav,
			FooterItems:     footer,
		},
	}

	templ.Handler(component.VirtualModelsPage(vmodel)).ServeHTTP(w, r)
}

func (h *Handler) getNewVirtualModelPage(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	user := httpCtx.User(ctx)
	orgSlug := r.PathValue("orgSlug")
	nav, footer := orgAdminNav(orgSlug)

	org, err := h.orgFromSlug(ctx, orgSlug)
	if err != nil {
		http.Error(w, "Organization not found", http.StatusNotFound)
		return
	}

	vmodel := component.VirtualModelFormVModel{
		Org:   org,
		IsNew: true,
		AppLayoutVModel: common.AppLayoutVModel{
			User:         user,
			SelectedItem: "org-" + orgSlug + "-virtual-models",
			Breadcrumbs: []common.BreadcrumbItem{
				{Label: org.Name(), Href: "/orgs/" + orgSlug + "/usage"},
				{Label: "Modèles virtuels", Href: "/orgs/" + orgSlug + "/admin/virtual-models"},
				{Label: "Nouveau", Href: ""},
			},
			NavigationItems: nav,
			FooterItems:     footer,
		},
	}

	templ.Handler(component.VirtualModelFormPage(vmodel)).ServeHTTP(w, r)
}

func (h *Handler) createVirtualModel(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	orgSlug := r.PathValue("orgSlug")

	org, err := h.orgFromSlug(ctx, orgSlug)
	if err != nil {
		http.Error(w, "Organization not found", http.StatusNotFound)
		return
	}

	if err := r.ParseForm(); err != nil {
		http.Error(w, "Invalid form", http.StatusBadRequest)
		return
	}

	name := r.FormValue("name")
	description := r.FormValue("description")

	if name == "" {
		http.Error(w, "Name is required", http.StatusBadRequest)
		return
	}

	// Check if a virtual model with this name already exists
	existing, err := h.virtualModelStore.GetVirtualModelByName(ctx, org.ID(), name)
	if err == nil && existing != nil {
		http.Redirect(w, r, "/orgs/"+orgSlug+"/admin/virtual-models?error=exists", http.StatusSeeOther)
		return
	}

	vm := model.NewVirtualModel(org.ID(), name, description)

	if err := h.virtualModelStore.CreateVirtualModel(ctx, vm); err != nil {
		slog.ErrorContext(ctx, "could not create virtual model", slog.Any("error", err))
		http.Redirect(w, r, "/orgs/"+orgSlug+"/admin/virtual-models?error=create_failed", http.StatusSeeOther)
		return
	}

	http.Redirect(w, r, "/orgs/"+orgSlug+"/admin/virtual-models?success=created", http.StatusSeeOther)
}

func (h *Handler) getEditVirtualModelPage(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	user := httpCtx.User(ctx)
	orgSlug := r.PathValue("orgSlug")
	modelID := r.PathValue("modelID")
	nav, footer := orgAdminNav(orgSlug)

	org, err := h.orgFromSlug(ctx, orgSlug)
	if err != nil {
		http.Error(w, "Organization not found", http.StatusNotFound)
		return
	}

	vm, err := h.virtualModelStore.GetVirtualModelByID(ctx, model.VirtualModelID(modelID))
	if err != nil {
		if errors.Is(err, port.ErrNotFound) {
			http.NotFound(w, r)
			return
		}
		slog.ErrorContext(ctx, "could not get virtual model", slog.Any("error", err))
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}

	vmodel := component.VirtualModelFormVModel{
		Org:          org,
		VirtualModel: vm,
		IsNew:        false,
		Name:         vm.Name(),
		Description:  vm.Description(),
		AppLayoutVModel: common.AppLayoutVModel{
			User:         user,
			SelectedItem: "org-" + orgSlug + "-virtual-models",
			Breadcrumbs: []common.BreadcrumbItem{
				{Label: org.Name(), Href: "/orgs/" + orgSlug + "/usage"},
				{Label: "Modèles virtuels", Href: "/orgs/" + orgSlug + "/admin/virtual-models"},
				{Label: vm.Name(), Href: ""},
			},
			NavigationItems: nav,
			FooterItems:     footer,
		},
	}

	templ.Handler(component.VirtualModelFormPage(vmodel)).ServeHTTP(w, r)
}

func (h *Handler) updateVirtualModel(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	orgSlug := r.PathValue("orgSlug")
	modelID := r.PathValue("modelID")

	_, err := h.orgFromSlug(ctx, orgSlug)
	if err != nil {
		http.Error(w, "Organization not found", http.StatusNotFound)
		return
	}

	vm, err := h.virtualModelStore.GetVirtualModelByID(ctx, model.VirtualModelID(modelID))
	if err != nil {
		if errors.Is(err, port.ErrNotFound) {
			http.NotFound(w, r)
			return
		}
		slog.ErrorContext(ctx, "could not get virtual model", slog.Any("error", err))
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}

	if err := r.ParseForm(); err != nil {
		http.Error(w, "Invalid form", http.StatusBadRequest)
		return
	}

	description := r.FormValue("description")

	// We know the concrete type from GORM wrapper.
	v, ok := vm.(*model.BaseVirtualModel)
	if !ok {
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}
	v.SetDescription(description)

	if err := h.virtualModelStore.SaveVirtualModel(ctx, vm); err != nil {
		slog.ErrorContext(ctx, "could not save virtual model", slog.Any("error", err))
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}

	http.Redirect(w, r, "/orgs/"+orgSlug+"/admin/virtual-models?success=updated", http.StatusSeeOther)
}

func (h *Handler) deleteVirtualModel(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	orgSlug := r.PathValue("orgSlug")
	modelID := r.PathValue("modelID")

	if err := h.virtualModelStore.DeleteVirtualModel(ctx, model.VirtualModelID(modelID)); err != nil {
		if errors.Is(err, port.ErrNotFound) {
			http.NotFound(w, r)
			return
		}
		slog.ErrorContext(ctx, "could not delete virtual model", slog.Any("error", err))
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}

	http.Redirect(w, r, "/orgs/"+orgSlug+"/admin/virtual-models?success=deleted", http.StatusSeeOther)
}
