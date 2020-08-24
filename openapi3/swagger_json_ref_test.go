package openapi3_test

import (
	"io/ioutil"
	"net/http"
	"testing"

	"github.com/getkin/kin-openapi/openapi3"
	"github.com/stretchr/testify/require"
)

func TestRecursiveSchemaJSON(t *testing.T) {

	cs := startTestServer(http.Dir("testdata"))
	defer cs()

	loader := openapi3.NewSwaggerLoader(openapi3.WithAllowExternalRefs(true))
	swagger, err := loader.LoadSwaggerFromFile("testdata/testref.openapi.inline.yml")
	require.NoError(t, err)

	openapi3.ClearResolvedExternalRefs(swagger)
	b, e := swagger.FixRefsAndMarshalJSON()
	require.NoError(t, e)

	// confirm no error from loading generated schema
	sl2 := openapi3.NewSwaggerLoader()
	_, e = sl2.LoadSwaggerFromData(b)
	require.NoError(t, e)

	g, e := ioutil.ReadFile("testdata/golden/testref.openapi.inline.json")
	require.NoError(t, e)
	require.Equal(t, b, g)
}
