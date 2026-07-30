package component

import (
	"context"
	"strconv"

	commonComp "github.com/bornholm/xolo/internal/http/handler/webui/common/component"
)

// usersPageURL builds the link of a pagination entry, carrying the search term
// over so paging through a filtered list keeps the filter.
func usersPageURL(ctx context.Context, vmodel UsersPageVModel, page int) string {
	values := []string{"page", strconv.Itoa(page)}
	if vmodel.Search != "" {
		values = append(values, "q", vmodel.Search)
	}

	return commonComp.BaseURLString(ctx,
		commonComp.WithPath("/admin/users"),
		commonComp.WithValues(values...),
	)
}
