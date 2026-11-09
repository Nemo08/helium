package xsd

import (
	"context"
	"strings"

	"github.com/lestrrat-go/helium"
	"github.com/lestrrat-go/helium/internal/lexicon"
)

// readAnnotation returns a snapshot of the xs:annotation child of a schema
// component, or nil when the component has none.
//
// It deliberately never reports diagnostics and never returns an error: the
// shape of xs:annotation is validated separately by checkAnnotations, and
// capturing metadata must not influence whether a schema compiles. A canceled
// context stops the walk and yields nil; the surrounding compile phase then
// surfaces ctx.Err().
func (c *compiler) readAnnotation(ctx context.Context, elem *helium.Element) *Annotation {
	if ctx.Err() != nil {
		return nil
	}
	annElem := firstXSDElementChild(elem, elemAnnotation)
	if annElem == nil {
		return nil
	}
	return c.readAnnotationElement(ctx, annElem)
}

// readAnnotationElement snapshots one xs:annotation element. It never fails and
// never emits diagnostics; a canceled context yields nil.
func (c *compiler) readAnnotationElement(ctx context.Context, annElem *helium.Element) *Annotation {
	if ctx.Err() != nil {
		return nil
	}
	ann := &Annotation{
		Line:   annElem.Line(),
		Source: c.diagSource(),
	}
	for child := range helium.Children(annElem) {
		if ctx.Err() != nil {
			return nil
		}
		ce, ok := child.(*helium.Element)
		if !ok || ce.URI() != lexicon.NamespaceXSD {
			continue
		}
		switch ce.LocalName() {
		case elemAppinfo:
			ann.AppInfos = append(ann.AppInfos, readAppInfo(ctx, ce))
		case elemDocumentation:
			ann.Documentation = append(ann.Documentation, readDocumentation(ctx, ce))
		}
	}
	ann.RawXML = serializeAnnotation(ctx, annElem)
	return ann
}

// serializeAnnotation serializes the whole xs:annotation element into a
// self-contained fragment: the writer re-declares every namespace prefix the
// subtree uses, so the result can be parsed without the surrounding document.
// A node that was parsed successfully is always serializable; if that ever
// fails, the typed fields remain available and RawXML is left empty rather than
// failing compilation over annotation metadata. The writer itself is not
// context-aware, so cancellation is only observed before serialization starts.
func serializeAnnotation(ctx context.Context, annElem *helium.Element) string {
	if ctx.Err() != nil {
		return ""
	}
	var buf strings.Builder
	if err := helium.NewWriter().WriteTo(&buf, annElem); err != nil {
		return ""
	}
	return buf.String()
}

// firstXSDElementChild returns the first direct child element in the XSD
// namespace with the given local name, or nil.
func firstXSDElementChild(elem *helium.Element, local string) *helium.Element {
	for child := range helium.Children(elem) {
		ce, ok := child.(*helium.Element)
		if ok && ce.URI() == lexicon.NamespaceXSD && ce.LocalName() == local {
			return ce
		}
	}
	return nil
}

// readAppInfo captures one xs:appinfo child: its @source, the in-scope
// namespaces, and the serialized child content. A canceled context yields the
// zero value.
func readAppInfo(ctx context.Context, elem *helium.Element) AppInfo {
	if ctx.Err() != nil {
		return AppInfo{}
	}
	ai := AppInfo{
		Source:     getAttr(elem, attrSource),
		Namespaces: collectNSContext(elem),
	}
	var buf strings.Builder
	w := helium.NewWriter().InheritedNamespaces(ai.Namespaces)
	for child := range helium.Children(elem) {
		if ctx.Err() != nil {
			return AppInfo{}
		}
		if err := w.WriteTo(&buf, child); err != nil {
			// A node that was parsed successfully is always serializable; if
			// that ever fails, expose the payload without inner XML rather
			// than failing compilation over annotation metadata.
			return AppInfo{Source: ai.Source, Namespaces: ai.Namespaces}
		}
	}
	ai.InnerXML = buf.String()
	return ai
}

// readDocumentation captures one xs:documentation child. A canceled context
// yields the zero value.
func readDocumentation(ctx context.Context, elem *helium.Element) Documentation {
	if ctx.Err() != nil {
		return Documentation{}
	}
	return Documentation{
		ID:     annotationXMLAttr(elem, lexicon.AttrID),
		Lang:   annotationXMLAttr(elem, lexicon.AttrLang),
		Source: getAttr(elem, attrSource),
		Text:   strings.TrimSpace(helium.CharacterData(elem)),
	}
}

// annotationXMLAttr returns the value of an XML-namespaced attribute such as
// xml:id or xml:lang. An unqualified attribute with the same local name is
// accepted as a fallback, matching how annotation consumers have historically
// read these attributes.
func annotationXMLAttr(elem *helium.Element, local string) string {
	if v := getAttr(elem, local); v != "" {
		return v
	}
	if attr := elem.GetAttributeNodeNS(local, lexicon.NamespaceXML); attr != nil {
		return attr.Value()
	}
	return ""
}
