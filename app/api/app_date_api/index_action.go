package app_date_api

import (
	"github.com/daqing/airway/app/services"
	"github.com/daqing/airway/lib/api_resp"
	"github.com/gin-gonic/gin"
)

func IndexAction(c *gin.Context) {
	api_resp.OK(c, gin.H{"date": services.Now()})
}
