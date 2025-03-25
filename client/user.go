package client

type User struct {
	Name  string `gorm:"column:name;type:varchar(255);not null"`
	Phone int64  `gorm:"column:phone;type:bigint;not null"`
}
