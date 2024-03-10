package database

import (
	"demo/model"
	"demo/utils"
	"encoding/csv"
	"fmt"
	"github.com/google/uuid"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
	"log"
	"os"
	"time"
)

var DB *Dao

type Dao struct {
	db *gorm.DB
}

func New(username, password, endpoint string) *Dao {
	res := &Dao{}
	dsn := fmt.Sprintf("%s:%s@tcp(%s)/csye6225?charset=utf8mb4&parseTime=True&loc=Local", username, password, endpoint)

	db, err := gorm.Open(mysql.Open(dsn), &gorm.Config{})
	if err != nil {
		log.Panic("failed to connect database")
	}
	res.db = db
	res.initSchemas()
	res.initData()
	DB = res
	return res
}

func (d *Dao) initSchemas() {
	err := d.db.AutoMigrate(&model.Account{})
	if err != nil {
		log.Panic("failed to creat table [Account]")
	}
	err = d.db.AutoMigrate(&model.Assignment{})
	if err != nil {
		log.Panic("failed to creat table [Assignment]")
	}
	err = d.db.AutoMigrate(&model.Submission{})
	if err != nil {
		log.Panic("failed to creat table [Submission]")
	}

}

func (d *Dao) initData() {
	file, err := os.Open("./opt/users.csv")
	if err != nil {
		log.Println("Error opening file:", err)
		return
	}
	defer file.Close()
	reader := csv.NewReader(file)
	_, _ = reader.Read()
	for {
		record, err := reader.Read()
		if err != nil {
			break
		}
		pwd, err := utils.HashPassword(record[3])
		if err != nil {
			continue
		}
		d.db.Clauses(clause.OnConflict{
			Columns:   []clause.Column{{Name: "email"}},
			DoNothing: true,
		}).Create(&model.Account{
			ID:             uuid.NewString(),
			FirstName:      record[0],
			LastName:       record[1],
			Password:       pwd,
			Email:          record[2],
			AccountCreated: time.Now().String(),
			AccountUpdated: time.Now().String(),
		})
	}
}

func (d *Dao) Ping() error {
	var err error
	if db, err := d.db.DB(); err == nil {
		err = db.Ping()
		if err != nil {
			return err
		}
	}
	return err
}
