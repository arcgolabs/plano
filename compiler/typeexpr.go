package compiler

import (
	"github.com/arcgolabs/plano/ast"
	"github.com/arcgolabs/plano/schema"
)

var builtinTypes = map[string]schema.Type{
	"string":   schema.TypeString,
	"int":      schema.TypeInt,
	"float":    schema.TypeFloat,
	"bool":     schema.TypeBool,
	"duration": schema.TypeDuration,
	"size":     schema.TypeSize,
	"path":     schema.TypePath,
	"any":      schema.TypeAny,
}

func convertTypeExpr(node ast.TypeExpr) schema.Type {
	switch current := node.(type) {
	case *ast.SimpleType:
		if typ, ok := builtinTypes[current.Name.Name]; ok {
			return typ
		}
		return schema.NamedType{Name: current.Name.Name}
	case *ast.QualifiedType:
		return schema.NamedType{Name: current.Name.String()}
	case *ast.ListType:
		return schema.ListType{Elem: convertTypeExpr(current.Elem)}
	case *ast.MapType:
		return schema.MapType{Elem: convertTypeExpr(current.Elem)}
	case *ast.RefType:
		return schema.RefType{Kind: current.Target.String()}
	default:
		return nil
	}
}
