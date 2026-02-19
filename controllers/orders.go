package controllers

import (
	"context"
	"fmt"
	"time"

	// "fmt"
	"net/http"
	// "strconv"
	//"time"

	"backend/config"
	"backend/models"

	"github.com/gin-gonic/gin"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	// "go.mongodb.org/mongo-driver/mongo/options"
)

func GetOrderByID(c *gin.Context) {
	orderID := c.Param("id")

	objID, err := primitive.ObjectIDFromHex(orderID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid order ID"})
		return
	}

	var order models.CustomerOrder
	err = config.OrderCollection.FindOne(context.TODO(), bson.M{"_id": objID}).Decode(&order)
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
		Type string `bson:"type"`
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
		Minimumorder     float64 `json:"minimumorder"`
		UnitPrice        float64 `json:"unit_price"`
		TotalPrice       float64 `json:"total_price"`
		TotalRetailPrice float64 `json:"totalretailprice"`
		Retailprice      float64 `json:"retailprice"`
		Name             string  `json:"name"`
		Unm              string  `json:"unm"`
		ProductPhotoURL  string  `json:"productphotourl"`
		Batches          []Batch `json:"batches"`
	}

	type Snapshot struct {
		Limit         float64 `json:"limit"`
		TotalPurchase float64 `json:"totalpurchase"`
		TotalLoan     float64 `json:"totalloan"`
		TotalOut      float64 `json:"totalout"`
		TotalSettle   float64 `json:"totalsettle"`
		Days          int64   `json:"days"`
		Retday        int64   `json:"retday"`
	}
	type Peshrafttran struct {
		Snapshot []Snapshot `json:"snapshot"`
	}

	var extendedProducts []ExtendedProduct

	for _, orderedProduct := range order.Products {
		var product models.Product
		err := config.ProductCollection.FindOne(context.TODO(), bson.M{"barcode": orderedProduct.Barcode}).Decode(&product)
		if err != nil {
			if err == mongo.ErrNoDocuments {
				extendedProducts = append(extendedProducts, ExtendedProduct{
					Barcode:          orderedProduct.Barcode,
					Quantity:         orderedProduct.Quantity,
					Minimumorder:     orderedProduct.Quantity,
					UnitPrice:        orderedProduct.UnitPrice,
					Retailprice:      orderedProduct.Retailprice,
					TotalRetailPrice: orderedProduct.TotalRetailprice,
					TotalPrice:       orderedProduct.TotalPrice,
					Name:             "Unknown",
					Unm:              "Unknown",
					Batches: func() []Batch {
						var result []Batch
						for _, b := range orderedProduct.Batches {
							result = append(result, Batch{
								ExpirationDate: b.ExpirationDate,
								UsedQuantity:   b.UsedQuantity,
							})
						}
						return result
					}(),
				})
				continue
			}
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to retrieve product details"})
			return
		}

		extendedProducts = append(extendedProducts, ExtendedProduct{
			Barcode:          orderedProduct.Barcode,
			Quantity:         orderedProduct.Quantity,
			Minimumorder:     orderedProduct.Quantity,
			UnitPrice:        orderedProduct.UnitPrice,
			TotalPrice:       orderedProduct.TotalPrice,
			Retailprice:      orderedProduct.Retailprice,
			TotalRetailPrice: orderedProduct.TotalRetailprice,
			Name:             product.Name,
			Unm:              product.Unm,
			ProductPhotoURL:  product.Productphotourl,
			Batches: func() []Batch {
				var result []Batch
				for _, b := range orderedProduct.Batches {
					result = append(result, Batch{
						ExpirationDate: b.ExpirationDate,
						UsedQuantity:   b.UsedQuantity,
					})
				}
				return result
			}(),
		})
	}

	var peshraftDetails interface{} = nil
	if order.PaymentMethod == "Peshraft" && order.PeshraftTransactionID != "" {
		peshraftID, err := primitive.ObjectIDFromHex(order.PeshraftTransactionID)
		if err == nil {
			var transaction Peshrafttran
			err = config.TransactionCollectionP.FindOne(context.TODO(), bson.M{"_id": peshraftID}).Decode(&transaction)
			if err == nil {
				peshraftDetails = transaction.Snapshot
			}
		}
	}

	// deliveryCost := 0.0
	// if order.DeliveryMethod == "courier" {
	// 	deliveryCost = 10.0
	// }
	// generalTotal := order.TotalAmount + order.DeliveryCost

	extendedOrder := struct {
		ID              primitive.ObjectID `json:"id"`
		Products        []ExtendedProduct  `json:"products"`
		DeliveryMethod  string             `json:"deliverymethod"`
		DeliveryAddress string             `json:"deliveryaddress"`
		PaymentMethod   string             `json:"paymentmethod"`
		Status          string             `json:"status"`
		TotalAmount     float64            `json:"total_amount"`
		DeliveryCost    float64            `json:"deliverycost,omitempty"`
		Total    float64            `json:"total,omitempty"`
		TranID          string             `json:"tranid"`
		FullName        string             `json:"fullname"`
		CreatedAt       time.Time          `json:"created_at"`
		ViewToken       string             `json:"view_token"`
		PeshraftDetails interface{}        `json:"peshraft_details,omitempty"`
		Type string `json:"type"`
	}{
		ID:              order.ID,
		Products:        extendedProducts,
		DeliveryMethod:  order.DeliveryMethod,
		DeliveryAddress: order.DeliveryAddress,
		PaymentMethod:   order.PaymentMethod,
		Status:          order.Status,
		TotalAmount:     order.TotalAmount,
		DeliveryCost:    order.DeliveryCost,
		Total:    order.Total,
		TranID:          order.Tranid,
		FullName:        fullName,
		CreatedAt:       order.CreatedAt,
		ViewToken:       order.ViewToken,
		PeshraftDetails: peshraftDetails,
		Type: client.Type,
	}

	c.JSON(http.StatusOK, extendedOrder)
}



func GetAllOrders(c *gin.Context) {
	cursor, err := config.OrderCollection.Find(context.TODO(), bson.M{})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to retrieve orders"})
		return
	}
	defer cursor.Close(context.TODO())

	var orders []models.CustomerOrder
	if err = cursor.All(context.TODO(), &orders); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to decode orders"})
		return
	}

	type ExtendedOrder struct {
		models.CustomerOrder
		FullName string `json:"fullname"`
	}

	var extendedOrders []ExtendedOrder

	for _, order := range orders {
		clientID, err := primitive.ObjectIDFromHex(order.Clientid)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid client ID format"})
			return
		}

		var client struct {
			FirstName string `bson:"first_name"`
			LastName  string `bson:"last_name"`
		}

		err = config.ClientCollection.FindOne(context.TODO(), bson.M{"_id": clientID}).Decode(&client)
		fullName := "Unknown"
		if err == nil {
			fullName = fmt.Sprintf("%s %s", client.FirstName, client.LastName)
		}

		extendedOrders = append(extendedOrders, ExtendedOrder{
			CustomerOrder: order,
			FullName:      fullName,
		})
	}

	c.JSON(http.StatusOK, extendedOrders)
}

func GetAllOrdersNEW(c *gin.Context) {
	cursor, err := config.OrderCollection.Find(context.TODO(), bson.M{})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to retrieve orders"})
		return
	}
	defer cursor.Close(context.TODO())

	var orders []models.CustomerOrder
	if err = cursor.All(context.TODO(), &orders); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to decode orders"})
		return
	}

	type TrimmedOrder struct {
		ID              primitive.ObjectID `json:"id"`
		DeliveryMethod  string             `json:"deliverymethod"`
		DeliveryAddress string             `json:"deliveryaddress,omitempty"`
		PaymentMethod   string             `json:"paymentmethod"`
		CardNumber      string             `json:"card_number,omitempty"`
		Status          string             `json:"status"`
		TotalAmount     float64            `json:"total_amount"`
		PeshraftTransactionID string       `json:"peshraft_transaction_id,omitempty"`
		Qrlink          string             `json:"qrlink"`
		Tranid          string             `json:"tranid"`
		ViewToken       string             `json:"view_token"`
		Clientid        string             `json:"clientid"`
		CreatedAt       time.Time          `json:"created_at"`
		FullName        string             `json:"fullname"`
	}

	var result []TrimmedOrder

	for _, order := range orders {
		clientID, err := primitive.ObjectIDFromHex(order.Clientid)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid client ID format"})
			return
		}

		var client struct {
			FirstName string `bson:"first_name"`
			LastName  string `bson:"last_name"`
		}

		err = config.ClientCollection.FindOne(context.TODO(), bson.M{"_id": clientID}).Decode(&client)
		fullName := "Unknown"
		if err == nil {
			fullName = fmt.Sprintf("%s %s", client.FirstName, client.LastName)
		}

		result = append(result, TrimmedOrder{
			ID:              order.ID,
			DeliveryMethod:  order.DeliveryMethod,
			DeliveryAddress: order.DeliveryAddress,
			PaymentMethod:   order.PaymentMethod,
			CardNumber:      order.CardNumber,
			Status:          order.Status,
			TotalAmount:     order.TotalAmount,
			PeshraftTransactionID: order.PeshraftTransactionID,
			Qrlink:          order.Qrlink,
			Tranid:          order.Tranid,
			ViewToken:       order.ViewToken,
			Clientid:        order.Clientid,
			CreatedAt:       order.CreatedAt,
			FullName:        fullName,
		})
	}

	c.JSON(http.StatusOK, result)
}


// routes/order_returns.go
func GetAllReturnOrders(c *gin.Context) {
	cursor, err := config.OrderReturnCollection.Find(context.TODO(), bson.M{})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to retrieve return orders"})
		return
	}
	defer cursor.Close(context.TODO())

	var returnOrders []models.CustomerOrderReturn
	if err = cursor.All(context.TODO(), &returnOrders); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to decode return orders"})
		return
	}

	type ExtendedReturnOrder struct {
		models.CustomerOrderReturn
		FullName string `json:"fullname"`
	}

	var extendedReturnOrders []ExtendedReturnOrder

	for _, returnOrder := range returnOrders {
		var originalOrder models.CustomerOrder
		err := config.OrderCollection.FindOne(context.TODO(), bson.M{"_id": returnOrder.OriginalOrderID}).Decode(&originalOrder)
		if err != nil {
			continue // пропускаем если заказ не найден
		}

		clientID, err := primitive.ObjectIDFromHex(originalOrder.Clientid)
		if err != nil {
			continue
		}

		var client struct {
			FirstName string `bson:"first_name"`
			LastName  string `bson:"last_name"`
		}
		err = config.ClientCollection.FindOne(context.TODO(), bson.M{"_id": clientID}).Decode(&client)

		fullName := "Unknown"
		if err == nil {
			fullName = fmt.Sprintf("%s %s", client.FirstName, client.LastName)
		}

		extendedReturnOrders = append(extendedReturnOrders, ExtendedReturnOrder{
			CustomerOrderReturn: returnOrder,
			FullName:            fullName,
		})
	}

	c.JSON(http.StatusOK, extendedReturnOrders)
}


func GetOrdersByCustomerID(c *gin.Context) {
    clientID := c.Param("clientid")

    // Only project needed fields
    projection := bson.M{
        "status":       1,
        "total_amount": 1,
        "created_at":   1,
        "products":     1,
    }

    cursor, err := config.OrderCollection.Find(context.TODO(), bson.M{"clientid": clientID}, options.Find().SetProjection(projection))
    if err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to retrieve orders"})
        return
    }
    defer cursor.Close(context.TODO())

    var orders []models.CustomerOrder
    if err = cursor.All(context.TODO(), &orders); err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to decode orders"})
        return
    }

    // Prepare final orders with product preview URLs
    var result []map[string]interface{}
    for _, order := range orders {
        orderMap := map[string]interface{}{
			"id": order.ID,
            "status":       order.Status,
            "total_amount": order.TotalAmount,
            "created_at":   order.CreatedAt,
        }

        var enrichedProducts []map[string]interface{}
        for _, prod := range order.Products {
            productDoc := config.ProductCollection.FindOne(context.TODO(), bson.M{"barcode": prod.Barcode}, options.FindOne().SetProjection(bson.M{"productphotopreviewurl": 1}))
            var productResult struct {
                ProductPhotoPreviewURL string `bson:"productphotopreviewurl"`
            }
            _ = productDoc.Decode(&productResult)

            enrichedProduct := map[string]interface{}{
                "barcode":                prod.Barcode,
                "productphotopreviewurl": productResult.ProductPhotoPreviewURL,
            }
            enrichedProducts = append(enrichedProducts, enrichedProduct)
        }

        orderMap["products"] = enrichedProducts
        result = append(result, orderMap)
    }

    c.JSON(http.StatusOK, result)
}


func ConfirmCustomerOrder(c *gin.Context) {
	orderID := c.Param("id")
	objID, err := primitive.ObjectIDFromHex(orderID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid order ID"})
		return
	}

	// Чтение подтверждённых продуктов из запроса
	var reqBody struct {
		Products []struct {
			Barcode string `json:"barcode"`
			Status  string `json:"status"`
		} `json:"products"`
	}

	if err := c.ShouldBindJSON(&reqBody); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request body"})
		return
	}

	// Получаем заказ из базы
	var order models.CustomerOrder
	err = config.OrderCollection.FindOne(context.TODO(), bson.M{"_id": objID}).Decode(&order)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Order not found"})
		return
	}

	if order.Status != "Order confirm, in process in stock!" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Невозможно подтвердить заказ: неверный статус"})
		return
	}

	reqMap := map[string]string{}
	for _, p := range reqBody.Products {
		reqMap[p.Barcode] = p.Status
	}

	// Проверим: фронт подтвердил ВСЕ товары?
	for _, p := range order.Products {
		status, ok := reqMap[p.Barcode]
		if !ok || status != "confirmed" {
			c.JSON(http.StatusBadRequest, gin.H{
				"error": fmt.Sprintf("Продукт с штрих-кодом %s не подтвержден фронтом", p.Barcode),
			})
			return
		}
	}

	// Обновляем статусы на confirmed
	for i := range order.Products {
		order.Products[i].Status = "confirmed"
	}

	// Обновляем статус заказа и продукты в базе
	update := bson.M{
		"$set": bson.M{
			"products":    order.Products,
			"status":      "Заказ собран, можете забрать со склада.",
			"updated_at":  time.Now(),
		},
	}
	_, err = config.OrderCollection.UpdateOne(context.TODO(), bson.M{"_id": objID}, update)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Ошибка при обновлении заказа"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"status": "Заказ собран, можете забрать со склада.",
	})
}

