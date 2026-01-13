package main

import (
	"fmt"
	"log"

	"github.com/gofiber/fiber/v3"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

type handler struct {
	DB *gorm.DB
}

func NewDB(db *gorm.DB) handler {
	return handler{db}
}

func ConnectToDB() *gorm.DB {

	dbURL := "postgres://postgres:mysecretpassword@localhost:5432/postgres"

	db, err := gorm.Open(postgres.Open(dbURL), &gorm.Config{})
	if err != nil {
		log.Fatal("error connecting db")
	}

	if err := db.AutoMigrate(&User{}, &Class{}, &Attendance{}); err != nil {
		log.Fatal("error migrating db")
	}

	fmt.Println("db connected successfully ")
	return db
}

type SignupUserReq struct {
	Username string `json:"username"`
	Email    string `json:"email"`
	Password string `json:"password"`
	Role     string `json:"role"`
}

func (h *handler) CreateUser(c fiber.Ctx) error {
	var req SignupUserReq
	if err := c.Bind().Body(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"Message": "error parsing body",
			"success": "false",
			"error":   err.Error(),
		})
	}
	// check does user already exists		
	var existinguser User
	if usrerro := h.DB.Where("username = ?",req.Email).Find(&existinguser); usrerro.Error != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"Message": "error getting user for existing check",
			"success": "false",
		})
	}

	if existinguser.Email == req.Email {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"Message": "error user with gmail already exist`",
			"success": "false",
		})	
	}
	// hash the password and store the user in db 
	hashp,err := HashPassword(req.Password)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"Message": "error hashing password`",
			"success": "false",
		})		
	}

	//TODO: create a new user and store it in the db	
	user := User{
		Name: req.Username,
		Email: req.Email,
		Password: hashp,
		Role: req.Role,
	}

	// store in the db	
	h.DB.Create(user)
	return nil 
}
