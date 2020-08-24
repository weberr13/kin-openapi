package openapi3

import (
	"context"
	"github.com/getkin/kin-openapi/jsoninfo"
)

type Swagger struct {
	Metadata `json:"-"`
	ExtensionProps
	OpenAPI      string               `json:"openapi"` // Required
	Info         Info                 `json:"info"`    // Required
	Servers      Servers              `json:"servers,omitempty"`
	Paths        Paths                `json:"paths,omitempty"`
	Components   Components           `json:"components,omitempty"`
	Security     SecurityRequirements `json:"security,omitempty"`
	ExternalDocs *ExternalDocs        `json:"externalDocs,omitempty"`
}

// FixRefsAndMarshalJSON rebuilds top level object->reference maps to convert all
// hydrated objects to references during marshaling.
// This avoids going off a cliff when dealing with recursive types
//
// NOTE: this works for schemas using references for properties etc, not for schemas that are aliases:
// in the example below, A will be fixed, B will not
// components:
//  schemas:
//    A:
//     properties
//      p1:
//       $ref: "#/components/schemas/C
//    B:
//     $ref: "#/components/schemas/C
//    C:
//     properties
//      name:
//       type: string
func (swagger *Swagger) FixRefsAndMarshalJSON() ([]byte, error) {

	// rebuild top level object->reference maps to convert all hydrated objects to references during marshaling
	// this avoids going off a cliff when dealing with recursive types
	m := BuildCompoenentRefMap(swagger)

	// todo: currently only schemas are constructed this way, extend to cover all other entities under components
	sm := swagger.Components.Schemas
	swagger.Components.Schemas = map[string]*SchemaRef{}
	for k := range m {
		switch t := k.(type) {
		case *Schema:
			swagger.Components.Schemas[t.ID] = &SchemaRef{Value: t}
		}
	}

	// pass the reference map to the json parser. It has to be a global since there's no way
	// to bundle this in the json encoder as the code interleaves jsoninfo and json.Marshal
	// todo: find a better way to do this
	jsoninfo.SetMarshalContext(m)

	b, e := swagger.MarshalJSON()

	// restore previous structures
	swagger.Components.Schemas = sm

	return b, e
}

func (swagger *Swagger) MarshalJSON() ([]byte, error) {
	return jsoninfo.MarshalStrictStruct(swagger)
}

func (swagger *Swagger) UnmarshalJSON(data []byte) error {
	return jsoninfo.UnmarshalStrictStruct(data, swagger)
}

func (swagger *Swagger) AddOperation(path string, method string, operation *Operation) {
	paths := swagger.Paths
	if paths == nil {
		paths = make(Paths)
		swagger.Paths = paths
	}
	pathItem := paths[path]
	if pathItem == nil {
		pathItem = &PathItem{}
		paths[path] = pathItem
	}
	pathItem.SetOperation(method, operation)
}

func (swagger *Swagger) AddServer(server *Server) {
	swagger.Servers = append(swagger.Servers, server)
}

func (swagger *Swagger) Validate(c context.Context) error {
	if err := swagger.Components.Validate(c); err != nil {
		return err
	}
	if v := swagger.Security; v != nil {
		if err := v.Validate(c); err != nil {
			return err
		}
	}
	if paths := swagger.Paths; paths != nil {
		if err := paths.Validate(c); err != nil {
			return err
		}
	}
	if v := swagger.Servers; v != nil {
		if err := v.Validate(c); err != nil {
			return err
		}
	}
	if v := swagger.Paths; v != nil {
		if err := v.Validate(c); err != nil {
			return err
		}
	}
	return nil
}

func (swagger *Swagger) buildRootComponentMarshalContext() map[interface{}]string {
	m := map[interface{}]string{}
	refPrefix := "#/components/"
	components := swagger.Components

	for k, v := range components.Schemas {
		m[v] = refPrefix + "schemas" + k
	}
	for k, v := range components.Parameters {
		m[v] = refPrefix + "parameters" + k
	}
	for k, v := range components.RequestBodies {
		m[v] = refPrefix + "requestBodies" + k
	}
	for k, v := range components.Responses {
		m[v] = refPrefix + "responses" + k
	}
	for k, v := range components.Headers {
		m[v] = refPrefix + "headers" + k
	}
	for k, v := range components.SecuritySchemes {
		m[v] = refPrefix + "securitySchemes" + k
	}

	return m
}
