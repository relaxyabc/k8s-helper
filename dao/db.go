package dao

import (
	"fmt"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"k8s.io/klog/v2"
)

var (
	dbConn *gorm.DB
)

func InitDBByArgs(host, port, dbname, user, password string) {
	dbInstance, err := initDB(host, port, dbname, user, password)
	if err != nil {
		klog.Fatalf("数据库连接失败: %v", err)
	}
	dbConn = dbInstance
	fmt.Println("数据库连接成功")
}

func initDB(host, port, dbname, user, password string) (*gorm.DB, error) {
	dsn := fmt.Sprintf("host=%s user=%s password=%s dbname=%s port=%s sslmode=disable", host, user, password, dbname, port)
	return gorm.Open(postgres.Open(dsn), &gorm.Config{})
}

func GetDB() *gorm.DB {
	return dbConn
}
