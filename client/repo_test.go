package client

import (
	"testing"
	"time"

	"gorm.io/gorm"
)

var (
	addr     = "127.0.0.1:3307"
	user     = "root"
	password = "cjh123"
	dbname   = "crocodile"
)

func TestGetUser(t *testing.T) {
	db, err := ConnnectMySQL(addr, user, password, dbname, true)
	if err != nil {
		t.Error(err)
	}

	if err = db.Transaction(func(tx *gorm.DB) error {
		for i := 0; i <= 10; i++ {
			tx.Exec("UPDATE user SET name = ? WHERE phone = ?", "jackson", 123456)
		}
		time.Sleep(time.Second * 3)
		return nil
	}); err != nil {
		t.Error(err)
	}

	db.Commit()

	// db, err := ConnnectMySQL("14.17.80.144:3307", "root", "f3r9Gy8Jyu7U", "downtime", true)
	// if err != nil {
	// 	t.Fatal(err)
	// }
	// tx := db.Raw("SELECT * FROM `task`")
	// if tx.Error != nil {
	// 	t.Fatal(tx.Error)
	// }
	// t.Log(tx.Rows())
	// // t.Log(u)

	// _, err = repo.GetUser("jackson", 111)
	// if err != nil {
	// 	t.Error(err)
	// }
	// t.Log(u)
	// 	}()
	// }
	// time.Sleep(time.Second * 3)
}

func TestCreateUser(t *testing.T) {

	db, err := ConnnectMySQL(addr, user, password, dbname, true)
	if err != nil {
		t.Error(err)
	}

	u := &User{
		Name:  "jackson",
		Phone: 123456,
	}
	err = NewUserRepo(db).CreateUser(u)
	if err != nil {
		t.Error(err)
	}
}

func TestExplainSQL(t *testing.T) {
	sql := "SELECT * FROM `user` WHERE name = ? AND phone = ? ORDER BY `user`.`name` LIMIT ?"
	t.Log(ExplainSQL(sql, nil, `'`, "jackson", 189593868))
}
