package main

import (
	"fmt"

	"github.com/gofiber/fiber/v3"
)

func main() {
	db := ConnectToDB()
	h := NewDB(db)
	r := fiber.New()

	r.Post("/auth/signup", h.CreateUser)
	r.Post("/auth/login", h.Login)

	fmt.Println("server started on 3000")
	r.Listen(":3000")
}
