package handlers

import (
	// "bytes"
	"context"
	"encoding/json"
	"sort"

	// "encoding/json"

	"fmt"
	// "io"
	// "log"
	"math"
	"net/http"

	// "sort"

	// "strings"
	"time"

	"github.com/google/uuid"

	"backend/config"
	// "backend/controllers"
	"backend/models"
	"backend/utils"

	"github.com/gin-gonic/gin"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	// "go.mongodb.org/mongo-driver/mongo"
	// "go.mongodb.org/mongo-driver/mongo/options"
)

// File: handlers/order_update_quantities.go
type ProductInput struct {
	Barcode  string  `json:"barcode" binding:"required"`
	Quantity float64 `json:"quantity" binding:"required"`
}

func CreateCustomerOrder(c *gin.Context) {
	var order struct {
		Products        []models.ProductQuantity `json:"products" binding:"required"`
		DeliveryMethod  string                   `json:"delivery_method" binding:"required"`
		DeliveryAddress string                   `json:"delivery_address"`
		DeliveryCost    float64                  `json:"deliverycost"`
		PaymentMethod   string                   `json:"payment_method" binding:"required"`
		CardNumber      string                   `json:"card_number"`
		Tranid          string                   `json:"tranid"`
		Clientid        string                   `json:"clientid"`
		CashierID       string                   `json:"cashierid"`
	}

	if err := c.ShouldBindJSON(&order); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	aggregatedProducts := aggregateProducts(order.Products)
	for i := range aggregatedProducts {
		aggregatedProducts[i].Quantity = math.Round(aggregatedProducts[i].Quantity*100) / 100
	}

	clientName := "Гость"
	isRetailClient := false
	if order.Clientid != "" && len(order.Clientid) == 24 {
		objID, err := primitive.ObjectIDFromHex(order.Clientid)
		if err == nil {
			var client models.Client
			err = config.ClientCollection.FindOne(context.TODO(), bson.M{"_id": objID}).Decode(&client)
			if err == nil {
				if client.Type == "retail" {
					isRetailClient = true
				}
				clientName = fmt.Sprintf("%s %s", client.FirstName, client.LastName)
			}
		}
	}

	total, orderedProducts, stockUpdates, err := processProducts(aggregatedProducts, isRetailClient)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	var peshraftTransactionID string
	var initialTxns []models.PeshraftTxn
	if order.PaymentMethod == "Peshraft" {
		if order.CardNumber == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Card number is required for Peshraft payment"})
			return
		}
		success, response, err := ProcessPeshraftTransaction(order.CardNumber, total+order.DeliveryCost, order.CashierID)
		if err != nil || !success {
			respText := response
			if err != nil {
				respText = err.Error()
			}
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to process Peshraft payment", "details": respText})
			return
		}
		var peshraftResp struct {
			Transaction struct {
				ID string `json:"id"`
			} `json:"transaction"`
		}
		if json.Unmarshal([]byte(response), &peshraftResp) != nil || peshraftResp.Transaction.ID == "" {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to parse Peshraft response"})
			return
		}
		peshraftTransactionID = peshraftResp.Transaction.ID
		amountPaid := math.Round((total+order.DeliveryCost)*100) / 100
		initialTxns = []models.PeshraftTxn{{ID: peshraftTransactionID, Amount: amountPaid}}
	} else if order.PaymentMethod == "DC" && order.Tranid == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Transaction ID is required for DC payment"})
		return
	}

	ctx := context.TODO()

	for _, product := range aggregatedProducts {
		update := stockUpdates[product.Barcode]
		remainingQuantity := product.Quantity
		usedBatches := []models.BatchUsage{}
		for i := 0; i < len(update.Quantities) && remainingQuantity > 0; i++ {
			used := minFloat(update.Quantities[i], remainingQuantity)
			update.Quantities[i] -= used
			remainingQuantity -= used
			usedBatches = append(usedBatches, models.BatchUsage{
				ExpirationDate: update.ExpirationDates[i],
				UsedQuantity:   used,
			})
		}
		if remainingQuantity > 0 {
			c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("Insufficient stock for: %s", product.Barcode)})
			return
		}

		_, err := config.ProductCollection.UpdateOne(ctx,
			bson.M{"barcode": product.Barcode},
			bson.M{"$set": bson.M{
				"quantities":     update.Quantities,
				"expirationdate": update.ExpirationDates,
				"updated_at":     time.Now(),
			}})
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update product stock"})
			return
		}

		for i := range orderedProducts {
			if orderedProducts[i].Barcode == product.Barcode {
				orderedProducts[i].Batches = usedBatches
				orderedProducts[i].StockRemaining = sumQuantitiesFloat(update.Quantities)
				break
			}
		}
	}

	totalAmount := math.Round((total+order.DeliveryCost)*100) / 100
	newOrder := models.CustomerOrder{
		ID:                    primitive.NewObjectID(),
		Products:              orderedProducts,
		DeliveryMethod:        order.DeliveryMethod,
		DeliveryAddress:       order.DeliveryAddress,
		DeliveryCost:          order.DeliveryCost,
		PaymentMethod:         order.PaymentMethod,
		PeshraftTransactionID: peshraftTransactionID,
		PeshraftTransactions:  initialTxns,
		Tranid:                order.Tranid,
		Clientid:              order.Clientid,
		Status:                "Order confirm, in process in stock!",
		Total:                 math.Round(total*100) / 100,
		TotalAmount:           totalAmount,
		CreatedAt:             time.Now(),
		ViewToken:             uuid.NewString(),
	}

	_, err = config.OrderCollection.InsertOne(ctx, newOrder)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to save order"})
		return
	}

	utils.SendSMS(removePlusFromPhone("+992937518880"), fmt.Sprintf("Клиент %s оформил заказ на сумму %.2f сомонӣ", clientName, newOrder.TotalAmount))

	if order.DeliveryMethod == "Рушон Вамар" || order.DeliveryMethod == "courier" {
		var storekeeper struct {
			Phone    string `bson:"phone"`
			Address  string `bson:"address"`
			FullName string `bson:"full_name"`
		}
		err := config.StorekeeperCollection.FindOne(ctx, bson.M{"location": "Рушон Вамар"}).Decode(&storekeeper)
		if err == nil {
			utils.SendSMS(removePlusFromPhone(storekeeper.Phone),
				fmt.Sprintf("Новый заказ от %s на сумму %.2f сомонӣ. Подготовьте заказ для доставки!", clientName, newOrder.TotalAmount))
		}
	}

	c.JSON(http.StatusOK, gin.H{"message": "Order created successfully", "order_id": newOrder.ID.Hex()})
}

func aggregateProducts(products []models.ProductQuantity) []models.ProductQuantity {
	agg := map[string]float64{}
	for _, p := range products {
		agg[p.Barcode] += p.Quantity
	}
	var result []models.ProductQuantity
	for barcode, qty := range agg {
		result = append(result, models.ProductQuantity{
			Barcode:  barcode,
			Quantity: qty,
		})
	}
	return result
}

func processProducts(products []models.ProductQuantity, isRetail bool) (
	float64,
	[]models.OrderedProduct,
	map[string]struct {
		Quantities      []float64
		ExpirationDates []string
	},
	error,
) {
	total := 0.0
	ordered := []models.OrderedProduct{}
	stockUpdates := make(map[string]struct {
		Quantities      []float64
		ExpirationDates []string
	})

	for _, p := range products {
		var stock models.Product
		err := config.ProductCollection.FindOne(context.TODO(), bson.M{"barcode": p.Barcode}).Decode(&stock)
		if err != nil {
			return 0, nil, nil, fmt.Errorf("Product not found: %s", p.Barcode)
		}
		totalStock := sumQuantitiesFloat(stock.Quantities)
		if totalStock < p.Quantity {
			return 0, nil, nil, fmt.Errorf("Недостаточно товара на складе: %s. Доступно %.2f, требуется %.2f", stock.Name, totalStock, p.Quantity)
		}
		unitPrice := stock.Whosaleprice
		subtotal := math.Round(p.Quantity*unitPrice*100) / 100
		total += subtotal
		sortedQuantities, sortedExp := sortBatchesByExpirationFloat(stock.Quantities, stock.ExpirationDate)
		stockUpdates[p.Barcode] = struct {
			Quantities      []float64
			ExpirationDates []string
		}{
			Quantities:      sortedQuantities,
			ExpirationDates: sortedExp,
		}
		ord := models.OrderedProduct{
			Barcode:    p.Barcode,
			Quantity:   p.Quantity,
			Unm:        stock.Unm,
			UnitPrice:  unitPrice,
			TotalPrice: subtotal,
		}
		if isRetail {
			ord.Retailprice = stock.Retailprice
			ord.TotalRetailprice = math.Round(p.Quantity*stock.Retailprice*100) / 100
		}
		ordered = append(ordered, ord)
	}
	return total, ordered, stockUpdates, nil
}

// func AdminUpdateOrderQuantities(c *gin.Context) {
// 	orderID := c.Param("orderID")
// 	var body struct {
// 		Updates []struct {
// 			Barcode      string  `json:"barcode" binding:"required"`
// 			Quantity     float64 `json:"quantity" binding:"required"`
// 			MinimumOrder float64 `json:"minimumorder" binding:"required"`
// 		} `json:"updates" binding:"required"`
// 		CashierID string `json:"cashier_id" binding:"required"`
// 	}

// 	if err := c.ShouldBindJSON(&body); err != nil {
// 		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
// 		return
// 	}

// 	orderObjID, err := primitive.ObjectIDFromHex(orderID)
// 	if err != nil {
// 		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid order ID format"})
// 		return
// 	}

// 	ctx := context.Background()
// 	var order models.CustomerOrder
// 	err = config.OrderCollection.FindOne(ctx, bson.M{"_id": orderObjID}).Decode(&order)
// 	if err != nil {
// 		c.JSON(http.StatusNotFound, gin.H{"error": "Order not found"})
// 		return
// 	}

// 	modLog := []models.OrderEditLog{}
// 	totalDiff := 0.0

// 	for _, upd := range body.Updates {
// 		// Если товар нужно удалить полностью
// 		if upd.MinimumOrder == 0 {
// 			found := false
// 			for i := range order.Products {
// 				if order.Products[i].Barcode == upd.Barcode {
// 					found = true
// 					oldQty := order.Products[i].Quantity
// 					totalPrice := order.Products[i].TotalPrice

// 					err := ReturnToStock(ctx, upd.Barcode, oldQty)
// 					if err != nil {
// 						c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
// 						return
// 					}

// 					modLog = append(modLog, models.OrderEditLog{
// 						Action:   "remove",
// 						Barcode:  upd.Barcode,
// 						Quantity: oldQty,
// 						Cashier:  body.CashierID,
// 						Time:     time.Now(),
// 					})

// 					order.Products = append(order.Products[:i], order.Products[i+1:]...)
// 					totalDiff -= totalPrice
// 					break
// 				}
// 			}
// 			if !found {
// 				c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("Product %s not found in order", upd.Barcode)})
// 				return
// 			}
// 			continue
// 		}

// 		found := false
// 		for i := range order.Products {
// 			if order.Products[i].Barcode == upd.Barcode {
// 				found = true
// 				oldQty := order.Products[i].Quantity
// 				diff := math.Round((upd.MinimumOrder - oldQty) * 100) / 100

// 				if diff > 0 {
// 					err := DecreaseStock(ctx, upd.Barcode, diff)
// 					if err != nil {
// 						c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
// 						return
// 					}
// 					totalDiff += diff * order.Products[i].UnitPrice
// 					order.Products[i].Quantity = upd.MinimumOrder
// 					order.Products[i].TotalPrice = RoundTo2(upd.MinimumOrder * order.Products[i].UnitPrice)
// 					modLog = append(modLog, models.OrderEditLog{
// 						Action:   "add",
// 						Barcode:  upd.Barcode,
// 						Quantity: diff,
// 						Cashier:  body.CashierID,
// 						Time:     time.Now(),
// 					})
// 				} else if diff < 0 {
// 					err := ReturnToStock(ctx, upd.Barcode, -diff)
// 					if err != nil {
// 						c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
// 						return
// 					}
// 					totalDiff += diff * order.Products[i].UnitPrice
// 					order.Products[i].Quantity = upd.MinimumOrder
// 					order.Products[i].TotalPrice = RoundTo2(upd.MinimumOrder * order.Products[i].UnitPrice)
// 					modLog = append(modLog, models.OrderEditLog{
// 						Action:   "remove",
// 						Barcode:  upd.Barcode,
// 						Quantity: -diff,
// 						Cashier:  body.CashierID,
// 						Time:     time.Now(),
// 					})
// 				}
// 				break
// 			}
// 		}
// 		if !found {
// 			c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("Product %s not found in order", upd.Barcode)})
// 			return
// 		}
// 	}

// 	order.Total = RoundTo2(order.Total + totalDiff)
// 	order.TotalAmount = RoundTo2(order.Total + order.DeliveryCost)
// 	order.EditLogs = append(order.EditLogs, modLog...)

// 	if totalDiff > 0 && order.PaymentMethod == "Peshraft" {
// 		peshraftTxn := models.PeshraftTxn{
// 			ID:     uuid.NewString(),
// 			Amount: RoundTo2(totalDiff),
// 		}
// 		_, _, err := ProcessPeshraftTransaction(order.CardNumber, peshraftTxn.Amount, body.CashierID)
// 		if err != nil {
// 			c.JSON(http.StatusInternalServerError, gin.H{"error": "Ошибка при доплате через Peshraft", "details": err.Error()})
// 			return
// 		}
// 		order.PeshraftTransactions = append(order.PeshraftTransactions, peshraftTxn)
// 	} else if totalDiff < 0 && order.PaymentMethod == "Peshraft" {
// 		err := ProcessPeshraftRefund(order.PeshraftTransactionID, -totalDiff, body.CashierID)
// 		if err != nil {
// 			c.JSON(http.StatusInternalServerError, gin.H{"error": "Ошибка возврата через Peshraft", "details": err.Error()})
// 			return
// 		}
// 		order.PeshraftTransactions = []models.PeshraftTxn{{ID: order.PeshraftTransactionID, Amount: RoundTo2(order.TotalAmount)}}
// 	}

// 	order.UpdatedAt = time.Now()
// 	_, err = config.OrderCollection.UpdateOne(ctx, bson.M{"_id": order.ID}, bson.M{"$set": order})
// 	if err != nil {
// 		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update order"})
// 		return
// 	}

// 	c.JSON(http.StatusOK, gin.H{"message": "Order updated successfully"})
// }

func extractPeshraftTransactionID(response string) (string, error) {
	var resp struct {
		Transaction struct {
			ID string `json:"id"`
		} `json:"transaction"`
	}
	err := json.Unmarshal([]byte(response), &resp)
	if err != nil {
		return "", err
	}
	if resp.Transaction.ID == "" {
		return "", fmt.Errorf("missing transaction ID in response")
	}
	return resp.Transaction.ID, nil
}

func ReturnToStock(ctx context.Context, barcode string, qty float64) error {
	var product models.Product
	err := config.ProductCollection.FindOne(ctx, bson.M{"barcode": barcode}).Decode(&product)
	if err != nil {
		return fmt.Errorf("product not found: %s", barcode)
	}

	exps := product.ExpirationDate
	quants := product.Quantities
	if len(exps) == 0 {
		return fmt.Errorf("no batch expiration data for %s", barcode)
	}

	quants[0] += qty

	_, err = config.ProductCollection.UpdateOne(ctx,
		bson.M{"barcode": barcode},
		bson.M{"$set": bson.M{
			"quantities":     quants,
			"expirationdate": exps,
			"updated_at":     time.Now(),
		}},
	)
	return err
}

func DecreaseStock(ctx context.Context, barcode string, qty float64) error {
	var product models.Product
	err := config.ProductCollection.FindOne(ctx, bson.M{"barcode": barcode}).Decode(&product)
	if err != nil {
		return fmt.Errorf("product not found: %s", barcode)
	}

	sortedQuantities, sortedDates := sortBatchesByExpirationFloat(product.Quantities, product.ExpirationDate)
	remaining := qty
	for i := range sortedQuantities {
		if remaining <= 0 {
			break
		}
		used := minFloat(sortedQuantities[i], remaining)
		sortedQuantities[i] -= used
		remaining -= used
	}

	if remaining > 0 {
		return fmt.Errorf("not enough stock for %s", barcode)
	}

	_, err = config.ProductCollection.UpdateOne(ctx,
		bson.M{"barcode": barcode},
		bson.M{"$set": bson.M{
			"quantities":     sortedQuantities,
			"expirationdate": sortedDates,
			"updated_at":     time.Now(),
		}},
	)
	return err
}

func RoundTo2(v float64) float64 {
	return math.Round(v*100) / 100
}

func minFloat(a, b float64) float64 {
	if a < b {
		return a
	}
	return b
}

func sortBatchesByExpirationFloat(qs []float64, dates []string) ([]float64, []string) {
	type batch struct {
		qty  float64
		date string
	}
	var batches []batch
	for i := range qs {
		batches = append(batches, batch{qty: qs[i], date: dates[i]})
	}
	sort.SliceStable(batches, func(i, j int) bool {
		return batches[i].date < batches[j].date
	})
	var sortedQs []float64
	var sortedDates []string
	for _, b := range batches {
		sortedQs = append(sortedQs, b.qty)
		sortedDates = append(sortedDates, b.date)
	}
	return sortedQs, sortedDates
}
