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
	StudentId string
	Status    string
}

func AMarkedHandler(c *websocket.Conn, data *Data) error {
	fmt.Println("called amarked")
	user := c.Locals("wsuser")
	ws, ok := user.(*WsUser)
	if !ok || ws == nil {
		return fmt.Errorf("wsuser missing or wrong type")
	}
	if ws.Role != "teacher" {
		return fmt.Errorf("only teacher can take attendance")
	}
	if ac == nil {
		return fmt.Errorf("start the attendance first")
	}
	fmt.Printf("user role is %s", ws.Role)
	return nil
}

func Unmarshall(message []byte, c *websocket.Conn) error {
	var req Incoming
	if err := json.Unmarshal(message, &req); err != nil {
		return fmt.Errorf("error marshalling into strcut")
	}

	switch req.Event {
	case "ATTENDANCE_MARKED":
		// TODO: error here fix by marshall and unmarshall
		dataBytes, err := json.Marshal(req.Data)
		if err != nil {
			return fmt.Errorf("error marshalling data")
		}
		var data Data
		if err := json.Unmarshal(dataBytes, &data); err != nil {
			return fmt.Errorf("error unmarshalling into data struct: %v", err)
		}
		if err := AMarkedHandler(c, &data); err != nil {
			return err
		}

	default:
		return fmt.Errorf("unknown event")
	}

	return nil
}
