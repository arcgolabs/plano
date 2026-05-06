package compiler

import (
	"go/token"
	"strconv"

	"github.com/arcgolabs/collectionx/list"
	"github.com/arcgolabs/plano/diag"
	"github.com/arcgolabs/plano/schema"
)

func (c *checker) checkSignature(
	kind string,
	name string,
	minArgs int,
	maxArgs int,
	paramTypes list.List[schema.Type],
	variadicType schema.Type,
	argTypes []schema.Type,
	pos token.Pos,
	end token.Pos,
) {
	if err := validateArity(kind, name, minArgs, maxArgs, len(argTypes)); err != nil {
		c.diagnostics.AddError(pos, end, err.Error())
		return
	}
	for idx, argType := range argTypes {
		want := signatureArgType(idx, paramTypes, variadicType)
		if want == nil {
			continue
		}
		if !isTypeAssignable(want, argType) {
			c.diagnostics.AddErrorCode(
				diag.CodeTypeMismatch,
				pos,
				end,
				typeMismatchError(kind+" argument "+strconv.Itoa(idx+1), want, argType).Error(),
			)
		}
	}
}
