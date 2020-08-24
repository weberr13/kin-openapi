package openapi3_test

import (
	"io/ioutil"
	"testing"

	"github.com/getkin/kin-openapi/openapi3"
	"github.com/stretchr/testify/require"
)

func TestRecursiveSchemaJSON(t *testing.T) {

	loader := openapi3.NewSwaggerLoader(openapi3.WithAllowExternalRefs(true))
	swagger, err := loader.LoadSwaggerFromFile("testdata/testref.openapi.inline.yml")
	require.NoError(t, err)

	openapi3.ClearResolvedExternalRefs(swagger)
	b, e := swagger.FixRefsAndMarshalJSON()
	require.NoError(t, e)

	g, e := ioutil.ReadFile("testdata/golden/testref.openapi.inline.json")
	require.NoError(t, e)
	require.Equal(t, b, g)
}
