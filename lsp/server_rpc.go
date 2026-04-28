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

func replyCall[T any, R any](ctx context.Context, reply jsonrpc2.Replier, req jsonrpc2.Request, params *T, fn func(context.Context, *T) (R, error)) error {
	if err := decodeParams(req.Params(), params); err != nil {
		return reply(ctx, nil, err)
	}
	result, err := fn(ctx, params)
	return reply(ctx, result, err)
}

func replyNotify[T any](ctx context.Context, reply jsonrpc2.Replier, req jsonrpc2.Request, params *T, fn func(context.Context, *T) error) error {
	if err := decodeParams(req.Params(), params); err != nil {
		return reply(ctx, nil, err)
	}
	return reply(ctx, nil, fn(ctx, params))
}
