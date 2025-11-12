package main

import "NewStudent/router"

func main() {
	//if err := dao.Db.AutoMigrate(&models.Project{}); err != nil {
	//	log.Fatalf("auto migrate Project failed: %v", err)
	//}
	r := router.Router()

	r.Run()

}
