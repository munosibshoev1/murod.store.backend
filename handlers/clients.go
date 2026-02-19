package handlers

import (
	"bytes"
	"context"
	"encoding/json"

	"fmt"
	"io"
	"log"
	"math"
	"net/http"

	// "sort"

	"strings"
	"time"

	// "github.com/google/uuid"

	"backend/config"
	"backend/controllers"
	"backend/models"
	"backend/utils"

	"github.com/gin-gonic/gin"
	// "github.com/google/uuid"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

func GetTransactionsByCard(c *gin.Context) {
	cardNumber := c.Param("cardNumber")
	var transactions []models.Transaction

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	cursor, err := config.TransactionCollection.Find(ctx, bson.M{"cardnumber": cardNumber})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Error retrieving transactions"})
		return
	}
	defer cursor.Close(ctx)

	if err = cursor.All(ctx, &transactions); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Error decoding transactions"})
		return
	}

	c.JSON(http.StatusOK, transactions)
}

func GetTransactionsByCardClient(c *gin.Context) {
	clientID, exists := c.Get("clientID")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	// Convert clientID to ObjectID
	clientObjectID, err := primitive.ObjectIDFromHex(clientID.(string))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid client ID"})
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Find the client details
	var client models.Client
	err = config.ClientCollection.FindOne(ctx, bson.M{"_id": clientObjectID}).Decode(&client)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			c.JSON(http.StatusNotFound, gin.H{"error": "Client not found"})
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Error retrieving client"})
		}
		return
	}

	// Find the card details
	var card models.Card
	err = config.CardCollection.FindOne(ctx, bson.M{"cardnumber": client.CardNumber}).Decode(&card)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			c.JSON(http.StatusNotFound, gin.H{"error": "Card not found"})
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Error retrieving card"})
		}
		return
	}

	// Find the transaction history
	var transactions []models.Transaction
	cursor, err := config.TransactionCollection.Find(ctx, bson.M{"cardnumber": card.CardNumber})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Error retrieving transactions"})
		return
	}
	defer cursor.Close(ctx)

	if err = cursor.All(ctx, &transactions); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Error decoding transactions"})
		return
	}

	// Create the response
	response := map[string]interface{}{
		"client_info": map[string]interface{}{
			"id":            card.ID,
			"cardnumber":    card.CardNumber,
			"status":        card.Status,
			"createdate":    card.CreateDate,
			"totalfast":     card.TotalFast,
			"totalout":      card.TotalOut,
			"totalloan":     card.TotalLoan,
			"limit":         card.Limit,
			"days":          card.Days,
			"totalpurchase": card.TotalPurchase,
			"fullname":      client.FirstName + " " + client.LastName,
		},
		"transaction_history": transactions,
	}

	c.JSON(http.StatusOK, response)
}

func GetClientInfo(c *gin.Context) {
	clientID, exists := c.Get("clientID")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	clientObjectID, err := primitive.ObjectIDFromHex(clientID.(string))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid client ID"})
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	var client models.Client
	err = config.ClientCollection.FindOne(ctx, bson.M{"_id": clientObjectID}).Decode(&client)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			c.JSON(http.StatusNotFound, gin.H{"error": "Client not found"})
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Error retrieving client"})
		}
		return
	}

	// Генерация правильного пути для photo_url
	var photoURL string
	if client.Photo_url != "" {
		// Используем только "/uploads" без повторений
		photoURL = fmt.Sprintf(client.Photo_url)
	}

	clientMap := map[string]interface{}{
		"email":     client.Email,
		"lastname":  client.LastName,
		"firstname": client.FirstName,
		"birthdate": client.BirthDate,
		"phone":     client.Phone,
		"gender":    client.Gender,
		"photo_url": photoURL, // URL фото для фронтенда
	}

	c.JSON(http.StatusOK, clientMap)
}

func UpdateClientInfo(c *gin.Context) {
	clientID, exists := c.Get("clientID")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	clientObjectID, err := primitive.ObjectIDFromHex(clientID.(string))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid client ID"})
		return
	}

	// Initialize updateFields map
	updateFields := make(map[string]interface{})

	// Manually retrieve fields from form-data
	updateFields["first_name"] = c.PostForm("firstname")
	updateFields["last_name"] = c.PostForm("lastname")
	updateFields["birth_date"] = c.PostForm("birthdate")
	updateFields["phone"] = c.PostForm("phone")
	updateFields["email"] = c.PostForm("email")
	updateFields["gender"] = c.PostForm("gender")

	if password := c.PostForm("password"); password != "" {
		hashedPassword, err := utils.HashPassword(password)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Error hashing password"})
			return
		}
		updateFields["password"] = hashedPassword
	}

	// Handle the photo file upload
	file, err := c.FormFile("photo_url")
	var photoPath string
	if err == nil {
		fmt.Printf("Received photo file: %s (size: %d bytes)\n", file.Filename, file.Size)

		// Save photo using SaveAvatar function
		photoPath, err = controllers.SaveAvatar(c, file, clientObjectID.Hex())
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Error saving photo"})
			return
		}
		updateFields["photo_url"] = photoPath
	} else {
		fmt.Println("No photo file received")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Prepare the MongoDB update document
	update := bson.M{"$set": updateFields}
	result, err := config.ClientCollection.UpdateOne(
		ctx,
		bson.M{"_id": clientObjectID},
		update,
		options.Update().SetUpsert(false),
	)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Error updating client information"})
		return
	}

	if result.MatchedCount == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "Client not found"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Client information updated successfully", "photo_url": photoPath})
}

// GetCategoryDetails - получение подкатегорий и продуктов по категории
func GetCategoryDetails(c *gin.Context) {
	categoryID := c.Param("id")

	// Проверяем валидность ObjectID
	objID, err := primitive.ObjectIDFromHex(categoryID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid category ID"})
		return
	}

	// Получаем основную категорию
	var mainCategory models.Category
	err = config.CategoryCollection.FindOne(context.TODO(), bson.M{"_id": objID}).Decode(&mainCategory)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Category not found"})
		return
	}

	// Список ID категорий для поиска продуктов
	categoryIDs := []string{mainCategory.CategoryID}

	// Рекурсивный поиск всех подкатегорий
	subcategories := []models.Category{}
	findSubcategories(mainCategory.CategoryID, &subcategories, &categoryIDs)

	// Поиск продуктов по всем категориям
	products := []models.Product{}
	cursor, err := config.ProductCollection.Find(context.TODO(), bson.M{"categoryid": bson.M{"$in": categoryIDs}})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch products"})
		return
	}
	defer cursor.Close(context.TODO())

	if err = cursor.All(context.TODO(), &products); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to decode products"})
		return
	}

	// Формируем ответ
	c.JSON(http.StatusOK, gin.H{
		//"category":     mainCategory,
		"subcategories": subcategories,
		"products":      products,
	})
}

// findSubcategories - рекурсивный поиск подкатегорий
func findSubcategories(topCategoryID string, subcategories *[]models.Category, categoryIDs *[]string) {
	cursor, err := config.CategoryCollection.Find(context.TODO(), bson.M{"topcategoryid": topCategoryID})
	if err != nil {
		return
	}
	defer cursor.Close(context.TODO())

	tempCategories := []models.Category{}
	if err = cursor.All(context.TODO(), &tempCategories); err != nil {
		return
	}

	// Добавляем найденные подкатегории и их ID
	for _, category := range tempCategories {
		*subcategories = append(*subcategories, category)
		*categoryIDs = append(*categoryIDs, category.CategoryID)

		// Рекурсивный вызов для поиска подкатегорий текущей категории
		findSubcategories(category.CategoryID, subcategories, categoryIDs)
	}
}

func GetCardLimit(c *gin.Context) {
	// Получение номера карты из параметра пути
	cardNumber := c.Param("cardnumber")

	if cardNumber == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Card number is required"})
		return
	}

	// Структура для хранения данных карты
	var cardData struct {
		Limit float64 `bson:"limit"`
	}

	// Поиск карты в коллекции cards базы данных peshraft
	err := config.PeshraftCollection.FindOne(
		context.TODO(),
		bson.M{"cardnumber": cardNumber},
	).Decode(&cardData)

	if err != nil {
		if err == mongo.ErrNoDocuments {
			c.JSON(http.StatusNotFound, gin.H{"error": "Card not found"})
		} else {
			log.Printf("Error retrieving card data: %v", err) // Логирование ошибки
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Error retrieving card data"})
		}
		return
	}

	// Возвращаем поле limit
	c.JSON(http.StatusOK, gin.H{"limit": cardData.Limit})
}

// RemovePlusFromPhone удаляет "+" из номера телефона
func removePlusFromPhone(phone string) string {
	return strings.TrimPrefix(phone, "+")
}

// sumQuantitiesFloat - функция для суммирования массива количеств (вес, дробные значения)
func sumQuantitiesFloat(quantities []float64) float64 {
	var total float64
	for _, quantity := range quantities {
		total += quantity
	}
	return math.Round(total*100) / 100
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

func GetOrderByToken(c *gin.Context) {
	token := c.Param("token")

	var order models.CustomerOrder
	err := config.OrderCollection.FindOne(context.TODO(), bson.M{"view_token": token}).Decode(&order)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			c.JSON(http.StatusNotFound, gin.H{"error": "Order not found"})
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to retrieve order"})
		}
		return
	}

	var client struct {
		FirstName string `bson:"first_name"`
		LastName  string `bson:"last_name"`
	}

	clID, err := primitive.ObjectIDFromHex(order.Clientid)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid client ID format"})
		return
	}

	err = config.ClientCollection.FindOne(context.TODO(), bson.M{"_id": clID}).Decode(&client)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			c.JSON(http.StatusNotFound, gin.H{"error": "Client not found"})
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to retrieve client details"})
		}
		return
	}

	fullName := fmt.Sprintf("%s %s", client.FirstName, client.LastName)

	type Batch struct {
		ExpirationDate string  `json:"expiration_date"`
		UsedQuantity   float64 `json:"used_quantity"`
	}

	type ExtendedProduct struct {
		Barcode          string  `json:"barcode"`
		Quantity         float64 `json:"quantity"`
		UnitPrice        float64 `json:"unit_price"`
		TotalPrice       float64 `json:"total_price"`
		TotalRetailPrice float64 `json:"totalretailprice"`
		Retailprice      float64 `json:"retailprice"`
		Name             string  `json:"name"`
		Unm              string  `json:"unm"`
		ProductPhotoURL  string  `json:"productphotourl"`
		Batches          []Batch `json:"batches"`
	}

	var extendedProducts []ExtendedProduct

	for _, orderedProduct := range order.Products {
		var product models.Product
		err := config.ProductCollection.FindOne(context.TODO(), bson.M{"barcode": orderedProduct.Barcode}).Decode(&product)

		batches := make([]Batch, 0, len(orderedProduct.Batches))
		for _, b := range orderedProduct.Batches {
			batches = append(batches, Batch{
				ExpirationDate: b.ExpirationDate,
				UsedQuantity:   b.UsedQuantity,
			})
		}

		if err != nil {
			if err == mongo.ErrNoDocuments {
				extendedProducts = append(extendedProducts, ExtendedProduct{
					Barcode:          orderedProduct.Barcode,
					Quantity:         orderedProduct.Quantity,
					UnitPrice:        orderedProduct.UnitPrice,
					TotalPrice:       orderedProduct.TotalPrice,
					Retailprice:      orderedProduct.Retailprice,
					TotalRetailPrice: orderedProduct.TotalRetailprice,
					Name:             "Unknown",
					Unm:              "Unknown",
					Batches:          batches,
				})
				continue
			}
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to retrieve product details"})
			return
		}

		extendedProducts = append(extendedProducts, ExtendedProduct{
			Barcode:          orderedProduct.Barcode,
			Quantity:         orderedProduct.Quantity,
			UnitPrice:        orderedProduct.UnitPrice,
			TotalPrice:       orderedProduct.TotalPrice,
			Retailprice:      orderedProduct.Retailprice,
			TotalRetailPrice: orderedProduct.TotalRetailprice,
			Name:             product.Name,
			Unm:              product.Unm,
			ProductPhotoURL:  product.Productphotourl,
			Batches:          batches,
		})
	}

	extendedOrder := struct {
		ID              primitive.ObjectID `json:"id"`
		Products        []ExtendedProduct  `json:"products"`
		DeliveryMethod  string             `json:"deliverymethod"`
		DeliveryAddress string             `json:"deliveryaddress"`
		PaymentMethod   string             `json:"paymentmethod"`
		Status          string             `json:"status"`
		TotalAmount     float64            `json:"total_amount"`
		TranID          string             `json:"tranid"`
		FullName        string             `json:"fullname"`
		CreatedAt       time.Time          `json:"created_at"`
	}{
		ID:              order.ID,
		Products:        extendedProducts,
		DeliveryMethod:  order.DeliveryMethod,
		DeliveryAddress: order.DeliveryAddress,
		PaymentMethod:   order.PaymentMethod,
		Status:          order.Status,
		TotalAmount:     order.TotalAmount,
		TranID:          order.Tranid,
		FullName:        fullName,
		CreatedAt:       order.CreatedAt,
	}

	c.JSON(http.StatusOK, extendedOrder)
}

// trimBatchUsageToQuantity returns slice of batches adjusted to sum exactly qty
func trimBatchUsageToQuantity(batches []models.BatchUsage, qty float64) []models.BatchUsage {
	if qty <= 0 {
		return nil
	}
	out := make([]models.BatchUsage, 0, len(batches))
	left := qty
	for _, b := range batches {
		if left <= 0 {
			break
		}
		use := math.Min(b.UsedQuantity, left)
		if use > 0 {
			out = append(out, models.BatchUsage{ExpirationDate: b.ExpirationDate, UsedQuantity: use})
			left -= use
		}
	}
	return out
}

// subtractBatchUsage subtracts "returned" from "existing" per expiration date
func subtractBatchUsage(existing, returned []models.BatchUsage) []models.BatchUsage {
	if len(returned) == 0 {
		return existing
	}
	m := make(map[string]float64, len(returned))
	for _, r := range returned {
		m[r.ExpirationDate] += r.UsedQuantity
	}
	res := make([]models.BatchUsage, 0, len(existing))
	for _, e := range existing {
		ret := m[e.ExpirationDate]
		left := e.UsedQuantity - ret
		if left > 1e-9 { // keep only positive remainders
			res = append(res, models.BatchUsage{ExpirationDate: e.ExpirationDate, UsedQuantity: left})
		}
	}
	return res
}

func restoreStock(barcode string, batches []models.BatchUsage) {
	filter := bson.M{"barcode": barcode}
	var product models.Product
	err := config.ProductCollection.FindOne(context.TODO(), filter).Decode(&product)
	if err != nil {
		return
	}
	for _, batch := range batches {
		for i, exp := range product.ExpirationDate {
			if exp == batch.ExpirationDate {
				product.Quantities[i] += batch.UsedQuantity
			}
		}
	}
	update := bson.M{"$set": bson.M{"quantities": product.Quantities, "expirationdate": product.ExpirationDate, "updated_at": time.Now()}}
	config.ProductCollection.UpdateOne(context.TODO(), filter, update)
}

func restoreStockPartial(barcode string, quantity float64) {
	filter := bson.M{"barcode": barcode}
	var product models.Product
	err := config.ProductCollection.FindOne(context.TODO(), filter).Decode(&product)
	if err != nil {
		return
	}
	for i := range product.Quantities {
		product.Quantities[i] += quantity
		break
	}
	update := bson.M{"$set": bson.M{"quantities": product.Quantities, "expirationdate": product.ExpirationDate, "updated_at": time.Now()}}
	config.ProductCollection.UpdateOne(context.TODO(), filter, update)
}

func deductFromStock(barcode string, quantity float64) error {
	filter := bson.M{"barcode": barcode}
	var product models.Product
	err := config.ProductCollection.FindOne(context.TODO(), filter).Decode(&product)
	if err != nil {
		return fmt.Errorf("Product not found")
	}
	totalStock := 0.0
	for _, q := range product.Quantities {
		totalStock += q
	}
	if totalStock < quantity {
		return fmt.Errorf("Not enough stock available")
	}
	remaining := quantity
	for i := 0; i < len(product.Quantities) && remaining > 0; i++ {
		if product.Quantities[i] >= remaining {
			product.Quantities[i] -= remaining
			remaining = 0
		} else {
			remaining -= product.Quantities[i]
			product.Quantities[i] = 0
		}
	}
	update := bson.M{"$set": bson.M{"quantities": product.Quantities, "expirationdate": product.ExpirationDate, "updated_at": time.Now()}}
	_, err = config.ProductCollection.UpdateOne(context.TODO(), filter, update)
	if err != nil {
		return fmt.Errorf("Failed to update product stock")
	}
	return nil
}

// Merge same-expiration batches to avoid duplicates in order doc.
func mergeBatchUsage(batches []models.BatchUsage) []models.BatchUsage {
	if len(batches) <= 1 {
		return batches
	}
	acc := make(map[string]float64)
	order := make([]string, 0, len(batches))
	seen := make(map[string]bool)
	for _, b := range batches {
		if b.UsedQuantity <= 0 {
			continue
		}
		acc[b.ExpirationDate] += b.UsedQuantity
		if !seen[b.ExpirationDate] {
			order = append(order, b.ExpirationDate)
			seen[b.ExpirationDate] = true
		}
	}
	out := make([]models.BatchUsage, 0, len(order))
	for _, d := range order {
		out = append(out, models.BatchUsage{ExpirationDate: d, UsedQuantity: acc[d]})
	}
	return out
}

// ----------------------------- Stock operations (FIFO) -----------------------------

func DistributeByBatches(barcode string, quantity float64) ([]models.BatchUsage, []float64, []string, float64, error) {
	var product models.Product
	if err := config.ProductCollection.FindOne(context.TODO(), bson.M{"barcode": barcode}).Decode(&product); err != nil {
		return nil, nil, nil, 0, fmt.Errorf("product not found: %v", err)
	}
	q, d := sortBatchesByExpirationFloat(product.Quantities, product.ExpirationDate)
	remaining := quantity
	used := make([]models.BatchUsage, 0, len(q))
	for i := 0; i < len(q) && remaining > 0; i++ {
		if q[i] <= 0 {
			continue
		}
		use := minFloat(q[i], remaining)
		q[i] -= use
		remaining -= use
		used = append(used, models.BatchUsage{ExpirationDate: d[i], UsedQuantity: use})
	}
	if remaining > 0 {
		return nil, nil, nil, 0, fmt.Errorf("not enough stock for barcode %s", barcode)
	}
	left := sumQuantitiesFloat(q)
	return used, q, d, left, nil
}

func ReturnToStockByBatches(barcode string, batches []models.BatchUsage) ([]float64, []string, error) {
	var product models.Product
	if err := config.ProductCollection.FindOne(context.TODO(), bson.M{"barcode": barcode}).Decode(&product); err != nil {
		return nil, nil, err
	}
	quantities := append([]float64(nil), product.Quantities...)
	expDates := append([]string(nil), product.ExpirationDate...)
	if len(quantities) != len(expDates) {
		return nil, nil, fmt.Errorf("stock data inconsistency")
	}
	idx := make(map[string]int, len(expDates))
	for i, date := range expDates {
		idx[date] = i
	}
	for _, b := range batches {
		if b.UsedQuantity <= 0 {
			continue
		}
		if i, ok := idx[b.ExpirationDate]; ok {
			quantities[i] += b.UsedQuantity
		} else {
			expDates = append(expDates, b.ExpirationDate)
			quantities = append(quantities, b.UsedQuantity)
			idx[b.ExpirationDate] = len(expDates) - 1
		}
	}
	if _, err := config.ProductCollection.UpdateOne(
		context.TODO(),
		bson.M{"barcode": barcode},
		bson.M{"$set": bson.M{"quantities": quantities, "expirationdate": expDates, "updated_at": time.Now()}},
	); err != nil {
		return nil, nil, fmt.Errorf("failed to update stock: %v", err)
	}
	return quantities, expDates, nil
}

func AdminUpdateCustomerOrder(c *gin.Context) {
	// frontend sends NEW quantity in MinimumOrder; Quantity is OLD one
	type Batch struct {
		ExpirationDate string  `json:"expiration_date"`
		UsedQuantity   float64 `json:"used_quantity"`
	}
	type UpdateOrderProduct struct {
		Barcode          string  `json:"barcode" binding:"required"`
		Quantity         float64 `json:"quantity" binding:"required"`
		MinimumOrder     float64 `json:"minimumorder"`
		UnitPrice        float64 `json:"unit_price"`
		TotalPrice       float64 `json:"total_price"`
		TotalRetailPrice float64 `json:"totalretailprice"`
		RetailPrice      float64 `json:"retailprice"`
		Name             string  `json:"name"`
		Unm              string  `json:"unm"`
		ProductPhotoURL  string  `json:"productphotourl"`
		Batches          []Batch `json:"batches"`
	}
	type UpdateOrderRequest struct {
		ID              string               `json:"id" binding:"required"`
		Products        []UpdateOrderProduct `json:"products" binding:"required"`
		DeliveryMethod  string               `json:"deliverymethod"`
		DeliveryAddress string               `json:"deliveryaddress"`
		DeliveryCost    float64              `json:"deliverycost"`
	}

	var req UpdateOrderRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	seen := map[string]bool{}
	for _, p := range req.Products {
		if seen[p.Barcode] {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Duplicate product", "barcode": p.Barcode})
			return
		}
		seen[p.Barcode] = true
	}

	orderID, err := primitive.ObjectIDFromHex(req.ID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid order ID"})
		return
	}

	var existingOrder models.CustomerOrder
	if err := config.OrderCollection.FindOne(context.TODO(), bson.M{"_id": orderID}).Decode(&existingOrder); err != nil {
		if err == mongo.ErrNoDocuments {
			c.JSON(http.StatusNotFound, gin.H{"error": "Order not found"})
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to retrieve order"})
		}
		return
	}
	if existingOrder.Status == "Заказ собран, можете забрать со склада." {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Order cannot be edited after being ready for pickup."})
		return
	}

	existingMap := map[string]models.OrderedProduct{}
	for _, p := range existingOrder.Products {
		existingMap[p.Barcode] = p
	}

	auditLog := []string{}
	updatedProducts := []models.OrderedProduct{}
	totalAmount := 0.0
	addedDelta := 0.0
	refundDelta := 0.0

	// обработка входящих товаров
	for _, upd := range req.Products {
		var product models.Product
		if err := config.ProductCollection.FindOne(context.TODO(), bson.M{"barcode": upd.Barcode}).Decode(&product); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Product not found", "barcode": upd.Barcode})
			return
		}
		ex := existingMap[upd.Barcode]
		oldQty := ex.Quantity
		newQty := upd.MinimumOrder
		diff := newQty - oldQty

		var newStock []float64
		var newExp []string
		var resulting []models.BatchUsage

		if diff > 0 {
			used, stock, expir, _, err := DistributeByBatches(upd.Barcode, diff)
			if err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
				return
			}
			newStock, newExp = stock, expir
			resulting = mergeBatchUsage(append(ex.Batches, used...))
			auditLog = append(auditLog, fmt.Sprintf("Добавлено %.2f к %s", diff, upd.Barcode))
			addedDelta += diff * upd.UnitPrice
		} else if diff < 0 {
			bret := trimBatchUsageToQuantity(ex.Batches, -diff)
			stock, expir, err := ReturnToStockByBatches(upd.Barcode, bret)
			if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to return stock", "barcode": upd.Barcode})
				return
			}
			newStock, newExp = stock, expir
			resulting = mergeBatchUsage(subtractBatchUsage(ex.Batches, bret))
			auditLog = append(auditLog, fmt.Sprintf("Удалено %.2f из %s", -diff, upd.Barcode))
			refundDelta += -diff * upd.UnitPrice
		} else {
			newStock = product.Quantities
			newExp = product.ExpirationDate
			resulting = mergeBatchUsage(ex.Batches)
		}

		unitPrice := upd.UnitPrice
		if unitPrice == 0 {
			unitPrice = product.Whosaleprice
		}
		totalPrice := math.Round(newQty*unitPrice*100) / 100
		totalAmount += totalPrice

		retailTotal := math.Round(newQty*upd.RetailPrice*100) / 100
		stockRemaining := sumQuantitiesFloat(newStock)

		updatedProducts = append(updatedProducts, models.OrderedProduct{
			Barcode:          upd.Barcode,
			Quantity:         newQty,
			MinimumOrder:     newQty,
			UnitPrice:        unitPrice,
			TotalPrice:       totalPrice,
			Retailprice:      upd.RetailPrice,
			TotalRetailprice: retailTotal,
			Batches:          resulting,
			StockRemaining:   stockRemaining,
			Unm:              upd.Unm,
		})

		if _, err := config.ProductCollection.UpdateOne(context.TODO(), bson.M{"barcode": upd.Barcode}, bson.M{"$set": bson.M{"quantities": newStock, "expirationdate": newExp, "updated_at": time.Now()}}); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update stock"})
			return
		}
	}

	// удаление отсутствующих товаров
	sent := map[string]bool{}
	for _, p := range req.Products {
		sent[p.Barcode] = true
	}
	for _, p := range existingOrder.Products {
		if !sent[p.Barcode] {
			bret := trimBatchUsageToQuantity(p.Batches, p.Quantity)
			if _, _, err := ReturnToStockByBatches(p.Barcode, bret); err == nil {
				auditLog = append(auditLog, fmt.Sprintf("Удалён товар %s, %.2f", p.Barcode, p.Quantity))
				refundDelta += p.TotalPrice
			}
		}
	}

	totalAmount = math.Round((totalAmount+req.DeliveryCost)*100) / 100
	if _, err := config.OrderCollection.UpdateOne(context.TODO(), bson.M{"_id": orderID}, bson.M{"$set": bson.M{
		"products":        updatedProducts,
		"total_amount":    totalAmount,
		"deliverymethod":  req.DeliveryMethod,
		"deliveryaddress": req.DeliveryAddress,
		"deliverycost":    req.DeliveryCost,
		"updated_at":      time.Now(),
	}}); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update order"})
		return
	}
	if len(auditLog) > 0 {
		_, _ = config.OrderCollection.UpdateOne(context.TODO(), bson.M{"_id": orderID}, bson.M{"$push": bson.M{"audit_log": bson.M{"$each": auditLog}}})
	}

	if existingOrder.PaymentMethod == "Peshraft" {
		delta := math.Round((addedDelta-refundDelta)*100) / 100
		if math.Abs(delta) > 0.000001 {
			defaultCashierID := "6789dda813e605d4bf8eec82"
			var client models.Client
			cid, err := primitive.ObjectIDFromHex(existingOrder.Clientid)
			if err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid client ID format"})
				return
			}
			if err := config.ClientCollection.FindOne(context.TODO(), bson.M{"_id": cid}).Decode(&client); err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": "Не удалось получить данные клиента"})
				return
			}
			card := client.HamrohCard
			if card == "" {
				c.JSON(http.StatusBadRequest, gin.H{"error": "У клиента нет привязанной карты Hamroh"})
				return
			}

			if delta > 0 {
				if success, resp, err := ProcessPeshraftTransaction(card, delta, defaultCashierID); err != nil || !success {
					c.JSON(http.StatusInternalServerError, gin.H{"error": "Peshraft charge failed", "details": err.Error()})
					return
				} else {
					transactionID := resp
					var parsed struct {
						Transaction struct {
							ID string `json:"id"`
						} `json:"transaction"`
					}
					if json.Valid([]byte(resp)) {
						_ = json.Unmarshal([]byte(resp), &parsed)
						if parsed.Transaction.ID != "" {
							transactionID = parsed.Transaction.ID
						}
					}
					if transactionID == "" {
						c.JSON(http.StatusInternalServerError, gin.H{"error": "Peshraft charge succeeded but transaction ID missing"})
						return
					}
					if _, err := config.OrderCollection.UpdateOne(context.TODO(), bson.M{"_id": orderID}, bson.M{"$push": bson.M{"peshraft_transactions": models.PeshraftTxn{ID: transactionID, Amount: delta}}}); err != nil {
						c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update Peshraft transactions"})
						return
					}
					_, _ = config.OrderCollection.UpdateOne(context.TODO(), bson.M{"_id": orderID}, bson.M{"$push": bson.M{"audit_log": fmt.Sprintf("Peshraft доплата %.2f (reconcile)", delta)}})
				}
			} else {
				refundRemaining := -delta
				txns := existingOrder.PeshraftTransactions
				if len(txns) == 0 && existingOrder.PeshraftTransactionID != "" {
					txns = append(txns, models.PeshraftTxn{ID: existingOrder.PeshraftTransactionID, Amount: existingOrder.TotalAmount})
				}
				updated := make([]models.PeshraftTxn, 0, len(txns))
				for i := len(txns) - 1; i >= 0 && refundRemaining > 0; i-- {
					t := txns[i]
					portion := math.Min(refundRemaining, t.Amount)
					if err := ProcessPeshraftRefund(t.ID, portion, defaultCashierID); err != nil {
						c.JSON(http.StatusInternalServerError, gin.H{"error": "Peshraft refund failed", "details": err.Error()})
						return
					}
					t.Amount -= portion
					refundRemaining -= portion
					if t.Amount > 0 {
						updated = append([]models.PeshraftTxn{t}, updated...)
					}
				}
				if refundRemaining > 1e-9 {
					c.JSON(http.StatusConflict, gin.H{"error": "Refund exceeds charged amount"})
					return
				}
				if _, err := config.OrderCollection.UpdateOne(context.TODO(), bson.M{"_id": orderID}, bson.M{"$set": bson.M{"peshraft_transactions": updated}}); err != nil {
					c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to persist refund changes"})
					return
				}
				_, _ = config.OrderCollection.UpdateOne(context.TODO(), bson.M{"_id": orderID}, bson.M{"$push": bson.M{"audit_log": fmt.Sprintf("Peshraft возврат %.2f (reconcile)", -delta)}})
			}
		}
	}

	c.JSON(http.StatusOK, gin.H{"message": "Order updated successfully"})
}

// ----------------------------- External Peshraft API callers (leave as-is) -----------------------------

func ProcessPeshraftRefund(transactionID string, amount float64, cashierID string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	apiKey, err := GetShopAPIKey(ctx)
	if err != nil {
		return fmt.Errorf("ошибка получения API ключа: %w", err)
	}

	url := fmt.Sprintf("https://bp.murod.store/api/transaction/%s/return", transactionID)

	body, err := json.Marshal(map[string]interface{}{"amount": amount})
	if err != nil {
		return fmt.Errorf("ошибка при создании JSON: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPut, url, bytes.NewBuffer(body))
	if err != nil {
		return fmt.Errorf("ошибка при создании запроса: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Cashier-ID", cashierID)
	req.Header.Set("X-API-Key", apiKey) // 🔐 ВАЖНО: Добавить API ключ

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("ошибка при отправке запроса: %w", err)
	}
	defer resp.Body.Close()

	respBody, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("ошибка возврата: %s", respBody)
	}

	return nil
}

func ProcessPeshraftTransaction(cardNumber string, amount float64, cashierid string) (bool, string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	apiKey, err := GetShopAPIKey(ctx)
	if err != nil {
		return false, "", fmt.Errorf("failed to get API key: %w", err)
	}

	url := fmt.Sprintf("https://bp.murod.store/api/card/%s/addpurchase", cardNumber)
	requestBody, _ := json.Marshal(map[string]interface{}{"amount": amount, "cashierid": cashierid})
	req, err := http.NewRequest("POST", url, bytes.NewBuffer(requestBody))
	if err != nil {
		return false, "", fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-API-Key", apiKey)

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return false, "", fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK {
		return false, string(body), fmt.Errorf("Peshraft error: %s", body)
	}
	return true, string(body), nil
}

// func ProcessPeshraftTransaction(cardNumber string, amount float64, cashierid string) (bool, string, error) {
// 	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
// 	defer cancel()

// 	apiKey, err := GetShopAPIKey(ctx)
// 	if err != nil {
// 		return false, "", fmt.Errorf("failed to get API key: %w", err)
// 	}

// 	url := fmt.Sprintf("https://bp.murod.store/api/card/%s/addpurchase", cardNumber)

// 	requestBody, _ := json.Marshal(map[string]interface{}{
// 		"amount":    amount,
// 		"cashierid": cashierid,
// 	})

// 	req, err := http.NewRequest("POST", url, bytes.NewBuffer(requestBody))
// 	if err != nil {
// 		return false, "", fmt.Errorf("failed to create request: %w", err)
// 	}

// 	req.Header.Set("Content-Type", "application/json")
// 	req.Header.Set("X-API-Key", apiKey)

// 	client := &http.Client{Timeout: 10 * time.Second}
// 	resp, err := client.Do(req)
// 	if err != nil {
// 		return false, "", fmt.Errorf("failed to send request: %w", err)
// 	}
// 	defer resp.Body.Close()

// 	body, _ := io.ReadAll(resp.Body)

// 	if resp.StatusCode != http.StatusOK {
// 		return false, string(body), fmt.Errorf("Peshraft error: %s", body)
// 	}

// 	return true, string(body), nil
// }
