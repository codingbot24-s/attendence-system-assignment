package main

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/gofiber/contrib/v3/websocket"
	"github.com/gofiber/fiber/v3"
	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"
)

func HashPassword(password string) (string, error) {
	bytes, err := bcrypt.GenerateFromPassword([]byte(password), 14)
	return string(bytes), err
}

func VerifyPassword(password, hash string) bool {
	err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(password))
	return err == nil
}

var secretKey = []byte("1234")

type MyCustomClaims struct {
	UserId uint
	jwt.RegisteredClaims
}

func CreateJWTToken(userId uint) (string, error) {

	claims := MyCustomClaims{
		UserId: userId,
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)

	tokenString, err := token.SignedString(secretKey)
	if err != nil {
		return "", fmt.Errorf("error signing token")
	}

	return tokenString, nil
}

func verifyToken(tokenString string) (*MyCustomClaims, error) {
	claims := &MyCustomClaims{}
	token, err := jwt.ParseWithClaims(tokenString, claims, func(token *jwt.Token) (interface{}, error) {
		// Return the secret key
		return secretKey, nil
	})

	if err != nil {
		return nil, err
	}

	if !token.Valid {
		return nil, fmt.Errorf("invalid token")
	}

	return claims, nil
}

func authMiddleware(c fiber.Ctx) error {

	authHeader := c.Get("Authorization")

	if authHeader == "" || !strings.HasPrefix(authHeader, "Bearer ") {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"message": "Missing or invalid authorization header",
		})
	}

	token := strings.TrimPrefix(authHeader, "Bearer ")
	claims, err := verifyToken(token)
	if err != nil || claims == nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"message": "invalid or expired token",
		})
	}
	c.Locals("userid", claims.UserId)

	return c.Next()
}

type WsUser struct {
	UserId uint
	Role   string
}

func UpgradeGuard() fiber.Handler {
	return func(c fiber.Ctx) error {
		if websocket.IsWebSocketUpgrade(c) {
			token := c.Query("token")
			if token == "" {
				// 1. --> extract the query params
				return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
					"success": "false",
					"message": "no token",
				})
			}
			// 2. --> Verify JWT
			claims, err := verifyToken(token)
			if err != nil {
				return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
					"success": "false",
					"message": "invalid token",
				})
			}

			// 3. --> Attach user info to websocket
			// TODO: change the claims for dynamiclly for user and teacher we can do it by fetching the user from db and we can set the role of that user
			// how can we query db here
			c.Locals("wsuser", &WsUser{
				UserId: claims.UserId,
				Role:   "teacher",
			})
			return c.Next()
		}
		return fiber.ErrUpgradeRequired
	}
}

type Data struct {
	StudentId string `json:"studentId"`
	Status    string `json:"status"`
}

func AMarkedHandler(c *websocket.Conn, data *Data, msg []byte) error {
	user := c.Locals("wsuser")
	ws, ok := user.(*WsUser)
	if !ok || ws == nil {
		return fmt.Errorf("wsuser missing or wrong type")
	}
	if ws.Role != "teacher" {
		return fmt.Errorf("only teacher can take attendance")
	}
	if ac.ClassId == 0 {
		return fmt.Errorf("first start the attendance")
	}
	fmt.Print("all checks have been passed")
	ac.Attendance[data.StudentId] = data.Status

	for client := range clients.Client {
		client.WriteMessage(websocket.TextMessage, msg)
	}
	return nil
}

type Summary struct {
	Event  string
	Smdata SummaryData
}

type SummaryData struct {
	Present int
	Absent  int
	Total   int
}

func SummaryHandler(c *websocket.Conn, data *Data) error {
	user := c.Locals("wsuser")
	ws, ok := user.(*WsUser)
	if !ok || ws == nil {
		return fmt.Errorf("wsuser missing or wrong type")
	}
	if ws.Role != "teacher" {
		return fmt.Errorf("only teacher can take attendance")
	}
	if ac.ClassId == 0 {
		return fmt.Errorf("first start the attendance")
	}

	var presentc int
	var absentc int

	for _, v := range ac.Attendance {
		switch v {
		case "present":
			presentc += 1
		case "absent":
			absentc += 1
		}
	}

	summary := Summary{
		Event: "TODAY_SUMMARY",
		Smdata: SummaryData{
			Present: presentc,
			Absent:  absentc,
			Total:   presentc + absentc,
		},
	}

	msg, err := json.Marshal(summary)
	if err != nil {
		return fmt.Errorf("summary reponse error ")
	}

	for client := range clients.Client {
		client.WriteMessage(websocket.TextMessage, msg)
	}
	return nil
}

type AttendanceCheck struct {
	Event string
	adata AttendanceData
}

type AttendanceData struct {
	Status string
}

func MyAttendanceHandler(c *websocket.Conn, data *Data) error {
	user := c.Locals("wsuser")
	ws, ok := user.(*WsUser)
	if !ok || ws == nil {
		return fmt.Errorf("wsuser missing or wrong type")
	}
	// note we dont have call to db for checking after that
	// if ws.Role != "teacher" {
	// 	return fmt.Errorf("only teacher can take attendance")
	// }
	if ac.ClassId == 0 {
		return fmt.Errorf("attendance not started")
	}
	usrIDstr := fmt.Sprintf("%d", ws.UserId)

	if _, ok := ac.Attendance[usrIDstr]; !ok {
		return fmt.Errorf("error user with %d not in the attendance map ", ws.UserId)
	}

	status := ac.Attendance[usrIDstr]
	switch status {
	case "":
		acc := AttendanceCheck{
			Event: "MY_ATTENDANCE",
			adata: AttendanceData{
				Status: "not yet updated",
			},
		}
		msg, err := json.Marshal(acc)
		if err != nil {
			return fmt.Errorf("marshall error in acc ")
		}
		c.Conn.WriteMessage(websocket.TextMessage, msg)
	case "present":
		acc := AttendanceCheck{
			Event: "MY_ATTENDANCE",
			adata: AttendanceData{
				Status: status,
			},
		}
		msg, err := json.Marshal(acc)
		if err != nil {
			return fmt.Errorf("summary reponse error ")
		}
		c.Conn.WriteMessage(websocket.TextMessage, msg)
	}

	return nil
}

func Unmarshall(message []byte, c *websocket.Conn) error {
	var req Incoming
	if err := json.Unmarshal(message, &req); err != nil {
		return fmt.Errorf("error marshalling into strcut")
	}
	dataBytes, err := json.Marshal(req.Data)
	if err != nil {
		return fmt.Errorf("error marshalling data")
	}
	var data Data
	if err := json.Unmarshal(dataBytes, &data); err != nil {
		return fmt.Errorf("error unmarshalling into data struct: %v", err)
	}
	switch req.Event {
	case "ATTENDANCE_MARKED":
		if err := AMarkedHandler(c, &data, message); err != nil {
			return err
		}
	case "TODAY_SUMMARY":
		if err := SummaryHandler(c, &data); err != nil {
			return err
		}

	case "MY_ATTENDANCE":
		if err := MyAttendanceHandler(c,&data); err != nil {
			return err
		}
	default:
		return fmt.Errorf("unknown event")
	}

	return nil
}
