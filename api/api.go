package api

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"

	// "log"
	"net/http"
	// "sort"
	"time"

	// "backend/config"
	// "backend/controllers"
	//  "backend/handlers"
	"backend/config"
	"backend/models"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	// "backend/utils"
	// "github.com/gin-gonic/gin"
	// "go.mongodb.org/mongo-driver/bson"
	// "go.mongodb.org/mongo-driver/bson/primitive"
	// "go.mongodb.org/mongo-driver/mongo"
	// "go.mongodb.org/mongo-driver/mongo/options"
)

func ProcessPeshraftRefund(transactionID string, amount float64) (bool, string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Получаем API-ключ из базы данных (как в ProcessPeshraftTransaction)
	apiKey, err := GetShopAPIKey(ctx)
	if err != nil {
		return false, "", fmt.Errorf("failed to get API key: %w", err)
	}

	// Формируем URL для возврата
	// Предположим, что эндпоинт выглядит так: POST /api/transaction/{transactionID}/return
	url := fmt.Sprintf("https://bp.murod.store/api/transaction/%s/return", transactionID)

	// Формируем тело запроса
	requestBody, _ := json.Marshal(map[string]interface{}{
		"amount": amount,
	})

	// Создаём запрос
	req, err := http.NewRequest("PUT", url, bytes.NewBuffer(requestBody))
	if err != nil {
		return false, "", fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-API-Key", apiKey)
	// Здесь важно: ваш код ReturnTransaction требует "Cashier-ID" в заголовке
	// Можете передавать какой-то системный ID, или хранить ID кассира, кто делал покупку
	req.Header.Set("Cashier-ID", "6789dda813e605d4bf8eec86")

	// Отправляем запрос
	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return false, "", fmt.Errorf("failed to send refund request: %w", err)
	}
	defer resp.Body.Close()

	// Читаем тело ответа
	body, _ := io.ReadAll(resp.Body)

	// Проверяем статус ответа
	if resp.StatusCode != http.StatusOK {
		return false, string(body), fmt.Errorf("Peshraft refund error: %s", body)
	}

	return true, string(body), nil
}

func GetShopAPIKey(ctx context.Context) (string, error) {
	var apiKey models.ShopAPIKey
	err := config.ShopAPIKeyCollection.FindOne(ctx, bson.M{
		"is_active":  true,
		"expires_at": bson.M{"$gt": time.Now()}, // Проверяем, что ключ не истек
	}).Decode(&apiKey)

	if err != nil {
		if err == mongo.ErrNoDocuments {
			return "", fmt.Errorf("no active API key found")
		}
		return "", fmt.Errorf("failed to retrieve API key: %w", err)
	}

	return apiKey.Key, nil
}
