package models

import (
    "go.mongodb.org/mongo-driver/bson/primitive"
    "time"
)

type Session struct {
    ID        primitive.ObjectID `bson:"_id,omitempty"`
    UserID    primitive.ObjectID `bson:"user_id"`
    Role      string             `bson:"role"`
    IP        string             `bson:"ip"`
    Device    string             `bson:"device"`
    Timestamp time.Time          `bson:"timestamp"`
}
type TemporarySession struct {
    ID        primitive.ObjectID `bson:"_id,omitempty" json:"id"`          // Уникальный ID сессии
    CashierID string             `bson:"cashier_id" json:"cashier_id"`     // ID кассира
    ClientID  string             `bson:"client_id" json:"client_id"`       // ID клиента
    Role      string             `bson:"role" json:"role"`                 // Роль пользователя (например, "client")
    IP        string             `bson:"ip" json:"ip"`                     // IP-адрес кассира
    Device    string             `bson:"device" json:"device"`             // Устройство, с которого вошел кассир
    CreatedAt time.Time          `bson:"created_at" json:"created_at"`     // Время создания сессии
    ExpiresAt time.Time          `bson:"expires_at" json:"expires_at"`     // Время истечения сессии
}

type SMSLog struct {
	Phone          string    `bson:"phone"`
	TotalSent      int       `bson:"total_sent"`      // Общее количество SMS за все время
	FailedAttempts int       `bson:"failed_attempts"` // Количество неудачных попыток
	LastSent       time.Time `bson:"last_sent"`       // Время последней отправки
	SMSLastMinute  int       `bson:"sms_last_minute"` // Счетчик SMS за последнюю минуту
}
