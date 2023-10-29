package media_plugin

import (
	"path/filepath"
	"time"

	"github.com/daqing/airway/lib/repo"
)

func SaveFile(
	userId int64,
	filename string,
	mime string,
	bytes int64,
) (*MediaFile, error) {
	return SaveFileExpiredAt(userId, filename, mime, bytes, repo.NeverExpires)
}

func SaveFileExpiredAt(
	userId int64,
	filename string,
	mime string,
	bytes int64,
	expiredAt time.Time,
) (*MediaFile, error) {

	return repo.Insert[MediaFile]([]repo.KVPair{
		repo.KV("user_id", userId),
		repo.KV("filename", filename),
		repo.KV("mime", mime),
		repo.KV("bytes", bytes),
		repo.KV("expired_at", expiredAt),
	})
}

// replace filename with part and origin extension
// replace("foo.pdf", "bar") -> "bar.pdf"
func replace(filename string, part string) string {
	return part + filepath.Ext(filename)
}

func hashDirPath(prefix string, path string) string {
	return assetFullPath(prefix, dirParts(path))
}

func dirParts(path string) string {
	if len(path) < 4 {
		return path
	}

	p1 := path[0:2]
	p2 := path[0:4]

	return "/" + p1 + "/" + p2
}

func assetFullPath(assetDir string, path string) string {
	return assetDir + path
}

func assetHostPath(prefix string, filename string) string {
	return prefix + dirParts(filename) + "/" + filename
}
