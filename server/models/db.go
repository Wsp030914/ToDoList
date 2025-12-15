package models

import (
	"gorm.io/gorm"
)

type DB struct {
	Db *gorm.DB
}

var d DB

func NewDB(db *gorm.DB) {
	d.Db = db
}
