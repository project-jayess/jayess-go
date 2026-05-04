package lowering

type nonReturnFlowKind int

const (
	nonReturnFlowKindNone nonReturnFlowKind = iota
	nonReturnFlowKindBreak
	nonReturnFlowKindContinue
)

type nonReturnFlow struct {
	kind  nonReturnFlowKind
	label string
}

var (
	nonReturnFlowNone     = nonReturnFlow{kind: nonReturnFlowKindNone}
	nonReturnFlowBreak    = nonReturnFlow{kind: nonReturnFlowKindBreak}
	nonReturnFlowContinue = nonReturnFlow{kind: nonReturnFlowKindContinue}
)

func nonReturnBreakFlow(label string) nonReturnFlow {
	return nonReturnFlow{kind: nonReturnFlowKindBreak, label: label}
}

func nonReturnContinueFlow(label string) nonReturnFlow {
	return nonReturnFlow{kind: nonReturnFlowKindContinue, label: label}
}
