package services

import (
	"log"

	"github.com/daqing/airway/app/models"
	"github.com/daqing/airway/lib/repo"
	"github.com/daqing/airway/lib/utils"
)

func GetHello(name string) *models.Hello {
	hellos, err := repo.Find[models.Hello](
		[]string{"id", "name", "age"},
		[]repo.KeyValueField{
			repo.NewCond("name", utils.TrimFull(name)),
		},
	)

	if err != nil {
		log.Println("Get Hello error", err)
		return nil
	}

	if len(hellos) == 0 {
		// hello not found
		return &models.Hello{}
	}

	return hellos[0]
}
