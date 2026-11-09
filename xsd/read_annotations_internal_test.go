package xsd

import (
	"context"
	"testing"

	helium "github.com/lestrrat-go/helium"
	"github.com/stretchr/testify/require"
)

// TestAnnotationReadersHonorCancelledContext checks the annotation readers obey
// the XSD cancellation contract at the layer that has no error return: a
// canceled context stops the walk and yields no metadata instead of doing more
// work. Compile surfaces ctx.Err() at its next checkpoint.
func TestAnnotationReadersHonorCancelledContext(t *testing.T) {
	t.Parallel()

	const src = `<xs:annotation xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:appinfo source="urn:src"><meta xmlns="urn:m">v</meta></xs:appinfo>
  <xs:documentation xml:lang="en">docs</xs:documentation>
</xs:annotation>`
	doc, err := helium.NewParser().Parse(t.Context(), []byte(src))
	require.NoError(t, err)

	annElem := findDocumentElement(doc)
	require.NotNil(t, annElem)
	appinfo := firstXSDElementChild(annElem, elemAppinfo)
	require.NotNil(t, appinfo)
	documentation := firstXSDElementChild(annElem, elemDocumentation)
	require.NotNil(t, documentation)

	c := &compiler{}

	// Sanity check that the same inputs yield metadata while the context lives,
	// so the canceled assertions below cannot pass vacuously.
	live := c.readAnnotationElement(t.Context(), annElem)
	require.NotNil(t, live)
	require.Len(t, live.AppInfos, 1)
	require.Len(t, live.Documentation, 1)
	require.NotEmpty(t, live.RawXML)
	require.Equal(t, "urn:src", readAppInfo(t.Context(), appinfo).Source)

	ctx, cancel := context.WithCancel(t.Context())
	cancel()

	require.Nil(t, c.readAnnotationElement(ctx, annElem))
	require.Equal(t, AppInfo{}, readAppInfo(ctx, appinfo))
	require.Equal(t, Documentation{}, readDocumentation(ctx, documentation))
	require.Empty(t, serializeAnnotation(ctx, annElem))
}
