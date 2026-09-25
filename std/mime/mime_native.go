package mime

import (
	"encoding/base64"
	"io"
	stdmime "mime"
	"mime/multipart"
	"mime/quotedprintable"
	"net/textproto"
	"path"
	"strings"

	. "github.com/candid82/joker/core"
)

type entityOptions struct {
	includeContent  bool
	maxDepth        int
	maxPartBytes    int64
	hasMaxPartBytes bool
}

func parseEntityOptions(opts Map) entityOptions {
	res := entityOptions{includeContent: true, maxDepth: 32}
	if opts == nil {
		return res
	}
	if ok, v := opts.Get(MakeKeyword("include-content?")); ok {
		res.includeContent = ToBool(v)
	}
	if ok, v := opts.Get(MakeKeyword("max-depth")); ok {
		res.maxDepth = EnsureObjectIsInt(v, "max-depth: %s").I
		if res.maxDepth < 0 {
			panic(RT.NewError("max-depth must be non-negative"))
		}
	}
	if ok, v := opts.Get(MakeKeyword("max-part-bytes")); ok {
		if _, isNil := v.(Nil); !isNil {
			max := EnsureObjectIsInt(v, "max-part-bytes: %s").I
			if max < 0 {
				panic(RT.NewError("max-part-bytes must be non-negative"))
			}
			res.maxPartBytes = int64(max)
			res.hasMaxPartBytes = true
		}
	}
	return res
}

func stringMap(m map[string]string) Map {
	res := EmptyArrayMap()
	for k, v := range m {
		res.Add(MakeString(k), MakeString(v))
	}
	return res
}

func headerMap(headers textproto.MIMEHeader) Map {
	res := EmptyArrayMap()
	for key, values := range headers {
		res.Add(MakeString(key), MakeStringVector(values))
	}
	return res
}

func objectString(obj Object) string {
	switch obj := obj.(type) {
	case String:
		return obj.S
	default:
		return obj.ToString(false)
	}
}

func firstHeader(headers Map, name string) string {
	if headers == nil {
		return ""
	}
	if ok, v := headers.Get(MakeString(textproto.CanonicalMIMEHeaderKey(name))); ok {
		return firstHeaderValue(v)
	}
	if ok, v := headers.Get(MakeString(name)); ok {
		return firstHeaderValue(v)
	}
	for iter := headers.Iter(); iter.HasNext(); {
		p := iter.Next()
		if strings.EqualFold(objectString(p.Key), name) {
			return firstHeaderValue(p.Value)
		}
	}
	return ""
}

func firstHeaderValue(obj Object) string {
	switch obj := obj.(type) {
	case String:
		return obj.S
	case Seqable:
		seq := obj.Seq()
		if seq.IsEmpty() {
			return ""
		}
		return objectString(seq.First())
	default:
		return objectString(obj)
	}
}

func normalizeEncoding(encoding string) string {
	return strings.ToLower(strings.TrimSpace(encoding))
}

func decodeTransferEncodingReader(encoding string, r io.Reader) io.Reader {
	switch normalizeEncoding(encoding) {
	case "", "7bit", "8bit", "binary":
		return r
	case "base64":
		return base64.NewDecoder(base64.StdEncoding, r)
	case "quoted-printable":
		return quotedprintable.NewReader(r)
	default:
		panic(RT.NewError("unsupported Content-Transfer-Encoding: " + encoding))
	}
}

func DecodeTransferEncoding(encoding string, content string) string {
	decoded, err := io.ReadAll(decodeTransferEncodingReader(encoding, strings.NewReader(content)))
	PanicOnErr(err)
	return string(decoded)
}

func decodeTransferEncoding(encoding Object, content string) string {
	var enc string
	if encoding != nil {
		switch encoding := encoding.(type) {
		case Nil:
			enc = ""
		case String:
			enc = encoding.S
		default:
			enc = encoding.ToString(false)
		}
	}
	return DecodeTransferEncoding(enc, content)
}

func ParseMediaType(s string) Map {
	mediaType, params, err := stdmime.ParseMediaType(s)
	PanicOnErr(err)
	res := EmptyArrayMap()
	res.Add(MakeKeyword("type"), MakeString(strings.ToLower(mediaType)))
	res.Add(MakeKeyword("params"), stringMap(params))
	return res
}

func parseMediaType(s string) Map {
	return ParseMediaType(s)
}

func parseMediaTypeParts(s string) (string, map[string]string) {
	mediaType, params, err := stdmime.ParseMediaType(s)
	PanicOnErr(err)
	return strings.ToLower(mediaType), params
}

func parseOptionalMediaType(s string, defaultType string) (string, map[string]string) {
	if strings.TrimSpace(s) == "" {
		return defaultType, map[string]string{}
	}
	return parseMediaTypeParts(s)
}

func addOptionalString(m *ArrayMap, key string, value string) {
	if value == "" {
		m.Add(MakeKeyword(key), NIL)
		return
	}
	m.Add(MakeKeyword(key), MakeString(value))
}

func pathVector(path []int) *Vector {
	res := EmptyVector()
	for _, n := range path {
		res = res.Conjoin(Int{I: n})
	}
	return res
}

func safeFilename(filename string) string {
	filename = strings.ReplaceAll(filename, "\x00", "")
	filename = strings.ReplaceAll(filename, "\\", "/")
	filename = path.Base(filename)
	if filename == "." || filename == "/" || filename == "" {
		return "attachment"
	}
	return filename
}

func readAllWithLimit(r io.Reader, opts entityOptions) []byte {
	if !opts.hasMaxPartBytes {
		b, err := io.ReadAll(r)
		PanicOnErr(err)
		return b
	}
	b, err := io.ReadAll(io.LimitReader(r, opts.maxPartBytes+1))
	PanicOnErr(err)
	if int64(len(b)) > opts.maxPartBytes {
		panic(RT.NewError("MIME part exceeds max-part-bytes"))
	}
	return b
}

func checkBytesLimit(n int, opts entityOptions) {
	if opts.hasMaxPartBytes && int64(n) > opts.maxPartBytes {
		panic(RT.NewError("MIME part exceeds max-part-bytes"))
	}
}

func ReadEntity(headers Map, body string, opts Map) Map {
	return readEntityAt(headers, body, parseEntityOptions(opts), nil, 0)
}

func readEntity(headers Map, body string, opts Map) Map {
	return ReadEntity(headers, body, opts)
}

func readEntityAt(headers Map, body string, opts entityOptions, partPath []int, depth int) Map {
	if depth > opts.maxDepth {
		panic(RT.NewError("MIME entity exceeds max-depth"))
	}
	checkBytesLimit(len(body), opts)

	contentType, contentTypeParams := parseOptionalMediaType(firstHeader(headers, "Content-Type"), "text/plain")
	disposition, dispositionParams := parseOptionalMediaType(firstHeader(headers, "Content-Disposition"), "")
	transferEncoding := normalizeEncoding(firstHeader(headers, "Content-Transfer-Encoding"))
	filename := dispositionParams["filename"]
	if filename == "" {
		filename = contentTypeParams["name"]
	}

	res := EmptyArrayMap()
	res.Add(MakeKeyword("headers"), headers)
	res.Add(MakeKeyword("content-type"), MakeString(contentType))
	res.Add(MakeKeyword("content-type-params"), stringMap(contentTypeParams))
	if disposition == "" {
		res.Add(MakeKeyword("disposition"), NIL)
	} else {
		res.Add(MakeKeyword("disposition"), MakeKeyword(disposition))
	}
	res.Add(MakeKeyword("disposition-params"), stringMap(dispositionParams))
	addOptionalString(res, "filename", filename)
	if filename == "" {
		res.Add(MakeKeyword("safe-filename"), NIL)
	} else {
		res.Add(MakeKeyword("safe-filename"), MakeString(safeFilename(filename)))
	}
	addOptionalString(res, "content-transfer-encoding", transferEncoding)
	res.Add(MakeKeyword("path"), pathVector(partPath))

	if strings.HasPrefix(contentType, "multipart/") {
		boundary := contentTypeParams["boundary"]
		if boundary == "" {
			panic(RT.NewError("multipart Content-Type missing boundary"))
		}
		mr := multipart.NewReader(strings.NewReader(body), boundary)
		parts := EmptyVector()
		for i := 0; ; i++ {
			part, err := mr.NextRawPart()
			if err == io.EOF {
				break
			}
			PanicOnErr(err)
			partBody := string(readAllWithLimit(part, opts))
			childPath := append(append([]int{}, partPath...), i)
			parts = parts.Conjoin(readEntityAt(headerMap(part.Header), partBody, opts, childPath, depth+1))
		}
		res.Add(MakeKeyword("parts"), parts)
		return res
	}

	decoded := DecodeTransferEncoding(transferEncoding, body)
	checkBytesLimit(len(decoded), opts)
	if opts.includeContent {
		res.Add(MakeKeyword("content"), MakeString(decoded))
	}
	res.Add(MakeKeyword("size"), Int{I: len(decoded)})
	return res
}

func mapValue(m Map, key string) Object {
	if m == nil {
		return NIL
	}
	if ok, v := m.Get(MakeKeyword(key)); ok {
		return v
	}
	return NIL
}

func mapStringValue(m Map, key string) string {
	v := mapValue(m, key)
	switch v := v.(type) {
	case String:
		return v.S
	default:
		return ""
	}
}

func mapKeywordName(m Map, key string) string {
	v := mapValue(m, key)
	switch v := v.(type) {
	case Keyword:
		return v.Name()
	default:
		return ""
	}
}

func mapParts(m Map) Vec {
	v := mapValue(m, "parts")
	if v == NIL {
		return nil
	}
	return EnsureObjectIsVec(v, "parts: %s")
}

func isAttachmentLike(entity Map, includeInline bool) bool {
	disposition := mapKeywordName(entity, "disposition")
	if disposition == "attachment" {
		return true
	}
	if includeInline && mapStringValue(entity, "filename") != "" {
		return true
	}
	return false
}

func Body(entity Map, _ Map) Map {
	var text Object = NIL
	var html Object = NIL
	var walk func(Map)
	walk = func(e Map) {
		if isAttachmentLike(e, true) {
			return
		}
		if parts := mapParts(e); parts != nil {
			for i := 0; i < parts.Count(); i++ {
				walk(EnsureObjectIsMap(parts.At(i), "part: %s"))
			}
			return
		}
		content := mapValue(e, "content")
		if content == NIL {
			return
		}
		switch mapStringValue(e, "content-type") {
		case "text/plain":
			if text == NIL {
				text = content
			}
		case "text/html":
			if html == NIL {
				html = content
			}
		}
	}
	walk(entity)
	res := EmptyArrayMap()
	res.Add(MakeKeyword("text"), text)
	res.Add(MakeKeyword("html"), html)
	return res
}

func body(entity Map, opts Map) Map {
	return Body(entity, opts)
}

func includeInline(opts Map) bool {
	if opts == nil {
		return true
	}
	if ok, v := opts.Get(MakeKeyword("include-inline?")); ok {
		return ToBool(v)
	}
	return true
}

func Attachments(entity Map, opts Map) Object {
	includeInline := includeInline(opts)
	res := EmptyVector()
	var walk func(Map)
	walk = func(e Map) {
		if parts := mapParts(e); parts != nil {
			for i := 0; i < parts.Count(); i++ {
				walk(EnsureObjectIsMap(parts.At(i), "part: %s"))
			}
			return
		}
		if isAttachmentLike(e, includeInline) {
			res = res.Conjoin(e)
		}
	}
	walk(entity)
	return res
}

func attachments(entity Map, opts Map) Object {
	return Attachments(entity, opts)
}
