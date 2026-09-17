package user

import "gorm.io/gorm"

type User struct {
	gorm.Model
	Name     *string
	Mobile   string
	Password string
	Status   bool
}
