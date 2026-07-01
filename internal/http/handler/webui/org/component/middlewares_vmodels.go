package component

import (
	"github.com/bornholm/xolo/internal/core/model"
	common "github.com/bornholm/xolo/internal/http/handler/webui/common/component"
)

type MiddlewaresPageVModel struct {
	common.AppLayoutVModel
	Org         model.Organization
	Middlewares []model.Middleware
	Success     string
	Error       string
}

// MiddlewareTargetOption is a selectable model (LLM or virtual) a middleware can target.
type MiddlewareTargetOption struct {
	Kind  string // model.ModelRefKindLLM | model.ModelRefKindVirtual
	ID    string
	Label string
}

type MiddlewareFormVModel struct {
	common.AppLayoutVModel
	Org          model.Organization
	Middleware   model.Middleware
	IsNew        bool
	Name         string
	Description  string
	Enabled      bool
	Priority     int
	AppliesToAll bool
	Options      []MiddlewareTargetOption
	Selected     map[string]bool // "kind\x00id" -> checked
	Error        string
}

// TargetKey builds the option value/key for a target model. It uses ":" as a
// separator (HTML-form-safe, unlike a NUL byte) — neither the kind (llm|virtual)
// nor the id (xid) contains a colon.
func TargetKey(kind, id string) string {
	return kind + ":" + id
}
