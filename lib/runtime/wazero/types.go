package wazero_runtime

import (
	"context"

	"github.com/tetratelabs/wazero/api"
)

type FnParamResultType interface {
	uint32 | uint64 | int32
}

type noArg func(ctx context.Context, m api.Module)
type noArgWithRet[R FnParamResultType] func(ctx context.Context, m api.Module) R

type singleArg[T FnParamResultType] func(ctx context.Context, m api.Module, a T)
type singleArgWithRet[T FnParamResultType, R FnParamResultType] func(ctx context.Context, m api.Module, a T) R

type doubleArg[T FnParamResultType, U FnParamResultType] func(ctx context.Context, m api.Module, a T, b U)
type doubleArgWithRet[T FnParamResultType, U FnParamResultType,
	R FnParamResultType] func(ctx context.Context, m api.Module, a T, b U) R

type tripleArg[T FnParamResultType, U FnParamResultType, V FnParamResultType] func(ctx context.Context, m api.Module,
	a T, b U, c V)
type tripleArgWithRet[T FnParamResultType, U FnParamResultType, V FnParamResultType,
	R FnParamResultType] func(ctx context.Context, m api.Module, a T, b U, c V) R

type quadArgWithRet[T FnParamResultType, U FnParamResultType, V FnParamResultType, W FnParamResultType,
	R FnParamResultType] func(ctx context.Context, m api.Module, a T, b U, c V, d W) R

type quintArgWithRet[T FnParamResultType, U FnParamResultType, V FnParamResultType, W FnParamResultType,
	X FnParamResultType, R FnParamResultType] func(ctx context.Context, m api.Module, a T, b U, c V, d W, e X) R

func noArgFn(f noArg) api.GoModuleFunc {
	return func(ctx context.Context, m api.Module, _ []uint64) {
		f(ctx, m)
	}
}

func singleArgFn[T FnParamResultType](f singleArg[T]) api.GoModuleFunc {
	return func(ctx context.Context, m api.Module, stack []uint64) {
		f(ctx, m, T(stack[0]))
	}
}

func doubleArgFn[T FnParamResultType, U FnParamResultType](f doubleArg[T, U]) api.GoModuleFunc {
	return func(ctx context.Context, m api.Module, stack []uint64) {
		f(ctx, m, T(stack[0]), U(stack[1]))
	}
}

func tripleArgFn[T FnParamResultType, U FnParamResultType, V FnParamResultType](
	f tripleArg[T, U, V]) api.GoModuleFunc {
	return func(ctx context.Context, m api.Module, stack []uint64) {
		f(ctx, m, T(stack[0]), U(stack[1]), V(stack[2]))
	}
}

func noArgWithReturn[R FnParamResultType](f noArgWithRet[R]) api.GoModuleFunc {
	return func(ctx context.Context, m api.Module, stack []uint64) {
		stack[0] = uint64(f(ctx, m))
	}
}

func singleArgWithReturnFn[T FnParamResultType, R FnParamResultType](f singleArgWithRet[T, R]) api.GoModuleFunc {
	return func(ctx context.Context, m api.Module, stack []uint64) {
		stack[0] = uint64(f(ctx, m, T(stack[0])))
	}
}

func doubleArgWithReturnFn[T FnParamResultType, U FnParamResultType, R FnParamResultType](
	f doubleArgWithRet[T, U, R]) api.GoModuleFunc {
	return func(ctx context.Context, m api.Module, stack []uint64) {
		stack[0] = uint64(f(ctx, m, T(stack[0]), U(stack[1])))
	}
}

func tripleArgWithReturnFn[T FnParamResultType, U FnParamResultType, V FnParamResultType, R FnParamResultType](
	f tripleArgWithRet[T, U, V, R]) api.GoModuleFunc {
	return func(ctx context.Context, m api.Module, stack []uint64) {
		stack[0] = uint64(f(ctx, m, T(stack[0]), U(stack[1]), V(stack[2])))
	}
}

func quadArgWithReturnFn[T FnParamResultType, U FnParamResultType, V FnParamResultType, W FnParamResultType,
	R FnParamResultType](f quadArgWithRet[T, U, V, W, R]) api.GoModuleFunc {
	return func(ctx context.Context, m api.Module, stack []uint64) {
		stack[0] = uint64(f(ctx, m, T(stack[0]), U(stack[1]), V(stack[2]), W(stack[3])))
	}
}

func quintArgWithReturnFn[T FnParamResultType, U FnParamResultType, V FnParamResultType, W FnParamResultType,
	X FnParamResultType, R FnParamResultType](f quintArgWithRet[T, U, V, W, X, R]) api.GoModuleFunc {
	return func(ctx context.Context, m api.Module, stack []uint64) {
		stack[0] = uint64(f(ctx, m, T(stack[0]), U(stack[1]), V(stack[2]), W(stack[3]), X(stack[4])))
	}
}
