package models

type Post struct {
	BaseModel

	UserId     IdType
	NodeId     IdType
	Title      string
	CustomPath string
	Place      string
	Content    string
	Fee        int
}

func (p Post) TableName() string { return "posts" }

const postPolyType = "post"

func (p *Post) PolyId() IdType   { return p.ID }
func (p *Post) PolyType() string { return postPolyType }
