package templui

import (
	"github.com/a-h/templ"
	"github.com/bornholm/go-x/templx/form"
)

// FieldRenderer provides TemplUI-based field rendering.
type FieldRenderer struct{}

// NewFieldRenderer creates a new TemplUI field renderer.
func NewFieldRenderer() *FieldRenderer {
	return &FieldRenderer{}
}

// RenderField dispatches to the appropriate field component based on field type.
func (r *FieldRenderer) RenderField(ctx form.FieldContext) templ.Component {
	switch ctx.Type {
	case "textarea":
		return Textarea(ctx)
	case "checkbox":
		return Checkbox(ctx)
	case "file":
		return FileInput(ctx)
	case "select":
		return Select(ctx)
	case "hidden":
		return Hidden(ctx)
	default:
		return Input(ctx)
	}
}

// errorClass returns CSS classes for fields with errors.
func errorClass(ctx form.FieldContext) string {
	if ctx.Error != "" {
		return "border-destructive focus-visible:ring-destructive"
	}
	return ""
}

// extraAttrs returns additional HTML attributes for the field.
func extraAttrs(ctx form.FieldContext) templ.Attributes {
	attrs := templ.Attributes{}
	if ctx.Required {
		attrs["required"] = "required"
	}
	// Check readonly in attributes
	if readonly, ok := ctx.Attributes["readonly"].(bool); ok && readonly {
		attrs["readonly"] = "readonly"
	}
	for k, v := range ctx.Attributes {
		attrs[k] = v
	}
	return attrs
}

var _ form.FieldRenderer = &FieldRenderer{}
