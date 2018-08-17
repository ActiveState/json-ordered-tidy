// This code was copied from https://gitlab.com/c0b/go-ordered-json/blob/master/ordered.go and then heavily adjusted
//
// Copyright ??
package jsontidier

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"regexp"
	"sort"
	"strings"
)

type sortFunc func([]string, bool)

// the JSONTidier type, has similar operations as the default map, but maintained
// the keys order of inserted; similar to map, all single key operations (Get/Set/Delete) runs at O(1).
type JSONTidier struct {
	indent        string
	ordering      map[*regexp.Regexp]sortFunc
	sorting       []*regexp.Regexp
	path          []string
	ourMap        map[string]interface{}
	keyOrder      []string
	arrayReplacer *regexp.Regexp
	debug         bool
}

type NewParams struct {
	Indent    *string
	KeyOrder  map[string][]string
	ArraySort []string
	Debug     bool
}

// Create a new JSONTidier
func NewJSONTidier(np NewParams) *JSONTidier {
	o := make(map[*regexp.Regexp]sortFunc)
	for k, v := range np.KeyOrder {
		o[pathToRegexp(k, np.Debug)] = makeKeySorter(v)
	}

	jt := &JSONTidier{
		ordering:      o,
		sorting:       pathsToRegexps(np.ArraySort, np.Debug),
		path:          []string{},
		ourMap:        make(map[string]interface{}),
		keyOrder:      []string{},
		arrayReplacer: regexp.MustCompile(`(?:\[\d+\])*$`),
		debug:         np.Debug,
	}
	if np.Indent == nil {
		jt.indent = "    "
	} else {
		jt.indent = *np.Indent
	}
	jt.pushPath("$")

	return jt
}

// All of this regexp stuff is really gross. It'd be much better to have a
// real JSON Path implementation that could say if two paths match.
func pathsToRegexps(paths []string, debug bool) []*regexp.Regexp {
	var r []*regexp.Regexp
	for _, path := range paths {
		r = append(r, pathToRegexp(path, debug))
	}

	return r
}

func pathToRegexp(path string, debug bool) *regexp.Regexp {
	var parts []string
	pieces := strings.Split(path, ".")
	for i := 0; i < len(pieces); i++ {
		// If the path is something like "$..foo" then we want to turn this
		// into a regexp like "^\$.*?\['foo'\". If we naively split into
		// pieces on "." and then rejoin later then we end up with a regex
		// like "^\$\..*?\.\['foo'\]". Note the extra literal "." before
		// "\['foo'\]".
		if pieces[i] == "" {
			i++
			if i == len(pieces) {
				parts = append(parts, ".*?")
			} else {
				parts = append(parts, ".*?"+pieceRegexp(pieces[i]))
			}
		} else if pieces[i] == "*" {
			parts = append(parts, `\[[^\]]+\]`)
		} else {
			parts = append(parts, pieceRegexp(pieces[i]))
		}
	}
	re := regexp.MustCompile(strings.Join(parts, "") + `$`)

	if debug {
		log.Printf("Converted path %s to regexp: %s", path, re.String())
	}

	return re
}

var isArray *regexp.Regexp
var singleQuote *regexp.Regexp

func pieceRegexp(piece string) string {
	if isArray == nil {
		isArray = regexp.MustCompile(`\[\*\]$`)
	}
	if singleQuote == nil {
		singleQuote = regexp.MustCompile(`'`)
	}

	if piece == "$" {
		return `^\$`
	}

	var arrayRe = ""
	if isArray.MatchString(piece) {
		piece = isArray.ReplaceAllLiteralString(piece, "")
		arrayRe = `\[\d+\]`
	}

	piece = singleQuote.ReplaceAllLiteralString(piece, `\'`)

	return regexp.QuoteMeta(fmt.Sprintf("['%s']", piece)) + arrayRe
}

func makeKeySorter(order []string) sortFunc {
	weights := make(map[string]int)
	for i, v := range order {
		weights[v] = i
	}

	return func(arr []string, debug bool) {
		var msg string
		if debug {
			w := make([]string, len(weights))
			for k, v := range weights {
				w[v] = k
			}
			msg = fmt.Sprintf("Reordered\n    weights = %v\n    keys    = %v", w, arr)
		}

		sort.SliceStable(arr, func(i, j int) bool {
			aw, exists := weights[arr[i]]
			if !exists {
				aw = len(weights)
			}
			bw, exists := weights[arr[j]]
			if !exists {
				bw = len(weights)
			}

			if aw == bw {
				// These should only be equal when both strings were _not_ in
				// the list of keys passed for sorting. In that case we sort
				// the keys alphanumerically.
				return strings.ToLower(arr[i]) < strings.ToLower(arr[j])
			} else {
				// Otherwise we sort based on the weighting given to us.
				return aw < bw
			}
		})

		if debug {
			log.Printf("%s\n    new     = %v", msg, arr)
		}
	}
}

func (jt *JSONTidier) TidyString(orig string) (string, error) {
	tidied, err := jt.TidyBytes([]byte(orig))
	return string(tidied), err
}

func (jt *JSONTidier) TidyBytes(orig []byte) ([]byte, error) {
	err := json.Unmarshal(orig, jt)
	if err != nil {
		return []byte{}, err
	}

	tidied, err := json.MarshalIndent(jt, "", jt.indent)
	if err != nil {
		return []byte{}, err
	}

	tidied = append(tidied, '\n')

	return tidied, nil
}

// this implements type json.Unmarshaler interface, so can be called in json.Unmarshal(data, om)
func (jt *JSONTidier) UnmarshalJSON(data []byte) error {
	dec := json.NewDecoder(bytes.NewReader(data))
	dec.UseNumber()

	// must open with a delim token '{'
	t, err := dec.Token()
	if err != nil {
		return err
	}
	if delim, ok := t.(json.Delim); !ok || delim != '{' {
		return fmt.Errorf("expect JSON object open with '{'")
	}

	err = jt.parseObject(dec)
	if err != nil {
		return err
	}

	t, err = dec.Token()
	if err != io.EOF {
		return fmt.Errorf("expect end of JSON object but got more token: %T: %v or err: %v", t, t, err)
	}

	return nil
}

func (jt *JSONTidier) parseObject(dec *json.Decoder) (err error) {
	if jt.debug {
		log.Printf("Parse object at %s", jt.currentPath())
	}

	var t json.Token
	for dec.More() {
		t, err = dec.Token()
		if err != nil {
			return err
		}

		key, ok := t.(string)
		if !ok {
			return fmt.Errorf("expecting JSON key should be always a string: %T: %v", t, t)
		}

		jt.pushPath(fmt.Sprintf(`['%s']`, key))

		t, err = dec.Token()
		if err == io.EOF {
			break
		} else if err != nil {
			return err
		}

		var value interface{}
		value, err = jt.handleDelim(t, dec)
		if err != nil {
			return err
		}

		jt.keyOrder = append(jt.keyOrder, key)
		jt.ourMap[key] = value

		jt.popPath()
	}

	t, err = dec.Token()
	if err != nil {
		return err
	}
	if delim, ok := t.(json.Delim); !ok || delim != '}' {
		return fmt.Errorf("expect JSON object close with '}'")
	}

	jt.maybeReorder()

	return nil
}

func (jt *JSONTidier) pushPath(p string) {
	jt.path = append(jt.path, p)
}

func (jt *JSONTidier) popPath() {
	jt.path = jt.path[:len(jt.path)-1]
}

func (jt *JSONTidier) maybeReorder() {
	cur := jt.currentPath()
	for k, s := range jt.ordering {
		match := k.MatchString(cur)

		if jt.debug {
			log.Printf("Reorder keys? %s =~ %s = %v", cur, k.String(), match)
		}

		if match {
			s(jt.keyOrder, jt.debug)
			break
		}
	}
}

func (jt *JSONTidier) currentPath() string {
	return strings.Join(jt.path, "")
}

func (jt *JSONTidier) handleDelim(t json.Token, dec *json.Decoder) (res interface{}, err error) {
	if delim, ok := t.(json.Delim); ok {
		switch delim {
		case '{':
			jt2 := NewJSONTidier(NewParams{})
			jt2.debug = jt.debug
			jt2.ordering = jt.ordering
			jt2.sorting = jt.sorting
			jt2.path = make([]string, len(jt.path))
			for i, p := range jt.path {
				jt2.path[i] = p
			}
			err = jt2.parseObject(dec)
			if err != nil {
				return
			}
			return jt2, nil
		case '[':
			var value []interface{}
			value, err = jt.parseArray(dec)
			if err != nil {
				return
			}
			return value, nil
		default:
			return nil, fmt.Errorf("Unexpected delimiter: %q", delim)
		}
	}
	return t, nil
}

func (jt *JSONTidier) parseArray(dec *json.Decoder) (arr []interface{}, err error) {
	if jt.debug {
		log.Printf("Parse array at %s", jt.currentPath())
	}

	var t json.Token
	arr = make([]interface{}, 0)
	i := 0
	for dec.More() {
		t, err = dec.Token()
		if err != nil {
			return
		}

		jt.pushPath(fmt.Sprintf("[%d]", i))
		i++

		var value interface{}
		value, err = jt.handleDelim(t, dec)
		if err != nil {
			return
		}
		arr = append(arr, value)

		jt.popPath()
	}
	t, err = dec.Token()
	if err != nil {
		return
	}
	if delim, ok := t.(json.Delim); !ok || delim != ']' {
		err = fmt.Errorf("expect JSON array close with ']'")
		return
	}

	if jt.shouldSortArray() {
		jt.sortArray(arr)
	}

	return
}

func (jt *JSONTidier) shouldSortArray() bool {
	cur := jt.currentPath()
	for _, re := range jt.sorting {
		match := re.MatchString(cur)

		if jt.debug {
			log.Printf("Sort array?    %s =~ %s = %v", cur, re.String(), match)
		}

		if match {
			return true
		}
	}

	return false
}

func (jt *JSONTidier) sortArray(arr []interface{}) {
	if _, ok := arr[0].(json.Number); ok {
		sort.SliceStable(arr, func(i, j int) bool {
			a := arr[i].(json.Number)
			b := arr[j].(json.Number)
			return a < b
		})
	} else if _, ok := arr[0].(string); ok {
		sort.SliceStable(arr, func(i, j int) bool {
			a := arr[i].(string)
			b := arr[j].(string)
			return strings.ToLower(a) < strings.ToLower(b)
		})
	}

	return
}

// this implements type json.Marshaler interface, so can be called in json.Marshal(om)
func (jt *JSONTidier) MarshalJSON() ([]byte, error) {
	res := []byte{'{'}
	for i, k := range jt.keyOrder {
		res = append(res, fmt.Sprintf("%q:", k)...)

		b, err := json.Marshal(jt.ourMap[k])
		if err != nil {
			return nil, err
		}
		res = append(res, b...)
		if i != len(jt.keyOrder)-1 {
			res = append(res, ',')
		}
	}
	res = append(res, '}')

	return res, nil
}
