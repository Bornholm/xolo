package webui

import (
	"net/http"

	"github.com/bornholm/xolo/internal/http/handler/webui/common"
)

func (h *Handler) getInactiveUserPage(w http.ResponseWriter, r *http.Request) {
	common.HandleError(w, r, common.NewError("forbidden", "Votre compte est inactif. Veuillez contacter un administrateur.", http.StatusForbidden))
}
