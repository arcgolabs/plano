package compiler

import (
	"github.com/arcgolabs/collectionx/mapping"
	"github.com/arcgolabs/plano/ast"
	"github.com/arcgolabs/plano/schema"
)

var builtinTypes = func() *mapping.Map[string, schema.Type] {
	types := mapping.NewMap[string, schema.Type]()
	types.Set("string", schema.TypeString)
	types.Set("int", schema.TypeInt)
	types.Set("float", schema.TypeFloat)
	types.Set("bool", schema.TypeBool)
	types.Set("duration", schema.TypeDuration)
	types.Set("size", schema.TypeSize)
	types.Set("path", schema.TypePath)
	types.Set("any", schema.TypeAny)
	return types
}()

func convertTypeExpr(node ast.TypeExpr) schema.Type {
	switch current := node.(type) {
	case *ast.SimpleType:
		if typ, ok := builtinTypes.Get(current.Name.Name); ok {
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
