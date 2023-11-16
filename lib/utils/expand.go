package utils

import (
	"fmt"
	"strings"
)

const DOT = "."
const UNDERSCORE = "_"
const CORE = "core"
const EXCLAMATION = "!"

// Get pkg name for page
//
// convert [core, blog!] to "blog"
// convert [ext, blog!] to "ext_blog"
//
// convert [core, foo.bar] to "foo_bar"
// convert [ext, foo.bar] to "ext_foo_bar"
//
// convert [core, home] to "home_page"
// convert [ext, home] to "ext_home_page"
func PagePkgName(topDir, page string) (pkgName string) {
	if strings.HasSuffix(page, EXCLAMATION) {
		pkgName = page[:len(page)-1]

		if topDir != CORE {
			pkgName = fmt.Sprintf("%s_%s", topDir, pkgName)
		}

		return
	}

	if strings.Contains(page, DOT) {
		pkgName = strings.Replace(page, DOT, UNDERSCORE, -1)

		if topDir != CORE {
			pkgName = fmt.Sprintf("%s_%s", topDir, pkgName)
		}

		return
	}

	pkgName = fmt.Sprintf("%s_page", page)
	if topDir != CORE {
		pkgName = fmt.Sprintf("%s_%s", topDir, pkgName)
	}

	return
}

// Get dir path for page
//
// convert [core, blog!] to "./core/pages/blog"
// convert [ext, blog!] to "./ext/pages/ext_blog"
//
// convert [core, foo.bar] to "./core/pages/foo/foo_bar"
// convert [ext, foo.bar] to "./ext/pages/foo/ext_foo_bar"
//
// convert [core, home] to "./core/pages/home_page"
// convert [ext, home] to "./ext/pages/ext_home_page"
func PageDirPath(topDir, page string) (dirPath string) {
	pkgName := PagePkgName(topDir, page)

	if strings.HasSuffix(page, EXCLAMATION) {
		dirPath = fmt.Sprintf("./%s/pages/%s", topDir, pkgName)
		return
	}

	if strings.Contains(page, DOT) {
		parts := strings.Split(page, DOT)

		dirPath = fmt.Sprintf("./%s/pages/%s/%s", topDir, parts[0], pkgName)
		return
	}

	dirPath = fmt.Sprintf("./%s/pages/%s", topDir, pkgName)
	return
}

func NormalizePage(page string) string {
	str := strings.Replace(page, EXCLAMATION, "", -1)

	return strings.Replace(str, DOT, UNDERSCORE, -1)
}
