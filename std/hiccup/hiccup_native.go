package hiccup

import (
	"html"
	"regexp"
	"sort"
	"strings"

	. "github.com/candid82/joker/core"
)

var (
	rawMarker = MakeKeyword("joker.hiccup/raw-string")
	modeKey   = MakeKeyword("mode")
	idKey     = MakeKeyword("id")
	classKey  = MakeKeyword("class")
	htmlMode  = MakeKeyword("html")
	xhtmlMode = MakeKeyword("xhtml")
	xmlMode   = MakeKeyword("xml")
	tagRE     = regexp.MustCompile(`([^\s\.#]+)(?:#([^\s\.#]+))?(?:\.([^\s#]+))?`)
	voidTags  = map[string]bool{
		"area": true, "base": true, "br": true, "col": true,
		"command": true, "embed": true, "hr": true, "img": true,
		"input": true, "keygen": true, "link": true, "meta": true,
		"param": true, "source": true, "track": true, "wbr": true,
	}
)

type renderer struct {
	output strings.Builder
	xml    bool
	html   bool
}

func rawString(content Object) Vec {
	return NewArrayVectorFrom(rawMarker, content)
}

func renderHTML(content []Object) string {
	mode := Object(xhtmlMode)
	if len(content) > 0 {
		if options, ok := attributeMap(content[0]); ok {
			if found, value := options.Get(modeKey); found && ToBool(value) {
				mode = value
			}
			content = content[1:]
		}
	}
	r := renderer{}
	// The original two-element mode sets compare keyword keys directly; other
	// objects cannot match (including unhashable transient values).
	if keyword, ok := mode.(Keyword); ok {
		r.xml = keyword.Equals(xmlMode) || keyword.Equals(xhtmlMode)
		r.html = keyword.Equals(htmlMode) || keyword.Equals(xhtmlMode)
	}
	// No execution is captured and the GIL is never released. Realizing lazy
	// sequences can synchronously call back into the caller's ambient VM.
	for _, value := range content {
		r.markup(value)
	}
	return r.output.String()
}

// Nil implements several collection interfaces in Go, but Joker's map?
// deliberately excludes it.
func attributeMap(value Object) (Map, bool) {
	if _, isNil := value.(Nil); isNil {
		return nil, false
	}
	m, ok := value.(Map)
	return m, ok
}

func named(value Object) (string, bool) {
	switch value := value.(type) {
	case String:
		return value.S, true
	case Keyword:
		return value.Name(), true
	case Symbol:
		return value.Name(), true
	default:
		return "", false
	}
}

// Match core str's printing rules rather than indiscriminately calling
// ToString(false): fallback collections contain quoted/escaped strings.
func stringValue(value Object) string {
	switch value := value.(type) {
	case Nil:
		return ""
	case String:
		return value.S
	case Char:
		return value.ToString(false)
	case *Regex:
		return value.ToString(false)
	default:
		return value.ToString(true)
	}
}

func asString(value Object) string {
	if name, ok := named(value); ok {
		return name
	}
	return stringValue(value)
}

func (r *renderer) markup(value Object) {
	switch value := value.(type) {
	case String:
		r.output.WriteString(html.EscapeString(value.S))
	case Nil:
		return
	case Vec:
		r.element(value)
	case Seq:
		for !value.IsEmpty() {
			r.markup(value.First())
			value = value.Rest()
		}
	case Keyword:
		r.output.WriteString(value.Name())
	case Symbol:
		r.output.WriteString(value.Name())
	default:
		r.output.WriteString(html.EscapeString(asString(value)))
	}
}

func invalidTag(tag Object) {
	// Use the standard exception constructor so ex-message/ex-data match the
	// interpreted renderer, including the original offending tag object.
	data := EmptyArrayMap()
	data.Add(MakeKeyword("tag"), tag)
	constructor := GLOBAL_ENV.CoreNamespace.Resolve("ex-info").Value.(Callable)
	panic(constructor.Call([]Object{
		MakeString(stringValue(tag) + " is not a valid element name"), data,
	}))
}

func parseTag(name string) (tag, id, class string) {
	if name != "" && !strings.ContainsAny(name, " \t\n\f\r.#") {
		return name, "", ""
	}
	match := tagRE.FindStringSubmatchIndex(name)
	// re-matches in joker.hiccup requires the entire match to equal the input.
	// A failed match produces a nil normalized tag, printed as an empty name.
	if match == nil || match[0] != 0 || match[1] != len(name) {
		return "", "", ""
	}
	tag = name[match[2]:match[3]]
	if match[4] >= 0 {
		id = name[match[4]:match[5]]
	}
	if match[6] >= 0 {
		class = strings.ReplaceAll(name[match[6]:match[7]], ".", " ")
	}
	return
}

// Shorthand merges use the same map construction/iteration order as Hiccup.
// This matters when printing attribute values realizes side-effecting sequences.
func mergeAttributes(id, class string, attrs Map) Map {
	var merged Map = attrs
	if id != "" {
		ids := EmptyArrayMap()
		ids.Add(idKey, MakeString(id))
		merged = ids.Merge(attrs)
	}
	var result Map = EmptyArrayMap()
	if class != "" {
		result = result.Assoc(classKey, MakeString(class)).(Map)
	}
	for iter := merged.Iter(); iter.HasNext(); {
		entry := iter.Next()
		value := entry.Value
		if found, previous := result.Get(entry.Key); found && ToBool(previous) {
			value = MakeString(stringValue(previous) + " " + stringValue(value))
		}
		result = result.Assoc(entry.Key, value).(Map)
	}
	return result
}

func (r *renderer) attribute(name, value Object) string {
	if !ToBool(value) {
		return ""
	}
	if boolean, ok := value.(Boolean); ok && boolean.B {
		if !r.xml {
			return " " + asString(name)
		}
		value = name
	}
	return " " + asString(name) + "=\"" + html.EscapeString(asString(value)) + "\""
}

func (r *renderer) attributes(attrs Map) {
	if attrs == nil || attrs.Count() == 0 {
		return
	}
	// Sort rendered strings, not keys; several keys may print the same name.
	attributes := make([]string, 0, attrs.Count())
	for iter := attrs.Iter(); iter.HasNext(); {
		entry := iter.Next()
		if ToBool(entry.Value) {
			attributes = append(attributes, r.attribute(entry.Key, entry.Value))
		}
	}
	sort.Strings(attributes)
	for _, attribute := range attributes {
		r.output.WriteString(attribute)
	}
}

func (r *renderer) element(element Vec) {
	count := element.Count()
	var originalTag Object = NIL
	if count > 0 {
		originalTag = element.At(0)
	}
	if originalTag.Equals(rawMarker) {
		// render-markup returns raw values unchanged; the surrounding join
		// prints them with ToString(false), even nil and non-string values.
		var raw Object = NIL
		if count > 1 {
			raw = element.At(1)
		}
		r.output.WriteString(raw.ToString(false))
		return
	}
	name, ok := named(originalTag)
	if !ok {
		invalidTag(originalTag)
	}
	tag, id, class := parseTag(name)
	start := 1
	var attrs Map
	if count > start {
		if explicit, ok := attributeMap(element.At(start)); ok {
			attrs = explicit
			start++
			if id != "" || class != "" {
				attrs = mergeAttributes(id, class, attrs)
			}
		}
	}
	if attrs == nil && (id != "" || class != "") {
		implicit := EmptyArrayMap()
		var idValue, classValue Object = NIL, NIL
		if id != "" {
			idValue = MakeString(id)
		}
		if class != "" {
			classValue = MakeString(class)
		}
		implicit.Add(idKey, idValue)
		implicit.Add(classKey, classValue)
		attrs = implicit
	}
	r.output.WriteByte('<')
	r.output.WriteString(tag)
	r.attributes(attrs)
	if start < count || (r.html && !voidTags[tag]) {
		r.output.WriteByte('>')
		for i := start; i < count; i++ {
			r.markup(element.At(i))
		}
		r.output.WriteString("</")
		r.output.WriteString(tag)
		r.output.WriteByte('>')
	} else if r.xml {
		r.output.WriteString(" />")
	} else {
		r.output.WriteByte('>')
	}
}
