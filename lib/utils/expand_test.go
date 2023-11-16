package utils

import "testing"

func TestExpandPagePkgName(t *testing.T) {
	var tests = []struct {
		TopDir string
		Page   string

		Expected string
	}{
		{"core", "blog!", "blog"},
		{"ext", "blog!", "ext_blog"},

		{"core", "foo.bar", "foo_bar"},
		{"ext", "foo.bar", "ext_foo_bar"},

		{"core", "home", "home_page"},
		{"ext", "home", "ext_home_page"},
	}

	for _, test := range tests {
		actual := PagePkgName(test.TopDir, test.Page)

		if actual != test.Expected {
			t.Errorf("Expected %s, got %s", test.Expected, actual)
		}
	}
}

func TestExpandPageDirPath(t *testing.T) {
	var tests = []struct {
		TopDir string
		Page   string

		Expected string
	}{
		{"core", "blog!", "./core/pages/blog"},
		{"ext", "blog!", "./ext/pages/ext_blog"},

		{"core", "foo.bar", "./core/pages/foo/foo_bar"},
		{"ext", "foo.bar", "./ext/pages/foo/ext_foo_bar"},

		{"core", "home", "./core/pages/home_page"},
		{"ext", "home", "./ext/pages/ext_home_page"},
	}

	for _, test := range tests {
		actual := PageDirPath(test.TopDir, test.Page)

		if actual != test.Expected {
			t.Errorf("Expected %s, got %s", test.Expected, actual)
		}
	}
}

func TestNormalizePage(t *testing.T) {
	var tests = []struct {
		Page string

		Expected string
	}{
		{"blog!", "blog"},
		{"foo.bar", "foo_bar"},
		{"foo.bar!", "foo_bar"},
		{"foo!.bar", "foo_bar"},
		{"!foo", "foo"},
	}

	for _, test := range tests {
		actual := NormalizePage(test.Page)

		if actual != test.Expected {
			t.Errorf("Expected %s, got %s", test.Expected, actual)
		}
	}
}
