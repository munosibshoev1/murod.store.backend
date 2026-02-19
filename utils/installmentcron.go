package utils

import (
	"context"
	"log"
	"time"
    "fmt"
    "math"
	"backend/config"
	
	"backend/models"
    "strconv"
	"go.mongodb.org/mongo-driver/bson"
)

func TruncateToTwoDecimals(value float64) float64 {
    factor := 100.0
    value, _ = strconv.ParseFloat(fmt.Sprintf("%.2f", value), 64)
    return math.Floor(value * factor) / factor
}
func CheckInstallmentRates() {
    log.Println("Начало выполнения функции CheckInstallmentRates")

    // Получаем текущую дату
    currentDate := time.Now()
    log.Printf("Текущая дата: %v\n", currentDate)

    // Фильтр: берем только карты со статусом "used"
    filter := bson.M{"status": "Used"}

    // Находим все документы, удовлетворяющие фильтру
    cursor, err := config.CardCollection.Find(context.TODO(), filter)
    if err != nil {
        log.Fatalf("Ошибка при поиске документов: %v\n", err)
    }
    defer func() {
        err := cursor.Close(context.TODO())
        if err != nil {
            log.Fatalf("Ошибка при закрытии курсора: %v\n", err)
        }
    }()

    // Проверяем количество найденных документов
    count, err := config.CardCollection.CountDocuments(context.TODO(), filter)
    if err != nil {
        log.Fatalf("Ошибка при подсчете документов: %v\n", err)
    }
    log.Printf("Найдено карт со статусом 'used': %d\n", count)
    
    // Обрабатываем каждый документ
    for cursor.Next(context.TODO()) {
        var card models.Card
        err := cursor.Decode(&card)
        if err != nil {
            log.Fatalf("Ошибка декодирования документа: %v\n", err)
        }

        log.Printf("Обработка карты: %s\n", card.CardNumber)
        if card.TotalLoan != 0 {
            // Проверяем, что StartDate имеет корректное значение
            if card.StartDate.Time().IsZero() {
                log.Printf("Пропускаем карту с пустым StartDate: %s\n", card.CardNumber)
                continue
            }

            // Вычисляем количество дней с момента StartDate
            daysPassed := int64(currentDate.Sub(card.StartDate.Time()).Hours() / 24)
            log.Printf("Дней с момента StartDate для карты %s: %d\n", card.CardNumber, daysPassed)

            // Обновляем поле Days
            card.Days += 1
		
            if card.Days >= 40 {
                // Прошло больше или равно 40 дней
                card.TotalLoan = card.TotalLoan * 1.06 // Добавляем 6%
                card.TotalLoan = TruncateToTwoDecimals(card.TotalLoan)
                card.TotalFast = card.TotalLoan * 0.995
                card.TotalFast = TruncateToTwoDecimals(card.TotalFast)
                card.TotalOut = card.TotalLoan * 1.06
                card.TotalOut = TruncateToTwoDecimals(card.TotalOut)
                card.AllDays += card.Days
                card.Days = 0 // Сбрасываем в 0
                log.Printf("Карта %s: прошло больше 40 дней, обновляем TotalLoan и сбрасываем Days\n", card.CardNumber)
            } else {
                // Прошло меньше 40 дней
                // card.TotalLoan = card.TotalPurchase * 0.995 // Уменьшаем на 0.5%
                log.Printf("Карта %s: прошло меньше 40 дней, уменьшаем TotalLoan\n", card.CardNumber)
            }

            // Обновляем документ в базе данных
            update := bson.M{
                "$set": bson.M{
                    "totalloan": card.TotalLoan,
				    "totalfast": card.TotalFast,
				    "totalout": card.TotalOut,				
                    "days":      card.Days,
                    "alldays":   card.AllDays,
                },
            }

            _, err = config.CardCollection.UpdateOne(context.TODO(), bson.M{"_id": card.ID}, update)
            if err != nil {
                log.Fatalf("Ошибка при обновлении документа для карты %s: %v\n", card.CardNumber, err)
            }

            log.Printf("Карта %s успешно обновлена\n", card.CardNumber)
        }
    }

    if err := cursor.Err(); err != nil {
        log.Fatalf("Ошибка при работе с курсором: %v\n", err)
    }

    log.Println("CheckInstallmentRates completed successfully")
}


