package forum_home

import "github.com/daqing/airway/lib/utils"

func ForumTitle() string {
	return utils.GetEnvMust("AW_FORUM_TITLE")
}

func ForumTagline() string {
	tagline, err := utils.GetEnv("AW_FORUM_TAGLINE")
	if err != nil {
		return ""
	}

	return tagline
}
