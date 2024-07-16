package models

type Setting struct {
	BaseModel

	Key string
	Val string
}

func (s Setting) TableName() string { return "settings" }
