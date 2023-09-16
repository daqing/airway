package models

type Hello struct {
	Id   int    `json:"id"`
	Name string `json:"name"`
	Age  int    `json:"age"`
}

func (h Hello) TableName() string {
	return "hello"
}
