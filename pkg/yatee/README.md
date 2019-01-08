# What is yatee?

Yatee is a basic YAML templating engine whose input is a valid YAML file.

# What does it support?

If, for, variable expansion, arithmetic expressions.

# Show me some examples!

    version: "3.4"
    services:
      "@for i in 0..2":
        replica$i:
          image: superduperserver:latest
          command: /run $i
          port:
            - $(5000 + ($i*2))
            - $(5001 + ($i * 2))
      "@if ${myapp.debug}":
        debugger:
          image: debug
      "@if ! ${myapp.debug}":
        monitor:
          image: monitor
      "@for i in $myapp.services":
        "$i":
          image: $i:latest

When processed with the following parameters file:

    myapp:
      debug: false
      services:
        - nginx
        - redis

Will produce the following output:

    services:
      monitor:
        image: monitor
      nginx:
        image: nginx:latest
      redis:
        image: redis:latest
      replica0:
        command: /run 0
        image: superduperserver:latest
        port:
        - "5000"
        - "5001"
      replica1:
        command: /run 1
        image: superduperserver:latest
        port:
        - "5002"
        - "5003"

# How do I invoke it?

    ./yatee TEMPLATE_FILE SETTINGS_FILES...

# How do I use it as a library?

The yatee go package exports the following two functions:

    // LoadParameters loads a set of parameters file and produce a property dictionary
    func LoadParameters(files []string) (map[string]interface{}, error)
    // Process resolves input templated yaml using values given in parameters
    func Process(inputString string, parameters map[string]interface{}) (map[interface{}] interface{}, error)

# Tell me more about the templating

## All features at a glance

- `$foo.bar and ${foo.bar}` are replaced by the value of `foo.bar` in the parameters structure. Nesting is allowed.
- `${foo?IF_TRUE:IF_FALSE}` is replaced by IF_TRUE if `foo` in parameters is true (not empty, 0 or `false`).
- `$(expr)` is evaluated as an arithmetic expression. Integers, parenthesis, and the operators
   '+-*/%' are supported. Note that there is no operator precedence, evaluation is from left to right.
- `$$` is replaced by a single literal `$` without any variable expansion.
- A YAML key of `@for VAR in begin..end` or `@for VAR in VALUE LIST` will inject the value in the
  parent node for each value of VAR.
- A YAML key of `@if VALUE` will inject its content in the parent node only if VAULE
  is not false (0, empty or 'false'). A prefix '!' is supported to negate the value. A `@else` dict can be specified
  under the `@if` node, and will be injected if the condition is false.
- A YAML key of `@switch VALUE` will inject it's sub-key's value matching VALUE to the parent node, or
  inject the value under the `default` key if present and no match is found.
- A YAML value of `@if (EXPR) VALUE` in a list will be replaced by `VALUE` if `EXPR` is true,
  suppressed otherwise

## Variable expansion examples

All examples below use the following parameters:

    app:
      debug: true
      release: false
    foo: bar
    bar: baz
    count: 2

Input | Output
----- | ------
${app.debug} | true
${foo}${bar} | barbaz
${$foo}      | baz
$(1+2*3)     | 9
$(1+(2*3))   | 7
$($count + 40) | 42
${app.debug?foo:bar}   | foo
${app.release?foo:bar} | bar
$$foo                  | $$foo
$$$foo                 | $$bar

## Control flow examples

Using the same parameters as above.

### If

    "@if !$app.release":
     shown: nope
     "@else":
       shown: yes
    somelist:
      - a
      - @if ($app.debug) b
      - @if ($app.release) c
      - d

produces:

    shown: yes
    somelist:
    - a
    - b
    - d

### For

    "@for v in 0..$(count)":
      key$v: val$($v + 1)
    "@for v in a b c":
      key$v: val$v

produces:

    key0: val1
    key1: val2
    keya: vala
    keyb: valb
    keyc: valc

### Switch

    "@switch $foo":
      baz:
        isbaz: 1
      bar:
        isbar: 1
      default:
        isother: 1
    "@switch $bar":
      foo:
        isfoo: 2
      default:
        isother:2

produces

    isbar: 1
    isother: 2
