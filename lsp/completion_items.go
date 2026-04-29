package lsp

import "github.com/arcgolabs/plano/schema"

func formCompletionItem(spec schema.FormSpec) CompletionItem {
	return CompletionItem{
		Label:         spec.Name,
		Kind:          CompletionForm,
		Detail:        "form",
		Documentation: formatFormSpec(spec),
	}
}

func formatFormSpec(spec schema.FormSpec) string {
	body := "```plano\n" + spec.Name + " { ... }\n```"
	if spec.Docs == "" {
		return body
	}
	return body + "\n\n" + spec.Docs
}

func formatFieldSpec(formKind string, field schema.FieldSpec) string {
	typ := "any"
	if field.Type != nil {
		typ = field.Type.String()
	}
	body := "```plano\n" + formKind + "." + field.Name + ": " + typ + "\n```"
	if field.Docs == "" {
		return body
	}
	return body + "\n\n" + field.Docs
}

func detailWithType(label string, typ schema.Type) string {
	if typ == nil {
		return label
	}
	return label + " " + typ.String()
}
