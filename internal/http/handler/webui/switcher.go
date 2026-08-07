package webui

import (
	"net/http"

	"github.com/a-h/templ"
	common "github.com/xolo-gateway/xolo/internal/http/handler/webui/common/component"
)

// getSwitcherFragment serves the filtered list of the context switcher. The
// popover renders the full list on page load; this endpoint only answers the
// search box, which is why it returns a fragment rather than a page.
func (h *Handler) getSwitcherFragment(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	query := r.URL.Query()

	entries := common.SwitcherEntries(ctx, common.Context(query.Get("context")), query.Get("slug"))
	entries = common.FilterSwitcherEntries(entries, query.Get("q"))

	templ.Handler(common.SwitcherList(entries)).ServeHTTP(w, r)
}
