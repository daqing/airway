package blog_page

import "github.com/daqing/airway/lib/utils"

func BlogTitle() string {
	return utils.GetEnvMust("AW_BLOG_TITLE")
}

func BlogTagline() string {
	tagline, err := utils.GetEnv("AW_BLOG_TAGLINE")
	if err != nil {
		return ""
	}

	return tagline
}
