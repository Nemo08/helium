package xsd_test

import (
	"io/fs"
	"testing"
	"testing/fstest"

	helium "github.com/lestrrat-go/helium"
	"github.com/lestrrat-go/helium/xsd"
	"github.com/stretchr/testify/require"
)

// compileAnnotations11 parses and compiles a single in-memory XSD 1.1 document
// and returns the compiled schema together with the non-fatal diagnostics
// rendered as text (warnings included). Tests assert on the errors string so a
// malformed fixture fails loudly instead of silently producing a partial model.
func compileAnnotations11(t *testing.T, schemaXML string) (*xsd.Schema, string) {
	t.Helper()
	doc := mustParseXSD(t, schemaXML)
	collector := helium.NewErrorCollector(t.Context(), helium.ErrorLevelNone)
	schema, err := xsd.NewCompiler().
		Version(xsd.Version11).
		Label("annotations.xsd").
		ErrorHandler(collector).
		Compile(t.Context(), doc)
	requireCompileResultErr(t, err)
	_ = collector.Close() // collector only buffers diagnostics; Close never fails
	_, errs := partitionCompileErrors(collector.Errors())
	return schema, errs
}

// compileAnnotationsFS11 compiles label from fsys as an XSD 1.1 schema.
func compileAnnotationsFS11(t *testing.T, fsys fs.FS, label string) (*xsd.Schema, string) {
	t.Helper()
	src, err := fs.ReadFile(fsys, label)
	require.NoError(t, err)
	doc := mustParseXSD(t, string(src))
	collector := helium.NewErrorCollector(t.Context(), helium.ErrorLevelNone)
	schema, err := xsd.NewCompiler().
		Version(xsd.Version11).
		Label(label).
		FS(fsys).
		ErrorHandler(collector).
		Compile(t.Context(), doc)
	requireCompileResultErr(t, err)
	_ = collector.Close() // collector only buffers diagnostics; Close never fails
	_, errs := partitionCompileErrors(collector.Errors())
	return schema, errs
}

// annText returns the text of the first xs:documentation in ann ("" when ann is
// nil or carries none).
func annText(ann *xsd.Annotation) string {
	if ann == nil || len(ann.Documentation) == 0 {
		return ""
	}
	return ann.Documentation[0].Text
}

// findAttrUse returns the attribute use with the given local name, or nil.
func findAttrUse(uses []*xsd.AttrUse, local string) *xsd.AttrUse {
	for _, au := range uses {
		if au.Name.Local == local {
			return au
		}
	}
	return nil
}

// findIDC returns the identity constraint with the given name, or nil.
func findIDC(idcs []*xsd.IDConstraint, name string) *xsd.IDConstraint {
	for _, idc := range idcs {
		if idc.Name == name {
			return idc
		}
	}
	return nil
}

// TestAnnotationComponentKinds covers the annotation snapshot of every component
// kind helium models: schema, notation, group, attributeGroup, attribute, simple
// type (with facets and every derivation wrapper), complex type (compositor,
// wildcards, assert, openContent) and element (alternatives, selector/field).
func TestAnnotationComponentKinds(t *testing.T) {
	t.Parallel()

	const schemaXML = `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema"
    xmlns:t="urn:t" targetNamespace="urn:t" elementFormDefault="qualified">
  <xs:annotation><xs:documentation>root-one</xs:documentation></xs:annotation>
  <xs:annotation><xs:documentation>root-two</xs:documentation></xs:annotation>

  <xs:notation name="jpeg" public="image/jpeg">
    <xs:annotation><xs:documentation>notation-doc</xs:documentation></xs:annotation>
  </xs:notation>

  <xs:attribute name="ga" type="xs:string">
    <xs:annotation><xs:documentation>global-attr</xs:documentation></xs:annotation>
  </xs:attribute>

  <xs:group name="grp">
    <xs:annotation><xs:documentation>group-def</xs:documentation></xs:annotation>
    <xs:sequence>
      <xs:annotation><xs:documentation>group-seq</xs:documentation></xs:annotation>
      <xs:element name="inner" type="xs:string"/>
    </xs:sequence>
  </xs:group>

  <xs:attributeGroup name="ag">
    <xs:annotation><xs:documentation>ag-def</xs:documentation></xs:annotation>
    <xs:attribute name="a" type="xs:string">
      <xs:annotation><xs:documentation>ag-attr</xs:documentation></xs:annotation>
    </xs:attribute>
    <xs:anyAttribute processContents="lax">
      <xs:annotation><xs:documentation>ag-anyattr</xs:documentation></xs:annotation>
    </xs:anyAttribute>
  </xs:attributeGroup>

  <xs:simpleType name="Sz">
    <xs:annotation><xs:documentation>sz-type</xs:documentation></xs:annotation>
    <xs:restriction base="xs:int">
      <xs:annotation><xs:documentation>sz-restriction</xs:documentation></xs:annotation>
      <xs:enumeration value="1">
        <xs:annotation><xs:documentation>enum-one</xs:documentation></xs:annotation>
      </xs:enumeration>
      <xs:enumeration value="2"/>
      <xs:minInclusive value="0">
        <xs:annotation><xs:documentation>min-doc</xs:documentation></xs:annotation>
      </xs:minInclusive>
    </xs:restriction>
  </xs:simpleType>

  <xs:simpleType name="Lst">
    <xs:list itemType="xs:int">
      <xs:annotation><xs:documentation>list-doc</xs:documentation></xs:annotation>
    </xs:list>
  </xs:simpleType>

  <xs:simpleType name="Un">
    <xs:union memberTypes="xs:int xs:string">
      <xs:annotation><xs:documentation>union-doc</xs:documentation></xs:annotation>
    </xs:union>
  </xs:simpleType>

  <xs:complexType name="Base">
    <xs:annotation><xs:documentation>base-type</xs:documentation></xs:annotation>
    <xs:sequence>
      <xs:annotation><xs:documentation>base-seq</xs:documentation></xs:annotation>
      <xs:element name="x" type="xs:string">
        <xs:annotation><xs:documentation>x-doc</xs:documentation></xs:annotation>
      </xs:element>
      <xs:any processContents="lax">
        <xs:annotation><xs:documentation>any-doc</xs:documentation></xs:annotation>
      </xs:any>
    </xs:sequence>
    <xs:anyAttribute processContents="lax">
      <xs:annotation><xs:documentation>base-anyattr</xs:documentation></xs:annotation>
    </xs:anyAttribute>
    <xs:assert test="true()">
      <xs:annotation><xs:documentation>assert-doc</xs:documentation></xs:annotation>
    </xs:assert>
  </xs:complexType>

  <xs:complexType name="SC">
    <xs:simpleContent>
      <xs:annotation><xs:documentation>sc-wrapper</xs:documentation></xs:annotation>
      <xs:extension base="xs:string">
        <xs:annotation><xs:documentation>sc-extension</xs:documentation></xs:annotation>
        <xs:attribute name="b" type="xs:string"/>
      </xs:extension>
    </xs:simpleContent>
  </xs:complexType>

  <xs:complexType name="CC">
    <xs:complexContent>
      <xs:annotation><xs:documentation>cc-wrapper</xs:documentation></xs:annotation>
      <xs:extension base="t:Base">
        <xs:annotation><xs:documentation>cc-extension</xs:documentation></xs:annotation>
        <xs:attribute name="c" type="xs:string"/>
      </xs:extension>
    </xs:complexContent>
  </xs:complexType>

  <xs:complexType name="Open">
    <xs:annotation><xs:documentation>open-type</xs:documentation></xs:annotation>
    <xs:openContent mode="interleave">
      <xs:annotation><xs:documentation>oc-doc</xs:documentation></xs:annotation>
      <xs:any processContents="lax"/>
    </xs:openContent>
    <xs:sequence>
      <xs:annotation><xs:documentation>open-seq</xs:documentation></xs:annotation>
      <xs:element name="y" type="xs:string"/>
    </xs:sequence>
  </xs:complexType>

  <xs:element name="root" type="t:Base">
    <xs:annotation><xs:documentation>root-elem</xs:documentation></xs:annotation>
    <xs:alternative test="true()" type="t:Base">
      <xs:annotation><xs:documentation>alt-doc</xs:documentation></xs:annotation>
    </xs:alternative>
    <xs:key name="k1">
      <xs:annotation><xs:documentation>key-doc</xs:documentation></xs:annotation>
      <xs:selector xpath="t:x">
        <xs:annotation><xs:documentation>sel-doc</xs:documentation></xs:annotation>
      </xs:selector>
      <xs:field xpath="@id">
        <xs:annotation><xs:documentation>field-doc</xs:documentation></xs:annotation>
      </xs:field>
    </xs:key>
    <xs:keyref name="kr1" refer="t:k1">
      <xs:annotation><xs:documentation>keyref-doc</xs:documentation></xs:annotation>
      <xs:selector xpath="t:x"/>
      <xs:field xpath="@ref"/>
    </xs:keyref>
  </xs:element>
</xs:schema>`

	schema, errs := compileAnnotations11(t, schemaXML)
	require.Empty(t, errs, "fixture must compile without diagnostics")

	t.Run("schema", func(t *testing.T) {
		t.Parallel()
		anns := schema.Annotations()
		require.Len(t, anns, 2, "xs:schema admits several annotations")
		require.Equal(t, "root-one", annText(anns[0]))
		require.Equal(t, "root-two", annText(anns[1]))
		for _, ann := range anns {
			require.Equal(t, "annotations.xsd", ann.Source)
			require.Greater(t, ann.Line, 0)
		}
	})

	t.Run("notation", func(t *testing.T) {
		t.Parallel()
		n, ok := schema.LookupNotation("jpeg", "urn:t")
		require.True(t, ok)
		require.Equal(t, "notation-doc", annText(n.Annotation))
	})

	t.Run("global attribute", func(t *testing.T) {
		t.Parallel()
		au, ok := schema.LookupAttribute("ga", "urn:t")
		require.True(t, ok)
		require.Equal(t, "global-attr", annText(au.Annotation))
		require.Equal(t, "annotations.xsd", au.Source)
		require.Greater(t, au.Line, 0)
	})

	t.Run("group definition and compositor", func(t *testing.T) {
		t.Parallel()
		require.Equal(t, "group-def", annText(schema.GroupAnnotation("grp", "urn:t")))
		g, ok := schema.LookupGroup("grp", "urn:t")
		require.True(t, ok)
		require.Equal(t, xsd.CompositorSequence, g.Compositor)
		require.Equal(t, "group-seq", annText(g.Annotation))
	})

	t.Run("attribute group definition", func(t *testing.T) {
		t.Parallel()
		require.Equal(t, "ag-def", annText(schema.AttributeGroupAnnotation("ag", "urn:t")))
		uses, ok := schema.LookupAttributeGroup("ag", "urn:t")
		require.True(t, ok)
		a := findAttrUse(uses, "a")
		require.NotNil(t, a)
		require.Equal(t, "ag-attr", annText(a.Annotation))
	})

	t.Run("simple type facets and derivation", func(t *testing.T) {
		t.Parallel()
		sz, ok := schema.LookupType("Sz", "urn:t")
		require.True(t, ok)
		require.Equal(t, "sz-type", annText(sz.Annotation))
		require.Len(t, sz.DerivationAnnotations, 1)
		require.Equal(t, "restriction", sz.DerivationAnnotations[0].Kind)
		require.Equal(t, "sz-restriction", annText(sz.DerivationAnnotations[0].Annotation))

		require.NotNil(t, sz.Facets)
		// EnumerationAnnotations is index-aligned with Enumeration, so the
		// unannotated enumeration keeps a nil slot.
		require.Len(t, sz.Facets.EnumerationAnnotations, len(sz.Facets.Enumeration))
		require.Equal(t, "enum-one", annText(sz.Facets.EnumerationAnnotations[0]))
		require.Nil(t, sz.Facets.EnumerationAnnotations[1])
		// FacetAnnotation is sparse: only facets that actually carry one.
		require.Len(t, sz.Facets.Annotations, 2)
		require.Equal(t, "enumeration", sz.Facets.Annotations[0].Kind)
		require.Equal(t, "1", sz.Facets.Annotations[0].Value)
		require.Equal(t, "minInclusive", sz.Facets.Annotations[1].Kind)
		require.Equal(t, "0", sz.Facets.Annotations[1].Value)
	})

	t.Run("list and union derivation", func(t *testing.T) {
		t.Parallel()
		lst, ok := schema.LookupType("Lst", "urn:t")
		require.True(t, ok)
		require.Len(t, lst.DerivationAnnotations, 1)
		require.Equal(t, "list", lst.DerivationAnnotations[0].Kind)
		require.Equal(t, "list-doc", annText(lst.DerivationAnnotations[0].Annotation))

		un, ok := schema.LookupType("Un", "urn:t")
		require.True(t, ok)
		require.Len(t, un.DerivationAnnotations, 1)
		require.Equal(t, "union", un.DerivationAnnotations[0].Kind)
		require.Equal(t, "union-doc", annText(un.DerivationAnnotations[0].Annotation))
	})

	t.Run("complex type content model and wildcards", func(t *testing.T) {
		t.Parallel()
		base, ok := schema.LookupType("Base", "urn:t")
		require.True(t, ok)
		require.Equal(t, "base-type", annText(base.Annotation))
		require.NotNil(t, base.ContentModel)
		require.Equal(t, "base-seq", annText(base.ContentModel.Annotation))
		require.Len(t, base.ContentModel.Particles, 2)

		xdecl, ok := base.ContentModel.Particles[0].Term.(*xsd.ElementDecl)
		require.True(t, ok)
		require.Equal(t, "x-doc", annText(xdecl.Annotation))

		anyW, ok := base.ContentModel.Particles[1].Term.(*xsd.Wildcard)
		require.True(t, ok)
		require.Equal(t, "any-doc", annText(anyW.Annotation))
		require.Equal(t, "annotations.xsd", anyW.Source)
		require.Greater(t, anyW.Line, 0)

		require.NotNil(t, base.AnyAttribute)
		require.Equal(t, "base-anyattr", annText(base.AnyAttribute.Annotation))

		require.Len(t, base.Assertions, 1)
		require.Equal(t, "assert-doc", annText(base.Assertions[0].Annotation))
	})

	t.Run("simpleContent and complexContent wrappers", func(t *testing.T) {
		t.Parallel()
		sc, ok := schema.LookupType("SC", "urn:t")
		require.True(t, ok)
		require.Len(t, sc.DerivationAnnotations, 2)
		require.Equal(t, "simpleContent", sc.DerivationAnnotations[0].Kind)
		require.Equal(t, "sc-wrapper", annText(sc.DerivationAnnotations[0].Annotation))
		require.Equal(t, "extension", sc.DerivationAnnotations[1].Kind)
		require.Equal(t, "sc-extension", annText(sc.DerivationAnnotations[1].Annotation))

		cc, ok := schema.LookupType("CC", "urn:t")
		require.True(t, ok)
		require.Len(t, cc.DerivationAnnotations, 2)
		require.Equal(t, "complexContent", cc.DerivationAnnotations[0].Kind)
		require.Equal(t, "cc-wrapper", annText(cc.DerivationAnnotations[0].Annotation))
		require.Equal(t, "extension", cc.DerivationAnnotations[1].Kind)
		require.Equal(t, "cc-extension", annText(cc.DerivationAnnotations[1].Annotation))
	})

	t.Run("open content", func(t *testing.T) {
		t.Parallel()
		open, ok := schema.LookupType("Open", "urn:t")
		require.True(t, ok)
		require.Equal(t, "open-type", annText(open.Annotation))
		require.NotNil(t, open.OpenContent)
		require.Equal(t, "oc-doc", annText(open.OpenContent.Annotation))
		require.NotNil(t, open.ContentModel)
		require.Equal(t, "open-seq", annText(open.ContentModel.Annotation))
	})

	t.Run("element, alternative and identity constraints", func(t *testing.T) {
		t.Parallel()
		root, ok := schema.LookupElement("root", "urn:t")
		require.True(t, ok)
		require.Equal(t, "root-elem", annText(root.Annotation))
		require.Equal(t, "annotations.xsd", root.Source)
		require.Greater(t, root.Line, 0)

		require.Len(t, root.Alternatives, 1)
		require.Equal(t, "alt-doc", annText(root.Alternatives[0].Annotation))

		k1 := findIDC(root.IDCs, "k1")
		require.NotNil(t, k1)
		require.Equal(t, "key-doc", annText(k1.Annotation))
		require.Equal(t, "sel-doc", annText(k1.SelectorAnnotation))
		require.Len(t, k1.FieldAnnotations, 1)
		require.Equal(t, "field-doc", annText(k1.FieldAnnotations[0]))

		kr1 := findIDC(root.IDCs, "kr1")
		require.NotNil(t, kr1)
		require.Equal(t, "keyref-doc", annText(kr1.Annotation))
		require.Nil(t, kr1.SelectorAnnotation, "keyref's own selector has no annotation")
	})
}

// TestAnnotationRefInheritance covers the ref rules: an annotation on the ref
// form wins, otherwise the global declaration's annotation is inherited. A group
// reference keeps its own annotation on the placeholder while the definition's
// annotation stays on Schema.GroupAnnotation. An identity-constraint @ref copies
// the selector/field annotations together with the expressions.
func TestAnnotationRefInheritance(t *testing.T) {
	t.Parallel()

	const schemaXML = `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema"
    xmlns:t="urn:t" targetNamespace="urn:t" elementFormDefault="qualified">
  <xs:element name="head" type="xs:string">
    <xs:annotation><xs:documentation>head-doc</xs:documentation></xs:annotation>
  </xs:element>
  <xs:element name="other" type="xs:string">
    <xs:annotation><xs:documentation>other-doc</xs:documentation></xs:annotation>
  </xs:element>
  <xs:attribute name="ga" type="xs:string">
    <xs:annotation><xs:documentation>ga-doc</xs:documentation></xs:annotation>
  </xs:attribute>
  <xs:group name="G">
    <xs:annotation><xs:documentation>g-def</xs:documentation></xs:annotation>
    <xs:sequence><xs:element name="g-in" type="xs:string"/></xs:sequence>
  </xs:group>

  <xs:complexType name="T">
    <xs:sequence>
      <xs:element ref="t:head"/>
      <xs:element ref="t:other">
        <xs:annotation><xs:documentation>other-use-doc</xs:documentation></xs:annotation>
      </xs:element>
      <xs:group ref="t:G">
        <xs:annotation><xs:documentation>g-use-doc</xs:documentation></xs:annotation>
      </xs:group>
    </xs:sequence>
    <xs:attribute ref="t:ga"/>
  </xs:complexType>

  <xs:element name="root" type="t:T">
    <xs:key name="k1">
      <xs:selector xpath="t:head"/>
      <xs:field xpath="@id"/>
    </xs:key>
    <xs:keyref name="k2" refer="t:k1">
      <xs:selector xpath="t:head">
        <xs:annotation><xs:documentation>k2-sel</xs:documentation></xs:annotation>
      </xs:selector>
      <xs:field xpath="@id">
        <xs:annotation><xs:documentation>k2-field</xs:documentation></xs:annotation>
      </xs:field>
    </xs:keyref>
    <xs:keyref ref="t:k2">
      <xs:annotation><xs:documentation>kr2-doc</xs:documentation></xs:annotation>
    </xs:keyref>
  </xs:element>
</xs:schema>`

	schema, errs := compileAnnotations11(t, schemaXML)
	require.Empty(t, errs, "fixture must compile without diagnostics")

	typ, ok := schema.LookupType("T", "urn:t")
	require.True(t, ok)
	require.NotNil(t, typ.ContentModel)
	require.Len(t, typ.ContentModel.Particles, 3)

	t.Run("element ref inherits when the ref has none", func(t *testing.T) {
		t.Parallel()
		decl, ok := typ.ContentModel.Particles[0].Term.(*xsd.ElementDecl)
		require.True(t, ok)
		require.True(t, decl.IsRef)
		require.Equal(t, "head-doc", annText(decl.Annotation))
	})

	t.Run("element ref keeps its own annotation", func(t *testing.T) {
		t.Parallel()
		decl, ok := typ.ContentModel.Particles[1].Term.(*xsd.ElementDecl)
		require.True(t, ok)
		require.True(t, decl.IsRef)
		require.Equal(t, "other-use-doc", annText(decl.Annotation))
	})

	t.Run("group ref and definition are separate sources", func(t *testing.T) {
		t.Parallel()
		g, ok := typ.ContentModel.Particles[2].Term.(*xsd.ModelGroup)
		require.True(t, ok)
		require.Equal(t, "g-use-doc", annText(g.Annotation))
		require.Equal(t, "g-def", annText(schema.GroupAnnotation("G", "urn:t")))
	})

	t.Run("attribute ref inherits", func(t *testing.T) {
		t.Parallel()
		ga := findAttrUse(typ.Attributes, "ga")
		require.NotNil(t, ga)
		require.True(t, ga.IsRef)
		require.Equal(t, "ga-doc", annText(ga.Annotation))
	})

	t.Run("identity constraint ref copies selector and field annotations", func(t *testing.T) {
		t.Parallel()
		root, ok := schema.LookupElement("root", "urn:t")
		require.True(t, ok)

		// The @ref form declares no name of its own (and adopts the target's
		// QName during resolution), so select it by IsConstraintRef.
		var kr2 *xsd.IDConstraint
		for _, idc := range root.IDCs {
			if idc.IsConstraintRef {
				kr2 = idc
				break
			}
		}
		require.NotNil(t, kr2)
		require.Equal(t, "kr2-doc", annText(kr2.Annotation))
		require.Equal(t, "k2-sel", annText(kr2.SelectorAnnotation))
		require.Len(t, kr2.FieldAnnotations, 1)
		require.Equal(t, "k2-field", annText(kr2.FieldAnnotations[0]))
	})
}

// TestAnnotationRawXML verifies the lossless escape hatch: @id, foreign
// namespaces and markup inside xs:documentation survive, while the typed fields
// keep exposing the text and the appinfo payload.
func TestAnnotationRawXML(t *testing.T) {
	t.Parallel()

	const schemaXML = `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema" xmlns:ui="urn:ui">
  <xs:element name="e">
    <xs:annotation id="ann1">
      <xs:appinfo source="hint-src">
        <ui:hint flag="1">hi</ui:hint>
      </xs:appinfo>
      <xs:documentation xml:lang="ru" source="doc-src">Текст <ui:b>жирный</ui:b></xs:documentation>
    </xs:annotation>
    <xs:complexType/>
  </xs:element>
</xs:schema>`

	schema, errs := compileAnnotations11(t, schemaXML)
	require.Empty(t, errs, "fixture must compile without diagnostics")

	e, ok := schema.LookupElement("e", "")
	require.True(t, ok)
	ann := e.Annotation
	require.NotNil(t, ann)

	require.Contains(t, ann.RawXML, "<xs:annotation")
	require.Contains(t, ann.RawXML, `id="ann1"`)
	// RawXML is self-contained: every element that uses the ui prefix carries
	// its own re-declaration instead of relying on an ancestor outside the
	// fragment.
	require.Contains(t, ann.RawXML, `<ui:b xmlns:ui="urn:ui">жирный</ui:b>`)
	// The fragment must be parseable on its own.
	_, err := helium.NewParser().Parse(t.Context(), []byte(ann.RawXML))
	require.NoError(t, err, "RawXML must be a self-contained, well-formed fragment")

	require.Len(t, ann.AppInfos, 1)
	require.Equal(t, "hint-src", ann.AppInfos[0].Source)
	require.Equal(t, "urn:ui", ann.AppInfos[0].Namespaces["ui"])
	require.Contains(t, ann.AppInfos[0].InnerXML, `<ui:hint flag="1">hi</ui:hint>`)

	require.Len(t, ann.Documentation, 1)
	require.Equal(t, "ru", ann.Documentation[0].Lang)
	require.Equal(t, "doc-src", ann.Documentation[0].Source)
	// CharacterData reads only direct text children, so the nested markup is
	// available through RawXML, not through Text.
	require.Equal(t, "Текст", ann.Documentation[0].Text)
}

// TestAnnotationsAcrossDocuments checks that include and import documents
// contribute both their schema-level annotations and the annotations on their
// components (with the declaring document's label in Source).
func TestAnnotationsAcrossDocuments(t *testing.T) {
	t.Parallel()

	const mainXSD = `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema"
    xmlns:t="urn:t" xmlns:i="urn:imp" targetNamespace="urn:t" elementFormDefault="qualified">
  <xs:annotation><xs:documentation>main-schema</xs:documentation></xs:annotation>
  <xs:include schemaLocation="inc.xsd"/>
  <xs:import namespace="urn:imp" schemaLocation="imp.xsd"/>
  <xs:element name="root" type="t:Included"/>
  <xs:element name="foreign" type="i:ImpType">
    <xs:annotation><xs:documentation>main-elem</xs:documentation></xs:annotation>
  </xs:element>
</xs:schema>`

	const incXSD = `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema"
    xmlns:t="urn:t" targetNamespace="urn:t" elementFormDefault="qualified">
  <xs:annotation><xs:documentation>inc-schema</xs:documentation></xs:annotation>
  <xs:complexType name="Included">
    <xs:annotation><xs:documentation>inc-type</xs:documentation></xs:annotation>
    <xs:sequence><xs:element name="v" type="xs:string"/></xs:sequence>
  </xs:complexType>
</xs:schema>`

	const impXSD = `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema"
    xmlns:i="urn:imp" targetNamespace="urn:imp">
  <xs:annotation><xs:documentation>imp-schema</xs:documentation></xs:annotation>
  <xs:complexType name="ImpType">
    <xs:annotation><xs:documentation>imp-type</xs:documentation></xs:annotation>
    <xs:sequence/>
  </xs:complexType>
</xs:schema>`

	fsys := fstest.MapFS{
		"main.xsd": &fstest.MapFile{Data: []byte(mainXSD)},
		"inc.xsd":  &fstest.MapFile{Data: []byte(incXSD)},
		"imp.xsd":  &fstest.MapFile{Data: []byte(impXSD)},
	}
	schema, errs := compileAnnotationsFS11(t, fsys, "main.xsd")
	require.Empty(t, errs, "fixture must compile without diagnostics")

	seen := make(map[string]string, 3)
	for _, ann := range schema.Annotations() {
		seen[annText(ann)] = ann.Source
	}
	require.Equal(t, map[string]string{
		"main-schema": "main.xsd",
		"inc-schema":  "inc.xsd",
		"imp-schema":  "imp.xsd",
	}, seen)

	inc, ok := schema.LookupType("Included", "urn:t")
	require.True(t, ok)
	require.Equal(t, "inc-type", annText(inc.Annotation))
	require.Equal(t, "inc.xsd", inc.Source)

	imp, ok := schema.LookupType("ImpType", "urn:imp")
	require.True(t, ok)
	require.Equal(t, "imp-type", annText(imp.Annotation))
	require.Equal(t, "imp.xsd", imp.Source)

	root, ok := schema.LookupElement("root", "urn:t")
	require.True(t, ok)
	require.Equal(t, "main.xsd", root.Source)

	foreign, ok := schema.LookupElement("foreign", "urn:t")
	require.True(t, ok)
	require.Equal(t, "main-elem", annText(foreign.Annotation))
}

// TestAnnotationsOverrideTarget checks that a document pulled in through
// xs:override still contributes its schema-level annotation, while the
// replacement component carries the override document's annotation.
func TestAnnotationsOverrideTarget(t *testing.T) {
	t.Parallel()

	const baseXSD = `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema"
    targetNamespace="urn:t" xmlns:t="urn:t">
  <xs:annotation><xs:documentation>base-schema</xs:documentation></xs:annotation>
  <xs:complexType name="T">
    <xs:annotation><xs:documentation>base-type</xs:documentation></xs:annotation>
    <xs:sequence/>
  </xs:complexType>
</xs:schema>`

	const mainXSD = `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema"
    targetNamespace="urn:t" xmlns:t="urn:t">
  <xs:annotation><xs:documentation>main-schema</xs:documentation></xs:annotation>
  <xs:override schemaLocation="base.xsd">
    <xs:complexType name="T">
      <xs:annotation><xs:documentation>override-type</xs:documentation></xs:annotation>
      <xs:sequence/>
    </xs:complexType>
  </xs:override>
  <xs:element name="root" type="t:T"/>
</xs:schema>`

	fsys := fstest.MapFS{
		"main.xsd": &fstest.MapFile{Data: []byte(mainXSD)},
		"base.xsd": &fstest.MapFile{Data: []byte(baseXSD)},
	}
	schema, errs := compileAnnotationsFS11(t, fsys, "main.xsd")
	require.Empty(t, errs, "fixture must compile without diagnostics")

	seen := make(map[string]string, 2)
	for _, ann := range schema.Annotations() {
		seen[annText(ann)] = ann.Source
	}
	require.Equal(t, map[string]string{
		"main-schema": "main.xsd",
		"base-schema": "base.xsd",
	}, seen)

	typ, ok := schema.LookupType("T", "urn:t")
	require.True(t, ok)
	require.Equal(t, "override-type", annText(typ.Annotation))
}

// TestAnnotationCapturedWithoutLabel makes sure capturing annotations does not
// depend on a diagnostic label: an unlabelled in-memory compile still records
// them, with an empty Source.
func TestAnnotationCapturedWithoutLabel(t *testing.T) {
	t.Parallel()

	doc := mustParseXSD(t, `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:element name="e" type="xs:string">
    <xs:annotation><xs:documentation>doc</xs:documentation></xs:annotation>
  </xs:element>
</xs:schema>`)
	schema, err := xsd.NewCompiler().Compile(t.Context(), doc)
	requireCompileResultErr(t, err)
	require.NoError(t, err)

	e, ok := schema.LookupElement("e", "")
	require.True(t, ok)
	require.Equal(t, "doc", annText(e.Annotation))
	// Without a Label the compiler's default display label is "(string)", the
	// same one diagnostics use; the annotation is still captured.
	require.Equal(t, "(string)", e.Annotation.Source)
	require.Greater(t, e.Annotation.Line, 0)
}
