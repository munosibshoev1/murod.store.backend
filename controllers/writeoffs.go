package controllers

import (
	"backend/config"
	"math"

	// "backend/handlers"
	"backend/models"
	"backend/utils"
	"context"

	"fmt"
	"net/http"

	// "strings"
	"time"

	"github.com/gin-gonic/gin"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// WriteOffProductsNEW - складовщик создает списание, batches заполняются, остаток не уменьшается
func WriteOffProductsNEW(c *gin.Context) {
	roleRaw, hasRole := c.Get("role")
	if !hasRole {
		c.JSON(http.StatusForbidden, gin.H{"error": "role not found"})
		return
	}
	role := roleRaw.(string)

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

		sortedQuantities, sortedExpDates := sortBatchesByExpirationFloat(product.Quantities, product.ExpirationDate)
		usedBatches := []models.BatchUsage{}
		remaining := item.Quantity

		for i := 0; i < len(sortedQuantities) && remaining > 0; i++ {
			if sortedQuantities[i] > 0 {
				used := minFloat(sortedQuantities[i], remaining)
				usedBatches = append(usedBatches, models.BatchUsage{
					ExpirationDate: sortedExpDates[i],
					UsedQuantity:   used,
				})
				remaining -= used
			}
		}

		if remaining > 0 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Not enough stock to plan write-off", "barcode": item.Barcode})
			return
		}

		// Только если admin - уменьшаем реальные остатки
		remainingStock := sumQuantitiesFloat(product.Quantities)
		if role == "admin" {
			// Списываем из actual product quantities
			remaining := item.Quantity
			for i := 0; i < len(sortedQuantities) && remaining > 0; i++ {
				if sortedQuantities[i] > 0 {
					used := minFloat(sortedQuantities[i], remaining)
					sortedQuantities[i] -= used
					remaining -= used
				}
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

			remainingStock = sumQuantitiesFloat(sortedQuantities)
		}

		purchasePrice := product.Purchaseprice
		writeOffValue := math.Round(item.Quantity*purchasePrice*100) / 100
		totalWriteOffValue += writeOffValue

		writeOffItems = append(writeOffItems, models.WriteOffItem{
			Barcode:        item.Barcode,
			Quantity:       item.Quantity,
			PurchasePrice:  purchasePrice,
			WriteOffValue:  writeOffValue,
			Comment:        item.Comment,
			Batches:        usedBatches,
			Status:         map[string]string{"admin": "Списан", "storekeeper": "В ожидании списания"}[role],
			RemainingStock: remainingStock,
		})
	}

	doc := models.WriteOffDocument{
		ID:         primitive.NewObjectID(),
		Products:   writeOffItems,
		TotalValue: totalWriteOffValue,
		CreatedAt:  time.Now(),
		Status:     map[string]string{"admin": "Списан", "storekeeper": "В ожидании списания"}[role],
	}

	_, err := config.WriteOffCollection.InsertOne(context.TODO(), doc)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create write-off document"})
		return
	}

	if role == "storekeeper" {
		adminPhone := "+992111143040"
		message := fmt.Sprintf("Складовщик оформил списание на сумму %.2f сомонӣ. Подтвердите в системе.", totalWriteOffValue)
		utils.SendSMS(removePlusFromPhone(adminPhone), message)
	}

	c.JSON(http.StatusOK, gin.H{"message": "Write-off created"})
}

// ConfirmWriteOff - админ подтверждает списание и уменьшает склад по batches или сам рассчитывает их при необходимости
func ConfirmWriteOff(c *gin.Context) {
	roleRaw, hasRole := c.Get("role")
	if !hasRole {
		c.JSON(http.StatusForbidden, gin.H{"error": "role not found"})
		return
	}
	role := roleRaw.(string)

	if role != "admin" {
		c.JSON(http.StatusForbidden, gin.H{"error": "Only admin can confirm write-offs"})
		return
	}

	writeOffID := c.Param("id")
	objID, err := primitive.ObjectIDFromHex(writeOffID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid document ID"})
		return
	}

	var doc models.WriteOffDocument
	err = config.WriteOffCollection.FindOne(context.TODO(), bson.M{"_id": objID}).Decode(&doc)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Document not found"})
		return
	}

	if doc.Status == "Списан" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Document already confirmed"})
		return
	}

	var input []models.WriteOffItem
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	var updateItems []models.WriteOffItem
	total := 0.0

	for _, item := range input {
		if item.Status != "Confirm" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "All items must be confirmed"})
			return
		}

		var product models.Product
		err := config.ProductCollection.FindOne(context.TODO(), bson.M{"barcode": item.Barcode}).Decode(&product)
		if err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "Product not found", "barcode": item.Barcode})
			return
		}

		sortedQuantities, sortedExpDates := sortBatchesByExpirationFloat(product.Quantities, product.ExpirationDate)
		remaining := item.Quantity

		var originalItem *models.WriteOffItem
		for _, prod := range doc.Products {
			if prod.Barcode == item.Barcode {
				originalItem = &prod
				break
			}
		}
		if originalItem == nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Original product not found", "barcode": item.Barcode})
			return
		}

		if len(item.Batches) == 0 && len(originalItem.Batches) > 0 {
			item.Batches = originalItem.Batches
		}

		if item.Quantity == 0 {
			item.Quantity = originalItem.Quantity
		}

		if item.PurchasePrice == 0 {
			item.PurchasePrice = product.Purchaseprice
		}

		if item.WriteOffValue == 0 {
			item.WriteOffValue = math.Round(item.Quantity*item.PurchasePrice*100) / 100
		}

		if len(item.Batches) == 0 {
			item.Batches = []models.BatchUsage{}
			for i := 0; i < len(sortedQuantities) && remaining > 0; i++ {
				if sortedQuantities[i] > 0 {
					used := minFloat(sortedQuantities[i], remaining)
					item.Batches = append(item.Batches, models.BatchUsage{
						ExpirationDate: sortedExpDates[i],
						UsedQuantity:   used,
					})
					sortedQuantities[i] -= used
					remaining -= used
				}
			}
		} else {
			for _, batch := range item.Batches {
				for i := 0; i < len(sortedExpDates); i++ {
					if sortedExpDates[i] == batch.ExpirationDate {
						used := minFloat(sortedQuantities[i], batch.UsedQuantity)
						sortedQuantities[i] -= used
						remaining -= used
						break
					}
				}
			}
		}

		if remaining > 0 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Batch info invalid or insufficient stock", "barcode": item.Barcode})
			return
		}

		remainingStock := sumQuantitiesFloat(sortedQuantities)

		_, err = config.ProductCollection.UpdateOne(context.TODO(), bson.M{"barcode": item.Barcode}, bson.M{
			"$set": bson.M{
				"quantities":     sortedQuantities,
				"expirationdate": sortedExpDates,
				"updated_at":     time.Now(),
			},
		})
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update product stock", "barcode": item.Barcode})
			return
		}

		item.Status = "Списан"
		item.RemainingStock = remainingStock
		total += item.WriteOffValue
		updateItems = append(updateItems, item)
	}

	total = math.Round(total*100) / 100

	_, err = config.WriteOffCollection.UpdateOne(context.TODO(), bson.M{"_id": objID}, bson.M{
		"$set": bson.M{
			"products":     updateItems,
			"total_value":  total,
			"status":       "Списан",
			"updated_at":   time.Now(),
			"confirmed_by": role,
		},
	})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update document"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Write-off confirmed and stock updated"})
}

func UpdateWriteOffDraft(c *gin.Context) {
	roleRaw, hasRole := c.Get("role")
	if !hasRole || roleRaw.(string) != "storekeeper" {
		c.JSON(http.StatusForbidden, gin.H{"error": "Access denied"})
		return
	}

	writeOffID := c.Param("id")
	objID, err := primitive.ObjectIDFromHex(writeOffID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid document ID"})
		return
	}

	var existing models.WriteOffDocument
	err = config.WriteOffCollection.FindOne(context.TODO(), bson.M{"_id": objID}).Decode(&existing)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Document not found"})
		return
	}

	if existing.Status == "Списан" {
		c.JSON(http.StatusForbidden, gin.H{"error": "Confirmed write-offs cannot be modified"})
		return
	}

	var rawItems []map[string]interface{}
	if err := c.ShouldBindJSON(&rawItems); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	var updatedProducts []models.WriteOffItem
	total := 0.0
	for _, item := range rawItems {
		barcode := item["barcode"].(string)
		quantity := item["qua"].(float64)
		comment := ""
		status := ""

		if v, ok := item["comment"].(string); ok {
			comment = v
		}
		if v, ok := item["status"].(string); ok {
			status = v
		}

		var product models.Product
		err := config.ProductCollection.FindOne(context.TODO(), bson.M{"barcode": barcode}).Decode(&product)
		if err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "Product not found", "barcode": barcode})
			return
		}

		purchasePrice := product.Purchaseprice
		writeOffValue := math.Round(quantity*purchasePrice*100) / 100
		remainingStock := sumQuantitiesFloat(product.Quantities)

		updatedProducts = append(updatedProducts, models.WriteOffItem{
			Barcode:        barcode,
			Quantity:       quantity,
			Comment:        comment,
			Status:         status,
			PurchasePrice:  purchasePrice,
			WriteOffValue:  writeOffValue,
			RemainingStock: remainingStock,
		})
		total += writeOffValue
	}

	update := bson.M{
		"$set": bson.M{
			"products":    updatedProducts,
			"total_value": total, // обновлённое поле в нужном формате
			"updated_at":  time.Now(),
		},
	}

	_, err = config.WriteOffCollection.UpdateOne(context.TODO(), bson.M{"_id": objID}, update)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update write-off document"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Write-off draft updated successfully"})
}

func GetWriteOffDocumentByID(c *gin.Context) {
	id := c.Param("id")
	docID, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid document ID"})
		return
	}

	var doc models.WriteOffDocument
	err = config.WriteOffCollection.FindOne(context.TODO(), bson.M{"_id": docID}).Decode(&doc)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Document not found"})
		return
	}

	type Batch struct {
		ExpirationDate string  `json:"expiration_date"`
		UsedQuantity   float64 `json:"used_quantity"`
	}
	type ExtendedProduct struct {
		ID              primitive.ObjectID ` json:"id,omitempty"`
		Barcode         string             `json:"barcode"`
		Quantity        float64            `json:"quantity"` // остаток на складе
		Qua             float64            `json:"qua"`      // сколько списано
		PurchasePrice   float64            `json:"purchaseprice"`
		WriteOffValue   float64            `json:"write_off_value"`
		TotalPrice      float64            `json:"total_price"`
		Comment         string             `json:"comment"`
		Status          string             `json:"status"`
		Name            string             `json:"name"`
		Unm             string             `json:"unm"`
		ProductPhotoURL string             `json:"productphotourl"`
		Batches         []Batch            `json:"batches"`
	}

	var extendedProducts []ExtendedProduct

	for _, p := range doc.Products {
		var product models.Product
		err := config.ProductCollection.FindOne(context.TODO(), bson.M{"barcode": p.Barcode}).Decode(&product)
		totalPrice := p.Quantity * p.PurchasePrice
		if err != nil {
			extendedProducts = append(extendedProducts, ExtendedProduct{

				Barcode:       p.Barcode,
				Quantity:      p.RemainingStock,
				Qua:           p.Quantity,
				PurchasePrice: p.PurchasePrice,
				WriteOffValue: p.WriteOffValue,
				TotalPrice:    totalPrice,
				Comment:       p.Comment,
				Status:        p.Status,
				Name:          "Unknown",
				Unm:           "Unknown",
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
			continue
		}

		extendedProducts = append(extendedProducts, ExtendedProduct{
			Barcode:         p.Barcode,
			Quantity:        p.RemainingStock,
			Qua:             p.Quantity,
			PurchasePrice:   p.PurchasePrice,
			WriteOffValue:   p.WriteOffValue,
			TotalPrice:      totalPrice,
			Comment:         p.Comment,
			Status:          p.Status,
			Name:            product.Name,
			Unm:             product.Unm,
			ProductPhotoURL: product.Productphotourl,
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

	extendedDoc := struct {
		ID         primitive.ObjectID `json:"id"`
		Products   []ExtendedProduct  `json:"products"`
		TotalValue float64            `json:"total_value"`
		CreatedAt  time.Time          `json:"created_at"`
	}{
		ID:         doc.ID,
		Products:   extendedProducts,
		TotalValue: doc.TotalValue,
		CreatedAt:  doc.CreatedAt,
	}

	c.JSON(http.StatusOK, extendedDoc)
}

func GetWriteOffDocuments(c *gin.Context) {
	cursor, err := config.WriteOffCollection.Find(context.TODO(), bson.M{})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch documents"})
		return
	}
	defer cursor.Close(context.TODO())

	var results []models.WriteOffDocument
	if err := cursor.All(context.TODO(), &results); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to decode documents"})
		return
	}

	var summaryList []map[string]interface{}
	for _, doc := range results {
		summaryList = append(summaryList, map[string]interface{}{
			"id":             doc.ID.Hex(),
			"created_at":     doc.CreatedAt,
			"total_products": len(doc.Products),
			"total_value":    doc.TotalValue,
			"status":         doc.Status,
		})
	}

	c.JSON(http.StatusOK, summaryList)
}
