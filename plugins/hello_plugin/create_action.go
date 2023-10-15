package hello_plugin

import (
	"github.com/daqing/airway/lib/repo"
	"github.com/daqing/airway/lib/resp"
	"github.com/daqing/airway/lib/utils"
	"github.com/gin-gonic/gin"
)

type CreateParams struct {
	Name string `json:"name"`
	Age  int    `json:"age"`
}

func CreateAction(c *gin.Context) {
	var p CreateParams

	if err := c.BindJSON(&p); err != nil {
		utils.LogError(c, err)
		return
	}

	hello, err := repo.Insert[Hello]([]repo.KeyValueField{
		repo.NewKV("name", p.Name),
		repo.NewKV("age", p.Age),
	})

	if err != nil {
		utils.LogError(c, err)
		return
	}

	resp.OK(c, gin.H{"hello": hello})
}
