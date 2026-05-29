package component

import (
	"github.com/a-h/templ"
	"github.com/bornholm/xolo/internal/core/model"
)

// AppLayoutVModel defines the view model for the app layout.
type AppLayoutVModel struct {
	User            model.User
	IsAdmin         bool
	SelectedItem    string
	Breadcrumbs     []BreadcrumbItem
	Version         string
	APIDocURL       string
	NavigationItems func(AppLayoutVModel) templ.Component
	FooterItems     func(AppLayoutVModel) templ.Component
	AdminSubtitle   string
	HomeLink        string
	// FullBleed removes padding from the main content area and disables overflow-auto,
	// useful for full-height canvas views like the pipeline editor.
	FullBleed bool
}

// BreadcrumbItem represents a single item in the breadcrumb navigation.
type BreadcrumbItem struct {
	Label string
	Href  string
}
