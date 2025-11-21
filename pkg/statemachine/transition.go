package statemachine

// Transition 定义状态转换规则
type Transition struct {
	From         State          // 源状态
	To           State          // 目标状态
	Event        Event          // 触发事件
	Guard        GuardFunc      // 守卫条件
	OnTransition TransitionFunc // 转换时回调
}

// transitionKey 唯一标识一个转换
type transitionKey struct {
	from  State
	event Event
}
