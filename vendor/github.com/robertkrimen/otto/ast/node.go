// Package ast declares types representing a JavaScript AST.
//
// # Warning
// The parser and AST interfaces are still works-in-progress (particularly where
// node types are concerned) and may change in the future.
package ast

import (
	"github.com/robertkrimen/otto/file"
	"github.com/robertkrimen/otto/token"
)

// Node is implemented by types that represent a node.
type Node interface {
	Idx0() file.Idx // The index of the first character belonging to the node
	Idx1() file.Idx // The index of the first character immediately after the node
}

// Expression is implemented by types that represent an Expression.
type Expression interface {
	Node
	expression()
}

// ArrayLiteral represents an array literal.
type ArrayLiteral struct {
	Value        []Expression
	LeftBracket  file.Idx
	RightBracket file.Idx
}

// Idx0 implements Node.
func (al *ArrayLiteral) Idx0() file.Idx {
	return al.LeftBracket
}

// Idx1 implements Node.
func (al *ArrayLiteral) Idx1() file.Idx {
	return al.RightBracket + 1
}

// expression implements Expression.
func (*ArrayLiteral) expression() {}

// AssignExpression represents an assignment expression.
type AssignExpression struct {
	Left     Expression
	Right    Expression
	Operator token.Token
}

// Idx0 implements Node.
func (ae *AssignExpression) Idx0() file.Idx {
	return ae.Left.Idx0()
}

// Idx1 implements Node.
func (ae *AssignExpression) Idx1() file.Idx {
	return ae.Right.Idx1()
}

// expression implements Expression.
func (*AssignExpression) expression() {}

// BadExpression represents a bad expression.
type BadExpression struct {
	From file.Idx
	To   file.Idx
}

// Idx0 implements Node.
func (be *BadExpression) Idx0() file.Idx {
	return be.From
}

// Idx1 implements Node.
func (be *BadExpression) Idx1() file.Idx {
	return be.To
}

// expression implements Expression.
func (*BadExpression) expression() {}

// BinaryExpression represents a binary expression.
type BinaryExpression struct {
	Left       Expression
	Right      Expression
	Operator   token.Token
	Comparison bool
}

// Idx0 implements Node.
func (be *BinaryExpression) Idx0() file.Idx {
	return be.Left.Idx0()
}

// Idx1 implements Node.
func (be *BinaryExpression) Idx1() file.Idx {
	return be.Right.Idx1()
}

// expression implements Expression.
func (*BinaryExpression) expression() {}

// BooleanLiteral represents a boolean expression.
type BooleanLiteral struct {
	Literal string
	Idx     file.Idx
	Value   bool
}

// Idx0 implements Node.
func (bl *BooleanLiteral) Idx0() file.Idx {
	return bl.Idx
}

// Idx1 implements Node.
func (bl *BooleanLiteral) Idx1() file.Idx {
	return file.Idx(int(bl.Idx) + len(bl.Literal))
}

// expression implements Expression.
func (*BooleanLiteral) expression() {}

// BracketExpression represents a bracketed expression.
type BracketExpression struct {
	Left         Expression
	Member       Expression
	LeftBracket  file.Idx
	RightBracket file.Idx
}

// Idx0 implements Node.
func (be *BracketExpression) Idx0() file.Idx {
	return be.Left.Idx0()
}

// Idx1 implements Node.
func (be *BracketExpression) Idx1() file.Idx {
	return be.RightBracket + 1
}

// expression implements Expression.
func (*BracketExpression) expression() {}

// CallExpression represents a call expression.
type CallExpression struct {
	Callee           Expression
	ArgumentList     []Expression
	LeftParenthesis  file.Idx
	RightParenthesis file.Idx
}

// Idx0 implements Node.
func (ce *CallExpression) Idx0() file.Idx {
	return ce.Callee.Idx0()
}

// Idx1 implements Node.
func (ce *CallExpression) Idx1() file.Idx {
	return ce.RightParenthesis + 1
}

// expression implements Expression.
func (*CallExpression) expression() {}

// ConditionalExpression represents a conditional expression.
type ConditionalExpression struct {
	Test       Expression
	Consequent Expression
	Alternate  Expression
}

// Idx0 implements Node.
func (ce *ConditionalExpression) Idx0() file.Idx {
	return ce.Test.Idx0()
}

// Idx1 implements Node.
func (ce *ConditionalExpression) Idx1() file.Idx {
	return ce.Alternate.Idx1()
}

// expression implements Expression.
func (*ConditionalExpression) expression() {}

// DotExpression represents a dot expression.
type DotExpression struct {
	Left       Expression
	Identifier *Identifier
}

// Idx0 implements Node.
func (de *DotExpression) Idx0() file.Idx {
	return de.Left.Idx0()
}

// Idx1 implements Node.
func (de *DotExpression) Idx1() file.Idx {
	return de.Identifier.Idx1()
}

// expression implements Expression.
func (*DotExpression) expression() {}

// EmptyExpression represents an empty expression.
type EmptyExpression struct {
	Begin file.Idx
	End   file.Idx
}

// Idx0 implements Node.
func (ee *EmptyExpression) Idx0() file.Idx {
	return ee.Begin
}

// Idx1 implements Node.
func (ee *EmptyExpression) Idx1() file.Idx {
	return ee.End
}

// expression implements Expression.
func (*EmptyExpression) expression() {}

// FunctionLiteral represents a function literal.
type FunctionLiteral struct {
	Body            Statement
	Name            *Identifier
	ParameterList   *ParameterList
	Source          string
	DeclarationList []Declaration
	Function        file.Idx
}

// Idx0 implements Node.
func (fl *FunctionLiteral) Idx0() file.Idx {
	return fl.Function
}

// Idx1 implements Node.
func (fl *FunctionLiteral) Idx1() file.Idx {
	return fl.Body.Idx1()
}

// expression implements Expression.
func (*FunctionLiteral) expression() {}

// Identifier represents an identifier.
type Identifier struct {
	Name string
	Idx  file.Idx
}

// Idx0 implements Node.
func (i *Identifier) Idx0() file.Idx {
	return i.Idx
}

// Idx1 implements Node.
func (i *Identifier) Idx1() file.Idx {
	return file.Idx(int(i.Idx) + len(i.Name))
}

// expression implements Expression.
func (*Identifier) expression() {}

// NewExpression represents a new expression.
type NewExpression struct {
	Callee           Expression
	ArgumentList     []Expression
	New              file.Idx
	LeftParenthesis  file.Idx
	RightParenthesis file.Idx
}

// Idx0 implements Node.
func (ne *NewExpression) Idx0() file.Idx {
	return ne.New
}

// Idx1 implements Node.
func (ne *NewExpression) Idx1() file.Idx {
	if ne.RightParenthesis > 0 {
		return ne.RightParenthesis + 1
	}
	return ne.Callee.Idx1()
}

// expression implements Expression.
func (*NewExpression) expression() {}

// NullLiteral represents a null literal.
type NullLiteral struct {
	Literal string
	Idx     file.Idx
}

// Idx0 implements Node.
func (nl *NullLiteral) Idx0() file.Idx {
	return nl.Idx
}

// Idx1 implements Node.
func (nl *NullLiteral) Idx1() file.Idx {
	return file.Idx(int(nl.Idx) + 4)
}

// expression implements Expression.
func (*NullLiteral) expression() {}

// NumberLiteral represents a number literal.
type NumberLiteral struct {
	Value   interface{}
	Literal string
	Idx     file.Idx
}

// Idx0 implements Node.
func (nl *NumberLiteral) Idx0() file.Idx {
	return nl.Idx
}

// Idx1 implements Node.
func (nl *NumberLiteral) Idx1() file.Idx {
	return file.Idx(int(nl.Idx) + len(nl.Literal))
}

// expression implements Expression.
func (*NumberLiteral) expression() {}

// ObjectLiteral represents an object literal.
type ObjectLiteral struct {
	Value      []Property
	LeftBrace  file.Idx
	RightBrace file.Idx
}

// Idx0 implements Node.
func (ol *ObjectLiteral) Idx0() file.Idx {
	return ol.LeftBrace
}

// Idx1 implements Node.
func (ol *ObjectLiteral) Idx1() file.Idx {
	return ol.RightBrace + 1
}

// expression implements Expression.
func (*ObjectLiteral) expression() {}

// ParameterList represents a parameter list.
type ParameterList struct {
	List    []*Identifier
	Opening file.Idx
	Closing file.Idx
}

// Property represents a property.
type Property struct {
	Value Expression
	Key   string
	Kind  string
}

// RegExpLiteral represents a regular expression literal.
type RegExpLiteral struct {
	Literal string
	Pattern string
	Flags   string
	Value   string
	Idx     file.Idx
}

// Idx0 implements Node.
func (rl *RegExpLiteral) Idx0() file.Idx {
	return rl.Idx
}

// Idx1 implements Node.
func (rl *RegExpLiteral) Idx1() file.Idx {
	return file.Idx(int(rl.Idx) + len(rl.Literal))
}

// expression implements Expression.
func (*RegExpLiteral) expression() {}

// SequenceExpression represents a sequence literal.
type SequenceExpression struct {
	Sequence []Expression
}

// Idx0 implements Node.
func (se *SequenceExpression) Idx0() file.Idx {
	return se.Sequence[0].Idx0()
}

// Idx1 implements Node.
func (se *SequenceExpression) Idx1() file.Idx {
	return se.Sequence[len(se.Sequence)-1].Idx1()
}

// expression implements Expression.
func (*SequenceExpression) expression() {}

// StringLiteral represents a string literal.
type StringLiteral struct {
	Literal string
	Value   string
	Idx     file.Idx
}

// Idx0 implements Node.
func (sl *StringLiteral) Idx0() file.Idx {
	return sl.Idx
}

// Idx1 implements Node.
func (sl *StringLiteral) Idx1() file.Idx {
	return file.Idx(int(sl.Idx) + len(sl.Literal))
}

// expression implements Expression.
func (*StringLiteral) expression() {}

// ThisExpression represents a this expression.
type ThisExpression struct {
	Idx file.Idx
}

// Idx0 implements Node.
func (te *ThisExpression) Idx0() file.Idx {
	return te.Idx
}

// Idx1 implements Node.
func (te *ThisExpression) Idx1() file.Idx {
	return te.Idx + 4
}

// expression implements Expression.
func (*ThisExpression) expression() {}

// UnaryExpression represents a unary expression.
type UnaryExpression struct {
	Operand  Expression
	Operator token.Token
	Idx      file.Idx
	Postfix  bool
}

// Idx0 implements Node.
func (ue *UnaryExpression) Idx0() file.Idx {
	if ue.Postfix {
		return ue.Operand.Idx0()
	}
	return ue.Idx
}

// Idx1 implements Node.
func (ue *UnaryExpression) Idx1() file.Idx {
	if ue.Postfix {
		return ue.Operand.Idx1() + 2 // ++ --
	}
	return ue.Operand.Idx1()
}

// expression implements Expression.
func (*UnaryExpression) expression() {}

// VariableExpression represents a variable expression.
type VariableExpression struct {
	Initializer Expression
	Name        string
	Idx         file.Idx
}

// Idx0 implements Node.
func (ve *VariableExpression) Idx0() file.Idx {
	return ve.Idx
}

// Idx1 implements Node.
func (ve *VariableExpression) Idx1() file.Idx {
	if ve.Initializer == nil {
		return file.Idx(int(ve.Idx) + len(ve.Name))
	}
	return ve.Initializer.Idx1()
}

// expression implements Expression.
func (*VariableExpression) expression() {}

// Statement is implemented by types which represent a statement.
type Statement interface {
	Node
	statement()
}

// BadStatement represents a bad statement.
type BadStatement struct {
	From file.Idx
	To   file.Idx
}

// Idx0 implements Node.
func (bs *BadStatement) Idx0() file.Idx {
	return bs.From
}

// Idx1 implements Node.
func (bs *BadStatement) Idx1() file.Idx {
	return bs.To
}

// expression implements Statement.
func (*BadStatement) statement() {}

// BlockStatement represents a block statement.
type BlockStatement struct {
	List       []Statement
	LeftBrace  file.Idx
	RightBrace file.Idx
}

// Idx0 implements Node.
func (bs *BlockStatement) Idx0() file.Idx {
	return bs.LeftBrace
}

// Idx1 implements Node.
func (bs *BlockStatement) Idx1() file.Idx {
	return bs.RightBrace + 1
}

// expression implements Statement.
func (*BlockStatement) statement() {}

// BranchStatement represents a branch statement.
type BranchStatement struct {
	Label *Identifier
	Idx   file.Idx
	Token token.Token
}

// Idx0 implements Node.
func (bs *BranchStatement) Idx0() file.Idx {
	return bs.Idx
}

// Idx1 implements Node.
func (bs *BranchStatement) Idx1() file.Idx {
	if bs.Label == nil {
		return file.Idx(int(bs.Idx) + len(bs.Token.String()))
	}
	return bs.Label.Idx1()
}

// expression implements Statement.
func (*BranchStatement) statement() {}

// CaseStatement represents a case statement.
type CaseStatement struct {
	Test       Expression
	Consequent []Statement
	Case       file.Idx
}

// Idx0 implements Node.
func (cs *CaseStatement) Idx0() file.Idx {
	return cs.Case
}

// Idx1 implements Node.
func (cs *CaseStatement) Idx1() file.Idx {
	return cs.Consequent[len(cs.Consequent)-1].Idx1()
}

// expression implements Statement.
func (*CaseStatement) statement() {}

// CatchStatement represents a catch statement.
type CatchStatement struct {
	Body      Statement
	Parameter *Identifier
	Catch     file.Idx
}

// Idx0 implements Node.
func (cs *CatchStatement) Idx0() file.Idx {
	return cs.Catch
}

// Idx1 implements Node.
func (cs *CatchStatement) Idx1() file.Idx {
	return cs.Body.Idx1()
}

// expression implements Statement.
func (*CatchStatement) statement() {}

// DebuggerStatement represents a debugger statement.
type DebuggerStatement struct {
	Debugger file.Idx
}

// Idx0 implements Node.
func (ds *DebuggerStatement) Idx0() file.Idx {
	return ds.Debugger
}

// Idx1 implements Node.
func (ds *DebuggerStatement) Idx1() file.Idx {
	return ds.Debugger + 8
}

// expression implements Statement.
func (*DebuggerStatement) statement() {}

// DoWhileStatement represents a do while statement.
type DoWhileStatement struct {
	Test             Expression
	Body             Statement
	Do               file.Idx
	RightParenthesis file.Idx
}

// Idx0 implements Node.
func (dws *DoWhileStatement) Idx0() file.Idx {
	return dws.Do
}

// Idx1 implements Node.
func (dws *DoWhileStatement) Idx1() file.Idx {
	return dws.RightParenthesis + 1
}

// expression implements Statement.
func (*DoWhileStatement) statement() {}

// EmptyStatement represents a empty statement.
type EmptyStatement struct {
	Semicolon file.Idx
}

// Idx0 implements Node.
func (es *EmptyStatement) Idx0() file.Idx {
	return es.Semicolon
}

// Idx1 implements Node.
func (es *EmptyStatement) Idx1() file.Idx {
	return es.Semicolon + 1
}

// expression implements Statement.
func (*EmptyStatement) statement() {}

// ExpressionStatement represents a expression statement.
type ExpressionStatement struct {
	Expression Expression
}

// Idx0 implements Node.
func (es *ExpressionStatement) Idx0() file.Idx {
	return es.Expression.Idx0()
}

// Idx1 implements Node.
func (es *ExpressionStatement) Idx1() file.Idx {
	return es.Expression.Idx1()
}

// expression implements Statement.
func (*ExpressionStatement) statement() {}

// ForInStatement represents a for in statement.
type ForInStatement struct {
	Into   Expression
	Source Expression
	Body   Statement
	For    file.Idx
}

// Idx0 implements Node.
func (fis *ForInStatement) Idx0() file.Idx {
	return fis.For
}

// Idx1 implements Node.
func (fis *ForInStatement) Idx1() file.Idx {
	return fis.Body.Idx1()
}

// expression implements Statement.
func (*ForInStatement) statement() {}

// ForStatement represents a for statement.
type ForStatement struct {
	Initializer Expression
	Update      Expression
	Test        Expression
	Body        Statement
	For         file.Idx
}

// Idx0 implements Node.
func (fs *ForStatement) Idx0() file.Idx {
	return fs.For
}

// Idx1 implements Node.
func (fs *ForStatement) Idx1() file.Idx {
	return fs.Body.Idx1()
}

// expression implements Statement.
func (*ForStatement) statement() {}

// FunctionStatement represents a function statement.
type FunctionStatement struct {
	Function *FunctionLiteral
}

// Idx0 implements Node.
func (fs *FunctionStatement) Idx0() file.Idx {
	return fs.Function.Idx0()
}

// Idx1 implements Node.
func (fs *FunctionStatement) Idx1() file.Idx {
	return fs.Function.Idx1()
}

// expression implements Statement.
func (*FunctionStatement) statement() {}

// IfStatement represents a if statement.
type IfStatement struct {
	Test       Expression
	Consequent Statement
	Alternate  Statement
	If         file.Idx
}

// Idx0 implements Node.
func (is *IfStatement) Idx0() file.Idx {
	return is.If
}

// Idx1 implements Node.
func (is *IfStatement) Idx1() file.Idx {
	if is.Alternate != nil {
		return is.Alternate.Idx1()
	}
	return is.Consequent.Idx1()
}

// expression implements Statement.
func (*IfStatement) statement() {}

// LabelledStatement represents a labelled statement.
type LabelledStatement struct {
	Statement Statement
	Label     *Identifier
	Colon     file.Idx
}

// Idx0 implements Node.
func (ls *LabelledStatement) Idx0() file.Idx {
	return ls.Label.Idx0()
}

// Idx1 implements Node.
func (ls *LabelledStatement) Idx1() file.Idx {
	return ls.Statement.Idx1()
}

// expression implements Statement.
func (*LabelledStatement) statement() {}

// ReturnStatement represents a return statement.
type ReturnStatement struct {
	Argument Expression
	Return   file.Idx
}

// Idx0 implements Node.
func (rs *ReturnStatement) Idx0() file.Idx {
	return rs.Return
}

// Idx1 implements Node.
func (rs *ReturnStatement) Idx1() file.Idx {
	if rs.Argument != nil {
		return rs.Argument.Idx1()
	}
	return rs.Return + 6
}

// expression implements Statement.
func (*ReturnStatement) statement() {}

// SwitchStatement represents a switch statement.
type SwitchStatement struct {
	Discriminant Expression
	Body         []*CaseStatement
	Switch       file.Idx
	Default      int
	RightBrace   file.Idx
}

// Idx0 implements Node.
func (ss *SwitchStatement) Idx0() file.Idx {
	return ss.Switch
}

// Idx1 implements Node.
func (ss *SwitchStatement) Idx1() file.Idx {
	return ss.RightBrace + 1
}

// expression implements Statement.
func (*SwitchStatement) statement() {}

// ThrowStatement represents a throw statement.
type ThrowStatement struct {
	Argument Expression
	Throw    file.Idx
}

// Idx0 implements Node.
func (ts *ThrowStatement) Idx0() file.Idx {
	return ts.Throw
}

// Idx1 implements Node.
func (ts *ThrowStatement) Idx1() file.Idx {
	return ts.Argument.Idx1()
}

// expression implements Statement.
func (*ThrowStatement) statement() {}

// TryStatement represents a try statement.
type TryStatement struct {
	Body    Statement
	Finally Statement
	Catch   *CatchStatement
	Try     file.Idx
}

// Idx0 implements Node.
func (ts *TryStatement) Idx0() file.Idx {
	return ts.Try
}

// Idx1 implements Node.
func (ts *TryStatement) Idx1() file.Idx {
	if ts.Finally != nil {
		return ts.Finally.Idx1()
	}
	return ts.Catch.Idx1()
}

// expression implements Statement.
func (*TryStatement) statement() {}

// VariableStatement represents a variable statement.
type VariableStatement struct {
	List []Expression
	Var  file.Idx
}

// Idx0 implements Node.
func (vs *VariableStatement) Idx0() file.Idx {
	return vs.Var
}

// Idx1 implements Node.
func (vs *VariableStatement) Idx1() file.Idx {
	return vs.List[len(vs.List)-1].Idx1()
}

// expression implements Statement.
func (*VariableStatement) statement() {}

// WhileStatement represents a while statement.
type WhileStatement struct {
	Test  Expression
	Body  Statement
	While file.Idx
}

// Idx0 implements Node.
func (ws *WhileStatement) Idx0() file.Idx {
	return ws.While
}

// Idx1 implements Node.
func (ws *WhileStatement) Idx1() file.Idx {
	return ws.Body.Idx1()
}

// expression implements Statement.
func (*WhileStatement) statement() {}

// WithStatement represents a with statement.
type WithStatement struct {
	Object Expression
	Body   Statement
	With   file.Idx
}

// Idx0 implements Node.
func (ws *WithStatement) Idx0() file.Idx {
	return ws.With
}

// Idx1 implements Node.
func (ws *WithStatement) Idx1() file.Idx {
	return ws.Body.Idx1()
}

// expression implements Statement.
func (*WithStatement) statement() {}

// Declaration is implemented by type which represent declarations.
type Declaration interface {
	declaration()
}

// FunctionDeclaration represents a function declaration.
type FunctionDeclaration struct {
	Function *FunctionLiteral
}

func (*FunctionDeclaration) declaration() {}

// VariableDeclaration represents a variable declaration.
type VariableDeclaration struct {
	List []*VariableExpression
	Var  file.Idx
}

// declaration implements Declaration.
func (*VariableDeclaration) declaration() {}

// Program represents a full program.
type Program struct {
	File            *file.File
	Comments        CommentMap
	Body            []Statement
	DeclarationList []Declaration
}

// Idx0 implements Node.
func (p *Program) Idx0() file.Idx {
	return p.Body[0].Idx0()
}

// Idx1 implements Node.
func (p *Program) Idx1() file.Idx {
	return p.Body[len(p.Body)-1].Idx1()
}
