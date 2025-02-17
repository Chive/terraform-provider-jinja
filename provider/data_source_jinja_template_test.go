package jinja

import (
	"fmt"
	"io/ioutil"
	"os"
	"path"
	"regexp"
	"testing"

	"github.com/MakeNowJust/heredoc"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
)

// args:
//   prefix [default is "tmp-"]
//   content [default is ""]
//   directory [default is current working directory]
// returns:
//   name of the file
//   content of the file
//   path to containing folder
//   callable to delete the file.
func mustCreateFile(args ...string) (string, string, string, func()) {
	if len(args) > 3 || len(args) == 0 {
		panic("mustCreateFile takes up to 3 arguments: prefix, content, directory")
	}
	var prefix, content, directory string
	switch len(args) {
	case 3:
		directory = args[2]
		fallthrough
	case 2:
		content = args[1]
		fallthrough
	case 1:
		prefix = args[0]
	case 0:
		prefix = "tmp-"
	}

	file, err := ioutil.TempFile(directory, prefix)
	if err != nil {
		panic(err)
	}
	_, err = file.WriteString(content)
	if err != nil {
		panic(err)
	}

	return path.Base(file.Name()), content, path.Dir(file.Name()), func() { os.Remove(file.Name()) }
}

func TestJinjaTemplateFormat(t *testing.T) {
	template, _, dir, remove := mustCreateFile(t.Name(), heredoc.Doc(`
	{{ "Hello %s!" | format("world") }}
	`))
	defer remove()

	resource.UnitTest(t, resource.TestCase{
		ProviderFactories: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: heredoc.Doc(`
				data "jinja_template" "render" {
					template = "` + path.Join(dir, template) + `"
				}`),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttrSet("data.jinja_template.render", "id"),
					resource.TestCheckResourceAttrWith("data.jinja_template.render", "result", func(got string) error {
						expected := heredoc.Doc(`
						Hello world!`)
						if expected != got {
							return fmt.Errorf("\nexpected:\n%s\ngot:\n%s", expected, got)
						}
						return nil
					}),
				),
			},
		},
	})
}

func TestJinjaTemplateSimple(t *testing.T) {
	template, _, dir, remove := mustCreateFile(t.Name(), heredoc.Doc(`
	{% if "foo" in "foo bar" %}
	show within loop!
	{% endif %}
	`))
	defer remove()

	resource.UnitTest(t, resource.TestCase{
		ProviderFactories: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: heredoc.Doc(`
				data "jinja_template" "render" {
					template = "` + path.Join(dir, template) + `"
				}`),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttrSet("data.jinja_template.render", "id"),
					resource.TestCheckResourceAttrWith("data.jinja_template.render", "result", func(got string) error {
						expected := heredoc.Doc(`

						show within loop!
						`)
						if expected != got {
							return fmt.Errorf("\nexpected:\n%s\ngot:\n%s", expected, got)
						}
						return nil
					}),
				),
			},
		},
	})
}

func TestJinjaTemplateWithInclude(t *testing.T) {
	nested, expected, dir, remove_nested := mustCreateFile("nested", heredoc.Doc(`
	I am nested !
	`))
	defer remove_nested()

	template, _, _, remove_template := mustCreateFile(t.Name(), heredoc.Doc(`
	{% include "`+nested+`" %}
	`), dir)
	defer remove_template()

	resource.UnitTest(t, resource.TestCase{
		ProviderFactories: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: heredoc.Doc(`
				data "jinja_template" "render" {
					template = "` + path.Join(dir, template) + `"
				}`),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttrSet("data.jinja_template.render", "id"),
					resource.TestCheckResourceAttrWith("data.jinja_template.render", "result", func(got string) error {
						if expected != got {
							return fmt.Errorf("\nexpected:\n%s\ngot:\n%s", expected, got)
						}
						return nil
					}),
				),
			},
		},
	})
}

func TestJinjaTemplateOtherDelimiters(t *testing.T) {
	template, _, dir, remove := mustCreateFile(t.Name(), heredoc.Doc(`
	|##- if "foo" in "foo bar" ##|
	I am cornered
	|##- endif ##|
	<< "but pointy" >>
	[#- "and can be invisible!" -#]
	`))
	defer remove()

	resource.UnitTest(t, resource.TestCase{
		ProviderFactories: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: heredoc.Doc(`
				data "jinja_template" "render" {
					template = "` + path.Join(dir, template) + `"
					delimiters {
						block_start = "|##"
						block_end = "##|"
						variable_start = "<<"
						variable_end = ">>"
						comment_start = "[#"
						comment_end = "#]"
					}
				}`),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttrSet("data.jinja_template.render", "id"),
					resource.TestCheckResourceAttrWith("data.jinja_template.render", "result", func(got string) error {
						expected := heredoc.Doc(`

						I am cornered
						but pointy
						`)
						if expected != got {
							return fmt.Errorf("\nexpected:\n%s\ngot:\n%s", expected, got)
						}
						return nil
					}),
				),
			},
		},
	})
}

func TestJinjaTemplateWithContextJSON(t *testing.T) {
	template, _, dir, remove := mustCreateFile(t.Name(), heredoc.Doc(`
	This is a very nested {{ top.middle.bottom.field }}

	`))
	defer remove()

	resource.UnitTest(t, resource.TestCase{
		ProviderFactories: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: heredoc.Doc(`
				data "jinja_template" "render" {
					template = "` + path.Join(dir, template) + `"
					context {
						type = "json"
						data = jsonencode({
							top = {
								middle = {
									bottom = {
										field = "surprise!"
									}
								}
							}
						})
					}
				}`),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttrSet("data.jinja_template.render", "id"),
					resource.TestCheckResourceAttrWith("data.jinja_template.render", "result", func(got string) error {
						expected := heredoc.Doc(`
						This is a very nested surprise!
						`)
						if expected != got {
							return fmt.Errorf("\nexpected:\n%s\ngot:\n%s", expected, got)
						}
						return nil
					}),
				),
			},
		},
	})
}

func TestJinjaTemplateWithContextYAML(t *testing.T) {
	template, _, dir, remove := mustCreateFile(t.Name(), heredoc.Doc(`
	The service name is {{ name }}
	{%- if flags %}
	The flags in the file are:
		{%- for flag in flags %}
	- {{ flag }}
		{%- endfor %}
	{% endif %}
	`))
	defer remove()

	resource.UnitTest(t, resource.TestCase{
		ProviderFactories: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: heredoc.Doc(`
				data "jinja_template" "render" {
					template = "` + path.Join(dir, template) + `"
					context {
						type = "yaml"
						data = yamlencode({
							name = "NATO"
							flags = [
								"fr",
								"us",
							]
						})
					}
				}`),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttrSet("data.jinja_template.render", "id"),
					resource.TestCheckResourceAttrWith("data.jinja_template.render", "result", func(got string) error {
						expected := heredoc.Doc(`
						The service name is NATO
						The flags in the file are:
						- fr
						- us
						`)
						if expected != got {
							return fmt.Errorf("\nexpected:\n%s\ngot:\n%s", expected, got)
						}
						return nil
					}),
				),
			},
		},
	})
}

func TestJinjaTemplateOtherDelimitersAtProviderLevel(t *testing.T) {
	template, _, dir, remove := mustCreateFile(t.Name(), heredoc.Doc(`
	[%- if "foo" in "foo bar" %]
	I am cornered
	[%- endif %]
	<< "but pointy" >>
	|#- "and can be invisible!" -#|
	`))
	defer remove()

	resource.UnitTest(t, resource.TestCase{
		ProviderFactories: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: heredoc.Doc(`
				provider "jinja" {
					delimiters {
						variable_start = "<<"
						variable_end = ">>"
						comment_start = "|#"
						comment_end = "#|"
					}
				}
				data "jinja_template" "render" {
					template = "` + path.Join(dir, template) + `"
					delimiters {
						block_start = "[%"
						block_end = "%]"
					}
				}`),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttrSet("data.jinja_template.render", "id"),
					resource.TestCheckResourceAttrWith("data.jinja_template.render", "result", func(got string) error {
						expected := heredoc.Doc(`

						I am cornered
						but pointy
						`)
						if expected != got {
							return fmt.Errorf("\nexpected:\n%s\ngot:\n%s", expected, got)
						}
						return nil
					}),
				),
			},
		},
	})
}

func TestJinjaTemplateWithPathContext(t *testing.T) {
	ctx, _, dir, remove_context := mustCreateFile("nested", heredoc.Doc(`
	name: remote-context
	`))
	defer remove_context()

	template, _, _, remove_template := mustCreateFile(t.Name(), heredoc.Doc(`
	The name field is: "{{ name }}"
	`), dir)
	defer remove_template()

	resource.UnitTest(t, resource.TestCase{
		ProviderFactories: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: heredoc.Doc(`
				data "jinja_template" "render" {
					template = "` + path.Join(dir, template) + `"
					context {
						type = "yaml"
						data = "` + path.Join(dir, ctx) + `"
					}
				}`),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttrSet("data.jinja_template.render", "id"),
					resource.TestCheckResourceAttrWith("data.jinja_template.render", "result", func(got string) error {
						expected := heredoc.Doc(`
						The name field is: "remote-context"`)
						if expected != got {
							return fmt.Errorf("\nexpected:\n%s\ngot:\n%s", expected, got)
						}
						return nil
					}),
				),
			},
		},
	})
}

func TestJinjaTemplateWithSchema(t *testing.T) {
	schema, _, dir, remove_schema := mustCreateFile("nested", heredoc.Doc(`
	{
		"type": "object",
		"required": [
			"name",
			"other"
		],
		"properties": {
			"name": {
				"type": "string"
			},
			"other": {
				"type": "object",
				"required": ["foo"],
				"properties": {
					"foo": {
						"type": "string"
					}
				}
			}
	
		}
	}
	`))
	defer remove_schema()

	template, _, _, remove_template := mustCreateFile(t.Name(), heredoc.Doc(`
	The name field is: "{{ name }}"
	`), dir)
	defer remove_template()

	resource.UnitTest(t, resource.TestCase{
		ProviderFactories: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: heredoc.Doc(`
				data "jinja_template" "render" {
					template = "` + path.Join(dir, template) + `"
					context {
						type = "yaml"
						data = yamlencode({
							name = "schema"
							other = {
								"foo" = "bar"
							}
						})
					}
					schema = "` + path.Join(dir, schema) + `"
				}`),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttrSet("data.jinja_template.render", "id"),
					resource.TestCheckResourceAttrWith("data.jinja_template.render", "result", func(got string) error {
						expected := heredoc.Doc(`
						The name field is: "schema"`)
						if expected != got {
							return fmt.Errorf("\nexpected:\n%s\ngot:\n%s", expected, got)
						}
						return nil
					}),
				),
			},
		},
	})
}

func TestJinjaTemplateWithInlineSchema(t *testing.T) {
	template, _, dir, remove := mustCreateFile(t.Name(), heredoc.Doc(`
	The name field is: "{{ name }}"
	`))
	defer remove()

	resource.UnitTest(t, resource.TestCase{
		ProviderFactories: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: heredoc.Doc(`
				data "jinja_template" "render" {
					template = "` + path.Join(dir, template) + `"
					context {
						type = "yaml"
						data = yamlencode({
							name = "schema"
						})
					}
					schema = <<-EOF
					{
						"type": "object",
						"required": [
							"name"
						],
						"properties": {
							"name": {
								"type": "string"
							}
						}
					}
					EOF
				}`),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttrSet("data.jinja_template.render", "id"),
					resource.TestCheckResourceAttrWith("data.jinja_template.render", "result", func(got string) error {
						expected := heredoc.Doc(`
						The name field is: "schema"`)
						if expected != got {
							return fmt.Errorf("\nexpected:\n%s\ngot:\n%s", expected, got)
						}
						return nil
					}),
				),
			},
		},
	})
}
func TestJinjaTemplateWithSchemaThatFails(t *testing.T) {
	schema, _, dir, remove_schema := mustCreateFile("nested", heredoc.Doc(`
	{
		"type": "object",
		"required": [
			"name"
		],
		"properties": {
			"name": {
				"type": "string"
			}	
		}
	}
	`))
	defer remove_schema()

	template, _, _, remove_template := mustCreateFile(t.Name(), heredoc.Doc(`
	The name field is: "{{ name }}"
	`), dir)
	defer remove_template()

	resource.UnitTest(t, resource.TestCase{
		ProviderFactories: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: heredoc.Doc(`
				data "jinja_template" "render" {
					template = "` + path.Join(dir, template) + `"
					context {
						type = "yaml"
						data = yamlencode({})
					}
					schema = "` + path.Join(dir, schema) + `"
				}`),
				ExpectError: regexp.MustCompile("Error: failed to validate context against schema: failed to pass JSON schema validation: jsonschema: '' does not validate with .*: missing properties: 'name'"),
			},
		},
	})
}
