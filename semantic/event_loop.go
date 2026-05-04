package semantic

import "jayess-go/ast"

type eventLoopQueue int

const (
	eventLoopQueueNone eventLoopQueue = iota
	eventLoopQueueMicrotask
	eventLoopQueueTimer
	eventLoopQueueCancellation
)

func eventLoopQueueForCall(name string) eventLoopQueue {
	switch name {
	case "queueMicrotask":
		return eventLoopQueueMicrotask
	case "setTimeout", "setInterval":
		return eventLoopQueueTimer
	case "clearTimeout", "clearInterval":
		return eventLoopQueueCancellation
	default:
		return eventLoopQueueNone
	}
}

func analyzeEventLoopCall(expr *ast.CallExpression) error {
	switch eventLoopQueueForCall(expr.Callee) {
	case eventLoopQueueMicrotask, eventLoopQueueTimer:
		if len(expr.Arguments) == 0 {
			return errorAt(expr, "%s requires callback", expr.Callee)
		}
	case eventLoopQueueCancellation:
		if len(expr.Arguments) == 0 {
			return errorAt(expr, "%s requires scheduled handle", expr.Callee)
		}
	}
	return nil
}
