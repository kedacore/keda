package kusto

/*
This file defines our Stmt, Definitions and Parameters types, which are used in Query() to query Kusto.
These provide injection safe querying for data retrieval and insertion.
*/

import (
	"encoding/json"
	"fmt"
	"github.com/Azure/azure-kusto-go/kusto/kql"
	"math/big"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/Azure/azure-kusto-go/kusto/data/types"
	"github.com/Azure/azure-kusto-go/kusto/data/value"
	ilog "github.com/Azure/azure-kusto-go/kusto/internal/log"
	"github.com/Azure/azure-kusto-go/kusto/unsafe"

	"github.com/google/uuid"
)

// stringConstant is an internal type that cannot be created outside the package.  The only two ways to build
// a stringConstant is to pass a string constant or use a local function to build the stringConstant.
// This allows us to enforce the use of constants or strings built with injection protection.
type stringConstant string

// String implements fmt.Stringer.
func (s stringConstant) String() string {
	return string(s)
}

// ParamTypes is a list of parameter types and corresponding type data.
type ParamTypes map[string]ParamType

func (p ParamTypes) clone() ParamTypes {
	c := make(ParamTypes, len(p))
	for k, v := range p {
		c[k] = v
	}
	return c
}

// ParamType provides type and default value information about the query parameter
type ParamType struct {
	// Type is the type of Column type this QueryParam will represent.
	Type types.Column
	// Default is a default value to use if the query doesn't provide this value.
	// The value that can be set is defined by the Type:
	// CTBool must be a bool
	// CTDateTime must be a time.Time
	// CTDynamic cannot have a default value
	// CTGuid must be an uuid.UUID
	// CTInt must be an int32
	// CTLong must be an int64
	// CTReal must be an float64
	// CTString must be a string
	// CTTimespan must be a time.Duration
	// CTDecimal must be a string or *big.Float representing a decimal value
	Default interface{}

	name string
}

func (p ParamType) validate() error {
	if !p.Type.Valid() {
		return fmt.Errorf("the .Type was not a valid value, must be one of the values in this package starting with CT<type name>, was %s", p.Type)
	}
	if p.Default == nil {
		return nil
	}

	switch p.Type {
	case types.Bool:
		if _, ok := p.Default.(bool); !ok {
			return fmt.Errorf("the .Type was %s, but the value was a %T", p.Type, p.Default)
		}
		return nil
	case types.DateTime:
		if _, ok := p.Default.(time.Time); !ok {
			return fmt.Errorf("the .Type was %s, but the value was a %T", p.Type, p.Default)
		}
		return nil
	case types.Dynamic:
		return fmt.Errorf("the .Type was %s, but Dynamic types cannot have default values", p.Type)
	case types.GUID:
		if _, ok := p.Default.(uuid.UUID); !ok {
			return fmt.Errorf("the .Type was %s, but the value was a %T", p.Type, p.Default)
		}
		return nil
	case types.Int:
		if _, ok := p.Default.(int32); !ok {
			return fmt.Errorf("the .Type was %s, but the value was a %T", p.Type, p.Default)
		}
		return nil
	case types.Long:
		if _, ok := p.Default.(int64); !ok {
			return fmt.Errorf("the .Type was %s, but the value was a %T", p.Type, p.Default)
		}
		return nil
	case types.Real:
		if _, ok := p.Default.(float64); !ok {
			return fmt.Errorf("the .Type was %s, but the value was a %T", p.Type, p.Default)
		}
		return nil
	case types.String:
		if _, ok := p.Default.(string); !ok {
			return fmt.Errorf("the .Type was %s, but the value was a %T", p.Type, p.Default)
		}
		return nil
	case types.Timespan:
		if _, ok := p.Default.(time.Duration); !ok {
			return fmt.Errorf("the .Type was %s, but the value was a %T", p.Type, p.Default)
		}
		return nil
	case types.Decimal:
		switch v := p.Default.(type) {
		case string:
			if !value.DecRE.MatchString(v) {
				return fmt.Errorf("string representing decimal does not appear to be a decimal number, was %v", v)
			}
			return nil
		case *big.Float:
			if v == nil {
				return fmt.Errorf("*big.Float type cannot be set to the nil value")
			}
			return nil
		case *big.Int:
			if v == nil {
				return fmt.Errorf("*big.Int type cannot be set to the nil value")
			}
			return nil
		}
		return fmt.Errorf("the .Type was %s, but the value was a %T", p.Type, p.Default)
	}
	return fmt.Errorf("received a field type %q we don't recognize", p.Type)
}

func (p ParamType) string() string {
	switch p.Type {
	case types.Bool:
		if p.Default == nil {
			return p.name + ":bool"
		}
		v := p.Default.(bool)
		return fmt.Sprintf("%s:bool = bool(%v)", p.name, v)
	case types.DateTime:
		if p.Default == nil {
			return p.name + ":datetime"
		}
		v := p.Default.(time.Time)
		return fmt.Sprintf("%s:datetime = datetime(%s)", p.name, v.Format(time.RFC3339Nano))
	case types.Dynamic:
		return p.name + ":dynamic"
	case types.GUID:
		if p.Default == nil {
			return p.name + ":guid"
		}
		v := p.Default.(uuid.UUID)
		return fmt.Sprintf("%s:guid = guid(%s)", p.name, v.String())
	case types.Int:
		if p.Default == nil {
			return p.name + ":int"
		}
		v := p.Default.(int32)
		return fmt.Sprintf("%s:int = int(%d)", p.name, v)
	case types.Long:
		if p.Default == nil {
			return p.name + ":long"
		}
		v := p.Default.(int64)
		return fmt.Sprintf("%s:long = long(%d)", p.name, v)
	case types.Real:
		if p.Default == nil {
			return p.name + ":real"
		}
		v := p.Default.(float64)
		return fmt.Sprintf("%s:real = real(%f)", p.name, v)
	case types.String:
		if p.Default == nil {
			return p.name + ":string"
		}
		v := p.Default.(string)
		return fmt.Sprintf(`%s:string = %s`, p.name, kql.QuoteString(v, false))
	case types.Timespan:
		if p.Default == nil {
			return p.name + ":timespan"
		}
		v := p.Default.(time.Duration)
		return fmt.Sprintf("%s:timespan = timespan(%s)", p.name, value.Timespan{Value: v, Valid: true}.Marshal())
	case types.Decimal:
		if p.Default == nil {
			return p.name + ":decimal"
		}

		var sval string
		switch v := p.Default.(type) {
		case string:
			sval = v
		case *big.Float:
			sval = v.String()
		}
		return fmt.Sprintf("%s:decimal = decimal(%s)", p.name, sval)
	}
	panic("internal bug: ParamType.string() called without a call to .validate()")
}

// Definitions represents definitions of parameters that are substituted for variables in
// a Kusto Query. This provides both variable substitution in a Stmt and provides protection against
// SQL-like injection attacks.
// See https://docs.microsoft.com/en-us/azure/kusto/query/queryparametersstatement?pivots=azuredataexplorer
// for internals. This object is not thread-safe and passing it as an argument to a function will create a
// copy that will share the internal state with the original.
type Definitions struct {
	m ParamTypes
}

// NewDefinitions is the constructor for Definitions.
func NewDefinitions() Definitions {
	return Definitions{}
}

// IsZero indicates if the Definitions object is the zero type.
func (p Definitions) IsZero() bool {
	if p.m == nil || len(p.m) == 0 {
		return true
	}
	return false
}

// With returns a copy of the Definitions object with the parameters names and types defined in "types".
func (p Definitions) With(types ParamTypes) (Definitions, error) {
	for name, param := range types {
		if strings.Contains(name, " ") {
			return p, fmt.Errorf("name %q cannot contain spaces", name)
		}
		if err := param.validate(); err != nil {
			return p, fmt.Errorf("parameter %q could not be added: %s", name, err)
		}
	}
	p.m = types
	return p, nil
}

// Must is the same as With(), but it must succeed or it panics.
func (p Definitions) Must(types ParamTypes) Definitions {
	var err error
	p, err = p.With(types)
	if err != nil {
		panic(err)
	}
	return p
}

var buildPool = sync.Pool{
	New: func() interface{} {
		return new(strings.Builder)
	},
}

// String implements fmt.Stringer.
func (p Definitions) String() string {
	const (
		declare   = "declare query_parameters("
		closeStmt = ");"
	)

	if len(p.m) == 0 {
		return ""
	}

	params := make([]ParamType, 0, len(p.m))

	for k, v := range p.m {
		v.name = k
		params = append(params, v)
	}

	sort.Slice(params, func(i, j int) bool { return params[i].name < params[j].name })

	build := buildPool.Get().(*strings.Builder)
	build.Reset()
	defer buildPool.Put(build)

	build.WriteString(declare)
	/*
		declare query_parameters ( Name1 : Type1 [= DefaultValue1] [,...] );
		declare query_parameters(UserName:string, Password:string)
	*/
	for i, param := range params {
		build.WriteString(param.string())
		if i+1 < len(params) {
			build.WriteString(", ")
		}
	}
	build.WriteString(closeStmt)
	return build.String()
}

// clone returns a clone of Definitions.
func (p Definitions) clone() Definitions {
	p.m = p.m.clone()
	return p
}

// QueryValues represents a set of values that are substituted in Parameters. Every QueryValue key
// must have a corresponding Parameter name. All values must be compatible with the Kusto Column type
// it will go into (int64 for a long, int32 for int, time.Time for datetime, ...)
type QueryValues map[string]interface{}

func (v QueryValues) clone() QueryValues {
	c := make(QueryValues, len(v))
	for k, v := range v {
		c[k] = v
	}
	return c
}

// Parameters represents values that will be substituted for a Stmt's Parameter. Keys are the names
// of corresponding Parameters, values are the value to be used. Keys must exist in the Parameter
// and value must be a Go type that corresponds to the ParamType.
type Parameters struct {
	m    QueryValues
	outM map[string]string // This is string keys and Kusto string query parameter values
}

// NewParameters is the construtor for Parameters.
func NewParameters() Parameters {
	return Parameters{
		m:    map[string]interface{}{},
		outM: map[string]string{},
	}
}

// IsZero returns if Parameters is the zero value.
func (q Parameters) IsZero() bool {
	return len(q.m) == 0
}

// With returns a Parameters set to "values". values' keys represents Definitions names
// that will substituted for and the values to be subsituted.
func (q Parameters) With(values QueryValues) (Parameters, error) {
	q.m = values
	return q, nil
}

// Must is the same as With() except any error is a panic.
func (q Parameters) Must(values QueryValues) Parameters {
	var err error
	q, err = q.With(values)
	if err != nil {
		panic(err)
	}
	return q
}

func (q Parameters) clone() Parameters {
	c := Parameters{
		m:    q.m.clone(),
		outM: make(map[string]string, len(q.outM)),
	}

	for k, v := range q.outM {
		c.outM[k] = v
	}

	return c
}

// toParameters creates a map[string]interface{} that is ready for JSON encoding to a REST query
// requests properties.Parameters. While output is a map[string]interface{}, this is not the same as
// Parameters itself (the values are converted to the appropriate string output).
func (q Parameters) toParameters(p Definitions) (map[string]string, error) {
	if q.outM == nil {
		return q.outM, nil
	}

	var err error
	q, err = q.validate(p)
	if err != nil {
		return nil, err
	}
	return q.outM, nil
}

// validate validates that Parameters is valid and has associated keys/types in Definitions.
// It returns a copy of Parameters that has out output map created using the values.
func (q Parameters) validate(p Definitions) (Parameters, error) {
	out := make(map[string]string, len(q.m))

	for k, v := range q.m {
		paramType, ok := p.m[k]
		if !ok {
			return q, fmt.Errorf("Parameters contains key %q that is not defined in the Stmt's Parameters", k)
		}
		switch paramType.Type {
		case types.Bool:
			b, ok := v.(bool)
			if !ok {
				return q, fmt.Errorf("Parameters[%s](bool) = %T, which is not a bool", k, v)
			}
			if b {
				out[k] = "bool(true)"
				break
			}
			out[k] = "bool(false)"
		case types.DateTime:
			t, ok := v.(time.Time)
			if !ok {
				return q, fmt.Errorf("Parameters[%s](datetime) = %T, which is not a time.Time", k, v)
			}
			out[k] = fmt.Sprintf("datetime(%s)", t.Format(time.RFC3339Nano))
		case types.Dynamic:
			b, err := json.Marshal(v)
			if err != nil {
				return q, fmt.Errorf("Parameters[%s](dynamic), %T could not be marshalled into JSON, err: %s", k, v, err)
			}
			out[k] = fmt.Sprintf("dynamic(%s)", string(b))
		case types.GUID:
			u, ok := v.(uuid.UUID)
			if !ok {
				return q, fmt.Errorf("Parameters[%s](guid) = %T, which is not a uuid.UUID", k, v)
			}
			out[k] = fmt.Sprintf("guid(%s)", u.String())
		case types.Int:
			i, ok := v.(int32)
			if !ok {
				return q, fmt.Errorf("Parameters[%s](int) = %T, which is not an int32", k, v)
			}
			out[k] = fmt.Sprintf("int(%d)", i)
		case types.Long:
			i, ok := v.(int64)
			if !ok {
				return q, fmt.Errorf("Parameters[%s](long) = %T, which is not an int64", k, v)
			}
			out[k] = fmt.Sprintf("long(%d)", i)
		case types.Real:
			i, ok := v.(float64)
			if !ok {
				return q, fmt.Errorf("Parameters[%s](real) = %T, which is not a float64", k, v)
			}
			out[k] = fmt.Sprintf("real(%f)", i)
		case types.String:
			s, ok := v.(string)
			if !ok {
				return q, fmt.Errorf("Parameters[%s](string) = %T, which is not a string", k, v)
			}
			out[k] = fmt.Sprint(s)
		case types.Timespan:
			d, ok := v.(time.Duration)
			if !ok {
				return q, fmt.Errorf("parameters[%s](timespan) = %T, which is not a time.Duration", k, v)
			}
			out[k] = fmt.Sprintf("timespan(%s)", value.Timespan{Value: d, Valid: true}.Marshal())
		case types.Decimal:
			var sval string
			switch v := v.(type) {
			case string:
				sval = v
			case *big.Float:
				sval = v.String()
			case *big.Int:
				sval = v.String()
			default:
				return q, fmt.Errorf("Parameters[%s](decimal) = %T, which is not a string or *big.Float", k, v)
			}
			out[k] = fmt.Sprintf("decimal(%s)", sval)
		}
	}
	q.outM = out
	return q, nil
}

// Statement is an interface designated to generalize query/management objects - both Stmt, and kql.StatementBuilder
type Statement interface {
	fmt.Stringer
	GetParameters() (map[string]string, error)
	SupportsInlineParameters() bool
}

// Stmt is a Kusto Query statement. A Stmt is thread-safe, but methods on the Stmt are not.
// All methods on a Stmt do not alter the statement, they return a new Stmt object with the changes.
// This includes a copy of the Definitions and Parameters objects, if provided.  This allows a
// root Stmt object that can be built upon. You should not pass *Stmt objects.
type Stmt struct {
	queryStr string
	defs     Definitions
	params   Parameters
	unsafe   unsafe.Stmt
}

// StmtOption is an optional argument to NewStmt().
type StmtOption func(s *Stmt)

func (s Stmt) GetParameters() (map[string]string, error) {
	return s.params.toParameters(s.defs)
}
func (s Stmt) SupportsInlineParameters() bool {
	return true
}

// UnsafeStmt enables unsafe actions on a Stmt and all Stmts derived from that Stmt.
// This turns off safety features that could allow a service client to compromise your data store.
// USE AT YOUR OWN RISK!
func UnsafeStmt(options unsafe.Stmt) StmtOption {
	return func(s *Stmt) {
		ilog.UnsafeWarning(options.SuppressWarning)
		s.unsafe.Add = true
	}
}

// Deprecated: Use kql.New and kql.NewParameters instead.
// NewStmt creates a Stmt from a string constant.
func NewStmt(query stringConstant, options ...StmtOption) Stmt {
	s := Stmt{queryStr: query.String()}
	for _, option := range options {
		option(&s)
	}
	return s
}

// Add will add more text to the Stmt. This is similar to the + operator on two strings, except
// it only can be done with string constants. This allows dynamically building of a query from a root
// Stmt.
func (s Stmt) Add(query stringConstant) Stmt {
	s.queryStr = s.queryStr + query.String()
	return s
}

// UnsafeAdd provides a method to add strings that are not injection protected to the Stmt.
// To utilize this method, you must create the Stmt with the UnsafeStmt() option and pass
// the unsafe.Stmt with .Add set to true. If not set, THIS WILL PANIC!
func (s Stmt) UnsafeAdd(query string) Stmt {
	if !s.unsafe.Add {
		panic("Stmt.UnsafeAdd() called, but the unsafe.Stmt.Add ability has not been enabled")
	}

	s.queryStr = s.queryStr + query
	return s
}

// WithDefinitions will return a Stmt that can be used in a Query() with Kusto
// Parameters to protect against SQL-like injection attacks. These Parameters must align with
// the placeholders in the statement. The new Stmt object will have a copy of the Parameters passed,
// not the original.
func (s Stmt) WithDefinitions(defs Definitions) (Stmt, error) {
	if len(defs.m) == 0 {
		return s, fmt.Errorf("cannot pass Definitions that are empty")
	}
	s.defs = defs.clone()

	return s, nil
}

// MustDefinitions is the same as WithDefinitions with the exceptions that an error causes a panic.
func (s Stmt) MustDefinitions(defs Definitions) Stmt {
	s, err := s.WithDefinitions(defs)
	if err != nil {
		panic(err)
	}

	return s
}

// WithParameters returns a Stmt that has the Parameters that will be substituted for
// Definitions in the query.  Must have supplied the appropriate Definitions using WithQueryParamaters().
func (s Stmt) WithParameters(params Parameters) (Stmt, error) {
	if s.defs.IsZero() {
		return s, fmt.Errorf("cannot call WithParameters() if WithDefinitions hasn't been called")
	}
	params = params.clone()
	var err error

	params, err = params.validate(s.defs)
	if err != nil {
		return s, err
	}

	s.params = params
	return s, nil
}

// MustParameters is the same as WithParameters with the exceptions that an error causes a panic.
func (s Stmt) MustParameters(params Parameters) Stmt {
	stmt, err := s.WithParameters(params)
	if err != nil {
		panic(err)
	}
	return stmt
}

// String implements fmt.Stringer. This can be used to see what the query statement to the server will be
// for debugging purposes.
func (s Stmt) String() string {
	build := buildPool.Get().(*strings.Builder)
	build.Reset()
	defer buildPool.Put(build)

	if len(s.defs.m) > 0 {
		build.WriteString(s.defs.String() + "\n")
	}
	build.WriteString(s.queryStr)
	return build.String()
}

// ValuesJSON returns a string in JSON format representing the Kusto QueryOptions.Parameters value
// that will be passed to the server. These values are substitued for Definitions in the Stmt and
// are represented by the Parameters that was passed.
func (s Stmt) ValuesJSON() (string, error) {
	m, err := s.params.toParameters(s.defs)
	if err != nil {
		return "", err
	}
	b, err := json.Marshal(m)
	if err != nil {
		return "", err
	}
	return string(b), nil
}
