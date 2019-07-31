// Package easytpl is a very simple template system.
//
// easytpl transforms to Go's template system as follows:
//
//  {%var%}                          -> {{Var}}
//  {%var.val%}                      -> {{Var.Val}}
//  {%var.val,fallback=some string%} -> {{if .Var.Val}}{{.Var.Val}}{{else}}some string{{end}}
//  {{var.val}}                      -> {{ "{{var.val}}" }}
//
// That's all :-) It doesn't support if, range, or anything else.
package easytpl

import (
	"bytes"
	"context"
	"fmt"
	htmlTemplate "html/template"
	"io"
	"regexp"
	"strings"
	textTemplate "text/template"

	"github.com/teamwork/utils/sliceutil"
)

// Templateable allows template substitution.
type Templateable interface {
	// TemplateKeys returns a zero or more keys to be used as template
	// parameters; e.g.
	//
	//   return easytpl.Params{
	//   	"ID": obj.ID,
	//   }
	//
	// The parentKeys are what you passed in with Text()/HTML()/HTMLSafe().
	TemplateKeys(ctx context.Context, parentKeys Keys) (params Keys)

	// TemplateCallbacks allows getting a template key dynamically only when
	// it's used. The advantage of this is performance (some variables are
	// expensive but not used a lot).
	//
	// If setIt is false, we won't set anything for this key.
	TemplateCallbacks(ctx context.Context, key string, parentKeys Keys) (value string, setIt bool)
}

// Keys are parameters passed to the template.
type Keys map[string]interface{}

var (
	escapeGo           = regexp.MustCompile(`\{\{.*?\}\}`)
	tagsToGo           = regexp.MustCompile(`(?i)\\?(\{\%|\%7B\%25)+(.*?)(\%\}|\%25\%7D)+`)
	findFallback       = regexp.MustCompile(`(?i)\s*(fallback)\s*=`)
	parseTemplateError = regexp.MustCompile(`executing "email" at \<\.(.*?)\>.*?\"(.*?)\"`)
)

// Text attempts to parse the template with text/template. The templatables in
// keys will be used as template parameters; for example with:
//
//   map[string]Templateable{
//       "Inbox": inbox,
//   }
//
// You get {%inbox.ID%} (or any other keys that the inbox type has).
//
// The parentKeys are always passed to every Templateable; it's useful to pass
// some global state around (e.g. context, session, etc.)
//
// If processing fails the original template and an error are returned.
func Text(ctx context.Context, body string, keys map[string]Templateable, parentKeys Keys) (string, error) {
	body, usedVars := prepareTemplateTags(body)
	tmpl, err := textTemplate.New("email").Parse(body)
	if err != nil {
		return body, err
	}
	return replaceVariables(ctx, tmpl, body, keys, parentKeys, usedVars)
}

// HTML works exactly like Text() but uses the html/template package so it will
// escape the variables values.
func HTML(ctx context.Context, body string, keys map[string]Templateable, parentKeys Keys) (string, error) {
	body, usedVars := prepareTemplateTags(body)
	tmpl, err := htmlTemplate.New("email").Parse(body)
	if err != nil {
		return body, err
	}
	return replaceVariables(ctx, tmpl, body, keys, parentKeys, usedVars)
}

// TestSafe will test the given data strictly against the HTMLSafe standard to
// find variable errors in particular.
func TestSafe(ctx context.Context, body string, keys map[string]Templateable, parentKeys Keys) (string, error) {
	body, usedVars := prepareTemplateTags(body)

	var (
		tmpl tpl
		err  error
	)
	tmpl, err = htmlTemplate.New("email").Option("missingkey=error").Parse(body)
	if err != nil {
		return "", err
	}

	out, err := replaceVariables(ctx, tmpl, body, keys, parentKeys, usedVars)
	if err != nil {
		tmpl, err = textTemplate.New("email").Option("missingkey=error").Parse(body)
		if err != nil {
			return "", err
		}

		out, err = replaceVariables(ctx, tmpl, body, keys, parentKeys, usedVars)
	}

	if err != nil {
		if _, ok := err.(textTemplate.ExecError); !ok {
			return out, err
		}

		variables := parseTemplateError.FindAllStringSubmatch(err.Error(), -1)
		if len(variables) < 1 {
			return out, err
		}

		if len(variables[0]) < 2 {
			return out, err
		}

		err = fmt.Errorf("unknown variable %s at %s", variables[0][2], variables[0][1])
	}

	if err != nil {
		return "", err
	}
	return out, nil
}

// HTMLSafe works exactly like HTML() but will fall back to Text() if HTML()
// fails.
// This is to allow people to use broken HTML. We can't fix the world,
// unfortunately.
//
// HTML() errors will be logged. Errors from Text() will be returned.
func HTMLSafe(ctx context.Context, body string, keys map[string]Templateable, parentKeys Keys) (string, error) {
	out, err := HTML(ctx, body, keys, parentKeys)
	if err != nil {
		out, err = Text(ctx, body, keys, parentKeys)
	}

	return out, err
}

type tpl interface {
	Execute(wr io.Writer, data interface{}) error
}

func replaceVariables(
	ctx context.Context,
	tmpl tpl,
	body string,
	keys map[string]Templateable,
	parentKeys Keys,
	usedVars map[string][]string,
) (string, error) {
	// Build params
	params := make(map[string]Keys)
	for key, value := range keys {
		params[key] = value.TemplateKeys(ctx, parentKeys)
		if params[key] == nil {
			params[key] = make(Keys)
		}

		if used, has := usedVars[key]; has {
			for _, v := range used {
				if result, setIt := value.TemplateCallbacks(ctx, v, parentKeys); setIt {
					params[key][v] = result
				}
			}
		}
	}

	// Parse template
	output := bytes.NewBufferString("")

	err := tmpl.Execute(output, params)
	if err != nil {
		return body, err
	}

	return output.String(), nil
}

// prepareTemplateTags parses the body input and converts:
//
// - {{anything.here}} to {{ "{{anything.here}}" }}
// - {%anything.here%} to {{.Anything.Here}}
//
// This allows us to parse this by the Go template engine.
func prepareTemplateTags(body string) (string, map[string][]string) {
	// Escape {{ text }} to {{ "{{ text }}" }}
	body = escapeGo.ReplaceAllStringFunc(body, func(match string) string {
		if strings.Contains(match, `"`) {
			return match
		}
		return fmt.Sprintf(`{{ "%s" }}`, match)
	})

	usedVars := map[string][]string{}
	// Convert {%user.firstName%} to {{.User.FirstName}}
	body = tagsToGo.ReplaceAllStringFunc(body, func(match string) string {
		// Allow escaping { with \{
		if match[0] == '\\' {
			return match[1:]
		}

		submatch := tagsToGo.FindStringSubmatch(match)
		if len(submatch) < 3 {
			return match
		}

		tagParts := strings.Split(strings.TrimSpace(submatch[2]), ".")
		if len(tagParts) < 2 {
			return match
		}

		if len(tagParts) > 0 {
			tagParts[0] = strings.ToUpper(tagParts[0][0:1]) + tagParts[0][1:]
		}

		if len(tagParts) > 1 {
			tagParts[1] = strings.ToUpper(tagParts[1][0:1]) + tagParts[1][1:]
		}

		if _, ok := usedVars[tagParts[0]]; !ok {
			usedVars[tagParts[0]] = []string{}
		}

		if !sliceutil.InStringSlice(usedVars[tagParts[0]], tagParts[1]) {
			usedVars[tagParts[0]] = append(usedVars[tagParts[0]], tagParts[1])
		}

		return replaceTemplateFallback(fmt.Sprintf("{{.%s.%s}}", tagParts[0], tagParts[1]))
	})

	return body, usedVars
}

// replaceTemplateFallback takes input like this:
//
//   {{.Inbox.Name,fallback=this inbox}}
//
// and turns it in to proper template tags:
//
//   {{if .Inbox.Name}}{{.Inbox.Name}}{{else}}this inbox{{end}}
func replaceTemplateFallback(tag string) string {
	if !strings.Contains(strings.ToLower(tag), "fallback") {
		return tag
	}

	tagBody := tag[2 : len(tag)-2]

	// Split out the variable name and fallback text
	parts := strings.SplitN(tagBody, ",", 2)
	if len(parts) != 2 {
		// This can never happen, I think... Leaving it here just in case.
		return tag
	}

	variable := strings.TrimSpace(parts[0])
	fallback := parts[1]

	// Now we have a string that looks like fallback=, so let's just get rid of
	// that, trim the string and move on
	fallback = strings.TrimSpace(findFallback.ReplaceAllString(fallback, ""))

	return fmt.Sprintf("{{if %s}}{{%s}}{{else}}%s{{end}}", variable, variable, fallback)
}
