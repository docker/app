package yatee

import (
	"fmt"
	"io/ioutil"
	"os"
	"strconv"
	"strings"

	yaml "gopkg.in/yaml.v2"
)

const (
	// OptionErrOnMissingKey if set will make rendering fail if a non-existing variable is used
	OptionErrOnMissingKey = "ErrOnMissingKey"
)

type options struct {
	errOnMissingKey bool
}

// flatten flattens a structure: foo.bar.baz -> 'foo.bar.baz'
func flatten(in map[string]interface{}, out map[string]string, prefix string) {
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
			out[prefix+k] = fmt.Sprintf("%v", v)
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

// LoadSettings loads a set of settings file and produce a property dictionary
func LoadSettings(files []string) (map[string]interface{}, error) {
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

func tokenize(expr string) ([]string, error) {
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
	return tokens, nil
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
func evalExpr(expr string) (string, error) {
	comps, err := tokenize(expr)
	if err != nil {
		return "", err
	}
	v, _, err := evalSub(comps, 0)
	return fmt.Sprintf("%v", v), err
}

// resolves and evaluate all ${foo.bar}, $foo.bar and $(expr) in epr
func eval(expr string, flattened map[string]string, o options) (string, error) {
	// Since we go from right to left to support nesting, handling $$ escape is
	// painful, so just hide them and restore them at the end
	expr = strings.Replace(expr, "$$", "\x00", -1)
	end := len(expr)
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
		comp, err := extract(expr[i+1:])
		if err != nil {
			return "", err
		}
		var val string
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
					if isTrue(val) {
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
		expr = expr[0:i] + val + expr[i+1+len(comp):]
		end = len(expr)
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

func recurseList(input []interface{}, settings map[string]interface{}, flattened map[string]string, o options) ([]interface{}, error) {
	var res []interface{}
	for _, v := range input {
		switch vv := v.(type) {
		case map[interface{}]interface{}:
			newv, err := recurse(vv, settings, flattened, o)
			if err != nil {
				return nil, err
			}
			res = append(res, newv)
		case []interface{}:
			newv, err := recurseList(vv, settings, flattened, o)
			if err != nil {
				return nil, err
			}
			res = append(res, newv)
		case string:
			vvv, err := eval(vv, flattened, o)
			if err != nil {
				return nil, err
			}
			trimed := strings.TrimLeft(vvv, " ")
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
			res = append(res, vvv)
		default:
			res = append(res, v)
		}
	}
	return res, nil
}

func recurse(input map[interface{}]interface{}, settings map[string]interface{}, flattened map[string]string, o options) (map[interface{}]interface{}, error) {
	res := make(map[interface{}]interface{})
	for k, v := range input {
		rk := k
		kstr, isks := k.(string)
		if isks {
			trimed := strings.TrimLeft(kstr, " ")
			if strings.HasPrefix(trimed, "@switch ") {
				mii, ok := v.(map[interface{}]interface{})
				if !ok {
					return nil, fmt.Errorf("@switch value must be a mapping")
				}
				key, err := eval(strings.TrimPrefix(trimed, "@switch "), flattened, o)
				if err != nil {
					return nil, err
				}
				var defaultValue interface{}
				hit := false
				for sk, sv := range mii {
					ssk, ok := sk.(string)
					if !ok {
						return nil, fmt.Errorf("@switch entry key must be a string")
					}
					if ssk == "default" {
						defaultValue = sv
					}
					if ssk == key {
						hit = true
						svv, ok := sv.(map[interface{}]interface{})
						if !ok {
							return nil, fmt.Errorf("@switch entry must be a mapping")
						}
						for valk, valv := range svv {
							res[valk] = valv
						}
					}
				}
				if !hit && defaultValue != nil {
					svv, ok := defaultValue.(map[interface{}]interface{})
					if !ok {
						return nil, fmt.Errorf("@switch entry must be a mapping")
					}
					for valk, valv := range svv {
						res[valk] = valv
					}
				}
				continue
			}
			if strings.HasPrefix(trimed, "@for ") {
				mii, ok := v.(map[interface{}]interface{})
				if !ok {
					return nil, fmt.Errorf("@for value must be a mapping")
				}
				comps := strings.SplitN(trimed, " ", 4)
				varname := comps[1]
				varrange, err := eval(comps[3], flattened, o)
				if err != nil {
					return nil, err
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
						val, err := recurse(mii, settings, flattened, o)
						if err != nil {
							return nil, err
						}
						for valk, valv := range val {
							res[valk] = valv
						}
					}
				} else {
					// treat range as a list
					rangevalues := strings.Split(varrange, " ")
					for _, i := range rangevalues {
						flattened[varname] = i
						val, err := recurse(mii, settings, flattened, o)
						if err != nil {
							return nil, err
						}
						for valk, valv := range val {
							res[valk] = valv
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
				mii, ok := v.(map[interface{}]interface{})
				if !ok {
					return nil, fmt.Errorf("@if value must be a mapping")
				}
				if isTrue(cond) {
					val, err := recurse(mii, settings, flattened, o)
					if err != nil {
						return nil, err
					}
					for valk, valv := range val {
						if valk != "@else" {
							res[valk] = valv
						}
					}
				} else {
					elseClause, ok := mii["@else"]
					if ok {
						elseDict, ok := elseClause.(map[interface{}]interface{})
						if !ok {
							return nil, fmt.Errorf("@else value must be a mapping")
						}
						for valk, valv := range elseDict {
							res[valk] = valv
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
		case map[interface{}]interface{}:
			newv, err := recurse(vv, settings, flattened, o)
			if err != nil {
				return nil, err
			}
			res[rk] = newv
		case []interface{}:
			newv, err := recurseList(vv, settings, flattened, o)
			if err != nil {
				return nil, err
			}
			res[rk] = newv
		case string:
			vvv, err := eval(vv, flattened, o)
			if err != nil {
				return nil, err
			}
			res[rk] = vvv
		default:
			res[rk] = v
		}
	}
	return res, nil
}

// ProcessStrings resolves input templated yaml using values in settings yaml
func ProcessStrings(input, settings string) (string, error) {
	ps := make(map[interface{}]interface{})
	err := yaml.Unmarshal([]byte(settings), ps)
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

// Process resolves input templated yaml using values given in settings
func Process(inputString string, settings map[string]interface{}, opts ...string) (map[interface{}]interface{}, error) {
	var o options
	for _, v := range opts {
		switch v {
		case OptionErrOnMissingKey:
			o.errOnMissingKey = true
		default:
			return nil, fmt.Errorf("unknown option '%s'", v)
		}
	}
	input := make(map[interface{}]interface{})
	err := yaml.Unmarshal([]byte(inputString), input)
	if err != nil {
		return nil, err
	}
	flattened := make(map[string]string)
	flatten(settings, flattened, "")
	return recurse(input, settings, flattened, o)
}
