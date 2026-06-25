package webui

import (
	"net/http"

	"github.com/bornholm/xolo/internal/http/handler/webui/common"
)

func (h *Handler) getInactiveUserPage(w http.ResponseWriter, r *http.Request) {
	common.HandleError(w, r, common.NewError("forbidden", "Votre compte est inactif. Veuillez contacter un administrateur.", http.StatusForbidden))
}

func (h *Handler) getForbiddenPage(w http.ResponseWriter, r *http.Request) {
	common.HandleError(w, r, common.NewError("forbidden", "Vous n'avez pas les permissions nécessaires pour accéder à cette page.", http.StatusForbidden))
}
