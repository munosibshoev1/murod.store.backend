package utils

import (
	"backend/config"
	"backend/models"
	
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"

	"net/http"
	"time"

	// "github.com/gin-gonic/gin"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// Ограничение на количество SMS в минуту
const smsLimitPerMinute = 6

// sendSMS отправляет SMS через внешний сервис и логирует в базе
func SendSMS(phone string, message string) error {
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


