package blog_page

import "os"

func BlogTitle() string   { return os.Getenv("AW_BLOG_TITLE") }
func BlogTagline() string { return os.Getenv("AW_BLOG_TAGLINE") }
