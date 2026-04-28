package compiler

import (
	"fmt"

	"github.com/arcgolabs/collectionx/list"
	"github.com/arcgolabs/plano/ast"
	"github.com/arcgolabs/plano/schema"
)

func (s *compileState) buildActionCall(current *ast.CallStmt, locals *env) (Call, ActionSpec, bool) {
	call := Call{Name: current.Callee.String(), Pos: current.Pos(), End: current.End()}
	args, ok := s.evalCallStatementArgs(current, locals)
	if !ok {
		return Call{}, ActionSpec{}, false
	}
	call.Args = *list.NewList(args...)
	spec, ok := s.compiler.actions.Get(call.Name)
	if !ok {
		s.diags.AddError(current.Pos(), current.End(), fmt.Sprintf("unknown action %q", call.Name))
		return Call{}, ActionSpec{}, false
	}
	if err := validateArity("action", call.Name, spec.MinArgs, spec.MaxArgs, call.Args.Len()); err != nil {
		s.diags.AddError(current.Pos(), current.End(), err.Error())
		return Call{}, ActionSpec{}, false
	}
	if spec.Validate != nil {
		if err := spec.Validate(call.Args); err != nil {
			s.diags.AddError(current.Pos(), current.End(), err.Error())
			return Call{}, ActionSpec{}, false
		}
	}
	return call, spec, true
}

func (s *compileState) evalCallStatementArgs(current *ast.CallStmt, locals *env) ([]any, bool) {
	args := make([]any, 0, len(current.Args))
	for _, arg := range current.Args {
		value, err := s.evalExpr(arg, locals)
		if err != nil {
			s.diags.AddError(arg.Pos(), arg.End(), err.Error())
			return nil, false
		}
		args = append(args, value)
	}
	return args, true
}

func (s *compileState) lowerActionCall(current *ast.CallStmt, scopeID string, call Call, spec ActionSpec) HIRCall {
	var resultType schema.Type = schema.TypeAny
	if callCheck, ok := s.callCheck(current.Pos(), current.End()); ok {
		scopeID = callCheck.ScopeID
		resultType = callCheck.Result
	}
	hirArgs := list.NewListWithCapacity[HIRArg](call.Args.Len())
	for idx, arg := range call.Args.Values() {
		hirArgs.Add(HIRArg{
			Type:  signatureArgType(idx, spec.ArgTypes, spec.VariadicType),
			Value: arg,
		})
	}
	return HIRCall{
		Name:    call.Name,
		ScopeID: scopeID,
		Args:    *hirArgs,
		Result:  resultType,
		Pos:     current.Pos(),
		End:     current.End(),
	}
}
