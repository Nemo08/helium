package xsd

import (
	"strings"

	"github.com/lestrrat-go/helium"
	"github.com/lestrrat-go/helium/internal/lexicon"
)

// readAnnotation returns a snapshot of the xs:annotation child of a schema
// component, or nil when the component has none.
//
// It deliberately never reports diagnostics and never returns an error: the
// shape of xs:annotation is validated separately by checkAnnotations, and
// capturing metadata must not influence whether a schema compiles.
func (c *compiler) readAnnotation(elem *helium.Element) *Annotation {
	annElem := firstXSDElementChild(elem, elemAnnotation)
	if annElem == nil {
		return nil
	}
	ann := &Annotation{
		Line:   annElem.Line(),
		Source: c.diagSource(),
	}
	for child := range helium.Children(annElem) {
		ce, ok := child.(*helium.Element)
		if !ok || ce.URI() != lexicon.NamespaceXSD {
			continue
		}
		switch ce.LocalName() {
		case elemAppinfo:
			ann.AppInfos = append(ann.AppInfos, readAppInfo(ce))
		case elemDocumentation:
			ann.Documentation = append(ann.Documentation, readDocumentation(ce))
		}
	}
	return ann
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
// namespaces, and the serialized child content.
func readAppInfo(elem *helium.Element) AppInfo {
	ai := AppInfo{
		Source:     getAttr(elem, attrSource),
		Namespaces: collectNSContext(elem),
	}
	var buf strings.Builder
	w := helium.NewWriter().InheritedNamespaces(ai.Namespaces)
	for child := range helium.Children(elem) {
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

// readDocumentation captures one xs:documentation child.
func readDocumentation(elem *helium.Element) Documentation {
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
