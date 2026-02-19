package controllers

import (
	"backend/api"
	"backend/config"
	"math"
	"strings"

	// "backend/handlers"
	"backend/models"
	"backend/utils"
	"context"
	"log"
	"mime/multipart"
	"os"
	"path/filepath"
	"sort"

	// "log"

	// "encoding/json"
	"fmt"
	// "log"
	"math/rand"
	"net/http"

	// "strings"
	"time"

	"github.com/gin-gonic/gin"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
)

func ListClients(c *gin.Context) {
    ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
    defer cancel()

    // Fetch clients
    cursor, err := config.ClientCollection.Find(ctx, bson.M{})
    if err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"error": "Error retrieving clients"})
        return
    }
    defer cursor.Close(ctx)

    var clientReports []map[string]interface{}

    for cursor.Next(ctx) {
        var client models.Client
        if err := cursor.Decode(&client); err != nil {
            c.JSON(http.StatusInternalServerError, gin.H{"error": "Error decoding client"})
            return
        }

        // // Fetch the card details based on the card number
        // var card models.Card
        // err = config.CardCollection.FindOne(ctx, bson.M{"cardnumber": client.CardNumber}).Decode(&card)
        // if err != nil {
        //     c.JSON(http.StatusInternalServerError, gin.H{"error": "Error retrieving card details"})
        //     return
        // }

        // Construct the client report
        clientReport := map[string]interface{}{
            "fullname":            client.FirstName + " " + client.LastName,
            "hamrohcard":          client.HamrohCard,
            // "totalpurchase":          card.TotalPurchase,
            // "totalsettle":          card.TotalSettle,
            // "alltotal":          card.AllTotal,
            // "totalcashbackspent": card.TotalCashbackSpent,
            "id":client.ID,
            "type":client.Type,
            // "interest":card.TotalSettle-card.AllTotal,
        }
        clientReports = append(clientReports, clientReport)
    }

    if err := cursor.Err(); err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"error": "Error processing clients"})
        return
    }

    c.JSON(http.StatusOK, clientReports)
}
// path: handlers/client_handler.go

func ListRetailClients(c *gin.Context) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Фильтр: только клиенты с типом "retail"
	filter := bson.M{"type": "retail"}

	cursor, err := config.ClientCollection.Find(ctx, filter)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Ошибка при получении клиентов"})
		return
	}
	defer cursor.Close(ctx)

	var retailClients []map[string]interface{}

	for cursor.Next(ctx) {
		var client models.Client
		if err := cursor.Decode(&client); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Ошибка при декодировании клиента"})
			return
		}

		retailClients = append(retailClients, map[string]interface{}{
			"name":     client.FirstName + " " + client.LastName,
			"clientid": client.ID.Hex(),
		})
	}

	if err := cursor.Err(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Ошибка при обработке курсора"})
		return
	}

	c.JSON(http.StatusOK, retailClients)
}

// Структура для приема данных с фронтенда
type CreateClientRequest struct {
    FirstName  string `json:"first_name" binding:"required"`
    LastName   string `json:"last_name" binding:"required"`
    BirthDate  string `json:"birth_date" binding:"required"`
    Phone      string `json:"phone" binding:"required"`
    Password   string `json:"password" binding:"required"`
    CardNumber string `json:"cardnumber" binding:"required"`
    Limit      float64    `json:"limit" binding:"required"` // Лимит для карты
    
}

// SaveAvatar сохраняет аватар пользователя и возвращает только имя файла
func SaveAvatar(c *gin.Context, file *multipart.FileHeader, clientID string) (string, error) {
    // Директория для сохранения аватарок
    avatarDir := "./uploads/avatars"
    if _, err := os.Stat(avatarDir); os.IsNotExist(err) {
        err := os.MkdirAll(avatarDir, os.ModePerm)
        if err != nil {
            return "", fmt.Errorf("failed to create avatar directory: %v", err)
        }
    }

    // Генерация уникального имени файла
    filename := fmt.Sprintf("%s_%d%s", clientID, time.Now().Unix(), filepath.Ext(file.Filename))
    fullPath := filepath.Join(avatarDir, filename)

    // Сохранение файла по полному пути
    if err := c.SaveUploadedFile(file, fullPath); err != nil {
        return "", fmt.Errorf("failed to save avatar: %v", err)
    }

    // Возвращаем только имя файла
    return filename, nil
}


func AddClient(c *gin.Context) {
    // Создаём объект клиента
    client := models.Client{
        Role: "client",
        ID:   primitive.NewObjectID(),
    }

    // Получаем и заполняем данные из form-data
    client.FirstName = c.PostForm("first_name")
    client.LastName = c.PostForm("last_name")
    client.BirthDate = c.PostForm("birth_date")
    client.Phone = c.PostForm("phone")
    client.Role = "client"
    client.Type = c.PostForm("type")
    // Проверка телефона
    isUsed, err := isPhoneNumberInUse(client.Phone)
    if err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"error": "Error checking phone number"})
        return
    }
    if isUsed {
        c.JSON(http.StatusBadRequest, gin.H{"error": "Phone number already in use"})
        return
    }

    // Хеширование пароля
    password := c.PostForm("password")
    hashedPassword, err := utils.HashPassword(password)
    if err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"error": "Error hashing password"})
        return
    }
    client.Password = hashedPassword

    // Проверка значения cardoption
    cardOption := c.PostForm("cardoption")
    cardOption = strings.ToLower(strings.TrimSpace(c.PostForm("cardoption")))
    switch cardOption {
    case "yes":
        var existingClient struct {
            CardNumber string `bson:"cardnumber"`
        }
        err = config.CardCollection.FindOne(
            context.TODO(),
            bson.M{"phone": client.Phone},
        ).Decode(&existingClient)
        if err != nil {
            c.JSON(http.StatusNotFound, gin.H{"error": "Client not found in external database"})
            return
        }
        client.HamrohCard = existingClient.CardNumber
    case "no":
        client.HamrohCard = ""
    default:
        c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid cardoption value, must be 'yes' or 'no'"})
        return
    }

    // Сохраняем фото клиента
    file, err := c.FormFile("photo_url")
    var photoPath string
    if err == nil {
        photoPath, err = SaveAvatar(c, file, client.ID.Hex())
        if err != nil {
            c.JSON(http.StatusInternalServerError, gin.H{"error": "Error saving photo"})
            return
        }
        client.Photo_url = photoPath
    } else {
        c.JSON(http.StatusBadRequest, gin.H{"error": "Photo file is required"})
        return
    }

    // Вставка клиента в базу данных
    ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
    defer cancel()

    _, err = config.ClientCollection.InsertOne(ctx, client)
    if err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"error": "Error adding client"})
        return
    }

    // Успешный ответ
    c.JSON(http.StatusCreated, gin.H{
        "message":   "Client created successfully",
    })
}





func GetUnusedCards(c *gin.Context) {
    var cards []models.Card

    ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
    defer cancel()

    cursor, err := config.CardCollection.Find(ctx, bson.M{"status": "Unused"})
    if err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"error": "Error fetching cards"})
        return
    }
   defer cursor.Close(ctx)

    if err = cursor.All(ctx, &cards); err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"error": "Error reading cards"})
        return
    }

    c.JSON(http.StatusOK, cards)
}

func UpdateClient(c *gin.Context) {
    clientID := c.Param("id")

    var updateData struct {
        FirstName  string `form:"first_name"`
        LastName   string `form:"last_name"`
        BirthDate  string `form:"birth_date"`
        Type       string `form:"type"`
        CardOption string `form:"cardoption"`
        Phone      string `form:"phone"`
        Password   string `form:"password"`
    }

    // Используем ShouldBind для multipart/form-data
    if err := c.ShouldBind(&updateData); err != nil {
        c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
        return
    }

    fmt.Println("Received Form Data:", updateData)

    update := bson.M{}

    // Photo upload handling
    file, err := c.FormFile("photo_url")
    var photoPath string
    if err == nil {
        fmt.Printf("Received photo file: %s (size: %d bytes)\n", file.Filename, file.Size)
        oid, err := primitive.ObjectIDFromHex(clientID)
        if err != nil {
            c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid client ID"})
            return
        }
        photoPath, err = SaveAvatar(c, file, oid.Hex())
        if err != nil {
            c.JSON(http.StatusInternalServerError, gin.H{"error": "Error saving photo"})
            return
        }
        update["photo_url"] = photoPath
    } else {
        fmt.Println("No photo file received")
    }

    // Убираем "undefined" значения
    if updateData.Phone != "" && updateData.Phone != "undefined" {
        isUsed, err := isPhoneNumberInUse(updateData.Phone)
        if err != nil {
            c.JSON(http.StatusInternalServerError, gin.H{"error": "Error checking phone number"})
            return
        }
        if isUsed {
            var existingClient models.Client
            err = config.ClientCollection.FindOne(context.TODO(), bson.M{"phone": updateData.Phone}).Decode(&existingClient)
            if err == nil && existingClient.ID.Hex() != clientID {
                c.JSON(http.StatusBadRequest, gin.H{"error": "Phone number already in use"})
                return
            }
        }
        update["phone"] = updateData.Phone
    }

    // Update client fields
    if updateData.FirstName != "" {
        update["first_name"] = updateData.FirstName
    }
    if updateData.LastName != "" {
        update["last_name"] = updateData.LastName
    }
    if updateData.BirthDate != "" {
        update["birth_date"] = updateData.BirthDate
    }
    if updateData.Type != "" {
        update["type"] = updateData.Type
    }

    if updateData.Password != "" && updateData.Password != "undefined" {
        hashedPassword, err := utils.HashPassword(updateData.Password)
        if err != nil {
            c.JSON(http.StatusInternalServerError, gin.H{"error": "Error hashing password"})
            return
        }
        update["password"] = hashedPassword
    }

    ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
    defer cancel()

    oid, err := primitive.ObjectIDFromHex(clientID)
    if err != nil {
        c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid client ID"})
        return
    }

    // Update client information
    _, err = config.ClientCollection.UpdateOne(
        ctx,
        bson.M{"_id": oid},
        bson.M{"$set": update},
    )
    if err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"error": "Error updating client"})
        return
    }

    c.JSON(http.StatusOK, gin.H{
        "message": "Client and card updated successfully",
        "photo_url": photoPath,
    })
}


func GetClient(c *gin.Context) {
    clientID := c.Param("id")

    oid, err := primitive.ObjectIDFromHex(clientID)
    if err != nil {
        c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid client ID"})
        return
    }

    var client models.Client
    ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
    defer cancel()

    err = config.ClientCollection.FindOne(ctx, bson.M{"_id": oid}).Decode(&client)
    if err != nil {
        c.JSON(http.StatusNotFound, gin.H{"error": "Client not found"})
        return
    }
    c.JSON(http.StatusOK, gin.H{
        "first_name":   client.FirstName,
        "last_name":    client.LastName,
        "birth_date":    client.BirthDate,
        "phone":        client.Phone,
        "photo_url":    client.Photo_url,
        "type":   client.Type,
        // Пароль не отправляем по соображениям безопасности
    })
}
func DeleteClient(c *gin.Context) {
    clientID := c.Param("id")

    oid, err := primitive.ObjectIDFromHex(clientID)
    if err != nil {
        c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid client ID"})
        return
    }

    ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
    defer cancel()

    _, err = config.ClientCollection.DeleteOne(ctx, bson.M{"_id": oid})
    if err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"error": "Error deleting client"})
        return
    }

    c.JSON(http.StatusOK, gin.H{"message": "Client deleted successfully"})
}
func ListCashiers(c *gin.Context) {
    ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
    defer cancel()

    cursor, err := config.CashierCollection.Find(ctx, bson.M{"role": "cashier"})
    if err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"error": "Error retrieving cashiers"})
        return
    }
    defer cursor.Close(ctx)

    var cashierReports []map[string]interface{}

    for cursor.Next(ctx) {
        var cashier models.Cashier
        if err := cursor.Decode(&cashier); err != nil {
            c.JSON(http.StatusInternalServerError, gin.H{"error": "Error decoding cashier"})
            return
        }

        matchStage := bson.D{{Key: "$match", Value: bson.D{{Key: "cashierid", Value: cashier.ID.Hex()}, {Key: "type", Value: "purchase"}}}}
        groupStage := bson.D{{Key: "$group", Value: bson.D{{Key: "_id", Value: nil}, {Key: "total", Value: bson.D{{Key: "$sum", Value: "$purchase"}}}}}}

        cursor, err := config.TransactionCollection.Aggregate(ctx, mongo.Pipeline{matchStage, groupStage})
        if err != nil {
            c.JSON(http.StatusInternalServerError, gin.H{"error": "Error retrieving transactions for cashier"})
            return
        }

        var result struct {
            Total float64 `bson:"total"`
        }
        if cursor.Next(ctx) {
            if err := cursor.Decode(&result); err != nil {
                c.JSON(http.StatusInternalServerError, gin.H{"error": "Error decoding transaction result"})
                return
            }
        }
        cursor.Close(ctx)

        // Construct the cashier report
        cashierReport := map[string]interface{}{
            "fullname":    cashier.FirstName + " " + cashier.LastName,
            "phone":        cashier.Phone,
            "totalsells": result.Total,
            "location":   cashier.Location,
            "birth_date":   cashier.BirthDate,
            "id":cashier.ID,
        }
        cashierReports = append(cashierReports, cashierReport)
    }

    if err := cursor.Err(); err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"error": "Error processing cashiers"})
        return
    }

    c.JSON(http.StatusOK, cashierReports)
}

func AddCashier(c *gin.Context) {
    var cashier models.Cashier
    if err := c.ShouldBindJSON(&cashier); err != nil {
        c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
        return
    }

    // Проверка, используется ли номер телефона
    isUsed, err := isPhoneNumberInUse(cashier.Phone)
    if err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"error": "Error checking phone number"})
        return
    }
    if isUsed {
        c.JSON(http.StatusBadRequest, gin.H{"error": "Phone number already in use"})
        return
    }

    hashedPassword, err := utils.HashPassword(cashier.Password)
    if err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"error": "Error hashing password"})
        return
    }
    cashier.Password = hashedPassword

    cashier.ID = primitive.NewObjectID()
    cashier.Role = "cashier"

    ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
    defer cancel()

    _, err = config.CashierCollection.InsertOne(ctx, cashier)
    if err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"error": "Error adding cashier"})
        return
    }

    c.JSON(http.StatusCreated, cashier)
}

func UpdateCashier(c *gin.Context) {
    cashierID := c.Param("id")
    var updateData models.Cashier

    if err := c.ShouldBindJSON(&updateData); err != nil {
        c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
        return
    }

    update := bson.M{}

    if updateData.Phone != "" {
        // Проверка, используется ли номер телефона
        isUsed, err := isPhoneNumberInUse(updateData.Phone)
        if err != nil {
            c.JSON(http.StatusInternalServerError, gin.H{"error": "Error checking phone number"})
            return
        }
        if isUsed {
            var existingCashier models.Cashier
            err = config.CashierCollection.FindOne(context.TODO(), bson.M{"phone": updateData.Phone}).Decode(&existingCashier)
            if err == nil && existingCashier.ID.Hex() != cashierID {
                c.JSON(http.StatusBadRequest, gin.H{"error": "Phone number already in use"})
                return
            }
        }
        update["phone"] = updateData.Phone
    }

    if updateData.FirstName != "" {
        update["first_name"] = updateData.FirstName
    }
    if updateData.LastName != "" {
        update["last_name"] = updateData.LastName
    }
    if updateData.BirthDate != "" {
        update["birth_date"] = updateData.BirthDate
    }
    if updateData.Location != "" {
        update["location"] = updateData.Location
    }
    if updateData.Password != "" {
        hashedPassword, err := utils.HashPassword(updateData.Password)
        if err != nil {
            c.JSON(http.StatusInternalServerError, gin.H{"error": "Error hashing password"})
            return
        }
        update["password"] = hashedPassword
    }

    ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
    defer cancel()

    oid, err := primitive.ObjectIDFromHex(cashierID)
    if err != nil {
        c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid cashier ID"})
        return
    }

    _, err = config.CashierCollection.UpdateOne(
        ctx,
        bson.M{"_id": oid},
        bson.M{"$set": update},
    )
    if err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"error": "Error updating cashier"})
        return
    }

    c.JSON(http.StatusOK, gin.H{"message": "Cashier updated successfully"})
}

func GetCashier(c *gin.Context) {
    cashierID := c.Param("id")

    oid, err := primitive.ObjectIDFromHex(cashierID)
    if err != nil {
        c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid cashier ID"})
        return
    }

    var cashier models.Cashier
    ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
    defer cancel()

    err = config.CashierCollection.FindOne(ctx, bson.M{"_id": oid}).Decode(&cashier)
    if err != nil {
        c.JSON(http.StatusNotFound, gin.H{"error": "Cashier not found"})
        return
    }

    // Предоставляем только поля, которые могут быть отредактированы
    c.JSON(http.StatusOK, gin.H{
        "first_name":   cashier.FirstName,
        "last_name":    cashier.LastName,
        "birth_date":    cashier.BirthDate,
        "phone":        cashier.Phone,
        "location":     cashier.Location,
        // Пароль не отправляем по соображениям безопасности
    })
}

func DeleteCashier(c *gin.Context) {
    cashierID := c.Param("id")

    oid, err := primitive.ObjectIDFromHex(cashierID)
    if err != nil {
        c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid cashier ID"})
        return
    }

    ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
    defer cancel()

    _, err = config.CashierCollection.DeleteOne(ctx, bson.M{"_id": oid})
    if err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"error": "Error deleting cashier"})
        return
    }

    c.JSON(http.StatusOK, gin.H{"message": "Cashier deleted successfully"})
}

func ListCards(c *gin.Context) {
    var cards []models.Card
    ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
    defer cancel()

    cursor, err := config.CardCollection.Find(ctx, bson.M{})
    if err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"error": "Error retrieving cards"})
        return
    }
    defer cursor.Close(ctx)

    for cursor.Next(ctx) {
        var card models.Card
        cursor.Decode(&card)
        cards = append(cards, card)
    }

    c.JSON(http.StatusOK, cards)
}

func GenerateCards(c *gin.Context) {
	var newCards []interface{}
	for i := 0; i < 10; i++ {
		cardNumber := generateUniqueCardNumber(config.CardCollection)
		newCard := models.Card{
			ID:              primitive.NewObjectID(),
			CardNumber:      cardNumber,
			Status:          "Unused",
            Limit: 0,
			CreateDate:      primitive.NewDateTimeFromTime(time.Now()),
		}
		newCards = append(newCards, newCard)
	}
	_, err := config.CardCollection.InsertMany(context.Background(), newCards)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Error generating cards"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "Cards generated successfully"})
}

func generateUniqueCardNumber(collection *mongo.Collection) string {
	for {
		cardNumber := rand.Intn(1000000) // генерируем число от 0 до 99999
		var card models.Card
		err := collection.FindOne(context.Background(), bson.M{"cardnumber": cardNumber}).Decode(&card)
		if err == mongo.ErrNoDocuments {
			return fmt.Sprintf("%05d", cardNumber) // возвращаем 5-значное число с ведущими нулями
		}
	}
}

func GetAllTransactions(c *gin.Context) {
    ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
    defer cancel()

    var transactions []models.Transaction

    cursor, err := config.TransactionCollection.Find(ctx, bson.M{})
    if err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"error": "Error retrieving transactions"})
        return
    }
    defer cursor.Close(ctx)

    for cursor.Next(ctx) {
        var transaction models.Transaction
        if err := cursor.Decode(&transaction); err != nil {
            c.JSON(http.StatusInternalServerError, gin.H{"error": "Error decoding transaction"})
            return
        }

        // Fetch the cashier name based on the cashier ID in the transaction
        var cashier models.Cashier
        cashierID, err := primitive.ObjectIDFromHex(transaction.CashierID)
        if err != nil {
            c.JSON(http.StatusInternalServerError, gin.H{"error": "Invalid cashier ID"})
            return
        }

        err = config.CashierCollection.FindOne(ctx, bson.M{"_id": cashierID}).Decode(&cashier)
        if err != nil {
            if err == mongo.ErrNoDocuments {
                transaction.CashierName = "Unknown"
            } else {
                c.JSON(http.StatusInternalServerError, gin.H{"error": "Error retrieving cashier"})
                return
            }
        } else {
            transaction.CashierName = cashier.FirstName + " " + cashier.LastName
        }

        transactions = append(transactions, transaction)
    }

    if err := cursor.Err(); err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"error": "Error processing transactions"})
        return
    }

    c.JSON(http.StatusOK, transactions)
}

func GetCardReport(c *gin.Context) {
    ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
    defer cancel()

    // var transactions []models.Transaction
    var report []map[string]interface{}

    cursor, err := config.TransactionCollection.Find(ctx, bson.M{})
    if err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"error": "Error retrieving transactions"})
        return
    }
    defer cursor.Close(ctx)

    for cursor.Next(ctx) {
        var transaction models.Transaction
        if err := cursor.Decode(&transaction); err != nil {
            c.JSON(http.StatusInternalServerError, gin.H{"error": "Error decoding transaction"})
            return
        }

        // Fetch the cashier name based on the cashier ID in the transaction
        var cashier models.Cashier
        cashierID, err := primitive.ObjectIDFromHex(transaction.CashierID)
        if err != nil {
            c.JSON(http.StatusInternalServerError, gin.H{"error": "Invalid cashier ID"})
            return
        }

        err = config.CashierCollection.FindOne(ctx, bson.M{"_id": cashierID}).Decode(&cashier)
        if err != nil {
            if err == mongo.ErrNoDocuments {
                transaction.CashierName = "Unknown"
            } else {
                c.JSON(http.StatusInternalServerError, gin.H{"error": "Error retrieving cashier"})
                return
            }
        } else {
            transaction.CashierName = cashier.FirstName + " " + cashier.LastName
        }

        // Fetch the client name based on the card number in the transaction
        var client models.Client
        err = config.ClientCollection.FindOne(ctx, bson.M{"cardnumber": transaction.CardNumber}).Decode(&client)
        if err != nil {
            if err == mongo.ErrNoDocuments {
                transaction.ClientName = "Unknown"
            } else {
                c.JSON(http.StatusInternalServerError, gin.H{"error": "Error retrieving client"})
                return
            }
        } else {
            transaction.ClientName = client.FirstName + " " + client.LastName
        }

        report = append(report, map[string]interface{}{
            "cashiername": transaction.CashierName,
            "cardnumber": transaction.CardNumber,
            "type": transaction.Type,
            "sumsettle": transaction.Sumsettle,
            "purchase": transaction.Purchase,
            "date": transaction.Date,
            "clientname": transaction.ClientName,
        })
    }

    if err := cursor.Err(); err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"error": "Error processing transactions"})
        return
    }

    c.JSON(http.StatusOK, report)
}

func GetCardReportByCashierID(c *gin.Context) {
    cashierIDParam := c.Param("cashierID")
    log.Println("Received cashierID:", cashierIDParam)

    ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
    defer cancel()

    var report []map[string]interface{}

    // Фильтруем транзакции по cashierID как по строке
    filter := bson.M{"cashierid": cashierIDParam}
    log.Println("Applying filter:", filter)

    cursor, err := config.TransactionCollection.Find(ctx, filter)
    if err != nil {
        log.Println("Error retrieving transactions:", err)
        c.JSON(http.StatusInternalServerError, gin.H{"error": "Error retrieving transactions"})
        return
    }
    defer cursor.Close(ctx)

    for cursor.Next(ctx) {
        var transaction models.Transaction
        if err := cursor.Decode(&transaction); err != nil {
            log.Println("Error decoding transaction:", err)
            c.JSON(http.StatusInternalServerError, gin.H{"error": "Error decoding transaction"})
            return
        }

        // Fetch the cashier name based on the cashier ID in the transaction
        var cashier models.Cashier
        err = config.CashierCollection.FindOne(ctx, bson.M{"_id": transaction.CashierID}).Decode(&cashier)
        if err != nil {
            if err == mongo.ErrNoDocuments {
                //log.Println("Cashier not found for ID:", cashierIDParam)
                transaction.CashierName = "Unknown"
            } else {
                //log.Println("Error retrieving cashier:", err)
                c.JSON(http.StatusInternalServerError, gin.H{"error": "Error retrieving cashier"})
                return
            }
        } else {
            transaction.CashierName = cashier.FirstName + " " + cashier.LastName
            //log.Println("Found cashier:", transaction.CashierName)
        }
        // Fetch the client name based on the card number in the transaction
        var client models.Client
        err = config.ClientCollection.FindOne(ctx, bson.M{"cardnumber": transaction.CardNumber}).Decode(&client)
        if err != nil {
            if err == mongo.ErrNoDocuments {
                //log.Println("Client not found for card number:", transaction.CardNumber)
                transaction.ClientName = "Unknown"
            } else {
                //log.Println("Error retrieving client:", err)
                c.JSON(http.StatusInternalServerError, gin.H{"error": "Error retrieving client"})
                return
            }
        } else {
            transaction.ClientName = client.FirstName + " " + client.LastName
            //log.Println("Found client:", transaction.ClientName)
        }

        report = append(report, map[string]interface{}{
            "cashiername": transaction.CashierName,
            "cardnumber": transaction.CardNumber,
            "type": transaction.Type,
            "purchase": transaction.Purchase,
            "date": transaction.Date,
            "clientname": transaction.ClientName,
            "sumsettle": transaction.Sumsettle,
            "location": transaction.Location,
        })

        //log.Println("Transaction added to report:", transaction)
    }
    if err := cursor.Err(); err != nil {
        log.Println("Error processing cursor:", err)
        c.JSON(http.StatusInternalServerError, gin.H{"error": "Error processing transactions"})
        return
    }

    log.Println("Successfully generated report:", report)
    c.JSON(http.StatusOK, report)
}

func RegisterClient(c *gin.Context) {
    var req models.Client
    if err := c.ShouldBindJSON(&req); err != nil {
        c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
        return
    }
    // Поиск и активация карты
    var card models.Card
    filter := bson.M{"cardnumber": req.CardNumber, "status": "Unused"}
    update := bson.M{"$set": bson.M{"status": "Used"}}
    err := config.CardCollection.FindOneAndUpdate(context.TODO(), filter, update).Decode(&card)
    if err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"error": "Card not found or already in use"})
        return
    }
    // Hash the client's password
    hashedPassword, err := utils.HashPassword(req.Password)
    if err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"error": "Error hashing password"})
        return
    }
    // Создание нового клиента
    newClient := models.Client{
        FirstName:         req.FirstName,
        LastName:          req.LastName,
        BirthDate:         req.BirthDate,
        Phone:             req.Phone,
        CardNumber: card.CardNumber,
        Role:              "client",
        Password: hashedPassword,
    }
    newClient.ID = primitive.NewObjectID()

    if _, err := config.ClientCollection.InsertOne(context.Background(), newClient); err != nil {
        // В случае ошибки возвращаем статус карты
        config.CardCollection.UpdateOne(context.TODO(), bson.M{"_id": card.ID}, bson.M{"$set": bson.M{"status": "Unused"}})
        c.JSON(http.StatusInternalServerError, gin.H{"error": "Error adding client"})
        return
    }

    c.JSON(http.StatusCreated, gin.H{"message": "Client registered successfully"})
}

func isPhoneNumberInUse(phone string) (bool, error) {
    var client models.Client
    var cashier models.Cashier
    var user models.User
    
    // Проверка в коллекции админов
    err := config.UserCollection.FindOne(context.TODO(), bson.M{"phone": phone}).Decode(&user)
    if err == nil {
        return true, nil
    } else if err != mongo.ErrNoDocuments {
        return false, err
    }

    // Проверка в коллекции клиентов
    err = config.ClientCollection.FindOne(context.TODO(), bson.M{"phone": phone}).Decode(&client)
    if err == nil {
        return true, nil
    } else if err != mongo.ErrNoDocuments {
        return false, err
    }

    // Проверка в коллекции кассиров
    err = config.CashierCollection.FindOne(context.TODO(), bson.M{"phone": phone}).Decode(&cashier)
    if err == nil {
        return true, nil
    } else if err != mongo.ErrNoDocuments {
        return false, err
    }

    return false, nil
}


func ReturnCustomerOrder(c *gin.Context) {
	orderID := c.Param("id")

	oid, err := primitive.ObjectIDFromHex(orderID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid order ID format"})
		return
	}

	var existingOrder models.CustomerOrder
	err = config.OrderCollection.FindOne(
		context.TODO(),
		bson.M{"_id": oid},
	).Decode(&existingOrder)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			c.JSON(http.StatusNotFound, gin.H{"error": "Order not found"})
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Error finding order"})
		}
		return
	}

	if existingOrder.Status == "Returned" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Order is already returned"})
		return
	}

	if existingOrder.PaymentMethod == "Peshraft" {
		if existingOrder.PeshraftTransactionID == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "No Peshraft transaction ID found in order"})
			return
		}

		refundAmount := existingOrder.TotalAmount
		success, refundResp, err := api.ProcessPeshraftRefund(existingOrder.PeshraftTransactionID, refundAmount)
		if err != nil || !success {
			c.JSON(http.StatusInternalServerError, gin.H{
				"error":   "Failed to process Peshraft refund",
				"details": err.Error(),
				"response": refundResp,
			})
			return
		}
	}

	for _, orderedProduct := range existingOrder.Products {
		var stockProduct models.Product
		err := config.ProductCollection.FindOne(
			context.TODO(),
			bson.M{"barcode": orderedProduct.Barcode},
		).Decode(&stockProduct)
		if err != nil {
			c.JSON(http.StatusNotFound, gin.H{
				"error":   "Product not found in stock while returning",
				"barcode": orderedProduct.Barcode,
			})
			return
		}

		quantities := stockProduct.Quantities       // []float64
		expirations := stockProduct.ExpirationDate  // []string

		for _, batchUsage := range orderedProduct.Batches {
			idx := -1
			for i, expDate := range expirations {
				if expDate == batchUsage.ExpirationDate {
					idx = i
					break
				}
			}

			if idx >= 0 {
				quantities[idx] += batchUsage.UsedQuantity
			} else {
				expirations = append(expirations, batchUsage.ExpirationDate)
				quantities = append(quantities, batchUsage.UsedQuantity)
			}
		}

		quantities, expirations = sortBatchesByExpirationFloat(quantities, expirations)

		_, err = config.ProductCollection.UpdateOne(
			context.TODO(),
			bson.M{"barcode": orderedProduct.Barcode},
			bson.M{
				"$set": bson.M{
					"quantities":     quantities,
					"expirationdate": expirations,
					"updated_at":     time.Now(),
				},
			},
		)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{
				"error":   "Failed to update stock in the database while returning",
				"barcode": orderedProduct.Barcode,
			})
			return
		}
	}

	_, err = config.OrderCollection.UpdateOne(
		context.TODO(),
		bson.M{"_id": oid},
		bson.M{
			"$set": bson.M{
				"status":     "Returned",
				"updated_at": time.Now(),
			},
		},
	)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update order status"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Order returned successfully"})
}


// sortBatchesByExpirationFloat сортирует партии по сроку годности с весами float64
func sortBatchesByExpirationFloat(quantities []float64, expirationDates []string) ([]float64, []string) {
    type batch struct {
        Quantity       float64
        ExpirationDate string
    }

    batches := make([]batch, len(quantities))
    for i := range quantities {
        batches[i] = batch{
            Quantity:       quantities[i],
            ExpirationDate: expirationDates[i],
        }
    }

    sort.Slice(batches, func(i, j int) bool {
        return batches[i].ExpirationDate < batches[j].ExpirationDate
    })

    sortedQuantities := make([]float64, len(batches))
    sortedExpirationDates := make([]string, len(batches))
    for i, b := range batches {
        sortedQuantities[i] = b.Quantity
        sortedExpirationDates[i] = b.ExpirationDate
    }

    return sortedQuantities, sortedExpirationDates
}


// Вспомогательная функция для выбора минимального значения
func min(a, b int64) int64 {
    if a < b {
        return a
    }
    return b
}


func ReturnCustomerOrderPartial(c *gin.Context) {
	orderID := c.Param("id")

	var refundRequest struct {
		Products []struct {
			Barcode string `json:"barcode"`
		} `json:"products"`
	}
	if err := c.ShouldBindJSON(&refundRequest); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid input"})
		return
	}

	oid, err := primitive.ObjectIDFromHex(orderID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid order ID"})
		return
	}

	var existingOrder models.CustomerOrder
	err = config.OrderCollection.FindOne(context.TODO(), bson.M{"_id": oid}).Decode(&existingOrder)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Order not found"})
		return
	}
    if existingOrder.Status == "Waiting for return to stock" || existingOrder.Status == "Returned" {
        c.JSON(http.StatusBadRequest, gin.H{"error": "Return already in process or completed"})
        return
    }
	var refundProducts []models.OrderedProduct
	var refundTotal float64

	for _, rProduct := range refundRequest.Products {
		for _, orderedProduct := range existingOrder.Products {
			if rProduct.Barcode == orderedProduct.Barcode {
				refundProducts = append(refundProducts, orderedProduct)
				refundTotal += orderedProduct.TotalPrice
				break
			}
		}
	}

	// Save refund record with pending status, do not refund yet
	refundDoc := models.CustomerOrderReturn{
		OriginalOrderID: existingOrder.ID,
		Products:        refundProducts,
		RefundAmount:    refundTotal,
		Status:          "Pending Stock Confirmation for return",
		CreatedAt:       time.Now(),
		PaymentMethod:   existingOrder.PaymentMethod,
		TransactionID:   existingOrder.PeshraftTransactionID,
	}
	_, err = config.OrderReturnCollection.InsertOne(context.TODO(), refundDoc)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to save refund record"})
		return
	}

	// Update original order (remove refunded products)
	var remainingProducts []models.OrderedProduct
	var remainingTotal float64
	for _, p := range existingOrder.Products {
		keep := true
		for _, r := range refundProducts {
			if p.Barcode == r.Barcode {
				keep = false
				break
			}
		}
		if keep {
			remainingProducts = append(remainingProducts, p)
			remainingTotal += p.TotalPrice
		}
	}
	_, err = config.OrderCollection.UpdateOne(
		context.TODO(),
		bson.M{"_id": oid},
		bson.M{"$set": bson.M{
			"products":     remainingProducts,
			"total_amount": remainingTotal,
            "status":       "Waiting for return to stock",
			"updated_at":   time.Now(),
		}},
	)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update original order"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Partial refund recorded, pending stock confirmation"})
}

func GetReturnOrderByID(c *gin.Context) {
    orderID := c.Param("id")

    oid, err := primitive.ObjectIDFromHex(orderID)
    if err != nil {
        c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid return order ID"})
        return
    }

    var returnOrder models.CustomerOrderReturn
    err = config.OrderReturnCollection.FindOne(context.TODO(), bson.M{"_id": oid}).Decode(&returnOrder)
    if err != nil {
        if err == mongo.ErrNoDocuments {
            c.JSON(http.StatusNotFound, gin.H{"error": "Return order not found"})
        } else {
            c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to retrieve return order"})
        }
        return
    }

    // Prepare product details with batch info
    type Batch struct {
        ExpirationDate string  `json:"expiration_date"`
        UsedQuantity   float64 `json:"used_quantity"`
    }

    type ExtendedProduct struct {
        Barcode          string  `json:"barcode"`
        Quantity         float64 `json:"quantity"`
        UnitPrice        float64 `json:"unit_price"`
        TotalPrice       float64 `json:"total_price"`
        Retailprice      float64 `json:"retailprice"`
        TotalRetailPrice float64 `json:"totalretailprice"`
        Unm              string  `json:"unm"`
        Batches          []Batch `json:"batches"`
    }

    var extendedProducts []ExtendedProduct
    for _, p := range returnOrder.Products {
        extendedProducts = append(extendedProducts, ExtendedProduct{
            Barcode:          p.Barcode,
            Quantity:         p.Quantity,
            UnitPrice:        p.UnitPrice,
            TotalPrice:       p.TotalPrice,
            Retailprice:      p.Retailprice,
            TotalRetailPrice: p.TotalRetailprice,
            Unm:              p.Unm,
            Batches: func() []Batch {
                var result []Batch
                for _, b := range p.Batches {
                    result = append(result, Batch{
                        ExpirationDate: b.ExpirationDate,
                        UsedQuantity:   b.UsedQuantity,
                    })
                }
                return result
            }(),
        })
    }

    // Construct response
    response := struct {
        ID             primitive.ObjectID `json:"id"`
        OriginalOrderID primitive.ObjectID `json:"originalorderid"`
        Products       []ExtendedProduct  `json:"products"`
        DeliveryMethod string             `json:"deliverymethod"`
        PaymentMethod  string             `json:"paymentmethod"`
        Status         string             `json:"status"`
        RefundAmount   float64            `json:"refundamount"`
        TotalAmount    float64            `json:"total_amount"`
        TransactionID  string             `json:"transactionid"`
        QRLink         string             `json:"qrlink"`
        ViewToken      string             `json:"view_token"`
        TranID         string             `json:"tranid"`
        ClientID       string             `json:"clientid"`
        CreatedAt      time.Time          `json:"created_at"`
    }{
        ID:              returnOrder.ID,
        OriginalOrderID: returnOrder.OriginalOrderID,
        Products:        extendedProducts,
        DeliveryMethod:  returnOrder.DeliveryMethod,
        PaymentMethod:   returnOrder.PaymentMethod,
        Status:          returnOrder.Status,
        RefundAmount:    returnOrder.RefundAmount,
        TotalAmount:     returnOrder.TotalAmount,
        TransactionID:   returnOrder.TransactionID,
        QRLink:          returnOrder.Qrlink,
        ViewToken:       returnOrder.ViewToken,
        TranID:          returnOrder.Tranid,
        ClientID:        returnOrder.Clientid,
        CreatedAt:       returnOrder.CreatedAt,
    }

    c.JSON(http.StatusOK, response)
}



func ConfirmReturnToStock(c *gin.Context) {
	returnID := c.Param("id")

	oid, err := primitive.ObjectIDFromHex(returnID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid return ID"})
		return
	}

	var returnDoc models.CustomerOrderReturn
	err = config.OrderReturnCollection.FindOne(context.TODO(), bson.M{"_id": oid}).Decode(&returnDoc)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Return document not found"})
		return
	}

	if returnDoc.Status != "Pending Stock Confirmation" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Return already processed or invalid status"})
		return
	}

	for _, product := range returnDoc.Products {
		var stock models.Product
		err = config.ProductCollection.FindOne(context.TODO(), bson.M{"barcode": product.Barcode}).Decode(&stock)
		if err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "Product not found in stock", "barcode": product.Barcode})
			return
		}

		q := stock.Quantities // []float64
		e := stock.ExpirationDate

		for _, batch := range product.Batches {
			found := false
			for i, exp := range e {
				if exp == batch.ExpirationDate {
					q[i] += batch.UsedQuantity // float64
					found = true
					break
				}
			}
			if !found {
				e = append(e, batch.ExpirationDate)
				q = append(q, batch.UsedQuantity)
			}
		}

		q, e = sortBatchesByExpirationFloat(q, e)

		_, err := config.ProductCollection.UpdateOne(
			context.TODO(),
			bson.M{"barcode": product.Barcode},
			bson.M{"$set": bson.M{
				"quantities":     q,
				"expirationdate": e,
				"updated_at":     time.Now(),
			}},
		)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update stock"})
			return
		}
	}

	if returnDoc.PaymentMethod == "Peshraft" && returnDoc.TransactionID != "" {
		success, resp, err := api.ProcessPeshraftRefund(returnDoc.TransactionID, returnDoc.RefundAmount)
		if err != nil || !success {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Refund failed", "details": err.Error(), "response": resp})
			return
		}
	}

	_, err = config.OrderReturnCollection.UpdateOne(
		context.TODO(),
		bson.M{"_id": oid},
		bson.M{"$set": bson.M{
			"status":       "Confirmed",
			"confirmed_at": time.Now(),
		}},
	)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update return status"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Return confirmed, refund processed"})
}

// func UpdateReturnOrderByID(c *gin.Context) {
//     orderID := c.Param("id")

//     oid, err := primitive.ObjectIDFromHex(orderID)
//     if err != nil {
//         c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid return order ID"})
//         return
//     }

//     var req UpdateReturnOrderRequest
//     if err := c.ShouldBindJSON(&req); err != nil {
//         c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request body"})
//         return
//     }

//     for _, p := range req.Products {
//         var product struct {
//             Quantities      []float64 `bson:"quantities"`
//             ExpirationDates []string  `bson:"expirationdate"`
//         }

//         filter := bson.M{"barcode": p.Barcode}
//         err := config.ProductsCollection.FindOne(context.TODO(), filter).Decode(&product)
//         if err != nil {
//             c.JSON(http.StatusNotFound, gin.H{"error": "Product not found: " + p.Barcode})
//             return
//         }

//         if len(product.Quantities) == 0 {
//             product.Quantities = []float64{0}
//         }

//         // Увеличиваем остаток товара
//         product.Quantities[0] += p.Quantity

//         // Обновляем срок годности, если необходимо
//         exists := false
//         for _, d := range product.ExpirationDates {
//             if d == p.ExpirationDate {
//                 exists = true
//                 break
//             }
//         }
//         if !exists {
//             product.ExpirationDates = append(product.ExpirationDates, p.ExpirationDate)
//         }

//         update := bson.M{"$set": bson.M{
//             "quantities":    product.Quantities,
//             "expirationdate": product.ExpirationDates,
//             "updated_at":     time.Now(),
//         }}

//         _, err = config.ProductsCollection.UpdateOne(context.TODO(), filter, update)
//         if err != nil {
//             c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update product stock or expiration date"})
//             return
//         }
//     }

//     updateFields := bson.M{
//         "status": "Возврат успешно принят и склад обновлен",
//         "updated_at": time.Now(),
//         "products": func() []bson.M {
//             var products []bson.M
//             for _, p := range req.Products {
//                 products = append(products, bson.M{
//                     "barcode": p.Barcode,
//                     "status":  p.Status,
//                 })
//             }
//             return products
//         }(),
//     }

//     filterOrder := bson.M{"_id": oid}

//     updateOrder := bson.M{"$set": updateFields}

//     result, err := config.OrderReturnCollection.UpdateOne(context.TODO(), filterOrder, updateOrder)
//     if err != nil {
//         c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update return order"})
//         return
//     }

//     if result.MatchedCount == 0 {
//         c.JSON(http.StatusNotFound, gin.H{"error": "Return order not found"})
//         return
//     }

//     c.JSON(http.StatusOK, gin.H{"message": "Return order updated, product stock and expiration dates handled successfully"})
// }


// path: handlers/dashboard.go
func Dashboard(c *gin.Context) {
    ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
    defer cancel()

    dailyStats, err := getLast30DaysOrderStats(ctx)
    if err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"error": "Error retrieving last 30 days statistics"})
        return
    }

    monthlyStats, err := getLast12MonthsOrderStats(ctx)
    if err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"error": "Error retrieving last 12 months statistics"})
        return
    }

    warehouseStats, err := calculateWarehouseStockValue(ctx)
    if err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"error": "Error calculating warehouse stock value"})
        return
    }

    writeOffStats, err := calculateTotalWriteOffValue(ctx)
    if err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"error": "Error calculating total write-off value"})
        return
    }

    totalPurchase := warehouseStats["totalPurchase"]
    totalWholesale := warehouseStats["totalWholesale"]
    totalRetail := warehouseStats["totalRetail"]
    totalWriteOff := writeOffStats["totalWriteOff"]

    c.JSON(http.StatusOK, gin.H{
        "totalPurchase":      totalPurchase,
        "totalWholesale":     totalWholesale,
        "totalRetail":        totalRetail,
        "writeoffsprice":      totalWriteOff,
        "last30DaysStats":    dailyStats,
        "last12MonthsStats":  monthlyStats,
    })
}




// path: stats/warehouse_value.go
func calculateWarehouseStockValue(ctx context.Context) (bson.M, error) {
    pipeline := mongo.Pipeline{
        bson.D{
            bson.E{Key: "$project", Value: bson.D{
                bson.E{Key: "quantities", Value: 1},
                bson.E{Key: "purchaseprice", Value: 1},
                bson.E{Key: "whosaleprice", Value: 1},
                bson.E{Key: "retailprice", Value: 1},
            }},
        },
        bson.D{
            bson.E{Key: "$unwind", Value: "$quantities"},
        },
        bson.D{
            bson.E{Key: "$project", Value: bson.D{
                bson.E{Key: "totalPurchaseValue", Value: bson.D{{Key: "$multiply", Value: bson.A{"$quantities", "$purchaseprice"}}}},
                bson.E{Key: "totalWholesaleValue", Value: bson.D{{Key: "$multiply", Value: bson.A{"$quantities", "$whosaleprice"}}}},
                bson.E{Key: "totalRetailValue", Value: bson.D{{Key: "$multiply", Value: bson.A{"$quantities", "$retailprice"}}}},
            }},
        },
        bson.D{
            bson.E{Key: "$group", Value: bson.D{
                bson.E{Key: "_id", Value: nil},
                bson.E{Key: "totalPurchase", Value: bson.D{{Key: "$sum", Value: "$totalPurchaseValue"}}},
                bson.E{Key: "totalWholesale", Value: bson.D{{Key: "$sum", Value: "$totalWholesaleValue"}}},
                bson.E{Key: "totalRetail", Value: bson.D{{Key: "$sum", Value: "$totalRetailValue"}}},
            }},
        },
        bson.D{
            bson.E{Key: "$project", Value: bson.D{
                bson.E{Key: "_id", Value: 0},
                bson.E{Key: "totalPurchase", Value: 1},
                bson.E{Key: "totalWholesale", Value: 1},
                bson.E{Key: "totalRetail", Value: 1},
            }},
        },
    }

    cursor, err := config.ProductCollection.Aggregate(ctx, pipeline)
    if err != nil {
        return nil, err
    }
    defer cursor.Close(ctx)

    var result []bson.M
    if err := cursor.All(ctx, &result); err != nil {
        return nil, err
    }

    if len(result) > 0 {
        return result[0], nil
    }
    return bson.M{"totalPurchase": 0, "totalWholesale": 0, "totalRetail": 0}, nil
}

func calculateTotalWriteOffValue(ctx context.Context) (bson.M, error) {
    pipeline := mongo.Pipeline{
        bson.D{{Key: "$match", Value: bson.D{{Key: "status", Value: "Списан"}}}},
        bson.D{
            {Key: "$group", Value: bson.D{
                {Key: "_id", Value: nil},
                {Key: "totalWriteOff", Value: bson.D{{Key: "$sum", Value: "$total_value"}}},
            }},
        },
        bson.D{
            {Key: "$project", Value: bson.D{
                {Key: "_id", Value: 0},
                {Key: "totalWriteOff", Value: 1},
            }},
        },
    }

    cursor, err := config.WriteOffCollection.Aggregate(ctx, pipeline)
    if err != nil {
        return nil, err
    }
    defer cursor.Close(ctx)

    var result []bson.M
    if err := cursor.All(ctx, &result); err != nil {
        return nil, err
    }

    if len(result) > 0 {
        return result[0], nil
    }
    return bson.M{"totalWriteOff": 0}, nil
}

func getLast30DaysOrderStats(ctx context.Context) ([]bson.M, error) {
    now := time.Now().UTC()
    thirtyDaysAgo := now.AddDate(0, 0, -30)

    pipeline := mongo.Pipeline{
        // 0) Преобразуем строку clientid в ObjectId
        bson.D{{
            Key: "$addFields",
            Value: bson.D{
                {Key: "clientObjId", Value: bson.D{
                    {Key: "$toObjectId", Value: "$clientid"},
                }},
            },
        }},
        // 1) Фильтрация по дате и статусу
        bson.D{{
            Key: "$match",
            Value: bson.D{
                {Key: "created_at", Value: bson.D{
                    {Key: "$gte", Value: thirtyDaysAgo},
                    {Key: "$lte", Value: now},
                }},
                {Key: "status", Value: "Заказ собран, можете забрать со склада."},
            },
        }},
        // 2) Подтягиваем данные клиента, теперь по полю clientObjId
        bson.D{{
            Key: "$lookup",
            Value: bson.D{
                {Key: "from",         Value: "clients"},
                {Key: "localField",   Value: "clientObjId"},
                {Key: "foreignField", Value: "_id"},
                {Key: "as",           Value: "client"},
            },
        }},
        bson.D{{Key: "$unwind", Value: "$client"}},
        // 3) Проекция нужных полей
        bson.D{{
            Key: "$project",
            Value: bson.D{
                {Key: "date",         Value: "$created_at"},
                {Key: "total_amount", Value: 1},
                {Key: "clientType",   Value: "$client.type"},
            },
        }},
        // 4) Группировка по дню
        bson.D{{
            Key: "$group",
            Value: bson.D{
                {Key: "_id", Value: bson.D{{
                    Key: "day", Value: bson.D{{
                        Key: "$dateToString",
                        Value: bson.D{
                            {Key: "format", Value: "%Y-%m-%d"},
                            {Key: "date",   Value: "$date"},
                        },
                    }},
                }}},
                {Key: "totalCount",     Value: bson.D{{Key: "$sum", Value: 1}}},
                {Key: "retailCount",    Value: bson.D{{Key: "$sum", Value: bson.D{{
                    Key: "$cond",
                    Value: bson.A{
                        bson.D{{Key: "$eq", Value: bson.A{"$clientType", "retail"}}},
                        1,
                        0,
                    },
                }}}}},
                {Key: "whosaleCount", Value: bson.D{{Key: "$sum", Value: bson.D{{
                    Key: "$cond",
                    Value: bson.A{
                        bson.D{{Key: "$eq", Value: bson.A{"$clientType", "whosale"}}},
                        1,
                        0,
                    },
                }}}}},
                {Key: "totalAmount",    Value: bson.D{{Key: "$sum", Value: "$total_amount"}}},
                {Key: "retailAmount",   Value: bson.D{{Key: "$sum", Value: bson.D{{
                    Key: "$cond",
                    Value: bson.A{
                        bson.D{{Key: "$eq", Value: bson.A{"$clientType", "retail"}}},
                        "$total_amount",
                        0,
                    },
                }}}}},
                {Key: "whosaleAmount",Value: bson.D{{Key: "$sum", Value: bson.D{{
                    Key: "$cond",
                    Value: bson.A{
                        bson.D{{Key: "$eq", Value: bson.A{"$clientType", "whosale"}}},
                        "$total_amount",
                        0,
                    },
                }}}}},
            },
        }},
        // 5) Собираем вложенные документы count и amount
        bson.D{{
            Key: "$project",
            Value: bson.D{
                {
                    Key: "count", Value: bson.D{
                        {Key: "total",     Value: "$totalCount"},
                        {Key: "retail",    Value: "$retailCount"},
                        {Key: "wholesale", Value: "$whosaleCount"},
                    },
                },
                {
                    Key: "amount", Value: bson.D{
                        {Key: "total",     Value: "$totalAmount"},
                        {Key: "retail",    Value: "$retailAmount"},
                        {Key: "wholesale", Value: "$whosaleAmount"},
                    },
                },
            },
        }},
        // 6) Сортировка по дате
        bson.D{{Key: "$sort", Value: bson.D{{Key: "_id.day", Value: 1}}}},
    }

    cursor, err := config.OrderCollection.Aggregate(ctx, pipeline)
    if err != nil {
        return nil, err
    }
    defer cursor.Close(ctx)

    var dailyStats []bson.M
    if err := cursor.All(ctx, &dailyStats); err != nil {
        return nil, err
    }

    return dailyStats, nil
}


func getLast12MonthsOrderStats(ctx context.Context) ([]bson.M, error) {
    now := time.Now().UTC()
    firstOfThisMonth := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, time.UTC)
    start := firstOfThisMonth.AddDate(0, -11, 0)

    pipeline := mongo.Pipeline{
        // 0) Преобразуем clientid → ObjectId
        bson.D{{
            Key: "$addFields",
            Value: bson.D{
                {Key: "clientObjId", Value: bson.D{
                    {Key: "$toObjectId", Value: "$clientid"},
                }},
            },
        }},
        // 1) Фильтрация по периоду и статусу
        bson.D{{
            Key: "$match",
            Value: bson.D{
                {Key: "created_at", Value: bson.D{
                    {Key: "$gte", Value: start},
                    {Key: "$lte", Value: now},
                }},
                {Key: "status", Value: "Заказ собран, можете забрать со склада."},
            },
        }},
        // 2) Lookup по clientObjId
        bson.D{{
            Key: "$lookup",
            Value: bson.D{
                {Key: "from",         Value: "clients"},
                {Key: "localField",   Value: "clientObjId"},
                {Key: "foreignField", Value: "_id"},
                {Key: "as",           Value: "client"},
            },
        }},
        bson.D{{Key: "$unwind", Value: "$client"}},
        // 3) Проекция
        bson.D{{
            Key: "$project",
            Value: bson.D{
                {Key: "date",         Value: "$created_at"},
                {Key: "total_amount", Value: 1},
                {Key: "clientType",   Value: "$client.type"},
            },
        }},
        // 4) Группировка по месяцу
        bson.D{{
            Key: "$group",
            Value: bson.D{
                {Key: "_id", Value: bson.D{{
                    Key: "month", Value: bson.D{{
                        Key: "$dateToString",
                        Value: bson.D{
                            {Key: "format", Value: "%Y-%m"},
                            {Key: "date",   Value: "$date"},
                        },
                    }},
                }}},
                {Key: "totalCount",     Value: bson.D{{Key: "$sum", Value: 1}}},
                {Key: "retailCount",    Value: bson.D{{Key: "$sum", Value: bson.D{{
                    Key: "$cond",
                    Value: bson.A{
                        bson.D{{Key: "$eq", Value: bson.A{"$clientType", "retail"}}},
                        1,
                        0,
                    },
                }}}}},
                {Key: "whosaleCount", Value: bson.D{{Key: "$sum", Value: bson.D{{
                    Key: "$cond",
                    Value: bson.A{
                        bson.D{{Key: "$eq", Value: bson.A{"$clientType", "whosale"}}},
                        1,
                        0,
                    },
                }}}}},
                {Key: "totalAmount",    Value: bson.D{{Key: "$sum", Value: "$total_amount"}}},
                {Key: "retailAmount",   Value: bson.D{{Key: "$sum", Value: bson.D{{
                    Key: "$cond",
                    Value: bson.A{
                        bson.D{{Key: "$eq", Value: bson.A{"$clientType", "retail"}}},
                        "$total_amount",
                        0,
                    },
                }}}}},
                {Key: "whosaleAmount",Value: bson.D{{Key: "$sum", Value: bson.D{{
                    Key: "$cond",
                    Value: bson.A{
                        bson.D{{Key: "$eq", Value: bson.A{"$clientType", "whosale"}}},
                        "$total_amount",
                        0,
                    },
                }}}}},
            },
        }},
        // 5) Сборка count/amount
        bson.D{{
            Key: "$project",
            Value: bson.D{
                {
                    Key: "count", Value: bson.D{
                        {Key: "total",     Value: "$totalCount"},
                        {Key: "retail",    Value: "$retailCount"},
                        {Key: "wholesale", Value: "$whosaleCount"},
                    },
                },
                {
                    Key: "amount", Value: bson.D{
                        {Key: "total",     Value: "$totalAmount"},
                        {Key: "retail",    Value: "$retailAmount"},
                        {Key: "wholesale", Value: "$whosaleAmount"},
                    },
                },
            },
        }},
        // 6) Сортировка по месяцу
        bson.D{{Key: "$sort", Value: bson.D{{Key: "_id.month", Value: 1}}}},
    }

    cursor, err := config.OrderCollection.Aggregate(ctx, pipeline)
    if err != nil {
        return nil, err
    }
    defer cursor.Close(ctx)

    var monthlyStats []bson.M
    if err := cursor.All(ctx, &monthlyStats); err != nil {
        return nil, err
    }

    return monthlyStats, nil
}



func WriteOffProducts(c *gin.Context) {
	type WriteOffProduct struct {
		Barcode      string  `json:"barcode" binding:"required"`
		Quantity     float64 `json:"qua" binding:"required"`
		SellingPrice float64 `json:"sellingprice"`
		Comment      string  `json:"comment"`
	}

	var input []WriteOffProduct
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	var writeOffItems []models.WriteOffItem
	totalWriteOffValue := 0.0

	for _, item := range input {
		var product models.Product
		err := config.ProductCollection.FindOne(context.TODO(), bson.M{"barcode": item.Barcode}).Decode(&product)
		if err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "Product not found", "barcode": item.Barcode})
			return
		}

		totalStock := sumQuantitiesFloat(product.Quantities)
		if totalStock < item.Quantity {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Not enough stock", "barcode": item.Barcode, "available": totalStock})
			return
		}

		sortedQuantities, sortedExpDates := sortBatchesByExpirationFloat(product.Quantities, product.ExpirationDate)

		remaining := item.Quantity
		usedBatches := []models.BatchUsage{}
		for i := 0; i < len(sortedQuantities) && remaining > 0; i++ {
			if sortedQuantities[i] > 0 {
				used := minFloat(sortedQuantities[i], remaining)
				sortedQuantities[i] -= used
				remaining -= used
				usedBatches = append(usedBatches, models.BatchUsage{
					ExpirationDate: sortedExpDates[i],
					UsedQuantity:   used,
				})
			}
		}

		if remaining > 0 {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update stock", "barcode": item.Barcode})
			return
		}

		_, err = config.ProductCollection.UpdateOne(context.TODO(), bson.M{"barcode": item.Barcode}, bson.M{
			"$set": bson.M{
				"quantities":     sortedQuantities,
				"expirationdate": sortedExpDates,
				"updated_at":     time.Now(),
			},
		})
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update stock DB", "barcode": item.Barcode})
			return
		}

		purchasePrice := product.Purchaseprice
		writeOffValue := math.Round(item.Quantity * purchasePrice * 100) / 100
		totalWriteOffValue += writeOffValue

		remainingStock := sumQuantitiesFloat(sortedQuantities)

		writeOffItems = append(writeOffItems, models.WriteOffItem{
			Barcode:        item.Barcode,
			Quantity:       item.Quantity,
			PurchasePrice:  purchasePrice,
			WriteOffValue:  writeOffValue,
			Comment:        item.Comment,
			Batches:        usedBatches,
			Status:         "Списан",
			RemainingStock: remainingStock,
		})
	}

	doc := models.WriteOffDocument{
		ID:         primitive.NewObjectID(),
		Products:   writeOffItems,
		TotalValue: totalWriteOffValue,
		CreatedAt:  time.Now(),
	}

	_, err := config.WriteOffCollection.InsertOne(context.TODO(), doc)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create write-off document"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Write-off completed", "doc_id": doc.ID})
}

// Вспомогательная функция для выбора минимального значения
func minFloat(a, b float64) float64 {
    if a < b {
        return a
    }
    return b
}


