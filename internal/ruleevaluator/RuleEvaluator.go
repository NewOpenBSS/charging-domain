package ruleevaluator

import (
	"fmt"
	"reflect"
	"strconv"
	"strings"
)

type tokenType int

const (
	tokEq tokenType = iota
	tokNe
	tokGt
	tokGe
	tokLt
	tokLe
	tokAnd
	tokOr
	tokLParen
	tokNot
	tokElse // :
	tokFunc
)

type EvaluatorFunc func(args []any) (any, error)

type RuleEvaluator struct {
	data any

	// function registry: name -> implementation
	funcs map[string]EvaluatorFunc
}

type funcMarker struct {
	fn EvaluatorFunc
}

func NewRuleEvaluator(data any) *RuleEvaluator {
	return &RuleEvaluator{
		data:  data,
		funcs: make(map[string]EvaluatorFunc),
	}
}

func (e *RuleEvaluator) RegisterFunction(name string, fn EvaluatorFunc) {
	e.funcs[name] = fn
}

func (e *RuleEvaluator) Evaluate(expression string) (any, error) {
	return e.EvaluateWithVars(expression, nil)
}

func compare(left, right any, operator tokenType) (bool, error) {
	// nil handling (match your Java behaviour: only == and != are allowed)
	if left == nil || right == nil {
		switch operator {
		case tokEq:
			return left == nil && right == nil, nil
		case tokNe:
			return !(left == nil && right == nil), nil
		default:
			return false, fmt.Errorf("cannot use operator %v with nil", operator)
		}
	}

	// type equality check (Java getClass().equals)
	lt, rt := reflect.TypeOf(left), reflect.TypeOf(right)
	if lt != rt {
		return false, fmt.Errorf("cannot compare different types: %v and %v", lt, rt)
	}

	switch l := left.(type) {
	case bool:
		r := right.(bool)
		switch operator {
		case tokEq:
			return l == r, nil
		case tokNe:
			return l != r, nil
		case tokGt:
			return l && !r, nil
		case tokLt:
			return !l && r, nil
		case tokGe:
			return l == r || (l && !r), nil
		case tokLe:
			return l == r || (!l && r), nil
		default:
			return false, fmt.Errorf("unknown comparison operator: %v", operator)
		}

	case string:
		r := right.(string)
		switch operator {
		case tokEq:
			return l == r, nil
		case tokNe:
			return l != r, nil
		case tokGt:
			return l > r, nil
		case tokLt:
			return l < r, nil
		case tokGe:
			return l >= r, nil
		case tokLe:
			return l <= r, nil
		default:
			return false, fmt.Errorf("unknown comparison operator: %v", operator)
		}

	case int64:
		r := right.(int64)
		switch operator {
		case tokEq:
			return l == r, nil
		case tokNe:
			return l != r, nil
		case tokGt:
			return l > r, nil
		case tokLt:
			return l < r, nil
		case tokGe:
			return l >= r, nil
		case tokLe:
			return l <= r, nil
		default:
			return false, fmt.Errorf("unknown comparison operator: %v", operator)
		}

	case float64:
		r := right.(float64)
		switch operator {
		case tokEq:
			return l == r, nil
		case tokNe:
			return l != r, nil
		case tokGt:
			return l > r, nil
		case tokLt:
			return l < r, nil
		case tokGe:
			return l >= r, nil
		case tokLe:
			return l <= r, nil
		default:
			return false, fmt.Errorf("unknown comparison operator: %v", operator)
		}

	default:
		return false, fmt.Errorf("type %v is not comparable", lt)
	}
}

func processOp(values *stack[any], operator tokenType) error {
	switch operator {
	case tokNot:
		// Unary NOT
		v, err := values.Pop()
		if err != nil {
			return fmt.Errorf("insufficient values for unary NOT")
		}
		bv, ok := v.(bool)
		if !ok {
			return fmt.Errorf("invalid type for unary NOT: %T", v)
		}
		values.Push(!bv)
		return nil

	case tokElse:
		// Ternary selection (condition ? trueVal : falseVal)
		fv, err := values.Pop()
		if err != nil {
			return fmt.Errorf("insufficient values for ternary operator")
		}
		tv, err := values.Pop()
		if err != nil {
			return fmt.Errorf("insufficient values for ternary operator")
		}
		cond, err := values.Pop()
		if err != nil {
			return fmt.Errorf("insufficient values for ternary operator")
		}
		b, ok := cond.(bool)
		if !ok {
			return fmt.Errorf("ternary condition must be bool, got %T", cond)
		}
		if b {
			values.Push(tv)
		} else {
			values.Push(fv)
		}
		return nil

	case tokFunc:
		return processFunc(values)

	case tokAnd, tokOr:
		right, err := values.Pop()
		if err != nil {
			return fmt.Errorf("insufficient values for operator: %d", operator)
		}
		left, err := values.Pop()
		if err != nil {
			return fmt.Errorf("insufficient values for operator: %d", operator)
		}
		lb, ok1 := left.(bool)
		rb, ok2 := right.(bool)
		if !ok1 || !ok2 {
			return fmt.Errorf("invalid types for logical operator: %T and %T", left, right)
		}
		if operator == tokAnd {
			values.Push(lb && rb)
		} else {
			values.Push(lb || rb)
		}
		return nil

	case tokEq, tokNe, tokGt, tokGe, tokLt, tokLe:
		right, err := values.Pop()
		if err != nil {
			return fmt.Errorf("insufficient values for operator: %d", operator)
		}
		left, err := values.Pop()
		if err != nil {
			return fmt.Errorf("insufficient values for operator: %d", operator)
		}
		b, err := compare(left, right, operator)
		if err != nil {
			return err
		}
		values.Push(b)
		return nil

	default:
		return fmt.Errorf("unknown operator: %v", operator)
	}
}

func processParen(values *stack[any], ops *stack[tokenType]) error {
	// Pop and apply operators until we reach the matching '('
	for ops.Len() > 0 {
		peek, _ := ops.Peek()
		if peek == tokLParen {
			break
		}
		op, _ := ops.Pop()
		if err := processOp(values, op); err != nil {
			return err
		}
	}

	// Ensure we have a matching '('
	peek, err := ops.Peek()
	if err != nil || peek != tokLParen {
		return fmt.Errorf("mismatched parentheses")
	}

	// Discard '('
	_, _ = ops.Pop()

	// If immediately before '(' there is a unary NOT or function marker, apply it now.
	if ops.Len() > 0 {
		op, _ := ops.Peek()
		if op == tokNot || op == tokFunc {
			op, _ := ops.Pop()
			if err := processOp(values, op); err != nil {
				return err
			}
		}
	}

	return nil
}

func processFunc(values *stack[any]) error {
	// Values stack layout (from bottom to top) around a function call:
	// ... , funcMarker{fn}, arg1, arg2, ...
	// We pop arguments until we hit the funcMarker.
	if values.Len() == 0 {
		return fmt.Errorf("function call with empty value stack")
	}

	argsReversed := make([]any, 0)
	for values.Len() > 0 {
		v, _ := values.Pop()

		if fm, ok := v.(funcMarker); ok {
			// reverse args into call order
			for i, j := 0, len(argsReversed)-1; i < j; i, j = i+1, j-1 {
				argsReversed[i], argsReversed[j] = argsReversed[j], argsReversed[i]
			}

			res, err := fm.fn(argsReversed)
			if err != nil {
				return err
			}

			values.Push(res)
			return nil
		}
		argsReversed = append(argsReversed, v)
	}

	return fmt.Errorf("function call missing func marker")
}

func tokenize(expression string) ([]string, error) {
	tokens := make([]string, 0, 32)

	isSpace := func(b byte) bool {
		return b == ' ' || b == '\t' || b == '\n' || b == '\r'
	}

	isIdentChar := func(b byte) bool {
		// Identifiers in this DSL include dotted paths and array selectors.
		// Examples: chargingData.foo.bar, recipientInfo[0], foo[$i]
		if (b >= 'a' && b <= 'z') || (b >= 'A' && b <= 'Z') || (b >= '0' && b <= '9') {
			return true
		}
		switch b {
		case '_', '.', '$', '[', ']', '-':
			return true
		default:
			return false
		}
	}

	// Returns true if s at position i starts with lit.
	startsWithAt := func(s string, i int, lit string) bool {
		if i < 0 || i+len(lit) > len(s) {
			return false
		}
		return s[i:i+len(lit)] == lit
	}

	for i := 0; i < len(expression); {
		// Skip whitespace
		if isSpace(expression[i]) {
			i++
			continue
		}

		// Two-char operators
		switch {
		case startsWithAt(expression, i, "=="):
			tokens = append(tokens, "==")
			i += 2
			continue
		case startsWithAt(expression, i, "!="):
			tokens = append(tokens, "!=")
			i += 2
			continue
		case startsWithAt(expression, i, ">="):
			tokens = append(tokens, ">=")
			i += 2
			continue
		case startsWithAt(expression, i, "<="):
			tokens = append(tokens, "<=")
			i += 2
			continue
		case startsWithAt(expression, i, "&&"):
			tokens = append(tokens, "&&")
			i += 2
			continue
		case startsWithAt(expression, i, "||"):
			tokens = append(tokens, "||")
			i += 2
			continue
		}

		ch := expression[i]

		// Single-char tokens
		switch ch {
		case '(', ')', '>', '<', '!', '?', ':':
			tokens = append(tokens, string(ch))
			i++
			continue
		case ',':
			// Ignore commas (function argument separators)
			i++
			continue
		case '\'', '"':
			// String literal
			quote := ch
			start := i
			i++
			for i < len(expression) {
				if expression[i] == '\\' {
					// Skip escaped char
					i += 2
					continue
				}
				if expression[i] == quote {
					i++
					tokens = append(tokens, expression[start:i])
					break
				}
				i++
			}
			// Fix string literal error check:
			if len(tokens) == 0 || tokens[len(tokens)-1] != expression[start:i] {
				return nil, fmt.Errorf("unterminated string literal")
			}
			continue
		}

		// Identifier / number / keywords
		start := i
		for i < len(expression) && isIdentChar(expression[i]) {
			i++
		}
		if start == i {
			return nil, fmt.Errorf("unexpected character: %q", expression[i])
		}

		word := expression[start:i]

		// Merge textual operator: `is not`
		if word == "is" {
			j := i
			for j < len(expression) && isSpace(expression[j]) {
				j++
			}
			if startsWithAt(expression, j, "not") {
				k := j + len("not")
				// Ensure boundary
				if k == len(expression) || !isIdentChar(expression[k]) {
					tokens = append(tokens, "is not")
					i = k
					continue
				}
			}
		}

		tokens = append(tokens, word)
	}

	return tokens, nil
}

func (e *RuleEvaluator) getFieldValue(field string, vars map[string]any) (any, error) {
	if field == "" {
		return nil, fmt.Errorf("field cannot be empty")
	}

	// Starting object is evaluator data.
	var cur any = e.data

	// Helper: dereference pointers/interfaces.
	deref := func(v reflect.Value) reflect.Value {
		for v.IsValid() && (v.Kind() == reflect.Pointer || v.Kind() == reflect.Interface) {
			if v.IsNil() {
				return reflect.Value{}
			}
			v = v.Elem()
		}
		return v
	}

	// Helper: parse a token part like "foo[0]" or "foo[$i]".
	parsePart := func(part string) (name string, hasIdx bool, idxSpec string, err error) {
		lb := strings.IndexByte(part, '[')
		if lb < 0 {
			return part, false, "", nil
		}
		rb := strings.LastIndexByte(part, ']')
		if rb < 0 || rb < lb {
			return "", false, "", fmt.Errorf("invalid index syntax in %q", part)
		}
		name = part[:lb]
		idxSpec = part[lb+1 : rb]
		if idxSpec == "" {
			return "", false, "", fmt.Errorf("empty index in %q", part)
		}
		return name, true, idxSpec, nil
	}

	// Helper: convert index spec to int.
	toIndex := func(spec string) (int, error) {
		if len(spec) > 0 && spec[0] == '$' {
			if vars == nil {
				return 0, fmt.Errorf("index variable %q not provided", spec)
			}
			v, ok := vars[spec]
			if !ok {
				return 0, fmt.Errorf("index variable %q not found", spec)
			}
			switch n := v.(type) {
			case int:
				return n, nil
			case int8:
				return int(n), nil
			case int16:
				return int(n), nil
			case int32:
				return int(n), nil
			case int64:
				return int(n), nil
			case uint:
				return int(n), nil
			case uint8:
				return int(n), nil
			case uint16:
				return int(n), nil
			case uint32:
				return int(n), nil
			case uint64:
				return int(n), nil
			case float32:
				return int(n), nil
			case float64:
				return int(n), nil
			case string:
				i, err := strconv.Atoi(n)
				if err != nil {
					return 0, fmt.Errorf("index variable %q is not an int: %v", spec, err)
				}
				return i, nil
			default:
				return 0, fmt.Errorf("index variable %q has unsupported type %T", spec, v)
			}
		}

		i, err := strconv.Atoi(spec)
		if err != nil {
			return 0, fmt.Errorf("invalid index %q: %v", spec, err)
		}
		return i, nil
	}

	// Walk dotted path.
	parts := strings.Split(field, ".")
	for _, rawPart := range parts {
		if rawPart == "" {
			// treat empty segments as missing
			return nil, nil
		}

		// Variables: a part like "$foo" replaces the current object.
		if len(rawPart) > 0 && rawPart[0] == '$' {
			if vars == nil {
				return nil, nil
			}
			v, ok := vars[rawPart]
			if !ok {
				return nil, nil
			}
			cur = v
			continue
		}

		name, hasIdx, idxSpec, err := parsePart(rawPart)
		if err != nil {
			return nil, err
		}

		if cur == nil {
			return nil, nil
		}

		cv := deref(reflect.ValueOf(cur))
		if !cv.IsValid() {
			return nil, nil
		}

		// Step 1: resolve named field/key if name is present.
		if name != "" {
			switch cv.Kind() {
			case reflect.Map:
				// Only support string keys.
				if cv.Type().Key().Kind() != reflect.String {
					return nil, nil
				}
				mv := cv.MapIndex(reflect.ValueOf(name))
				if !mv.IsValid() {
					// missing key -> nil (Java-like)
					return nil, nil
				}
				cur = mv.Interface()

			case reflect.Struct:
				// Try exact exported field name.
				fv := cv.FieldByName(name)
				if !fv.IsValid() {
					// Try Title-cased (common when config uses lowerCamel and struct uses UpperCamel).
					if len(name) > 0 {
						alt := strings.ToUpper(name[:1]) + name[1:]
						fv = cv.FieldByName(alt)
					}
				}
				if !fv.IsValid() || !fv.CanInterface() {
					return nil, nil
				}
				cur = fv.Interface()

			default:
				// Not a map/struct: cannot resolve named field.
				return nil, nil
			}
		}

		// Step 2: apply index if present.
		if hasIdx {
			idx, err := toIndex(idxSpec)
			if err != nil {
				return nil, err
			}

			if cur == nil {
				return nil, nil
			}

			iv := deref(reflect.ValueOf(cur))
			if !iv.IsValid() {
				return nil, nil
			}

			switch iv.Kind() {
			case reflect.Slice, reflect.Array:
				if idx < 0 || idx >= iv.Len() {
					return nil, nil
				}
				el := iv.Index(idx)
				if !el.IsValid() {
					return nil, nil
				}
				if !el.CanInterface() {
					return nil, nil
				}
				cur = el.Interface()

			default:
				return nil, nil
			}
		}
	}

	return cur, nil
}

func (e *RuleEvaluator) resolveValue(token string, ops *stack[tokenType], vars map[string]any) (any, error) {

	if token == "true" {
		return true, nil
	}

	if token == "false" {
		return false, nil
	}

	if token == "null" || token == "nil" {
		return nil, nil
	}

	// Check if the token is a number
	if v, err := strconv.ParseInt(token, 10, 64); err == nil {
		return v, nil
	}
	if v, err := strconv.ParseFloat(token, 64); err == nil {
		return v, nil
	}

	// Strips surrounding quotes from string literal
	if len(token) >= 2 {
		first := token[0]
		last := token[len(token)-1]
		if (first == '\'' && last == '\'') || (first == '"' && last == '"') {
			return token[1 : len(token)-1], nil
		}
	}

	// Check if the token is a function call
	if e.funcs[token] != nil {
		ops.Push(tokFunc)
		return funcMarker{e.funcs[token]}, nil
	}

	// lookup field value
	val, err := e.getFieldValue(token, vars)
	if err != nil {
		return nil, err
	}

	// Dereference if it's a pointer to a simple type (bool, string, numbers)
	if val != nil {
		v := reflect.ValueOf(val)
		if v.Kind() == reflect.Ptr && !v.IsNil() {
			elem := v.Elem()
			switch elem.Kind() {
			case reflect.Bool, reflect.String, reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64,
				reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Float32, reflect.Float64:
				return elem.Interface(), nil
			}
		}
	}

	return val, nil
}

func (e *RuleEvaluator) EvaluateWithVars(expression string, vars map[string]any) (any, error) {
	tokens, err := tokenize(expression)
	if err != nil {
		return nil, err
	}

	values := newStack[any]()
	ops := newStack[tokenType]()

	for _, t := range tokens {
		switch t {
		case "==":
			ops.Push(tokEq)
		case "!=":
			ops.Push(tokNe)
		case ">":
			ops.Push(tokGt)
		case ">=":
			ops.Push(tokGe)
		case "<":
			ops.Push(tokLt)
		case "<=":
			ops.Push(tokLe)
		case "&&":
			ops.Push(tokAnd)
		case "||":
			ops.Push(tokOr)
		case "!":
			ops.Push(tokNot)
		case "not":
			ops.Push(tokNot)
		case "is":
			ops.Push(tokEq)
		case "is not":
			ops.Push(tokNe)
		case "(":
			ops.Push(tokLParen)
		case ")":
			if err := processParen(values, ops); err != nil {
				return nil, err
			}
		case ":":
			ops.Push(tokElse)
		case "?":
			//pop an operator from the stack (if any)
			if ops.Len() > 0 {
				op, _ := ops.Pop()
				if err := processOp(values, op); err != nil {
					return nil, err
				}
			}

		default:
			v, err := e.resolveValue(t, ops, vars)
			if err != nil {
				return nil, err
			}
			values.Push(v)
		}
	}

	// Process remaining ops
	for ops.Len() > 0 {
		op, _ := ops.Pop()
		if err := processOp(values, op); err != nil {
			return nil, err
		}
	}

	return values.Pop()
}
