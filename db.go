package main

import (
	"errors"
	"fmt"
	"log"
	"strconv"
	"strings"
	"time"

	"github.com/gofiber/contrib/v3/websocket"
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

func (h *handler) checkTeacher(userId uint) (*User, error) {
	var user User
	res := h.DB.First(&user, userId)
	if res.Error != nil {
		return nil, res.Error
	}
	if user.Role != "teacher" {
		return nil, fmt.Errorf("user is not a teacher")
	} else {
		return &user, nil
	}
}

type CreateClassReq struct {
	ClassName string `json:"className"`
}

func (h *handler) CreateClass(c fiber.Ctx) error {
	fmt.Println("create class started")
	var req CreateClassReq
	if err := c.Bind().Body(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"Message": "error parsing body",
			"success": "false",
			"error":   err.Error(),
		})
	}
	userid := c.Locals("userid")
	teacher, err := h.checkTeacher(userid.(uint))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"Message": "error user is not a teacher",
			"success": "false",
			"error":   err.Error(),
		})
	}

	class := Class{
		ClassName: req.ClassName,
		TeacherID: teacher.ID,
	}

	if cerr := h.DB.Create(&class); cerr.Error != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"Message": "error creating class",
			"success": "false",
		})
	}

	return c.Status(fiber.StatusOK).JSON(fiber.Map{
		"Message":  "class created",
		"success":  "true",
		"id":       class.ID,
		"name":     class.ClassName,
		"teacher":  teacher.ID,
		"students": class.Students,
	})
}

type AddStudentReq struct {
	StudentID uint `json:"studentId"`
}

/*
// 1. check if class exist
// 2. check if user is teacher of that class
// 3. check if student exist
*/
func (h *handler) AddStudentToClass(c fiber.Ctx) error {
	var req AddStudentReq
	if err := c.Bind().Body(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"Message": "error parsing body",
			"success": "false",
			"error":   err.Error(),
		})
	}
	classId := c.Params("id")
	if after, ok := strings.CutPrefix(classId, ":id="); ok {
		classId = after
	}
	idInt, err := strconv.Atoi(classId)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"Message": "invalid class id",
			"success": "false",
			"error":   err.Error(),
		})
	}

	var class Class
	res := h.DB.First(&class, idInt)
	if res.Error != nil {
		if errors.Is(res.Error, gorm.ErrRecordNotFound) {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
				"Message": "class not found",
				"success": "false",
			})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"Message": "error getting class",
			"success": "false",
		})
	}

	teacherId := c.Locals("userid")
	if class.TeacherID != teacherId.(uint) {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"Message": "error user is not the teacher of this class",
			"success": "false",
		})
	}

	var student User
	if serr := h.DB.First(&student, req.StudentID); serr.Error != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"Message": "error getting student",
			"success": "false",
		})
	}
	// add student to class
	if aerr := h.DB.Model(&class).Association("Students").Append(&student); aerr != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"Message": "error adding student to class",
			"success": "false",
		})
	}

	return c.Status(fiber.StatusOK).JSON(fiber.Map{
		"Message":   "student added to class",
		"classname": class.ClassName,
		"teacherid": class.TeacherID,
		"studentid": student.ID,
		"students":  class.Students,
		"success":   "true",
	})
}

func (h *handler) GetClassDetails(c fiber.Ctx) error {
	classId := c.Params("id")
	if after, ok := strings.CutPrefix(classId, ":id="); ok {
		classId = after
	}
	idInt, err := strconv.Atoi(classId)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"Message": "invalid class id",
			"success": "false",
			"error":   err.Error(),
		})
	}
	userid := c.Locals("userid")
	var class Class
	if cerr := h.DB.First(&class, idInt); cerr.Error != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"Message": "error getting class",
			"success": "false",
		})
	}

	// NOTE ---> we dont need to fetch all the students here just to check if the user is part of the class change this later
	students := []User{}
	if serr := h.DB.Model(&class).Association("Students").Find(&students); serr != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"Message": "error getting students of class",
			"success": "false",
		})
	}

	if len(students) == 0 {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{})
	}
	if class.TeacherID != userid.(uint) {
		for _, student := range students {
			if student.ID == userid.(uint) {
				return c.Status(fiber.StatusOK).JSON(fiber.Map{
					"Message":   "class details",
					"success":   "true",
					"classname": class.ClassName,
					"teacherid": class.TeacherID,
					"students":  students,
				})
			} else {
				return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
					"Message": "error user is not authorized to view this class",
					"success": "false",
				})
			}
		}
	}
	return c.Status(fiber.StatusOK).JSON(fiber.Map{
		"Message":   "class details",
		"success":   "true",
		"classname": class.ClassName,
		"teacherid": class.TeacherID,
		"students":  students,
	})
}

func (h *handler) GetAllStudets(c fiber.Ctx) error {
	teacherId := c.Locals("userid").(uint)
	var teacher User
	if terr := h.DB.First(&teacher, teacherId); terr.Error != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"Message": "error getting teacher",
			"success": "false",
		})
	}
	if teacher.Role != "teacher" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"Message": "user is not a teacher",
			"success": "false",
		})
	}
	students := []User{}
	if serr := h.DB.Where("role = ?", "student").Find(&students); serr.Error != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"Message": "error getting students",
			"success": "false",
		})
	}
	return c.Status(fiber.StatusOK).JSON(fiber.Map{
		"Message":  "students fetched successfully",
		"success":  "true",
		"students": students,
	})
}

// TODO: TEST THIS ROUTE
func (h *handler) GetMyAttendance(c fiber.Ctx) error {
	classId := c.Params("id")
	if after, ok := strings.CutPrefix(classId, ":id="); ok {
		classId = after

	}
	idInt, err := strconv.Atoi(classId)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"Message": "invalid class id",
			"success": "false",
			"error":   err.Error(),
		})
	}
	// check if class exist
	var class Class
	if cerr := h.DB.First(&class, idInt); cerr.Error != nil {
		if errors.Is(cerr.Error, gorm.ErrRecordNotFound) {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
				"Message": "class not found",
				"success": "false",
			})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"Message": "error getting class",
			"success": "false",
		})
	}

	studentId := c.Locals("userid").(uint)

	// Get attendance for this class and student
	var attendance Attendance
	res := h.DB.Where("class_id = ? AND student_id = ?", idInt, studentId).First(&attendance)
	if errors.Is(res.Error, gorm.ErrRecordNotFound) {
		return c.Status(fiber.StatusOK).JSON(fiber.Map{
			"success": true,
			"data": fiber.Map{
				"classId": idInt,
				"status":  nil,
			},
		})
	}

	if res.Error != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"Message": "error getting attendance",
			"success": "false",
		})
	}

	return c.Status(fiber.StatusOK).JSON(fiber.Map{
		"success": true,
		"data": fiber.Map{
			"classId": idInt,
			"status":  attendance.Status,
		},
	})
}

type attendanceReq struct {
	ClassId string `json:"classid"`
}

type ActiveSession struct {
	ClassId   uint
	StartedAt int64
	// Attendance is map
	Session map[uint]string
}

func (h *handler) StartAttendance(c fiber.Ctx) error {
	userId := c.Locals("userid").(uint)
	var teacher User
	if t := h.DB.First(&teacher, userId); t.Error != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"Message": "error getting user",
			"success": "false",
		})
	}
	if teacher.Role != "teacher" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"Message": "error user is not teacher",
			"success": "false",
		})
	}

	var req attendanceReq
	if bindErr := c.Bind().Body(&req); bindErr != nil {
		return c.Status(fiber.ErrBadRequest.Code).JSON(fiber.Map{
			"message": "error parsing body",
			"suceess": "false",
		})
	}

	var class Class
	if dbclass := h.DB.First(&class, req.ClassId); dbclass.Error != nil {
		return c.Status(fiber.ErrBadRequest.Code).JSON(fiber.Map{
			"message": "error parsing body",
			"suceess": "false",
		})
	}

	if class.ID == 0 {
		return c.Status(fiber.ErrBadRequest.Code).JSON(fiber.Map{
			"message": "class not exist with this id",
			"suceess": "false",
		})
	}

	// start attendance session in memory
	ac := ActiveSession{
		ClassId:   class.ID,
		StartedAt: time.Now().Unix(),
		Session:   make(map[uint]string),
	}
	return c.Status(fiber.StatusOK).JSON(fiber.Map{
		"success": "true",
		"data": fiber.Map{
			"classid":   class.ID,
			"startedAt": ac.StartedAt,
		},
	})
}

func (h *handler) HandleWebSocket(c *websocket.Conn) {
	fmt.Println("client connected successfully")
	c.WriteMessage(websocket.TextMessage, []byte("Connected to attendance system"))

	for {
		_, msg, err := c.ReadMessage()
		if err != nil {
			fmt.Println("error reading message:", err)
			break
		}
		fmt.Printf("received message: %s\n", msg)

		
	}
}
