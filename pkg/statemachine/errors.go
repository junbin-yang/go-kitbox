package statemachine

import "fmt"

var (
	// ErrInvalidTransition 当状态转换不被允许时返回
	ErrInvalidTransition = fmt.Errorf("invalid transition")

	// ErrTransitionDenied 当守卫拒绝转换时返回
	ErrTransitionDenied = fmt.Errorf("transition denied by guard")

	// ErrStateNotFound 当状态不存在时返回
	ErrStateNotFound = fmt.Errorf("state not found")

	// ErrEventNotFound 当事件不存在时返回
	ErrEventNotFound = fmt.Errorf("event not found")

	// ErrDuplicateTransition 当转换规则已存在时返回
	ErrDuplicateTransition = fmt.Errorf("duplicate transition")
)
