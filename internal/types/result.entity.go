package types

type ResultEntity[T any] struct {
	success bool
	message string
	data    T
	err     error
}

func NewSuccess[T any](data T) *ResultEntity[T] {
	return &ResultEntity[T]{
		success: true,
		data:    data,
	}
}

func NewSuccessWithMessage[T any](data T, message string) *ResultEntity[T] {
	return &ResultEntity[T]{
		success: true,
		message: message,
		data:    data,
	}
}

func NewError[T any](err error) *ResultEntity[T] {
	return &ResultEntity[T]{
		success: false,
		err:     err,
	}
}

func NewErrorWithMessage[T any](err error, message string) *ResultEntity[T] {
	return &ResultEntity[T]{
		success: false,
		message: message,
		err:     err,
	}
}

func NewFailure[T any](message string) *ResultEntity[T] {
	return &ResultEntity[T]{
		success: false,
		message: message,
	}
}

func (r *ResultEntity[T]) IsSuccess() bool {
	return r.success
}

func (r *ResultEntity[T]) IsFailure() bool {
	return !r.success
}

func (r *ResultEntity[T]) GetData() T {
	return r.data
}

func (r *ResultEntity[T]) GetMessage() string {
	return r.message
}

func (r *ResultEntity[T]) GetError() error {

	return r.err
}
