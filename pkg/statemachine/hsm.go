package statemachine

import (
	"context"
	"sync"
)

// HSM 层次状态机实现
type HSM struct {
	mu          sync.RWMutex
	current     State
	initial     State
	transitions map[transitionKey]*Transition
	onEnter     map[State]ActionFunc
	onExit      map[State]ActionFunc
	parent      map[State]State // 状态的父状态
	children    map[State][]State // 状态的子状态
}

// NewHSM 创建新的层次状态机
func NewHSM(initial State) *HSM {
	return &HSM{
		current:     initial,
		initial:     initial,
		transitions: make(map[transitionKey]*Transition),
		onEnter:     make(map[State]ActionFunc),
		onExit:      make(map[State]ActionFunc),
		parent:      make(map[State]State),
		children:    make(map[State][]State),
	}
}

// AddState 添加子状态
func (h *HSM) AddState(child, parent State) {
	h.mu.Lock()
	defer h.mu.Unlock()

	h.parent[child] = parent
	h.children[parent] = append(h.children[parent], child)
}

// AddTransition 添加状态转换规则
func (h *HSM) AddTransition(from, to State, event Event) error {
	h.mu.Lock()
	defer h.mu.Unlock()

	key := transitionKey{from: from, event: event}
	if _, exists := h.transitions[key]; exists {
		return ErrDuplicateTransition
	}

	h.transitions[key] = &Transition{
		From:  from,
		To:    to,
		Event: event,
	}
	return nil
}

// SetOnEnter 设置状态进入时的回调
func (h *HSM) SetOnEnter(state State, action ActionFunc) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.onEnter[state] = action
}

// SetOnExit 设置状态退出时的回调
func (h *HSM) SetOnExit(state State, action ActionFunc) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.onExit[state] = action
}

// Current 返回当前状态
func (h *HSM) Current() State {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return h.current
}

// Can 检查是否可以触发事件
func (h *HSM) Can(event Event) bool {
	h.mu.RLock()
	defer h.mu.RUnlock()

	// 检查当前状态及其所有父状态
	state := h.current
	for {
		key := transitionKey{from: state, event: event}
		if _, exists := h.transitions[key]; exists {
			return true
		}

		parent, hasParent := h.parent[state]
		if !hasParent {
			break
		}
		state = parent
	}

	return false
}

// Trigger 触发事件进行状态转换
func (h *HSM) Trigger(ctx context.Context, event Event) error {
	h.mu.Lock()
	defer h.mu.Unlock()

	// 查找转换规则（支持继承）
	var trans *Transition
	state := h.current
	for {
		key := transitionKey{from: state, event: event}
		if t, exists := h.transitions[key]; exists {
			trans = t
			break
		}

		parent, hasParent := h.parent[state]
		if !hasParent {
			return ErrInvalidTransition
		}
		state = parent
	}

	// 检查守卫条件
	if trans.Guard != nil && !trans.Guard(ctx, h.current, trans.To) {
		return ErrTransitionDenied
	}

	from := h.current

	// 执行退出回调（从当前状态到共同祖先）
	exitStates := h.getExitPath(from, trans.To)
	for _, s := range exitStates {
		if exitFn, ok := h.onExit[s]; ok {
			if err := exitFn(ctx, s); err != nil {
				return err
			}
		}
	}

	// 执行转换回调
	if trans.OnTransition != nil {
		if err := trans.OnTransition(ctx, from, trans.To); err != nil {
			return err
		}
	}

	// 更新状态
	h.current = trans.To

	// 执行进入回调（从共同祖先到目标状态）
	enterStates := h.getEnterPath(from, trans.To)
	for _, s := range enterStates {
		if enterFn, ok := h.onEnter[s]; ok {
			if err := enterFn(ctx, s); err != nil {
				return err
			}
		}
	}

	return nil
}

// Reset 重置到初始状态
func (h *HSM) Reset() error {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.current = h.initial
	return nil
}

// getExitPath 获取退出路径
func (h *HSM) getExitPath(from, to State) []State {
	fromAncestors := h.getAncestors(from)
	toAncestors := h.getAncestors(to)

	// 找到共同祖先
	commonIdx := 0
	for i := 0; i < len(fromAncestors) && i < len(toAncestors); i++ {
		if fromAncestors[len(fromAncestors)-1-i] == toAncestors[len(toAncestors)-1-i] {
			commonIdx = i + 1
		} else {
			break
		}
	}

	// 返回需要退出的状态
	if commonIdx == 0 {
		return fromAncestors
	}
	return fromAncestors[:len(fromAncestors)-commonIdx]
}

// getEnterPath 获取进入路径
func (h *HSM) getEnterPath(from, to State) []State {
	fromAncestors := h.getAncestors(from)
	toAncestors := h.getAncestors(to)

	// 找到共同祖先
	commonIdx := 0
	for i := 0; i < len(fromAncestors) && i < len(toAncestors); i++ {
		if fromAncestors[len(fromAncestors)-1-i] == toAncestors[len(toAncestors)-1-i] {
			commonIdx = i + 1
		} else {
			break
		}
	}

	// 返回需要进入的状态（逆序）
	enterStates := toAncestors[:len(toAncestors)-commonIdx]
	for i := 0; i < len(enterStates)/2; i++ {
		enterStates[i], enterStates[len(enterStates)-1-i] = enterStates[len(enterStates)-1-i], enterStates[i]
	}
	return enterStates
}

// getAncestors 获取状态的所有祖先（包括自己）
func (h *HSM) getAncestors(state State) []State {
	ancestors := []State{state}
	current := state
	for {
		parent, hasParent := h.parent[current]
		if !hasParent {
			break
		}
		ancestors = append(ancestors, parent)
		current = parent
	}
	return ancestors
}
