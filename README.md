[![Build Status](https://travis-ci.com/Teamwork/easytpl.svg?token=VszHEX46e27fhnkZbvFm&branch=master)](https://travis-ci.com/Teamwork/easytpl)
[![codecov](https://codecov.io/gh/Teamwork/easytpl/branch/master/graph/badge.svg)](https://codecov.io/gh/Teamwork/easytpl)
[![GoDoc](https://godoc.org/github.com/Teamwork/easytpl?status.svg)](https://godoc.org/github.com/Teamwork/easytpl)

A very simple template system, intended for simple customer-facing templates.

easytpl transforms to Go's template system as follows:

```go
{%var%}                          -> {{Var}}
{%var.val%}                      -> {{Var.Val}}
{%var.val,fallback=some string%} -> {{if .Var.Val}}{{.Var.Val}}{{else}}some string{{end}}
{{var.val}}                      -> {{ "{{var.val}}" }}
```

Template variables can also be escaped so that they are not translated into Go's template system, like so:

```
\{%var%} -> {%var%}
```

Function calls

```go
{%@user.HasPermission "feature-x"%} -> {{call .user.HasPermission "feature-x"}}
```

That's all :-) It doesn't support if, range, or anything else.
