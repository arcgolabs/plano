package lsp

import (
	"context"
	"encoding/json"
	"fmt"

	"go.lsp.dev/jsonrpc2"
)

func decodeParams(params []byte, target any) error {
	if len(params) == 0 {
		return nil
	}
	if err := json.Unmarshal(params, target); err != nil {
		return fmt.Errorf("decode params: %w", err)
	}
	return nil
}

func replyCall[T any, R any](ctx context.Context, req *jsonrpc2.Request, params *T, fn func(context.Context, *T) (R, error)) (any, error) {
	if err := decodeParams(req.Params(), params); err != nil {
		return nil, err
	}
	return fn(ctx, params)
}

func replyNotify[T any](ctx context.Context, req *jsonrpc2.Request, params *T, fn func(context.Context, *T) error) (any, error) {
	if err := decodeParams(req.Params(), params); err != nil {
		return nil, err
	}
	return nil, fn(ctx, params)
}
