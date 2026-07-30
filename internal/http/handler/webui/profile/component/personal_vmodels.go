package component

import (
	"github.com/bornholm/xolo/internal/core/model"
	common "github.com/bornholm/xolo/internal/http/handler/webui/common/component"
)

type PersonalModelsPageVModel struct {
	common.AppLayoutVModel
	VirtualModels []model.PersonalVirtualModel
	// Selected is the model whose pipeline the right pane previews. It is the
	// first of the list unless `?vm=` names another, exactly as on the
	// organisation screen.
	Selected model.PersonalVirtualModel
	BaseURL  string
	Success       string
	Error         string
}

type PersonalModelFormVModel struct {
	common.AppLayoutVModel
	VirtualModel model.PersonalVirtualModel
	IsNew        bool
	Name         string
	Description  string
	Error        string
}

type PersonalModelEditorVModel struct {
	common.AppLayoutVModel
	VM      model.PersonalVirtualModel
	APIBase string
}
