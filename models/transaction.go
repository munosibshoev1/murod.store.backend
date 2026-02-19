package models

import (
    "time"

    "go.mongodb.org/mongo-driver/bson/primitive"
)

type Transaction struct {
    ID          primitive.ObjectID `bson:"_id,omitempty" json:"id,omitempty"`
    CardNumber  string             `bson:"cardnumber" json:"cardnumber"`
    Type        string             `bson:"type" json:"type"`
    Amount      float64            `bson:"amount" json:"amount"`
    Purchase float64               `bson:"purchase" json:"purchase"`
    Date        time.Time          `bson:"date" json:"date"`
    CashierID   string             `bson:"cashierid" json:"cashierid"`
    CashierName    string          `bson:"cashiername,omitempty" json:"cashiername,omitempty"`
    ClientName     string          `bson:"clientname,omitempty" json:"clientname,omitempty"`
    Sumsettle      float64         `bson:"sumsettle" json:"sumsettle"`
    Procent        int64           `bson:"procent" json:"procent"`
    Location       string          `bson:"location" json:"location"`
    Without        float64         `bson:"without" json:"without"`
    Days           int64           `json:"days" binding:"required"`
}
// OrderEditLog фиксирует каждое изменение товара в заказе
type OrderEditLog struct {
	Action   string    `bson:"action" json:"action"`       // "add" или "remove"
	Barcode  string    `bson:"barcode" json:"barcode"`     // Штрихкод товара
	Quantity float64   `bson:"quantity" json:"quantity"`   // Сколько добавлено/удалено
	Cashier  string    `bson:"cashier" json:"cashier"`     // ID кассира, кто сделал изменение
	Time     time.Time `bson:"time" json:"time"`           // Когда было изменение
}
type ProductQuantity struct {
	Barcode  string  `json:"barcode"`
	Quantity float64 `json:"quantity"`
}