package websocket

import "github.com/daqing/airway/lib/openapi"

// The /ws upgrade itself has no HTTP contract to document; the publish
// endpoint is a plain form POST broadcasting to every connected client.
func init() {
	openapi.Post("/ws/publish", func(o *openapi.Operation) {
		o.Summary("Broadcast a message").Tag("ws").
			Description(`Accepts a "message" form field and broadcasts it to every client connected via /ws.`)

		o.Form(map[string]*openapi.Schema{
			"message": openapi.Str(),
		})

		o.Respond(200, "application/json", openapi.Obj(map[string]*openapi.Schema{
			"status": openapi.Str(),
		})).Description("Message queued for broadcast")
		o.Respond(400, "application/json", openapi.Obj(map[string]*openapi.Schema{
			"error": openapi.Str(),
		})).Description("Empty message")
	})
}
