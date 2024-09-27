package main

import (
	"fmt"
)

// 定义状态和条件
const (
	ConditionA = 1 << iota // 0001
	ConditionB             // 0010
	ConditionC             // 0100
	ConditionD             // 1000
)

// 检查状态是否满足任意一个条件 0b0101
func checkAnyCondition(state int, conditions []int) bool {
	for _, cond := range conditions {
		if state|cond == state {
			return true
		}
	}
	return false
}

// 检查状态是否同时满足多个条件  0b0101
func checkAllConditions(state int, conditions []int) bool {
	for _, cond := range conditions {
		if state&cond != cond {
			return false
		}
	}
	return true
}

// 检查状态是否不满足某个条件
func checkNotCondition(state int, condition int) bool {
	return state&condition == 0
}

// 更新状态以设置或清除某个条件
func updateState(state int, condition int, set bool) int {
	if set {
		return state | condition
	} else {
		return state & (^condition)
	}
}

func main() {
	var state int = 0b0101 // 初始状态

	// 检查状态是否满足任意一个条件
	var conditions []int = []int{ConditionA, ConditionC}
	fmt.Println("检查是否满足任意一个条件:", checkAnyCondition(state, conditions)) // 输出：true

	// 检查状态是否同时满足多个条件
	fmt.Println("检查是否同时满足多个条件:", checkAllConditions(state, conditions)) // 输出：false

	// 检查状态是否不满足某个条件
	fmt.Println("检查是否不满足ConditionB:", checkNotCondition(state, ConditionB)) // 输出：false

	// 更新状态以设置或清除某个条件
	newState := updateState(state, ConditionC, true)
	fmt.Println("更新状态后的新状态:", fmt.Sprintf("%04b", newState)) // 输出：0111

	newState = updateState(newState, ConditionB, false)
	fmt.Println("再次更新状态后的新状态:", fmt.Sprintf("%04b", newState)) // 输出：0101
}
