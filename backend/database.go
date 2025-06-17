package main

import (
	"fmt"
	"log"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

var DB *gorm.DB

type UserSymbols struct {
	gorm.Model
	Symbol   string `gorm:"size:100; NOT NULL; UNIQUE;"`
	UserId   string `gorm:"size:200; NOT NULL;"`
	Type     string `gorm:"size:200; NOT NULL; VALUE IN (STOCK, CRYPTO)"`
	CryptoId string `gorm:"size:200; DEFAULT NULL;"` //only if crypto
}

func initDB() {
	connectionStr := "host=localhost user=abdullah password=edhi dbname=mydb1 port=5432"
	var err error
	DB, err = gorm.Open(postgres.Open(connectionStr), &gorm.Config{})
	if err != nil {
		log.Fatal("error connecting to database")
	}
	DB.AutoMigrate(&UserSymbols{})
	fmt.Println("Database connected successfully")
}
