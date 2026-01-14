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
	r.Get("/auth/me", authMiddleware, h.GetMe)

	class := r.Group("/class")
	class.Use(authMiddleware)
	class.Post("/createclass", h.CreateClass)

	fmt.Println("server started on 3000")
	r.Listen(":3000")
}
