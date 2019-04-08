package yatee

import (
	"fmt"
	"io/ioutil"
	"os"
	"strconv"
	"strings"

	"github.com/docker/app/internal/yaml"
	yml "gopkg.in/yaml.v2"
)

const (
	// OptionErrOnMissingKey if set will make rendering fail if a non-existing variable is used
	OptionErrOnMissingKey = "ErrOnMissingKey"
)

type options struct {
	errOnMissingKey bool
}

// flatten flattens a structure: foo.bar.baz -> 'foo.bar.baz'
func flatten(in map[string]interface{}, out map[string]interface{}, prefix string) {
	for k, v := range in {
		switch vv := v.(type) {
		case string:
			out[prefix+k] = vv
		case map[string]interface{}:
			flatten(vv, out, prefix+k+".")
		case []interface{}:
			values := []string{}
			for _, i := range vv {
				values = append(values, fmt.Sprintf("%v", i))
			}
			out[prefix+k] = strings.Join(values, " ")
		default:
			out[prefix+k] = v
		}
	}
}

func merge(res map[string]interface{}, src map[interface{}]interface{}) {
	for k, v := range src {
		kk, ok := k.(string)
		if !ok {
			panic(fmt.Sprintf("fatal error, key %v in %#v is not a string", k, src))
		}
		eval, ok := res[kk]
		switch vv := v.(type) {
		case map[interface{}]interface{}:
			if !ok {
				res[kk] = make(map[string]interface{})
			} else {
				if _, ok2 := eval.(map[string]interface{}); !ok2 {
					res[kk] = make(map[string]interface{})
				}
			}
			merge(res[kk].(map[string]interface{}), vv)
		default:
			res[kk] = vv
		}
	}
}

// LoadParameters loads a set of parameters file and produce a property dictionary
func LoadParameters(files []string) (map[string]interface{}, error) {
	res := make(map[string]interface{})
	for _, f := range files {
		data, err := ioutil.ReadFile(f)
		if err != nil {
			return nil, err
		}
		s := make(map[interface{}]interface{})
		err = yaml.Unmarshal(data, &s)
		if err != nil {
			return nil, err
		}
		merge(res, s)
	}
	return res, nil
}

func isIdentNumChar(r byte) bool {
	return (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') ||
		r == '.' || r == '_'
}

// extract extracts an expression from a string
// nolint: gocyclo
func extract(expr string) (string, error) {
	if expr == "" {
		return "", nil
	}
	if expr[0] == '{' {
		closing := strings.Index(expr, "}")
		if closing == -1 {
			return "", fmt.Errorf("Missing '}' at end of expression")
		}
		return expr[0 : closing+1], nil
	}
	if expr[0] == '(' {
		indent := 1
		i := 1
		for ; i < len(expr); i++ {
			if expr[i] == '(' {
				indent++
			}
			if expr[i] == ')' {
				indent--
			}
			if indent == 0 {
				break
			}
		}
		if indent != 0 {
			return "", fmt.Errorf("Missing ')' at end of expression")
		}
		return expr[0 : i+1], nil
	}
	i := 0
	for ; i < len(expr); i++ {
		if !((expr[i] >= 'a' && expr[i] <= 'z') || (expr[i] >= 'A' && expr[i] <= 'Z') ||
			expr[i] == '.' || expr[i] == '_') {
			break
		}
	}
	return expr[0:i], nil
}

func tokenize(expr string) []string {
	var tokens []string
	p := 0
	for p < len(expr) {
		if isIdentNumChar(expr[p]) {
			pp := p + 1
			for ; pp < len(expr) && isIdentNumChar(expr[pp]); pp++ {
			}
			tokens = append(tokens, expr[p:pp])
			p = pp
		} else {
			if expr[p] != ' ' {
				tokens = append(tokens, expr[p:p+1])
			}
			p++
		}
	}
	return tokens
}

func evalValue(comps []string, i int) (int64, int, error) {
	c := comps[i]
	if c == "(" {
		value, ni, error := evalSub(comps, i+1)
		if error != nil {
			return 0, 0, error
		}
		return value, ni, nil
	}
	v, err := strconv.ParseInt(c, 0, 64)
	return v, i + 1, err
}

func evalSub(comps []string, i int) (int64, int, error) {
	current, next, err := evalValue(comps, i)
	if err != nil {
		return 0, 0, err
	}
	i = next
	for i < len(comps) {
		c := comps[i]
		if c == ")" {
			return current, i + 1, nil
		}
		if c == "*" || c == "+" || c == "-" || c == "/" || c == "%" {
			rhs, next, err := evalValue(comps, i+1)
			if err != nil {
				return 0, 0, err
			}
			switch c {
			case "+":
				current += rhs
			case "-":
				current -= rhs
			case "/":
				current /= rhs
			case "*":
				current *= rhs
			case "%":
				current %= rhs
			}
			i = next
		} else {
			return 0, 0, fmt.Errorf("expected operator")
		}
	}
	return current, i, nil
}

// resolves an arithmetic expression
func evalExpr(expr string) (int64, error) {
	comps := tokenize(expr)
	v, _, err := evalSub(comps, 0)
	return v, err
}

// resolves and evaluate all ${foo.bar}, $foo.bar and $(expr) in epr
// nolint: gocyclo
func eval(expr string, flattened map[string]interface{}, o options) (interface{}, error) {
	// Since we go from right to left to support nesting, handling $$ escape is
	// painful, so just hide them and restore them at the end
	expr = strings.Replace(expr, "$$", "\x00", -1)
	end := len(expr)
	// If evaluation resolves to a single value, return the type value, not a string
	var bypass interface{}
	iteration := 0
	for {
		iteration++
		if iteration > 100 {
			return "", fmt.Errorf("eval loop detected")
		}
		i := strings.LastIndex(expr[0:end], "$")
		if i == -1 {
			break
		}
		bypass = nil
		comp, err := extract(expr[i+1:])
		if err != nil {
			return "", err
		}
		var val interface{}
		if len(comp) != 0 && comp[0] == '(' {
			var err error
			val, err = evalExpr(comp[1 : len(comp)-1])
			if err != nil {
				return "", err
			}
		} else {
			var ok bool
			if len(comp) != 0 && comp[0] == '{' {
				content := comp[1 : len(comp)-1]
				q := strings.Index(content, "?")
				if q != -1 {
					s := strings.Index(content, ":")
					if s == -1 {
						return "", fmt.Errorf("parse error in ternary '%s', missing ':'", content)
					}
					variable := content[0:q]
					val, ok = flattened[variable]
					if isTrue(fmt.Sprintf("%v", val)) {
						val = content[q+1 : s]
					} else {
						val = content[s+1:]
					}
				} else {
					val, ok = flattened[comp[1:len(comp)-1]]
				}
			} else {
				val, ok = flattened[comp]
			}
			if !ok {
				if o.errOnMissingKey {
					return "", fmt.Errorf("variable '%s' not set", comp)
				}
				fmt.Fprintf(os.Stderr, "variable '%s' not set, expanding to empty string", comp)
			}
		}
		valstr := fmt.Sprintf("%v", val)
		expr = expr[0:i] + valstr + expr[i+1+len(comp):]
		if strings.Trim(expr, " ") == valstr {
			bypass = val
		}
		end = len(expr)
	}
	if bypass != nil {
		return bypass, nil
	}
	expr = strings.Replace(expr, "\x00", "$", -1)
	return expr, nil
}

func isTrue(cond string) bool {
	ct := strings.TrimLeft(cond, " ")
	reverse := len(cond) != 0 && cond[0] == '!'
	if reverse {
		cond = ct[1:]
	}
	cond = strings.Trim(cond, " ")
	return (cond != "" && cond != "false" && cond != "0") != reverse
}

func recurseList(input []interface{}, parameters map[string]interface{}, flattened map[string]interface{}, o options) ([]interface{}, error) {
	var res []interface{}
	for _, v := range input {
		switch vv := v.(type) {
		case yml.MapSlice:
			newv, err := recurse(vv, parameters, flattened, o)
			if err != nil {
				return nil, err
			}
			res = append(res, newv)
		case []interface{}:
			newv, err := recurseList(vv, parameters, flattened, o)
			if err != nil {
				return nil, err
			}
			res = append(res, newv)
		case string:
			vvv, err := eval(vv, flattened, o)
			if err != nil {
				return nil, err
			}
			if vvvs, ok := vvv.(string); ok {
				trimed := strings.TrimLeft(vvvs, " ")
				if strings.HasPrefix(trimed, "@if") {
					be := strings.Index(trimed, "(")
					ee := strings.Index(trimed, ")")
					if be == -1 || ee == -1 || be > ee {
						return nil, fmt.Errorf("parse error looking for if condition in '%s'", vvv)
					}
					cond := trimed[be+1 : ee]
					if isTrue(cond) {
						res = append(res, strings.Trim(trimed[ee+1:], " "))
					}
					continue
				}
			}
			res = append(res, vvv)
		default:
			res = append(res, v)
		}
	}
	return res, nil
}

// FIXME complexity on this is 47… get it lower than 16
// nolint: gocyclo
func recurse(input yml.MapSlice, parameters map[string]interface{}, flattened map[string]interface{}, o options) (yml.MapSlice, error) {
	res := yml.MapSlice{}
	for _, kvp := range input {
		k := kvp.Key
		v := kvp.Value
		rk := k
		kstr, isks := k.(string)
		if isks {
			trimed := strings.TrimLeft(kstr, " ")
			if strings.HasPrefix(trimed, "@switch ") {
				mii, ok := v.(yml.MapSlice)
				if !ok {
					return nil, fmt.Errorf("@switch value must be a mapping")
				}
				key, err := eval(strings.TrimPrefix(trimed, "@switch "), flattened, o)
				if err != nil {
					return nil, err
				}
				var defaultValue interface{}
				hit := false
				for _, sval := range mii {
					sk := sval.Key
					sv := sval.Value
					ssk, ok := sk.(string)
					if !ok {
						return nil, fmt.Errorf("@switch entry key must be a string")
					}
					if ssk == "default" {
						defaultValue = sv
					}
					if ssk == key {
						hit = true
						svv, ok := sv.(yml.MapSlice)
						if !ok {
							return nil, fmt.Errorf("@switch entry must be a mapping")
						}
						for _, vval := range svv {
							res = append(res, yml.MapItem{Key: vval.Key, Value: vval.Value})
						}
					}
				}
				if !hit && defaultValue != nil {
					svv, ok := defaultValue.(yml.MapSlice)
					if !ok {
						return nil, fmt.Errorf("@switch entry must be a mapping")
					}
					for _, vval := range svv {
						res = append(res, yml.MapItem{Key: vval.Key, Value: vval.Value})
					}
				}
				continue
			}
			if strings.HasPrefix(trimed, "@for ") {
				mii, ok := v.(yml.MapSlice)
				if !ok {
					return nil, fmt.Errorf("@for value must be a mapping")
				}
				comps := strings.SplitN(trimed, " ", 4)
				varname := comps[1]
				varrangeraw, err := eval(comps[3], flattened, o)
				if err != nil {
					return nil, err
				}
				varrange, ok := varrangeraw.(string)
				if !ok {
					return nil, fmt.Errorf("@for argument must be a string")
				}
				mayberange := strings.Split(varrange, "..")
				if len(mayberange) == 2 {
					rangestart, err := strconv.ParseInt(mayberange[0], 0, 64)
					if err != nil {
						return nil, err
					}
					rangeend, err := strconv.ParseInt(mayberange[1], 0, 64)
					if err != nil {
						return nil, err
					}
					for i := rangestart; i < rangeend; i++ {
						flattened[varname] = fmt.Sprintf("%v", i)
						val, err := recurse(mii, parameters, flattened, o)
						if err != nil {
							return nil, err
						}
						for _, vval := range val {
							res = append(res, yml.MapItem{Key: vval.Key, Value: vval.Value})
						}
					}
				} else {
					// treat range as a list
					rangevalues := strings.Split(varrange, " ")
					for _, i := range rangevalues {
						flattened[varname] = i
						val, err := recurse(mii, parameters, flattened, o)
						if err != nil {
							return nil, err
						}
						for _, vval := range val {
							res = append(res, yml.MapItem{Key: vval.Key, Value: vval.Value})
						}
					}
				}
				continue
			}
			if strings.HasPrefix(trimed, "@if ") {
				cond, err := eval(strings.TrimPrefix(trimed, "@if "), flattened, o)
				if err != nil {
					return nil, err
				}
				mii, ok := v.(yml.MapSlice)
				if !ok {
					return nil, fmt.Errorf("@if value must be a mapping")
				}
				if isTrue(fmt.Sprintf("%v", cond)) {
					val, err := recurse(mii, parameters, flattened, o)
					if err != nil {
						return nil, err
					}
					for _, vval := range val {
						if vval.Key != "@else" {
							res = append(res, yml.MapItem{Key: vval.Key, Value: vval.Value})
						}
					}
				} else {
					var elseClause interface{}
					for _, miiv := range mii {
						if miiv.Key == "@else" {
							elseClause = miiv.Value
							break
						}
					}
					if elseClause != nil {
						elseDict, ok := elseClause.(yml.MapSlice)
						if !ok {
							return nil, fmt.Errorf("@else value must be a mapping")
						}
						for _, vval := range elseDict {
							res = append(res, yml.MapItem{Key: vval.Key, Value: vval.Value})
						}
					}
				}
				continue
			}
			rstr, err := eval(kstr, flattened, o)
			if err != nil {
				return nil, err
			}
			rk = rstr
		}
		switch vv := v.(type) {
		case yml.MapSlice:
			newv, err := recurse(vv, parameters, flattened, o)
			if err != nil {
				return nil, err
			}
			res = append(res, yml.MapItem{Key: rk, Value: newv})
		case []interface{}:
			newv, err := recurseList(vv, parameters, flattened, o)
			if err != nil {
				return nil, err
			}
			res = append(res, yml.MapItem{Key: rk, Value: newv})
		case string:
			vvv, err := eval(vv, flattened, o)
			if err != nil {
				return nil, err
			}
			res = append(res, yml.MapItem{Key: rk, Value: vvv})
		default:
			res = append(res, yml.MapItem{Key: rk, Value: v})
		}
	}
	return res, nil
}

// ProcessStrings resolves input templated yaml using values in parameters yaml
func ProcessStrings(input, parameters string) (string, error) {
	ps := make(map[interface{}]interface{})
	err := yaml.Unmarshal([]byte(parameters), ps)
	if err != nil {
		return "", err
	}
	s := make(map[string]interface{})
	merge(s, ps)
	res, err := Process(input, s)
	if err != nil {
		return "", err
	}
	sres, err := yaml.Marshal(res)
	if err != nil {
		return "", err
	}
	return string(sres), nil
}

// ProcessWithOrder resolves input templated yaml using values given in parameters, returning a MapSlice with order preserved
func ProcessWithOrder(inputString string, parameters map[string]interface{}, opts ...string) (yml.MapSlice, error) {
	var o options
	for _, v := range opts {
		switch v {
		case OptionErrOnMissingKey:
			o.errOnMissingKey = true
		default:
			return nil, fmt.Errorf("unknown option %q", v)
		}
	}
	var input yml.MapSlice
	err := yaml.Unmarshal([]byte(inputString), &input)
	if err != nil {
		return nil, err
	}
	flattened := make(map[string]interface{})
	flatten(parameters, flattened, "")
	return recurse(input, parameters, flattened, o)
}

// Process resolves input templated yaml using values given in parameters, returning a map
func Process(inputString string, parameters map[string]interface{}, opts ...string) (map[interface{}]interface{}, error) {
	mapSlice, err := ProcessWithOrder(inputString, parameters, opts...)
	if err != nil {
		return nil, err
	}

	res, err := convert(mapSlice)
	if err != nil {
		return nil, err
	}
	return res, nil
}

func convert(mapSlice yml.MapSlice) (map[interface{}]interface{}, error) {
	res := make(map[interface{}]interface{})
	for _, kv := range mapSlice {
		v := kv.Value
		castValue, ok := v.(yml.MapSlice)
		if !ok {
			res[kv.Key] = kv.Value
		} else {
			recursed, err := convert(castValue)
			if err != nil {
				return nil, err
			}
			res[kv.Key] = recursed
		}
	}
	return res, nil
}
