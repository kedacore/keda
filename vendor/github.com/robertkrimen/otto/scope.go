package otto

// An ECMA-262 ExecutionContext.
type scope struct {
	lexical  stasher
	variable stasher
	this     *object
	outer    *scope
	frame    frame
	depth    int
	eval     bool
}

func newScope(lexical stasher, variable stasher, this *object) *scope {
	return &scope{
		lexical:  lexical,
		variable: variable,
		this:     this,
	}
}
