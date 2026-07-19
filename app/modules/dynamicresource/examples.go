package dynamicresource

// Example describes a reusable demonstration resource used by documentation,
// tests, and local bootstrap tooling.
type Example struct {
	Code      string
	Name      string
	TableName string
	Schema    Schema
}

// Examples returns two intentionally different resource shapes to exercise the
// generic engine without relying on resource-specific handlers.
func Examples() []Example {
	return []Example{
		{
			Code: "articles", Name: "文章", TableName: "articles",
			Schema: Schema{
				Fields: []Field{
					{Code: "title", Name: "标题", Type: "string", Required: true, List: true, Searchable: true, Sortable: true, Input: "text"},
					{Code: "body", Name: "正文", Type: "text", Required: true, Input: "textarea"},
					{Code: "views", Name: "浏览量", Type: "integer", List: true, Filterable: true, Sortable: true, Input: "number"},
					{Code: "published", Name: "已发布", Type: "boolean", Required: true, List: true, Filterable: true, Input: "checkbox"},
				},
				Permissions: examplePermissions("articles"),
			},
		},
		{
			Code: "products", Name: "商品", TableName: "products",
			Schema: Schema{
				Fields: []Field{
					{Code: "sku", Name: "SKU", Type: "string", Required: true, List: true, Searchable: true, Filterable: true, Input: "text"},
					{Code: "name", Name: "商品名称", Type: "string", Required: true, List: true, Searchable: true, Sortable: true, Input: "text"},
					{Code: "price_cents", Name: "价格（分）", Type: "bigint", Required: true, List: true, Sortable: true, Input: "number"},
					{Code: "stock", Name: "库存", Type: "integer", Required: true, List: true, Filterable: true, Sortable: true, Input: "number"},
					{Code: "metadata", Name: "扩展信息", Type: "json", Input: "json"},
				},
				Permissions: examplePermissions("products"),
			},
		},
	}
}

func examplePermissions(code string) ActionPermissions {
	return ActionPermissions{
		List: code + ":list", Read: code + ":read", Create: code + ":create",
		Update: code + ":update", Delete: code + ":delete",
	}
}
