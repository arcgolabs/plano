package compiler

import (
	"go/token"

	"github.com/arcgolabs/collectionx/list"
	"github.com/arcgolabs/collectionx/mapping"
	"github.com/arcgolabs/plano/schema"
)

func (b *ArtifactBinding) binding() (*Binding, error) {
	out := emptyBinding()
	if b == nil {
		return out, nil
	}
	out.Files = b.Files
	decodeBindingScopes(out, b.Scopes)
	if err := decodeBindingLocals(out, b.Locals); err != nil {
		return nil, err
	}
	decodeBindingUses(out, b.Uses)
	out.Symbols = decodeArtifactSymbolMap(b.Symbols)
	if err := decodeBindingConsts(out, b.Consts); err != nil {
		return nil, err
	}
	if err := decodeBindingFunctions(out, b.Functions); err != nil {
		return nil, err
	}
	return out, nil
}

func decodeBindingScopes(out *Binding, scopes *mapping.OrderedMap[string, ArtifactScope]) {
	if out == nil || scopes == nil {
		return
	}
	for _, id := range scopes.Keys() {
		item, _ := scopes.Get(id)
		out.Scopes.Set(id, ScopeBinding{
			ID:       item.ID,
			Kind:     item.Kind,
			FormKind: item.FormKind,
			ParentID: item.ParentID,
			Pos:      token.NoPos,
			End:      token.NoPos,
		})
	}
}

func decodeBindingLocals(out *Binding, locals *mapping.OrderedMap[string, ArtifactLocal]) error {
	if out == nil || locals == nil {
		return nil
	}
	for _, id := range locals.Keys() {
		item, _ := locals.Get(id)
		typ, err := item.Type.Type()
		if err != nil {
			return err
		}
		out.Locals.Set(id, LocalBinding{
			ID:      item.ID,
			Name:    item.Name,
			Kind:    item.Kind,
			ScopeID: item.ScopeID,
			Type:    typ,
			Pos:     token.NoPos,
			End:     token.NoPos,
		})
	}
	return nil
}

func decodeBindingUses(out *Binding, uses *mapping.OrderedMap[string, ArtifactUse]) {
	if out == nil || uses == nil {
		return
	}
	for _, id := range uses.Keys() {
		item, _ := uses.Get(id)
		out.Uses.Set(id, NameUse{
			ID:       item.ID,
			Name:     item.Name,
			Kind:     item.Kind,
			ScopeID:  item.ScopeID,
			TargetID: item.TargetID,
			Pos:      token.NoPos,
			End:      token.NoPos,
		})
	}
}

func decodeBindingConsts(out *Binding, consts *mapping.OrderedMap[string, ArtifactConst]) error {
	if out == nil || consts == nil {
		return nil
	}
	for _, name := range consts.Keys() {
		item, _ := consts.Get(name)
		typ, err := item.Type.Type()
		if err != nil {
			return err
		}
		out.Consts.Set(name, ConstBinding{
			Name: item.Name,
			Type: typ,
			Pos:  token.NoPos,
			End:  token.NoPos,
		})
	}
	return nil
}

func decodeBindingFunctions(out *Binding, functions *mapping.OrderedMap[string, ArtifactFunction]) error {
	if out == nil || functions == nil {
		return nil
	}
	for _, name := range functions.Keys() {
		item, _ := functions.Get(name)
		params, err := decodeBindingParams(item.Params)
		if err != nil {
			return err
		}
		result, err := item.Result.Type()
		if err != nil {
			return err
		}
		out.Functions.Set(name, FunctionBinding{
			Name:   item.Name,
			Params: params,
			Result: result,
			Pos:    token.NoPos,
			End:    token.NoPos,
		})
	}
	return nil
}

func decodeBindingParams(items list.List[ArtifactParam]) (list.List[ParamBinding], error) {
	out := list.NewListWithCapacity[ParamBinding](items.Len())
	for index := range items.Len() {
		item, _ := items.Get(index)
		typ, err := item.Type.Type()
		if err != nil {
			return list.List[ParamBinding]{}, err
		}
		out.Add(ParamBinding{Name: item.Name, Type: typ})
	}
	return *out, nil
}

func (c *ArtifactChecks) checks() (*CheckInfo, error) {
	out := emptyChecks()
	if c == nil {
		return out, nil
	}
	if err := decodeExprChecks(out, c.Exprs); err != nil {
		return nil, err
	}
	if err := decodeFieldChecks(out, c.Fields); err != nil {
		return nil, err
	}
	if err := decodeCallChecks(out, c.Calls); err != nil {
		return nil, err
	}
	return out, nil
}

func decodeExprChecks(out *CheckInfo, exprs *mapping.OrderedMap[string, ArtifactExprCheck]) error {
	if out == nil || exprs == nil {
		return nil
	}
	for _, id := range exprs.Keys() {
		item, _ := exprs.Get(id)
		typ, err := item.Type.Type()
		if err != nil {
			return err
		}
		out.Exprs.Set(id, ExprCheck{
			ID:      item.ID,
			Kind:    item.Kind,
			ScopeID: item.ScopeID,
			Type:    typ,
		})
	}
	return nil
}

func decodeFieldChecks(out *CheckInfo, fields *mapping.OrderedMap[string, ArtifactFieldCheck]) error {
	if out == nil || fields == nil {
		return nil
	}
	for _, id := range fields.Keys() {
		item, _ := fields.Get(id)
		expected, err := item.Expected.Type()
		if err != nil {
			return err
		}
		actual, err := item.Actual.Type()
		if err != nil {
			return err
		}
		out.Fields.Set(id, FieldCheck{
			ID:       item.ID,
			FormKind: item.FormKind,
			Field:    item.Field,
			ScopeID:  item.ScopeID,
			Expected: expected,
			Actual:   actual,
		})
	}
	return nil
}

func decodeCallChecks(out *CheckInfo, calls *mapping.OrderedMap[string, ArtifactCallCheck]) error {
	if out == nil || calls == nil {
		return nil
	}
	for _, id := range calls.Keys() {
		item, _ := calls.Get(id)
		args, err := decodeCallCheckArgs(item.Args)
		if err != nil {
			return err
		}
		result, err := item.Result.Type()
		if err != nil {
			return err
		}
		out.Calls.Set(id, CallCheck{
			ID:      item.ID,
			Name:    item.Name,
			ScopeID: item.ScopeID,
			Args:    args,
			Result:  result,
		})
	}
	return nil
}

func decodeCallCheckArgs(items list.List[ArtifactType]) (list.List[schema.Type], error) {
	out := list.NewListWithCapacity[schema.Type](items.Len())
	for index := range items.Len() {
		item, _ := items.Get(index)
		typ, err := item.Type()
		if err != nil {
			return list.List[schema.Type]{}, err
		}
		out.Add(typ)
	}
	return *out, nil
}

func (s ArtifactSymbol) symbol() Symbol {
	return Symbol{Name: s.Name, Kind: s.Kind}
}

func (s *ArtifactSymbol) symbolPtr() *Symbol {
	if s == nil {
		return nil
	}
	symbol := s.symbol()
	return &symbol
}

func decodeArtifactList[W any, T any](items list.List[W], decode func(W) (T, error)) (list.List[T], error) {
	if decode == nil {
		return list.List[T]{}, errNilArtifactListCodec
	}
	out := list.NewListWithCapacity[T](items.Len())
	for index := range items.Len() {
		item, _ := items.Get(index)
		value, err := decode(item)
		if err != nil {
			return list.List[T]{}, err
		}
		out.Add(value)
	}
	return *out, nil
}

func decodeArtifactMapToOrdered[W any, V any](items *mapping.OrderedMap[string, W], decode func(W) (V, error)) (*mapping.OrderedMap[string, V], error) {
	if decode == nil {
		return nil, errNilArtifactMapCodec
	}
	out := mapping.NewOrderedMap[string, V]()
	if items == nil {
		return out, nil
	}
	for _, key := range items.Keys() {
		item, _ := items.Get(key)
		value, err := decode(item)
		if err != nil {
			return nil, err
		}
		out.Set(key, value)
	}
	return out, nil
}
