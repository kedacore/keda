package otto

type resultKind int

const (
	_ resultKind = iota
	resultReturn
	resultBreak
	resultContinue
)

type result struct {
	value  Value
	target string
	kind   resultKind
}

func newReturnResult(value Value) result {
	return result{kind: resultReturn, value: value, target: ""}
}

func newContinueResult(target string) result {
	return result{kind: resultContinue, value: emptyValue, target: target}
}

func newBreakResult(target string) result {
	return result{kind: resultBreak, value: emptyValue, target: target}
}
