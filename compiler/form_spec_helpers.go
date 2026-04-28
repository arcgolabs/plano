package compiler

import "github.com/arcgolabs/plano/schema"

func formFieldSpec(spec schema.FormSpec, name string) (schema.FieldSpec, bool) {
	if spec.Fields == nil {
		return schema.FieldSpec{}, false
	}
	return spec.Fields.Get(name)
}

func hasFormField(spec schema.FormSpec, name string) bool {
	_, ok := formFieldSpec(spec, name)
	return ok
}

func allowsNestedFormName(spec schema.FormSpec, name string) bool {
	return spec.NestedForms == nil || spec.NestedForms.Len() == 0 || spec.NestedForms.Contains(name)
}
