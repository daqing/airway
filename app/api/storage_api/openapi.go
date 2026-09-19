package storage_api

import "github.com/daqing/airway/lib/openapi"

// storageError is the raw error shape used by the storage endpoints
// ({"error": "..."}), which bypass the render envelope.
func storageError() *openapi.Schema {
	return openapi.Obj(map[string]*openapi.Schema{"error": openapi.Str()})
}

func init() {
	openapi.Post("/api/v1/storage", func(o *openapi.Operation) {
		o.Summary("Upload a file").Tag("storage").
			Description(`Stores the multipart "file" field under an optional "dir" prefix and returns the generated key.`)

		o.FormUpload(map[string]*openapi.Schema{
			"file": openapi.File(),
			"dir":  openapi.Str(),
		})

		o.Respond(200, "application/json", openapi.Obj(map[string]*openapi.Schema{
			"key":  openapi.Str(),
			"url":  openapi.Str(),
			"size": openapi.Int(),
		})).Description("Stored object key, URL and size")
		o.Respond(400, "application/json", storageError()).
			Description(`Missing multipart "file" field`)
		o.Respond(500, "application/json", storageError()).
			Description("Storage backend failure")
	})

	openapi.Get("/api/v1/storage/{key}", func(o *openapi.Operation) {
		o.Summary("Download a file").Tag("storage").
			Path("key", openapi.Str(), "File key; may contain slashes").
			Description("Streams the stored object with its detected content type.")

		o.Respond(200, "application/octet-stream", openapi.File()).
			Description("The file contents")
		o.Respond(404, "application/json", storageError()).
			Description("Unknown key")
		o.Respond(500, "application/json", storageError()).
			Description("Storage backend failure")
	})

	openapi.Delete("/api/v1/storage/{key}", func(o *openapi.Operation) {
		o.Summary("Delete a file").Tag("storage").
			Path("key", openapi.Str(), "File key; may contain slashes")

		o.Respond(200, "application/json", openapi.Obj(map[string]*openapi.Schema{
			"deleted": openapi.Str(),
		})).Description("The deleted key")
		o.Respond(500, "application/json", storageError()).
			Description("Storage backend failure")
	})
}
