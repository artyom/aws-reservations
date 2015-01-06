package model

import (
	"bytes"
	"encoding/json"
	"fmt"
	"go/format"
	"io"
	"strings"
	"text/template"
)

// A Constraint is a set of constraints on input types.
type Constraint []interface{}

// Condition returns a Go fragment which matches the constraint.
func (c Constraint) Condition() string {
	str := func(i interface{}) string {
		if i == nil {
			return ""
		}
		return i.(string)
	}

	switch c[1] {
	case "startsWith":
		return fmt.Sprintf("strings.HasPrefix(%s, %q)", str(c[0]), str(c[2]))
	case "notStartsWith":
		return fmt.Sprintf("!strings.HasPrefix(%s, %q)", str(c[0]), str(c[2]))
	case "equals":
		return fmt.Sprintf("%s == %q", str(c[0]), str(c[2]))
	case "notEquals":
		return fmt.Sprintf("%s != %q", str(c[0]), str(c[2]))
	case "oneOf":
		var values []string
		for _, v := range c[2].([]interface{}) {
			values = append(values, str(v))
		}

		var conditions []string
		for _, v := range values {
			conditions = append(conditions, fmt.Sprintf("%s == %q", str(c[0]), v))
		}

		return strings.Join(conditions, " || ")
	default:
		panic(fmt.Sprintf("unknown operator: %v", c[1]))
	}
}

// CredentialScope is a set of overrides for the service region and name.
type CredentialScope struct {
	Region  string
	Service string
}

// Properties is a set of properties associated with an Endpoint.
type Properties struct {
	CredentialScope CredentialScope
}

// An Endpoint is an URL where a service is available.
type Endpoint struct {
	Name        string
	URI         string
	Properties  Properties
	Constraints []Constraint
}

// Service returns the Go literal or variable for the service.
func (e Endpoint) Service() string {
	if e.Properties.CredentialScope.Service != "" {
		return fmt.Sprintf("%q", e.Properties.CredentialScope.Service)
	}
	return "service"
}

// Region returns the Go literal or variable for the region.
func (e Endpoint) Region() string {
	if e.Properties.CredentialScope.Region != "" {
		return fmt.Sprintf("%q", e.Properties.CredentialScope.Region)
	}
	return "region"
}

// Conditions returns the conjunction of the conditions for the endpoint.
func (e Endpoint) Conditions() string {
	var conds []string
	for _, c := range e.Constraints {
		conds = append(conds, "("+c.Condition()+")")
	}
	return strings.Join(conds, " && ")
}

// Endpoints are a set of named endpoints.
type Endpoints map[string][]Endpoint

// Parse parses the JSON description of the endpoints.
func (e *Endpoints) Parse(r io.Reader) error {
	return json.NewDecoder(r).Decode(e)
}

// Generate writes a Go file to the given writer.
func (e Endpoints) Generate(w io.Writer) error {
	tmpl, err := template.New("endpoints").Parse(t)
	if err != nil {
		return err
	}

	out := bytes.NewBuffer(nil)
	if err := tmpl.Execute(out, e); err != nil {
		return err
	}

	b, err := format.Source(bytes.TrimSpace(out.Bytes()))
	if err != nil {
		return err
	}

	_, err = io.Copy(w, bytes.NewReader(b))
	return err
}

const t = `
// Package endpoints provides lookups for all AWS service endpoints.
package endpoints

import (
  "strings"
)

// Lookup returns the endpoint for the given service in the given region plus
// any overrides for the service name and region.
func Lookup(service, region string) (uri, newService, newRegion string) {
  switch service {
    {{ range $name, $endpoints := . }}
    {{ if ne $name "_default" }}
    case "{{ $name}}" :
    {{ range $endpoints }}
      {{ if .Constraints }}if {{ .Conditions }} { {{ end }}
        return format("{{ .URI }}", service, region), {{ .Service }}, {{ .Region }}
      {{ if .Constraints }} } {{ end }}
    {{ end }}
    {{ end }}
    {{ end }}
  }

  {{ with $endpoints := index . "_default" }}
  {{ range $endpoints }}
    {{ if .Constraints }}if {{ .Conditions }} { {{ end }}
      return format("{{ .URI }}", service, region), {{ .Service }}, {{ .Region }}
    {{ if .Constraints }} } {{ end }}
  {{ end }}
  {{ end }}

  panic("unknown endpoint for " + service + " in " + region)
}

func format(uri, service, region string) string {
  uri = strings.Replace(uri, "{scheme}", "https", -1)
  uri = strings.Replace(uri, "{service}", service, -1)
  uri = strings.Replace(uri, "{region}", region, -1)
  return uri
}
`
