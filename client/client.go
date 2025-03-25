package client

import (
	"time"

	mysqldriver "github.com/go-sql-driver/mysql"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func ConnnectMySQL(addr, user, password, dbname string, debug bool) (*gorm.DB, error) {
	cfg := mysqldriver.NewConfig()
	cfg.Addr = addr
	cfg.User = user
	cfg.Passwd = password
	cfg.DBName = ""
	cfg.ParseTime = true
	cfg.Loc = time.Local
	cfg.Net = "tcp"
	cfg.MultiStatements = true
	orm, err := gorm.Open(mysql.Open(cfg.FormatDSN()))
	if err != nil {
		return nil, err
	}
	db, err := orm.DB()
	if err != nil {
		return nil, err
	}
	_, err = db.Exec("CREATE DATABASE IF NOT EXISTS " + dbname + " DEFAULT CHARACTER SET utf8mb4 COLLATE utf8mb4_general_ci")
	if err != nil {
		return nil, err
	}
	cfg.DBName = dbname
	opt := &gorm.Config{}
	if !debug {
		opt.Logger = logger.Discard
	}
	orm, err = gorm.Open(mysql.Open(cfg.FormatDSN()), opt)
	if err != nil {
		return nil, err
	}
	db, err = orm.DB()
	if err != nil {
		return nil, err
	}
	db.SetMaxOpenConns(50)
	return orm, err
}
