package controllers

import (
	"backend/config"
	"backend/models"
	"backend/utils"
	"bytes"
	"context"
	// "crypto/rand"
	// "encoding/base64"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"math/rand"
	"log"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// Temporary storage for verification codes
var verificationCodes = make(map[string]string)
var codeExpiry = make(map[string]time.Time)
// Generate random verification code
func generateVerificationCode() string {
	rand.Seed(time.Now().UnixNano())
	return fmt.Sprintf("%06d", rand.Intn(1000000))
}


// RequestPasswordReset handles password reset requests
func RequestPasswordReset(c *gin.Context) {
    var input struct {
        Phone string `json:"phone" binding:"required"`
    }
    if err := c.ShouldBindJSON(&input); err != nil {
        c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
        return
    }

    ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
    defer cancel()

    var user models.User
    var cashier models.Cashier
    var client models.Client

    err := config.UserCollection.FindOne(ctx, bson.M{"phone": input.Phone}).Decode(&user)
    if err != nil {
        err = config.CashierCollection.FindOne(ctx, bson.M{"phone": input.Phone}).Decode(&cashier)
        if err != nil {
            err = config.ClientCollection.FindOne(ctx, bson.M{"phone": input.Phone}).Decode(&client)
            if err != nil {
                c.JSON(http.StatusNotFound, gin.H{"error": "User not found"})
                return
            }
        }
        
    }

    // Генерируем и сохраняем код верификации
	code := generateVerificationCode()
	verificationCodes[input.Phone] = code
	codeExpiry[input.Phone] = time.Now().Add(2 * time.Minute)

	// Формируем сообщение с кодом
	message := fmt.Sprintf("Рамзи Шумо барои тасдиқ: %s. Рамзро ба шахси сеюм надиҳед!", code)

	// Отправляем SMS через внешний сервис
	err = sendSMS(removePlusFromPhone(input.Phone), message)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to send SMS", "details": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Verification code sent"})
}

// Ограничение на количество SMS в минуту
const smsLimitPerMinute = 6

// sendSMS отправляет SMS через внешний сервис и логирует в базе
func sendSMS(phone string, message string) error {
	// Подключение к базе данных
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Проверяем, сколько сообщений уже отправлено за последнюю минуту
	var smsLog models.SMSLog
	err := config.SMSLogCollection.FindOne(ctx, bson.M{"phone": phone}).Decode(&smsLog)

	// Определяем, нужно ли сбросить счетчик `sms_last_minute`
	shouldReset := false
	if err == nil {
		elapsed := time.Since(smsLog.LastSent).Minutes()
		if elapsed >= 1 {
			shouldReset = true
		}

		// Проверяем ограничение: не более `smsLimitPerMinute` SMS за минуту
		if !shouldReset && smsLog.SMSLastMinute >= smsLimitPerMinute {
			return fmt.Errorf("ограничение на отправку SMS: попробуйте позже")
		}
	}

	// URL вашего сервиса для отправки SMS
	smsServiceURL := "https://sms.matrix.tj/smsmurod"

	// Формируем запрос
	payload := map[string]string{
		"phone":   phone,
		"message": message,
	}
	reqBody, _ := json.Marshal(payload)

	// Логируем тело запроса
	log.Printf("Sending SMS request: %s", reqBody)

	// Отправляем POST-запрос
	resp, err := http.Post(smsServiceURL, "application/json", bytes.NewBuffer(reqBody))
	if err != nil {
		// Логируем неудачную попытку в базе
		update := bson.M{"$inc": bson.M{"failed_attempts": 1}}
		config.SMSLogCollection.UpdateOne(ctx, bson.M{"phone": phone}, update, options.Update().SetUpsert(true))
		return fmt.Errorf("failed to send SMS: %v", err)
	}
	defer resp.Body.Close()

	// Читаем ответ
	respBody, _ := ioutil.ReadAll(resp.Body)
	log.Printf("SMS service response status: %d, body: %s", resp.StatusCode, string(respBody))

	// Проверяем статус ответа
	if resp.StatusCode != http.StatusOK {
		// Логируем неудачную попытку
		update := bson.M{"$inc": bson.M{"failed_attempts": 1}}
		config.SMSLogCollection.UpdateOne(ctx, bson.M{"phone": phone}, update, options.Update().SetUpsert(true))
		return fmt.Errorf("received non-200 response from SMS service: %d", resp.StatusCode)
	}

	// Формируем обновление для базы данных
	update := bson.M{
		"$set": bson.M{"last_sent": time.Now()},
		"$inc": bson.M{"total_sent": 1},
	}

	// Если прошло больше 1 минуты, сбрасываем счетчик SMS за минуту
	if shouldReset {
		update["$set"].(bson.M)["sms_last_minute"] = 1
	} else {
		update["$inc"].(bson.M)["sms_last_minute"] = 1
	}

	// Обновляем данные в базе
	config.SMSLogCollection.UpdateOne(ctx, bson.M{"phone": phone}, update, options.Update().SetUpsert(true))

	return nil
}

// VerifyCode handles code verification
func VerifyCode(c *gin.Context) {
    var input struct {
        Phone string `json:"phone" binding:"required"`
        Code  string `json:"code" binding:"required"`
    }
    if err := c.ShouldBindJSON(&input); err != nil {
        c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
        return
    }

    storedCode, exists := verificationCodes[input.Phone]
    if !exists || storedCode != input.Code || time.Now().After(codeExpiry[input.Phone]) {
        c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid or expired code"})
        return
    }

    c.JSON(http.StatusOK, gin.H{"message": "Code verified"})
}

// ResetPassword handles password resetting
func ResetPassword(c *gin.Context) {
    var input struct {
        Phone       string `json:"phone" binding:"required"`
        Code        string `json:"code" binding:"required"`
        NewPassword string `json:"newpassword" binding:"required"`
    }
    if err := c.ShouldBindJSON(&input); err != nil {
        c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
        return
    }

    // Verify the code
    storedCode, exists := verificationCodes[input.Phone]
    if !exists || storedCode != input.Code || time.Now().After(codeExpiry[input.Phone]) {
        c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid or expired code"})
        return
    }

    // Hash the new password
    hashedPassword, err := utils.HashPassword(input.NewPassword)
    if err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"error": "Error hashing password"})
        return
    }

    // Update the password in the database
    ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
    defer cancel()

    // Attempt to update user
    updateResult, err := config.UserCollection.UpdateOne(ctx, bson.M{"phone": input.Phone}, bson.M{"$set": bson.M{"password": hashedPassword}})
    if err != nil {
        log.Println("Error updating user password:", err)
        c.JSON(http.StatusInternalServerError, gin.H{"error": "Error updating password"})
        return
    }
	// Attempt to update user
    updateResult, err = config.ClientCollection.UpdateOne(ctx, bson.M{"phone": input.Phone}, bson.M{"$set": bson.M{"password": hashedPassword}})
    if err != nil {
        log.Println("Error updating user password:", err)
        c.JSON(http.StatusInternalServerError, gin.H{"error": "Error updating password"})
        return
    }

    // If no user updated, attempt to update cashier
    if updateResult.MatchedCount == 0 {
        updateResult, err = config.CashierCollection.UpdateOne(ctx, bson.M{"phone": input.Phone}, bson.M{"$set": bson.M{"password": hashedPassword}})
        if err != nil {
            log.Println("Error updating cashier password:", err)
            c.JSON(http.StatusInternalServerError, gin.H{"error": "Error updating password"})
            return
        }
    }
    

    // Log the update result for debugging
    log.Printf("Password update result: MatchedCount=%d, ModifiedCount=%d", updateResult.MatchedCount, updateResult.ModifiedCount)

    // Remove the code after successful reset
    delete(verificationCodes, input.Phone)
    delete(codeExpiry, input.Phone)

    c.JSON(http.StatusOK, gin.H{"message": "Password reset successful"})
}
