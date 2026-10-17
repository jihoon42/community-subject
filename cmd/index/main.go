// Command index builds the static site of community subjects.
//
// It reads subjects/*/SUBJECT.md (frontmatter and body), the signed
// records under records/<evaluatee>/, verifies every signature with
// kit/record/verify.sh against the allowed_signers file in the repository
// root, and writes plain HTML to site/public/. No server, no JavaScript.
package main

import (
	"bytes"
	"embed"
	"flag"
	"fmt"
	"html/template"
	"log"
	"os"
	"path/filepath"
	"sort"

	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/extension"
	"github.com/yuin/goldmark/parser"
	"github.com/yuin/goldmark/renderer/html"
)

//go:embed templates/*.html
var templateFS embed.FS

//go:embed static/style.css
var styleCSS []byte

// Subject is one entry of subjects/.
type Subject struct {
	ID       string
	Title    string
	Subtitle string
	Version  string
	Status   string
	Front    Frontmatter
	Body     template.HTML
	Dir      string // relative path of the subject directory
	HasTests bool

	Completed    int // distinct evaluatees with a passing human record
	HumanCount   int
	MachineCount int
}

// Site is everything the templates see.
type Site struct {
	Subjects []*Subject
	Owners   []*Owner
}

// Page is the data passed to one template execution.
type Page struct {
	Root    string // relative prefix back to the site root, "" or "../../"
	Title   string
	Site    *Site
	Subject *Subject
	Owner   *Owner
}

func main() {
	root := flag.String("root", ".", "repository root")
	out := flag.String("out", "site/public", "output directory")
	flag.Parse()
	files, err := build(*root, *out)
	if err != nil {
		log.Fatal(err)
	}
	for _, f := range files {
		fmt.Println(f)
	}
}

// build generates the whole site and returns the files it wrote.
func build(root, out string) ([]string, error) {
	absRoot, err := filepath.Abs(root)
	if err != nil {
		return nil, err
	}
	site, err := loadSite(absRoot)
	if err != nil {
		return nil, err
	}
	if err := os.RemoveAll(out); err != nil {
		return nil, err
	}
	w := &writer{out: out}
	w.file("style.css", styleCSS)
	w.page("index.html", "home.html", Page{Root: "", Title: "커뮤니티 과제", Site: site})
	for _, s := range site.Subjects {
		w.page(filepath.Join("subjects", s.ID, "index.html"), "subject.html",
			Page{Root: "../../", Title: s.Title, Site: site, Subject: s})
	}
	for _, o := range site.Owners {
		w.page(filepath.Join("records", o.Name, "index.html"), "record.html",
			Page{Root: "../../", Title: o.Name + "의 기록", Site: site, Owner: o})
		for _, r := range o.Records {
			w.copy(filepath.Join("records", o.Name, r.File), r.Path)
			w.copy(filepath.Join("records", o.Name, r.File+".sig"), r.Path+".sig")
		}
	}
	return w.written, w.err
}

func loadSite(root string) (*Site, error) {
	site := &Site{}
	paths, err := filepath.Glob(filepath.Join(root, "subjects", "*", "SUBJECT.md"))
	if err != nil {
		return nil, err
	}
	sort.Strings(paths)
	for _, p := range paths {
		s, err := loadSubject(root, p)
		if err != nil {
			return nil, err
		}
		site.Subjects = append(site.Subjects, s)
	}
	v := Verifier{
		Script:         filepath.Join(root, "kit", "record", "verify.sh"),
		AllowedSigners: filepath.Join(root, "allowed_signers"),
	}
	site.Owners, err = loadOwners(filepath.Join(root, "records"), v)
	if err != nil {
		return nil, err
	}
	count(site)
	return site, nil
}

func loadSubject(root, path string) (*Subject, error) {
	src, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	fm, body, err := splitFrontmatter(src)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", path, err)
	}
	dir := filepath.Dir(path)
	s := &Subject{
		ID:       fm.Get("id"),
		Title:    fm.Get("title"),
		Subtitle: fm.Get("subtitle"),
		Version:  fm.Get("version"),
		Status:   fm.Get("status"),
		Front:    fm,
	}
	if s.ID == "" {
		s.ID = filepath.Base(dir)
	}
	if s.Title == "" {
		s.Title = s.ID
	}
	s.Dir, _ = filepath.Rel(root, dir)
	if st, err := os.Stat(filepath.Join(dir, "tests")); err == nil && st.IsDir() {
		s.HasTests = true
	}
	s.Body, err = renderMarkdown(body)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", path, err)
	}
	return s, nil
}

func renderMarkdown(src []byte) (template.HTML, error) {
	md := goldmark.New(
		goldmark.WithExtensions(extension.Table, extension.Strikethrough),
		goldmark.WithParserOptions(parser.WithAutoHeadingID()),
		goldmark.WithRendererOptions(html.WithUnsafe()),
	)
	var buf bytes.Buffer
	if err := md.Convert(src, &buf); err != nil {
		return "", err
	}
	return template.HTML(buf.String()), nil
}

// count fills the per-subject figures. Only numbers are shown on the
// subject page; who evaluated whom stays in the record pages.
func count(site *Site) {
	byID := map[string]*Subject{}
	for _, s := range site.Subjects {
		byID[s.ID] = s
	}
	passed := map[string]map[string]bool{}
	for _, o := range site.Owners {
		for _, r := range o.Records {
			s := byID[r.SubjectID]
			if s == nil {
				continue
			}
			switch r.Kind {
			case "human":
				s.HumanCount++
				if r.OK && (r.Flag == "Ok" || r.Flag == "Outstanding project") {
					if passed[s.ID] == nil {
						passed[s.ID] = map[string]bool{}
					}
					passed[s.ID][o.Name] = true
				}
			case "machine":
				s.MachineCount++
			}
		}
	}
	for id, set := range passed {
		byID[id].Completed = len(set)
	}
}

type writer struct {
	out     string
	written []string
	err     error
}

func (w *writer) file(rel string, data []byte) {
	if w.err != nil {
		return
	}
	dst := filepath.Join(w.out, rel)
	if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
		w.err = err
		return
	}
	if err := os.WriteFile(dst, data, 0o644); err != nil {
		w.err = err
		return
	}
	w.written = append(w.written, dst)
}

func (w *writer) copy(rel, src string) {
	if w.err != nil {
		return
	}
	data, err := os.ReadFile(src)
	if err != nil {
		if os.IsNotExist(err) {
			return
		}
		w.err = err
		return
	}
	w.file(rel, data)
}

func (w *writer) page(rel, tmpl string, p Page) {
	if w.err != nil {
		return
	}
	data, err := renderPage(tmpl, p)
	if err != nil {
		w.err = fmt.Errorf("%s: %w", rel, err)
		return
	}
	w.file(rel, data)
}

func renderPage(name string, p Page) ([]byte, error) {
	t, err := template.New("base.html").Funcs(template.FuncMap{
		"kindName": kindName,
	}).ParseFS(templateFS, "templates/base.html", "templates/"+name)
	if err != nil {
		return nil, err
	}
	var buf bytes.Buffer
	if err := t.ExecuteTemplate(&buf, "base.html", p); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

func kindName(kind string) string {
	switch kind {
	case "machine":
		return "기계"
	case "human":
		return "사람"
	}
	return kind
}
