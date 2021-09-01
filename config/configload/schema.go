package configload

import (
	"reflect"
	"regexp"

	"github.com/avenga/couper/config"
	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/gohcl"
)

const (
	noLabelForErrorHandler = "No labels are expected for error_handler blocks."
	summUnsupportedAttr    = "Unsupported argument"
	summUnsupportedBlock   = "Unsupported block type"
)

var (
	reFetchUnsupportedName = regexp.MustCompile(`\"(.*)\"`)
	reFetchLabeledName     = regexp.MustCompile(`All (.*) blocks must have .* labels \(.*\).`)
	reFetchUnlabeledName   = regexp.MustCompile(`No labels are expected for (.*) blocks.`)
	reFetchUnexpectedArg   = regexp.MustCompile(`An argument named (.*) is not expected here.`)
)

func ValidateConfigSchema(body hcl.Body, obj interface{}) hcl.Diagnostics {
	var errors hcl.Diagnostics

	attrs, blocks, diags := getSchemaComponents(body, obj)
	if diags.HasErrors() {
		for _, err := range diags {
			if err.Detail == noLabelForErrorHandler {
				continue
			}

			matches := reFetchUnsupportedName.FindStringSubmatch(err.Detail)
			if len(matches) != 2 {
				match := reFetchLabeledName.MatchString(err.Detail)
				if match {
					errors = errors.Append(err)
					continue
				}

				match = reFetchUnlabeledName.MatchString(err.Detail)
				if match {
					errors = errors.Append(err)
					continue
				}

				match = reFetchUnexpectedArg.MatchString(err.Detail)
				if match {
					errors = errors.Append(err)
					continue
				}

				errors = errors.Append(&hcl.Diagnostic{
					Severity: hcl.DiagError,
					Subject:  err.Subject,
					Summary:  "cannot match argument name from: " + err.Detail,
				})

				continue
			}

			name := matches[1]

			if err.Summary == summUnsupportedAttr {
				if _, ok := attrs[name]; ok {
					continue
				}

				errors = errors.Append(err)
			} else if err.Summary == summUnsupportedBlock {
				if len(blocks.OfType(name)) > 0 {
					continue
				}

				errors = errors.Append(err)
			}
		}
	}

	typ := reflect.TypeOf(obj)
	if typ.Kind() == reflect.Ptr {
		typ = typ.Elem()
	}

	val := reflect.ValueOf(obj)
	if val.Kind() == reflect.Ptr {
		val = val.Elem()
	}

	for _, block := range blocks {
		for i := 0; i < typ.NumField(); i++ {
			field := typ.Field(i)

			if _, ok := field.Tag.Lookup("hcl"); !ok {
				continue
			}
			if field.Tag.Get("hcl") != block.Type+",block" {
				continue
			}

			if field.Type.Kind() == reflect.Ptr {
				o := reflect.New(field.Type.Elem()).Interface()
				errors = errors.Extend(ValidateConfigSchema(block.Body, o))

				continue
			} else if field.Type.Kind() == reflect.Slice {
				v := reflect.TypeOf(val.Field(i).Interface())
				if v.Kind() == reflect.Slice {
					v = v.Elem()
				}

				field := reflect.ValueOf(v)
				if field.Kind() == reflect.Ptr {
					field = field.Elem()
				}

				if field.Kind() == reflect.Struct {
					o := reflect.New(v.Elem()).Interface()
					errors = errors.Extend(ValidateConfigSchema(block.Body, o))

					continue
				}
			}
		}
	}

	return errors
}

func getSchemaComponents(body hcl.Body, obj interface{}) (hcl.Attributes, hcl.Blocks, hcl.Diagnostics) {
	var (
		attrs  hcl.Attributes = make(hcl.Attributes)
		blocks hcl.Blocks
		diags  hcl.Diagnostics
	)

	schema, _ := gohcl.ImpliedBodySchema(obj)
	ty := reflect.TypeOf(obj)
	if ty.Kind() == reflect.Ptr {
		ty = ty.Elem()
	}

	ct := ty.NumField()
	for i := 0; i < ct; i++ {
		field := ty.Field(i)

		if field.Type.String() == "config.AccessControlSetter" {
			schema = config.SchemaWithACSetter(schema)
			break
		}
	}

	content, errors := body.Content(schema)
	diags = diags.Extend(errors)

	if content != nil {
		for name, attr := range content.Attributes {
			attrs[name] = attr
		}

		blocks = append(blocks, content.Blocks...)
	}

	if i, ok := obj.(config.Inline); ok {
		schema := i.Schema(true)
		content, errors := body.Content(schema)

		for _, diag := range errors {
			if match := reFetchLabeledName.MatchString(diag.Detail); match {
				bodyContent := bodyToContent(body)

				added := false
				for _, block := range bodyContent.Blocks {
					if block.Type == "proxy" || block.Type == "request" || block.Type == "backend" {
						blocks = append(blocks, block)

						added = true
					}
				}

				if !added {
					diags = diags.Append(diag)
				}
			} else {
				diags = diags.Append(diag)
			}
		}

		if content != nil {
			for name, attr := range content.Attributes {
				attrs[name] = attr
			}

			blocks = append(blocks, content.Blocks...)
		}
	}

	return attrs, blocks, diags
}
