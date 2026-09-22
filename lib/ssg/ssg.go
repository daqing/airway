// Package ssg turns Airway into a static showcase generator: a Site
// registers pages against a Theme, and Build exports them as a plain HTML
// directory deployable to any static host. Themes are ordinary Go modules
// supplying templ components and embedded assets (see
// github.com/daqing/airway/themes/corporate), so a showcase site needs no
// database, no Node toolchain, and no running server.
package ssg

import (
	"fmt"
	"io/fs"
	"strings"

	"github.com/a-h/templ"
)

// Meta describes the site-wide information every page shares.
type Meta struct {
	// Title is the site name (e.g. "Acme Inc"). Themes use it for the brand
	// mark, footers, and page title suffixes.
	Title string
	// Description is the fallback description for search engines and
	// social cards on pages that do not carry one of their own.
	Description string
	// Language is the <html lang> value; empty means "en".
	Language string
}

// Lang returns the HTML document language code.
func (m Meta) Lang() string {
	if m.Language == "" {
		return "en"
	}
	return m.Language
}

// Page is one output page of the site. Data carries theme-specific page
// content; each theme documents the data types it understands.
type Page struct {
	// Slug is the URL path of the page, normalized to a clean absolute
	// path ("/", "/about"). It decides the export location:
	// <out>/<slug>/index.html.
	Slug string
	// Title labels the page in navigation and the document title.
	Title string
	// Data is the page content, interpreted by the selected theme.
	Data any
}

// Context is what a Theme receives to render one page.
type Context struct {
	Meta Meta
	// Page is the page being rendered.
	Page Page
	// Pages lists every registered page in registration order, for
	// building navigation.
	Pages []Page
}

// Theme supplies the layout, styling, and static assets for a site.
type Theme interface {
	// Name is the theme identifier (e.g. "corporate").
	Name() string
	// Render builds the full HTML document for ctx.Page.
	Render(ctx *Context) templ.Component
	// Assets returns the static files (CSS, fonts, images) that Build
	// copies into the site's assets/ directory and Handler serves under
	// /assets/. Return nil when the theme ships no assets.
	Assets() fs.FS
}

// Builder constructs a Site definition. Host projects register one through
// cmd.SetSSGBuilder from an init function in a root-package file (ssg.go),
// which is how `airway ssg:build` and `ssg:serve` find the pages to work
// on.
type Builder func() (*Site, error)

// Site is a set of registered pages plus the theme that renders them.
type Site struct {
	meta  Meta
	theme Theme
	pages []Page
}

// New starts a site definition with the given site-wide metadata.
func New(meta Meta) *Site {
	return &Site{meta: meta}
}

// Use selects the theme that renders every page. Calling it again replaces
// the previous theme.
func (s *Site) Use(t Theme) *Site {
	s.theme = t
	return s
}

// Page registers a page. Registering the same normalized slug twice panics,
// as does a slug that does not resolve to an absolute path — both are
// authoring errors that should fail fast, like duplicate plugin names.
func (s *Site) Page(slug string, title string, data any) *Site {
	p := Page{Slug: NormalizeSlug(slug), Title: title, Data: data}

	if !strings.HasPrefix(p.Slug, "/") {
		panic(fmt.Sprintf("site: page slug %q does not resolve to an absolute path", slug))
	}

	for _, existing := range s.pages {
		if existing.Slug == p.Slug {
			panic(fmt.Sprintf("site: duplicate page slug %q", p.Slug))
		}
	}

	s.pages = append(s.pages, p)
	return s
}

// NormalizeSlug canonicalizes a page slug to a clean absolute path: a leading
// slash is added and trailing slashes are dropped, so "about" and "/about/"
// both become "/about", while "" and "/" both stay "/".
func NormalizeSlug(slug string) string {
	slug = strings.TrimSpace(slug)
	if slug == "" {
		return "/"
	}
	if !strings.HasPrefix(slug, "/") {
		slug = "/" + slug
	}
	if trimmed := strings.TrimRight(slug, "/"); trimmed != "" {
		slug = trimmed
	}
	return slug
}

// Meta returns the site metadata.
func (s *Site) Meta() Meta { return s.meta }

// Theme returns the selected theme, or nil when Use has not been called.
func (s *Site) Theme() Theme { return s.theme }

// Pages returns the registered pages in registration order.
func (s *Site) Pages() []Page {
	pages := make([]Page, len(s.pages))
	copy(pages, s.pages)
	return pages
}

// DirName maps the page slug to its output directory under the export root:
// "/" becomes the root itself, "/about" becomes "about".
func (p Page) DirName() string {
	return strings.Trim(p.Slug, "/")
}
