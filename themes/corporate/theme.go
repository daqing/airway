// Package corporate is a ready-to-use Airway site theme for company
// showcase websites: a landing page with hero and feature grid, plain text
// pages, shared header navigation, and a footer. It implements
// github.com/daqing/airway/lib/ssg.Theme, so a site selects it with:
//
//	s.Use(corporate.Theme)
//
// Page data types:
//
//	corporate.Home     — landing page (hero heading, tagline, features)
//	corporate.Content  — text page (lead, paragraphs)
//	templ.Component    — custom body rendered inside the shared layout
//	nil                — header and footer only
package corporate

import (
	"io/fs"
	"strings"

	"github.com/a-h/templ"

	"github.com/daqing/airway/lib/ssg"
)

// Theme is the corporate showcase theme. Select it with Site.Use.
var Theme ssg.Theme = corporateTheme{}

type corporateTheme struct{}

func (corporateTheme) Name() string { return "corporate" }

// Render dispatches on the page data type; see the package documentation
// for the supported types.
func (corporateTheme) Render(ctx *ssg.Context) templ.Component {
	switch data := ctx.Page.Data.(type) {
	case Home:
		return layout(ctx, homeBody(data))
	case Content:
		return layout(ctx, contentBody(data))
	case templ.Component:
		return layout(ctx, data)
	default:
		return layout(ctx, nil)
	}
}

func (corporateTheme) Assets() fs.FS { return assetsFS }

// Home is the landing page data. Heading and Tagline fill the hero; the
// feature grid renders one card per Feature.
type Home struct {
	Heading  string
	Tagline  string
	Features []Feature
}

// Feature is one card in the landing page's feature grid.
type Feature struct {
	Title string
	Body  string
}

// Content is a plain text page (about, services, contact, ...).
type Content struct {
	Lead       string
	Paragraphs []string
}

// pageTitle composes the document title: the home page carries the site
// name alone, every other page appends it after the page title.
func pageTitle(ctx *ssg.Context) string {
	if ctx.Page.Slug == "/" {
		return ctx.Meta.Title
	}
	if ctx.Page.Title == "" {
		return ctx.Meta.Title
	}
	return strings.Join([]string{ctx.Page.Title, ctx.Meta.Title}, " · ")
}

// navLabel is the navigation text for a page: its title, or a word derived
// from the slug when the title is empty.
func navLabel(p ssg.Page) string {
	if p.Title != "" {
		return p.Title
	}
	if p.Slug == "/" {
		return "Home"
	}
	return strings.ReplaceAll(strings.Trim(p.Slug, "/"), "-", " ")
}
