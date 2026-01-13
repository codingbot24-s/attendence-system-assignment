package main

import "gorm.io/gorm"




type User struct {
	gorm.Model
	Name string 
	Email string 
	Password string 
	Role  	string  
}

type Class struct {
	gorm.Model
	ClassName  string
	TeacherID  uint
	Teacher    User
	Students   []User `gorm:"many2many:class_students;"`
}
type Attendance struct {
	gorm.Model
	ClassID   uint
	StudentID uint
	Status    string 
}

