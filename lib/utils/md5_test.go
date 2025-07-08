package utils

import "testing"

func TestDirNameFromMD5(t *testing.T) {
	tests := []struct {
		md5Hash string
		extName string
		want    string
	}{
		{md5Hash: "1234567890", extName: ".jpg", want: "12/34/567890.jpg"},
		{md5Hash: "abcdabcdef", extName: ".png", want: "ab/cd/abcdef.png"},
	}

	for _, test := range tests {
		got := FilePathFromMD5(test.md5Hash, test.extName)
		if got != test.want {
			t.Errorf("FilePathFromMD5(%s, %s) = %s, want %s", test.md5Hash, test.extName, got, test.want)
		}
	}
}
