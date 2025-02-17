---
page_title: "Jinja provider"
subcategory: ""
description: |-
 The Jinja provider is used render Jinja templates within terraform context. 
---

# Jinja provider

The "jinja" provider allows the use of [Jinja](https://jinja.palletsprojects.com) templating within Terraform configurations. This is a *logical provider*, which means that it works entirely within Terraform's logic, and doesn't interact with any other services.

The Jinja engine used under the hood is based on [the `gonja` Golang library](https://github.com/noirbizarre/gonja) and aims to be "mostly" compliant with the Jinja API.

The JSON schema validation engine is based on [the `jsonschema` Golang library](https://github.com/santhosh-tekuri/jsonschema).

## Example

```terraform
provider "jinja" {
  delimiters {
    // The values below are the defaults
    variable_start = "{{"
    variable_end   = "}}"
    block_start    = "{%"
    block_end      = "%}"
    comment_start  = "{#"
    comment_end    = "#}"
  }
}
```

<!-- schema generated by tfplugindocs -->
## Schema

### Optional

- `delimiters` (Block List, Max: 1) Provider-wide custom delimiters for the jinja engine (see [below for nested schema](#nestedblock--delimiters))

<a id="nestedblock--delimiters"></a>
### Nested Schema for `delimiters`

Optional:

- `block_end` (String)
- `block_start` (String)
- `comment_end` (String)
- `comment_start` (String)
- `variable_end` (String)
- `variable_start` (String)