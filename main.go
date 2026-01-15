package main

import (
	"fmt"

	"github.com/gofiber/contrib/v3/websocket"
	"github.com/gofiber/fiber/v3"
)

func main() {
	db := ConnectToDB()
	h := NewDB(db)
	r := fiber.New()

	r.Post("/auth/signup", h.CreateUser)
	r.Post("/auth/lo	gin", h.Login)
	r.Get("/auth/me", authMiddleware, h.GetMe)

	class := r.Group("/class")
	class.Use(authMiddleware)
	class.Post("/createclass", h.CreateClass)
	class.Post("/:id/addstudent", h.AddStudentToClass)
	class.Get("/students", h.GetAllStudets)
	class.Get("/:id/class", h.GetClassDetails)
	class.Get("/:id/my-attendance", h.GetMyAttendance)
	class.Post("/attendance/start", authMiddleware)

	r.Get("/ws", UpgradeGuard(), websocket.New(h.HandleWebSocket))

	fmt.Println("server started on 3000")
	r.Listen(":3000")
}
