package controllers

import (
	"context"
	// "fmt"
	"net/http"
	// "strconv"
	"time"

	"backend/config"
	"backend/models"

	"github.com/gin-gonic/gin"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	// "go.mongodb.org/mongo-driver/mongo/options"
)

func GetClientCardInfo(c *gin.Context) {
	clientID, exists := c.Get("clientID")
	c.JSON(http.StatusBadRequest, gin.H{"error": clientID})
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	// Convert clientID to ObjectID
	clientObjectID, err := primitive.ObjectIDFromHex(clientID.(string))
	c.JSON(http.StatusBadRequest, gin.H{"error": clientID})
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

	// Add the fullname to the card data
	cardMap := map[string]interface{}{
		"id":                 card.ID,
		"cardnumber":         card.CardNumber,
		"status":             card.Status,
		"createdate":         card.CreateDate,
		"limit": 			  card.Limit,
		"totalout":           card.TotalOut,
		"totalloan":          card.TotalLoan,
		"totalfast":          card.TotalFast,
		"totalpurchase":      card.TotalPurchase,
		"fullname":           client.FirstName + " " + client.LastName,
		"day":card.Days,
	}

	c.JSON(http.StatusOK, cardMap)
}


















// AddToCart добавляет товар в корзину
// func AddToCart(c *gin.Context) {
// 	clientID := c.Param("clientID")

// 	var item models.OrderItem
// 	if err := c.ShouldBindJSON(&item); err != nil {
// 		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid input data"})
// 		return
// 	}

// 	// Проверяем, что ProductID передаётся как строка и не пустой
// 	if item.ProductID == "" {
// 		c.JSON(http.StatusBadRequest, gin.H{"error": "ProductID is required"})
// 		return
// 	}

// 	// Преобразуем строку ProductID в ObjectID
// 	productID, err := primitive.ObjectIDFromHex(item.ProductID)
// 	if err != nil {
// 		c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("Invalid ProductID format: %s", item.ProductID)})
// 		return
// 	}

// 	// Проверяем наличие товара по ProductID в базе данных
// 	var product models.Product
// 	err = config.ProductCollection.FindOne(context.TODO(), bson.M{"_id": productID}).Decode(&product)
// 	if err != nil {
// 		c.JSON(http.StatusNotFound, gin.H{"error": fmt.Sprintf("Product not found for ID: %s", item.ProductID)})
// 		return
// 	}

// 	// Проверка минимального количества для заказа
// 	minOrder, err := strconv.Atoi(product.Minimumorder)
// 	if err != nil {
// 		c.JSON(http.StatusInternalServerError, gin.H{"error": "Invalid minimum order quantity"})
// 		return
// 	}

// 	if item.Quantity < minOrder {
// 		c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("Minimum order for product '%s' is %d", product.Name, minOrder)})
// 		return
// 	}

// 	// Проверка, достаточно ли товара на складе для нового добавления
// 	stockQuantity, err := strconv.Atoi(product.Quantity)
// 	if err != nil {
// 		c.JSON(http.StatusInternalServerError, gin.H{"error": "Invalid product quantity in stock"})
// 		return
// 	}

// 	// Ищем существующую корзину
// 	var order models.Order
// 	err = config.OrderCollection.FindOne(context.TODO(), bson.M{"clientid": clientID, "status": "Pending"}).Decode(&order)

// 	if err != nil && err != mongo.ErrNoDocuments {
// 		c.JSON(http.StatusInternalServerError, gin.H{"error": "Error fetching cart"})
// 		return
// 	}

// 	// Если корзина не найдена, создаём новую
// 	if err == mongo.ErrNoDocuments {
// 		order = models.Order{
// 			ID:          primitive.NewObjectID(),
// 			ClientID:    clientID,
// 			Items:       []models.OrderItem{},
// 			TotalAmount: 0,
// 			Status:      "Pending",
// 			CreatedAt:   time.Now(),
// 			UpdatedAt:   time.Now(),
// 		}
// 	}

// 	// Проверка, если товар уже в корзине
// 	itemUpdated := false
// 	for i := range order.Items {
// 		if order.Items[i].ProductID == item.ProductID {
// 			// Текущие количество и новое количество суммируются
// 			newQuantity := order.Items[i].Quantity + item.Quantity

// 			// Проверяем, достаточно ли товара на складе для нового суммарного количества
// 			if newQuantity > stockQuantity {
// 				c.JSON(http.StatusBadRequest, gin.H{
// 					"error": fmt.Sprintf("Not enough stock for product '%s'. Available: %d", product.Name, stockQuantity),
// 				})
// 				return
// 			}

// 			// Обновляем количество и общую стоимость
// 			order.Items[i].Quantity = newQuantity
// 			order.Items[i].TotalPrice = order.Items[i].Price * float64(newQuantity)
// 			itemUpdated = true
// 			break
// 		}
// 	}

// 	// Если товара нет в корзине, добавляем его
// 	if !itemUpdated {
// 		// Заполняем информацию о товаре и рассчитываем общую цену
// 		item.ProductName = product.Name
// 		item.Price = product.Sellingprice
// 		item.TotalPrice = item.Price * float64(item.Quantity)

// 		// Проверяем, достаточно ли товара на складе
// 		if item.Quantity > stockQuantity {
// 			c.JSON(http.StatusBadRequest, gin.H{
// 				"error": fmt.Sprintf("Not enough stock for product '%s'. Available: %d", product.Name, stockQuantity),
// 			})
// 			return
// 		}

// 		order.Items = append(order.Items, item)
// 	}

// 	// Пересчитываем общую сумму заказа
// 	order.TotalAmount = 0
// 	for _, item := range order.Items {
// 		order.TotalAmount += item.TotalPrice
// 	}
// 	order.UpdatedAt = time.Now()

// 	// Сохраняем изменения в базе данных
// 	_, err = config.OrderCollection.UpdateOne(context.TODO(), bson.M{"_id": order.ID}, bson.M{"$set": order}, options.Update().SetUpsert(true))
// 	if err != nil {
// 		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update cart"})
// 		return
// 	}

// 	c.JSON(http.StatusOK, gin.H{"message": "Item added to cart", "order": order})
// }





// // Checkout оформляет заказ и уменьшает количество товаров на складе
// func Checkout(c *gin.Context) {
// 	clientID := c.Param("clientID")

// 	var requestBody struct {
// 		PaymentMethod string `json:"payment_method"`
// 		DeliveryType  string `json:"delivery_type"`
// 	}

// 	// Получаем метод оплаты и тип доставки из запроса
// 	if err := c.ShouldBindJSON(&requestBody); err != nil {
// 		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid input data"})
// 		return
// 	}

// 	if requestBody.PaymentMethod == "" || requestBody.DeliveryType == "" {
// 		c.JSON(http.StatusBadRequest, gin.H{"error": "Payment method and delivery type are required"})
// 		return
// 	}

// 	// Ищем корзину с товарами для клиента
// 	var order models.Order
// 	err := config.OrderCollection.FindOne(context.TODO(), bson.M{"clientid": clientID, "status": "Pending"}).Decode(&order)
// 	if err != nil {
// 		c.JSON(http.StatusNotFound, gin.H{"error": "No pending order found"})
// 		return
// 	}

// 	// Проверка наличия всех товаров на складе и уменьшение количества
// 	for _, item := range order.Items {
// 		var product models.Product
// 		productID, _ := primitive.ObjectIDFromHex(item.ProductID)
// 		err := config.ProductCollection.FindOne(context.TODO(), bson.M{"_id": productID}).Decode(&product)
// 		if err != nil {
// 			c.JSON(http.StatusNotFound, gin.H{"error": fmt.Sprintf("Product not found: %s", item.ProductName)})
// 			return
// 		}

// 		// Проверка, достаточно ли товара на складе
// 		stockQuantity, err := strconv.Atoi(product.Quantity)
// 		if err != nil {
// 			c.JSON(http.StatusInternalServerError, gin.H{"error": "Invalid product quantity in stock"})
// 			return
// 		}

// 		if stockQuantity < item.Quantity {
// 			c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("Not enough stock for product '%s'. Available: %d", product.Name, stockQuantity)})
// 			return
// 		}

// 		// Уменьшаем количество товара на складе
// 		newQuantity := stockQuantity - item.Quantity
// 		_, err = config.ProductCollection.UpdateOne(
// 			context.TODO(),
// 			bson.M{"_id": productID},
// 			bson.M{"$set": bson.M{"quantity": strconv.Itoa(newQuantity)}},
// 		)
// 		if err != nil {
// 			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update product quantity"})
// 			return
// 		}
// 	}

// 	// Обновляем статус заказа на "Completed" и добавляем метод оплаты и тип доставки
// 	order.Status = "Completed"
// 	order.PaymentMethod = requestBody.PaymentMethod
// 	order.DeliveryType = requestBody.DeliveryType
// 	order.UpdatedAt = time.Now()

// 	_, err = config.OrderCollection.UpdateOne(context.TODO(), bson.M{"_id": order.ID}, bson.M{"$set": order})
// 	if err != nil {
// 		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to complete the order"})
// 		return
// 	}

// 	c.JSON(http.StatusOK, gin.H{"message": "Order completed successfully", "order": order})
// }



// GetOrderByID - получение информации о заказе по его OrderID
func GetOrderByID1(c *gin.Context) {
	orderID := c.Param("orderid")
	objID, err := primitive.ObjectIDFromHex(orderID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid order ID"})
		return
	}

	var order models.Order
	err = config.OrderCollection.FindOne(context.TODO(), bson.M{"_id": objID}).Decode(&order)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Order not found"})
		return
	}

	c.JSON(http.StatusOK, order)
}


// GetOrdersByClientID - получение всех заказов клиента по ClientID
func GetOrdersByClientID(c *gin.Context) {
	clientID := c.Param("clientID")

	var orders []models.Order
	cursor, err := config.OrderCollection.Find(context.TODO(), bson.M{"clientID": clientID})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch orders"})
		return
	}
	defer cursor.Close(context.TODO())

	if err = cursor.All(context.TODO(), &orders); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to decode orders"})
		return
	}

	if len(orders) == 0 {
		c.JSON(http.StatusNotFound, gin.H{"message": "No orders found for this client"})
		return
	}

	c.JSON(http.StatusOK, orders)
}
