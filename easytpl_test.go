package easytpl

import (
	"context"
	"fmt"
	"reflect"
	"strings"
	"testing"

	"github.com/teamwork/test"
)

type TestKeys struct {
	keys Keys
}

func (k TestKeys) TemplateKeys(ctx context.Context, p Keys) Keys { return k.keys }
func (k TestKeys) TemplateCallbacks(ctx context.Context, key string, p Keys) (string, bool) {
	switch strings.ToLower(key) {
	case "useit":
		return "cb value", true
	default:
		return "", false
	}
}

func TestText(t *testing.T) {
	cases := []struct {
		in          string
		keys        map[string]Templateable
		expected    string
		expectedErr error
	}{
		// Basic test
		{
			`<a href="http://example.com/{%Test.Test%}?a={%Test.Test%}">{%Test.Test%}</a>`,
			map[string]Templateable{
				"Test": TestKeys{keys: Keys{
					"Test": "hello <world>",
				}},
			},
			`<a href="http://example.com/hello <world>?a=hello <world>">hello <world></a>`,
			nil,
		},
		// Allow nil
		{
			`Hello {%world%}`,
			nil,
			`Hello {%world%}`,
			nil,
		},
		// Don't fret about spaces
		{
			`Hello, {%     Test.Test     %}! {% Test.Test%}`,
			map[string]Templateable{
				"Test": TestKeys{keys: Keys{
					"Test": "world",
				}},
			},
			`Hello, world! world`,
			nil,
		},
		// Reasonably case-insensitive ({%TEST.TEST%} and such won't work, which
		// is probably okay.
		{
			`{%test.test%} {%Test.test%} {%test.Test%} {%TEst.test%}`,
			map[string]Templateable{
				"Test": TestKeys{keys: Keys{
					"Test": "world",
				}},
			},
			`world world world <no value>`,
			nil,
		},
		// Fallback
		{
			//`Thank you for sending an e-mail to {%inbox.name,fallback=föö bäρ%}. We will respond shortly.`,
			`Thank you for sending an e-mail to {%inbox.name,fallback=foo bar%}. We will respond shortly.`,
			map[string]Templateable{
				"inbox": TestKeys{keys: Keys{
					"name": false,
				}},
			},
			`Thank you for sending an e-mail to foo bar. We will respond shortly.`,
			nil,
		},
		{
			`Thank you for sending an e-mail to {%inbox.xxx,fallback=föö bäρ%}. We will respond shortly.`,
			map[string]Templateable{
				"inbox": TestKeys{keys: Keys{
					"name": false,
				}},
			},
			`Thank you for sending an e-mail to föö bäρ. We will respond shortly.`,
			nil,
		},
		// Don't panic on bad input (return original+err).
		{
			`Hey {%Black Lighting%} is really great {%beer%}.`,
			map[string]Templateable{
				"inbox": TestKeys{keys: Keys{
					"name": false,
				}},
			},
			`Hey {%Black Lighting%} is really great {%beer%}.`,
			nil,
		},
		// <no value>
		{
			`Thank you for sending an e-mail to {%Inbox.Name%}.  We will respond shortly.`,
			map[string]Templateable{},
			`Thank you for sending an e-mail to <no value>.  We will respond shortly.`,
			nil,
		},
		// Callbacks
		{
			`Hello {%test.useit%}`,
			map[string]Templateable{"Test": TestKeys{}},
			`Hello cb value`,
			nil,
		},
		{
			// Too many parameters; just return original.
			`Hello {%Test,fallback=foo,another%}`,
			map[string]Templateable{},
			`Hello {%Test,fallback=foo,another%}`,
			nil,
		},
	}

	for i, tc := range cases {
		t.Run(fmt.Sprintf("%v", i), func(t *testing.T) {
			out, outErr := Text(context.Background(), tc.in, tc.keys, nil)
			if !reflect.DeepEqual(outErr, tc.expectedErr) {
				t.Errorf("wrong error\nout:      %v\nexpected: %v\n", outErr, tc.expectedErr)
			}
			if out != tc.expected {
				t.Errorf("\nout:      %v\nexpected: %v\n", out, tc.expected)
			}
		})
	}
}

func TestHTML(t *testing.T) {
	cases := []struct {
		in          string
		keys        map[string]Templateable
		expected    string
		expectedErr error
	}{
		{
			`<a href="http://example.com/{%Test.Test%}?a={%Test.Test%}">{%Test.Test%}</a>`,
			map[string]Templateable{
				"Test": TestKeys{keys: Keys{
					"Test": "hello <world>",
				}},
			},
			`<a href="http://example.com/hello%20%3cworld%3e?a=hello%20%3cworld%3e">hello &lt;world&gt;</a>`,
			nil,
		},
	}

	for i, tc := range cases {
		t.Run(fmt.Sprintf("%v", i), func(t *testing.T) {
			out, outErr := HTML(context.Background(), tc.in, tc.keys, Keys{})
			if !reflect.DeepEqual(outErr, tc.expectedErr) {
				t.Errorf("wrong error\nout:      %v\nexpected: %v\n", outErr, tc.expectedErr)
			}
			if out != tc.expected {
				t.Errorf("\nout:      %v\nexpected: %v\n", out, tc.expected)
			}
		})
	}
}

func TestHTMLSafe(t *testing.T) {
	cases := []struct {
		in          string
		keys        map[string]Templateable
		expected    string
		expectedErr error
	}{
		{
			`<a href="http://example.com/{%Test.Test%}?a={%Test.Test%}">{%Test.Test%}</a>`,
			map[string]Templateable{
				"Test": TestKeys{keys: Keys{
					"Test": "hello <world>",
				}},
			},
			`<a href="http://example.com/hello%20%3cworld%3e?a=hello%20%3cworld%3e">hello &lt;world&gt;</a>`,
			nil,
		},

		{
			`<a href="http://example.com/{%Test.Test%}?a={%Test.Test%}">{%Test.Test%}</a><br`,
			map[string]Templateable{
				"Test": TestKeys{keys: Keys{
					"Test": "hello <world>",
				}},
			},
			`<a href="http://example.com/hello <world>?a=hello <world>">hello <world></a><br`,
			nil,
		},
	}

	for i, tc := range cases {
		t.Run(fmt.Sprintf("%v", i), func(t *testing.T) {
			out, outErr := HTMLSafe(context.Background(), tc.in, tc.keys, Keys{})
			if !reflect.DeepEqual(outErr, tc.expectedErr) {
				t.Errorf("wrong error\nout:      %v\nexpected: %v\n", outErr, tc.expectedErr)
			}
			if out != tc.expected {
				t.Errorf("\nout:      %v\nexpected: %v\n", out, tc.expected)
			}
		})
	}
}

func TestReplaceTemplateFallback(t *testing.T) {
	cases := []struct {
		in       string
		expected string
	}{
		{
			"{{.Inbox.Name}}",
			"{{.Inbox.Name}}",
		},
		{
			"{{.Inbox.Name,fallback=single}}",
			"{{if .Inbox.Name}}{{.Inbox.Name}}{{else}}single{{end}}",
		},
		{
			"{{.Inbox.Name,fallback=multiple words}}",
			"{{if .Inbox.Name}}{{.Inbox.Name}}{{else}}multiple words{{end}}",
		},
		{
			"{{.Inbox.Name,fallback= words with spaces}}",
			"{{if .Inbox.Name}}{{.Inbox.Name}}{{else}}words with spaces{{end}}",
		},
		{
			"{{.Inbox.Name,fallback = space after fallback word}}",
			"{{if .Inbox.Name}}{{.Inbox.Name}}{{else}}space after fallback word{{end}}",
		},
		{
			"{{.Inbox.Name, fallback = space before fallback word}}",
			"{{if .Inbox.Name}}{{.Inbox.Name}}{{else}}space before fallback word{{end}}",
		},
		{
			"{{.Inbox.Name , fallback = space before comma}}",
			"{{if .Inbox.Name}}{{.Inbox.Name}}{{else}}space before comma{{end}}",
		},
	}

	for _, tc := range cases {
		t.Run(tc.in, func(t *testing.T) {
			out := replaceTemplateFallback(tc.in)
			if out != tc.expected {
				t.Errorf("\nout:      %v\nexpected: %v\n", out, tc.expected)
			}
		})
	}
}

func TestPrepareTemplateTags(t *testing.T) {
	cases := []struct {
		in       string
		expected string
	}{
		{
			`this is a test {% template.tag %} and another one {% t.fe, fallback = testing %} and done`,
			`this is a test {{.Template.Tag}} and another one {{if .T.Fe}}{{.T.Fe}}{{else}}testing{{end}} and done`,
		},
		{
			`this is a test \{% template.tag %} and another one {% t.fe, fallback = testing %} and done`,
			`this is a test {% template.tag %} and another one {{if .T.Fe}}{{.T.Fe}}{{else}}testing{{end}} and done`,
		},

		{
			`this is a test {% template.tag %} and another one \{% t.fe, fallback = testing %} and done`,
			`this is a test {{.Template.Tag}} and another one {% t.fe, fallback = testing %} and done`,
		},
		{
			`this is a test {{ foo }}`,
			`this is a test {{ "{{ foo }}" }}`,
		},
		{
			`this is a test {{ foo }} bar {% template.tag %} {{hello}}`,
			`this is a test {{ "{{ foo }}" }} bar {{.Template.Tag}} {{ "{{hello}}" }}`,
		},

		{
			`this is a test {{ "{{ foo }}" }} bar {% template.tag %} {{hello}}`,
			`this is a test {{ "{{ foo }}" }} bar {{.Template.Tag}} {{ "{{hello}}" }}`,
		},
		// This is how Squire sends multiple spaces.
		// TODO: Fix this!
		//{
		//	`hello {% &nbsp;  template.tag&nbsp;&nbsp;&nbsp; %} world`,
		//	`hello {{.Template.Tag}} world`,
		//},
		//{
		//	`&nbsp; hello &nbsp; {% &nbsp;  template.tag&nbsp;&nbsp;&nbsp; %} &nbsp; world &nbsp;`,
		//	`hello {{.Template.Tag}} world`,
		//},
	}

	for i, tc := range cases {
		t.Run(fmt.Sprintf("%v", i), func(t *testing.T) {
			out, _ := prepareTemplateTags(tc.in)
			if out != tc.expected {
				t.Errorf("\nout:      %v\nexpected: %v\n", out, tc.expected)
			}
		})
	}
}

func TestTestSafe(t *testing.T) {
	cases := []struct {
		in      string
		keys    map[string]Templateable
		want    string
		wantErr string
	}{
		{
			"Hello",
			map[string]Templateable{},
			"Hello",
			"",
		},
		{
			"Hello {% asd.zxc %}",
			map[string]Templateable{
				"Asd": TestKeys{keys: Keys{
					"Zxc": "world",
				}},
			},
			"Hello world",
			"",
		},
		{
			"Hello {% asd.zxc %}",
			map[string]Templateable{},
			"",
			"unknown variable Asd at Asd.Zxc",
		},

		// TODO: ideally this should error out too, but not doing so is okay for
		// now.
		{
			"Hello {% asd %}",
			map[string]Templateable{},
			"Hello {% asd %}",
			"",
		},
	}

	for i, tc := range cases {
		t.Run(fmt.Sprintf("%v", i), func(t *testing.T) {
			out, err := TestSafe(context.Background(), tc.in, tc.keys, nil)
			if out != tc.want {
				t.Errorf("wrong output\nout:      %#v\nwant: %#v\n", out, tc.want)
			}
			if !test.ErrorContains(err, tc.wantErr) {
				t.Errorf("wrong error\nout:      %#v\nwant: %#v\n", err, tc.wantErr)
			}
		})
	}
}
