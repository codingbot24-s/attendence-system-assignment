package main

import (
	"errors"
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
	if usrerro := h.DB.Where("name = ?", req.Username).Find(&existinguser); usrerro.Error != nil {
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
	hashp, err := HashPassword(req.Password)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"Message": "error hashing password`",
			"success": "false",
		})
	}

	user := User{
		Name:     req.Username,
		Email:    req.Email,
		Password: hashp,
		Role:     req.Role,
	}

	// store in the db
	if cerr := h.DB.Create(&user); cerr.Error != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"Message": "error creating user",
			"success": "false",
		})
	}

	return c.Status(fiber.StatusOK).JSON(fiber.Map{
		"Message": "user created",
		"success": "true",
		"id":      user.ID,
		"name":    user.Name,
		"email":   user.Email,
		"role":    user.Role,
	})
}

type LoginReq struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

func (h *handler) Login(c fiber.Ctx) error {

	var req LoginReq
	if err := c.Bind().Body(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"Message": "error parsing body",
			"success": "false",
			"error":   err.Error(),
		})
	}

	var user User
	if usrerro := h.DB.Where("Email = ?", req.Email).Find(&user); usrerro.Error != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"Message": "error getting user for existing check",
			"success": "false",
		})
	}
	if user.ID == 0 {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"Message": "user does not exist",
			"success": "false",
		})
	}

	if !VerifyPassword(req.Password, user.Password) {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"Message": "invalid password",
			"success": "false",
		})
	}

	token, err := CreateJWTToken(user.ID)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"Message": "error creating jwt token",
			"success": "false",
		})
	}

	return c.Status(fiber.StatusOK).JSON(fiber.Map{
		"Message": "login successful",
		"success": "true",
		"token":   token,
	})
}

func (h *handler) GetMe(c fiber.Ctx) error {
	

	userId := c.Locals("userid")

	var user User
	res := h.DB.First(&user, userId)
	if res.Error != nil {
		if errors.Is(res.Error, gorm.ErrRecordNotFound) {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
				"Message": "user not found",
				"success": "false",
			})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"Message": "error getting user",
			"success": "false",
		})
	}

	return c.Status(fiber.StatusOK).JSON(fiber.Map{
		"Message": "getting user successful",
		"success": "true",
		"id":      user.ID,
		"name":    user.Name,
		"email":   user.Email,
	})
}
