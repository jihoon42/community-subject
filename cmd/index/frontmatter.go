package main

import (
	"bufio"
	"bytes"
	"errors"
	"strings"
)

// Field is one frontmatter entry. List is set when the value was written
// as [a, b, c]; Value then holds the items joined with ", ".
type Field struct {
	Key   string
	Value string
	List  []string
}

// Frontmatter keeps fields in file order so the page table reads like the
// source file.
type Frontmatter struct {
	Fields []Field
}

// Get returns the value of key, or "" when it is absent.
func (f Frontmatter) Get(key string) string {
	for _, fl := range f.Fields {
		if fl.Key == key {
			return fl.Value
		}
	}
	return ""
}

// splitFrontmatter separates a "---" delimited YAML-like header from the
// Markdown body. Only the subset the spec uses is understood: "key: value"
// lines, optional double quotes around scalars, and [a, b] lists.
func splitFrontmatter(src []byte) (Frontmatter, []byte, error) {
	var fm Frontmatter
	sc := bufio.NewScanner(bytes.NewReader(src))
	sc.Buffer(make([]byte, 0, 64*1024), 1024*1024)
	if !sc.Scan() || strings.TrimRight(sc.Text(), "\r") != "---" {
		return fm, nil, errors.New("frontmatter: file does not start with ---")
	}
	closed := false
	for sc.Scan() {
		line := strings.TrimRight(sc.Text(), "\r")
		if line == "---" {
			closed = true
			break
		}
		t := strings.TrimSpace(line)
		if t == "" || strings.HasPrefix(t, "#") {
			continue
		}
		k, v, ok := strings.Cut(line, ":")
		if !ok {
			return fm, nil, errors.New("frontmatter: line without a colon: " + line)
		}
		fm.Fields = append(fm.Fields, parseField(strings.TrimSpace(k), strings.TrimSpace(v)))
	}
	if !closed {
		return fm, nil, errors.New("frontmatter: closing --- not found")
	}
	// The remainder of the file is the body.
	var body bytes.Buffer
	for sc.Scan() {
		body.Write(sc.Bytes())
		body.WriteByte('\n')
	}
	if err := sc.Err(); err != nil {
		return fm, nil, err
	}
	return fm, body.Bytes(), nil
}

func parseField(key, raw string) Field {
	f := Field{Key: key}
	if strings.HasPrefix(raw, "[") && strings.HasSuffix(raw, "]") {
		inner := strings.TrimSpace(raw[1 : len(raw)-1])
		if inner != "" {
			for _, item := range strings.Split(inner, ",") {
				f.List = append(f.List, unquote(strings.TrimSpace(item)))
			}
		}
		f.Value = strings.Join(f.List, ", ")
		return f
	}
	f.Value = unquote(raw)
	return f
}

func unquote(s string) string {
	if len(s) >= 2 {
		if (s[0] == '"' && s[len(s)-1] == '"') || (s[0] == '\'' && s[len(s)-1] == '\'') {
			return s[1 : len(s)-1]
		}
	}
	return s
}
