package main

import (
	"NewStudent/router"
)

func main() {
	//if err := dao.Db.AutoMigrate(&models.User{}); err != nil {
	//	log.Fatalf("auto migrate user failed: %v", err)
	//}
	r := router.Router()

	r.Run()
}
