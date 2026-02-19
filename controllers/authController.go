package controllers

import (
	"backend/config"
	"backend/models"
	"backend/utils"
	"context"

	// "fmt"
	"strings"

	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"go.mongodb.org/mongo-driver/bson"
)

func Login(c *gin.Context) {
	var input models.User
	var foundUser models.User
	var foundCashier models.Cashier
	var foundClient models.Client
	var foundStorekeeper models.Storekeeper
	var foundOperator models.Storekeeper

	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	clientIP := getClientIP(c)

	err := config.UserCollection.FindOne(ctx, bson.M{"phone": input.Phone}).Decode(&foundUser)
	if err != nil {
		err = config.CashierCollection.FindOne(ctx, bson.M{"phone": input.Phone}).Decode(&foundCashier)
		if err != nil {
			err = config.StorekeeperCollection.FindOne(ctx, bson.M{"phone": input.Phone, "role": "storekeeper"}).Decode(&foundStorekeeper)
			if err != nil {
				err = config.OperatorCollection.FindOne(ctx, bson.M{"phone": input.Phone}).Decode(&foundOperator)
				if err == nil {
					err = utils.VerifyPassword(foundOperator.Password, input.Password)
					if err != nil {
						c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid credentials"})
						return
					}
					token, err := utils.GenerateToken(foundOperator.ID.Hex(), foundOperator.Role)
					if err != nil {
						c.JSON(http.StatusInternalServerError, gin.H{"error": "Error while generating token"})
						return
					}
					session := models.Session{
						UserID:    foundOperator.ID,
						Role:      foundOperator.Role,
						IP:        clientIP,
						Device:    c.Request.UserAgent(),
						Timestamp: time.Now(),
					}
					_, err = config.SessionCollection.InsertOne(ctx, session)
					if err != nil {
						c.JSON(http.StatusInternalServerError, gin.H{"error": "Error recording session"})
						return
					}
					c.SetCookie("token", token, 3600*24, "/", "murod.store", true, true)
					c.JSON(http.StatusOK, gin.H{
						"token":      token,
						"operatorID": foundOperator.ID.Hex(),
						"role":       foundOperator.Role,
						"fullName":   foundOperator.FirstName + " " + foundOperator.LastName,
					})
					return
				}

				err = config.ClientCollection.FindOne(ctx, bson.M{"phone": input.Phone}).Decode(&foundClient)
				if err != nil {
					c.JSON(http.StatusUnauthorized, gin.H{"error": "Incorrect Phone number", "phone": input.Phone})
					return
				}
				err = utils.VerifyPassword(foundClient.Password, input.Password)
				if err != nil {
					c.JSON(http.StatusUnauthorized, gin.H{"error": "Incorrect Password"})
					return
				}
				token, err := utils.GenerateToken(foundClient.ID.Hex(), foundClient.Role)
				if err != nil {
					c.JSON(http.StatusInternalServerError, gin.H{"error": "Error while generating token"})
					return
				}
				session := models.Session{
					UserID:    foundClient.ID,
					Role:      foundClient.Role,
					IP:        clientIP,
					Device:    c.Request.UserAgent(),
					Timestamp: time.Now(),
				}
				_, err = config.SessionCollection.InsertOne(ctx, session)
				if err != nil {
					c.JSON(http.StatusInternalServerError, gin.H{"error": "Error recording session"})
					return
				}
				c.SetCookie("token", token, 3600*24, "/", "murod.store", true, true)
				c.JSON(http.StatusOK, gin.H{
					"token":      token,
					"clientID":   foundClient.ID.Hex(),
					"role":       foundClient.Role,
					"fullName":   foundClient.FirstName + " " + foundClient.LastName,
					"photo_url":  foundClient.Photo_url,
					"hamrohcard": foundClient.HamrohCard,
					"type":       foundClient.Type,
				})
				return
			}

			err = utils.VerifyPassword(foundStorekeeper.Password, input.Password)
			if err != nil {
				c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid credentials"})
				return
			}
			token, err := utils.GenerateToken(foundStorekeeper.ID.Hex(), foundStorekeeper.Role)
			if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": "Error while generating token"})
				return
			}
			session := models.Session{
				UserID:    foundStorekeeper.ID,
				Role:      foundStorekeeper.Role,
				IP:        clientIP,
				Device:    c.Request.UserAgent(),
				Timestamp: time.Now(),
			}
			_, err = config.SessionCollection.InsertOne(ctx, session)
			if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": "Error recording session"})
				return
			}
			c.SetCookie("token", token, 3600*24, "/", "murod.store", true, true)
			c.JSON(http.StatusOK, gin.H{
				"token":         token,
				"storekeeperID": foundStorekeeper.ID.Hex(),
				"role":          foundStorekeeper.Role,
				"fullName":      foundStorekeeper.FirstName + " " + foundStorekeeper.LastName,
			})
			return
		}

		err = utils.VerifyPassword(foundCashier.Password, input.Password)
		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid credentials"})
			return
		}
		token, err := utils.GenerateToken(foundCashier.ID.Hex(), foundCashier.Role)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Error while generating token"})
			return
		}
		session := models.Session{
			UserID:    foundCashier.ID,
			Role:      foundCashier.Role,
			IP:        clientIP,
			Device:    c.Request.UserAgent(),
			Timestamp: time.Now(),
		}
		_, err = config.SessionCollection.InsertOne(ctx, session)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Error recording session"})
			return
		}
		c.SetCookie("token", token, 3600*24, "/", "murod.store", true, true)
		c.JSON(http.StatusOK, gin.H{
			"token":     token,
			"cashierID": foundCashier.ID.Hex(),
			"role":      foundCashier.Role,
			"fullName":  foundCashier.FirstName + " " + foundCashier.LastName,
		})
		return
	}

	err = utils.VerifyPassword(foundUser.Password, input.Password)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid credentials"})
		return
	}
	token, err := utils.GenerateToken(foundUser.ID.Hex(), foundUser.Role)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Error while generating token"})
		return
	}
	session := models.Session{
		UserID:    foundUser.ID,
		Role:      foundUser.Role,
		IP:        clientIP,
		Device:    c.Request.UserAgent(),
		Timestamp: time.Now(),
	}
	_, err = config.SessionCollection.InsertOne(ctx, session)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Error recording session"})
		return
	}
	cookie := &http.Cookie{
		Name:     "token",
		Value:    token,
		MaxAge:   3600 * 24,
		Path:     "/",
		Domain:   "murod.store",
		Secure:   true,
		HttpOnly: true,
		SameSite: http.SameSiteNoneMode,
	}
	http.SetCookie(c.Writer, cookie)
	c.JSON(http.StatusOK, gin.H{
		"token":    token,
		"role":     foundUser.Role,
		"fullName": foundUser.FirstName + " " + foundUser.LastName,
		"photo":    foundUser.ImageURL,
	})
}

func getClientIP(c *gin.Context) string {
	ip := c.Request.Header.Get("X-Forwarded-For")
	if ip == "" {
		ip = c.ClientIP()
	}
	return ip
}

func LoginCashierByCard(c *gin.Context) {
	var input struct {
		CashierID   string `json:"cashierid"`
		ClientCard  string `json:"clientcard"`
		Cashiername string `json:"cashiername"`
	}

	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request"})
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Проверяем, есть ли клиент с таким номером карты
	var client models.Client
	err := config.ClientCollection.FindOne(ctx, bson.M{"hamrohcard": input.ClientCard}).Decode(&client)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Client not found"})
		return
	}

	// Создаем временную сессию (ограничена 10 минутами)
	session := models.TemporarySession{
		CashierID: input.CashierID,
		ClientID:  client.ID.Hex(),
		CreatedAt: time.Now(),
		ExpiresAt: time.Now().Add(30 * time.Minute),
	}

	_, err = config.TemporarySessionCollection.InsertOne(ctx, session)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create session"})
		return
	}

	// Генерируем токен клиента
	token, err := utils.GenerateToken(client.ID.Hex(), client.Role)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to generate token"})
		return
	}

	// Отправляем SMS клиенту
	//utils.SendSMS(removePlusFromPhone(client.Phone), fmt.Sprintf("Кассир %s начал оформлят заказ за вас.", input.Cashiername))

	c.JSON(http.StatusOK, gin.H{
		"token1":     token,
		"clientID":   client.ID.Hex(),
		"cashierID":  input.CashierID,
		"role":       client.Role,
		"fullName":   client.FirstName + " " + client.LastName,
		"hamrohcard": client.HamrohCard,
		"type":       client.Type,
	})
}

// RemovePlusFromPhone удаляет "+" из номера телефона
func removePlusFromPhone(phone string) string {
	return strings.TrimPrefix(phone, "+")
}
