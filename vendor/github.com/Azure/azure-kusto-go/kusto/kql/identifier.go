package kql

import "fmt"

func (b *Builder) AddDatabase(database string) *Builder {
	return b.addBase(stringConstant(fmt.Sprintf("%s(%s)", "database", QuoteString(database, false))))
}

func (b *Builder) AddTable(table string) *Builder {
	return b.addBase(stringConstant(NormalizeName(table)))
}

func (b *Builder) AddKeyword(keyword string) *Builder {
	if RequiresQuoting(keyword) {
		panic("Invalid keyword. Cannot add a keyword that requires escaping.")
	}
	return b.addBase(stringConstant(keyword))
}

func (b *Builder) AddColumn(column string) *Builder {
	return b.addBase(stringConstant(NormalizeName(column)))
}

func (b *Builder) AddFunction(function string) *Builder {
	return b.addBase(stringConstant(NormalizeName(function)))
}

// NormalizeName normalizes a string in order to be used safely in the engine - given "query" will produce [\"query\"].
func NormalizeName(name string) string {
	if name == "" {
		return name
	}

	if !RequiresQuoting(name) {
		return name
	}

	return "[" + QuoteString(name, false) + "]"
}
