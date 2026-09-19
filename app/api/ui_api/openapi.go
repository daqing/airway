package ui_api

import "github.com/daqing/airway/lib/openapi"

type uiDemoItem struct {
	ID    int    `json:"id"`
	Name  string `json:"name"`
	Stock int    `json:"stock"`
}

type uiDemoItems struct {
	Items []uiDemoItem `json:"items"`
}

func init() {
	openapi.Get("/api/v1/ui-demo/items", func(o *openapi.Operation) {
		o.Summary("Demo items for the airway-ui showcase").Tag("ui").
			OK(openapi.Item[uiDemoItems]())
	})
}
