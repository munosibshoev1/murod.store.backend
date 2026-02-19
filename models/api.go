package models

import (
	// "time"

	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

type ShopAPIKey struct {
    ID         primitive.ObjectID `bson:"_id,omitempty"`       // Уникальный идентификатор
    Key        string             `bson:"key"`                 // Сам ключ
    IsActive   bool               `bson:"is_active"`           // Активен ли ключ
    CreatedAt  time.Time          `bson:"created_at"`          // Дата создания ключа
    UpdatedAt  time.Time          `bson:"updated_at"`          // Последнее обновление
    ExpiresAt  time.Time          `bson:"expires_at"`          // Дата истечения срока действия ключа
    Description string            `bson:"description"`         // Описание ключа, например, "Для транзакций интернет-магазина"
}

