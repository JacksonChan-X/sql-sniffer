package client

import (
	"database/sql"

	"gorm.io/gorm"
)

type UserRepo struct {
	db *gorm.DB
}

func NewUserRepo(db *gorm.DB) *UserRepo {
	return &UserRepo{db: db}
}

// 预编译语句
var (
	insertStmt *sql.Stmt
	queryStmt  *sql.Stmt
	deleteStmt *sql.Stmt
)

// 初始化预编译语句
func initPreparedStatements(db *sql.DB) error {
	// 插入语句
	var err error
	insertStmt, err = db.Prepare("INSERT INTO users (name, phone) VALUES (?, ?)")
	if err != nil {
		return err
	}

	// 查询语句
	queryStmt, err = db.Prepare("SELECT name, phone FROM users WHERE phone = ?")
	if err != nil {
		return err
	}

	// 删除语句
	deleteStmt, err = db.Prepare("DELETE FROM users WHERE phone = ?")
	if err != nil {
		return err
	}

	return nil
}

func (r *UserRepo) GetUser(name string, phone int64) (*User, error) {
	var user User
	if err := r.db.Where("name = ? AND phone = ?", name, phone).First(&user).Error; err != nil {
		return nil, err
	}
	return &user, nil
}

func (u *User) TableName() string {
	return "user"
}

func (r *UserRepo) CreateUser(user *User) error {
	tx := r.db.Exec("INSERT INTO user (name,phone) VALUES ('jackson',123124)")
	return tx.Error
	// if err := r.db.Create(user).Error; err != nil {
	// 	return err
	// }
	// return nil
}
