package controllers

import (
	"backend/config"
	"backend/models"
	"bytes"
	"context"
	"encoding/json"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"

	// "fmt"
	// "strconv"
	"time"
	// "github.com/shopspring/decimal"
	"github.com/gin-gonic/gin"
	"github.com/minio/minio-go/v7"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	// "go.mongodb.org/mongo-driver/mongo"
	// "go.mongodb.org/mongo-driver/mongo/options"
)

// GetAllProducts - получение списка всех товаров
func GetAllProductsTemplate(c *gin.Context) {
    cursor, err := config.ProductTemplateCollection.Find(context.TODO(), bson.M{})
    if err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch products"})
        return
    }
    defer cursor.Close(context.TODO())

    var products []models.ProductTemplate
    if err = cursor.All(context.TODO(), &products); err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to decode products", "details": err.Error()})
        return
    }

    c.JSON(http.StatusOK, products)
}



func AddProductTemplate(c *gin.Context) {
    product := models.ProductTemplate{
        ID: primitive.NewObjectID(),
    }

    // Получаем и заполняем данные из form-data
    product.CategoryID = c.PostForm("categoryid")
    product.Name = c.PostForm("name")
    product.Unm = c.PostForm("unm")
    minimumOrderStr := c.PostForm("minimumorder")
    product.Barcode = c.PostForm("barcode")
	grossWeightStr := c.PostForm("grossweight")

    // Преобразование строки minimumorder в int
    minimumOrder, err := strconv.Atoi(minimumOrderStr)
    if err != nil || minimumOrder <= 0 {
        c.JSON(http.StatusBadRequest, gin.H{"error": "Minimumorder must be a positive integer"})
        return
    }
    product.MinimumOrder = minimumOrder

	grossWeight, err := strconv.ParseFloat(grossWeightStr, 64)
	if err != nil || grossWeight <= 0 {
    	c.JSON(http.StatusBadRequest, gin.H{"error": "grossweight must be a positive number"})
    	return
	}
	product.Grossweight = grossWeight
    // Проверка наличия всех необходимых данных
    if product.CategoryID == "" || product.Name == "" || product.Unm == "" || product.Barcode == "" {
        c.JSON(http.StatusBadRequest, gin.H{"error": "All fields are required"})
        return
    }

    // Проверка уникальности barcode
    ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
    defer cancel()

    var existingProduct models.ProductTemplate
    err = config.ProductTemplateCollection.FindOne(ctx, bson.M{"barcode": product.Barcode}).Decode(&existingProduct)
    if err == nil {
        // Если штрих-код уже существует
        c.JSON(http.StatusBadRequest, gin.H{"error": "Barcode must be unique"})
        return
    }
    if err != nil && err != mongo.ErrNoDocuments {
        // Ошибка подключения или другая ошибка базы данных
        c.JSON(http.StatusInternalServerError, gin.H{"error": "Error checking barcode uniqueness"})
        return
    }

    // Сохранение фото товара
    file, err := c.FormFile("productphoto")
    if err == nil {
		photoURL, previewURL, err := SaveProductPhotoToS3(c, file, product.ID.Hex())
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		product.Productphotourl = photoURL
		product.Productphotopreviewurl = previewURL
	}

	_, err = config.ProductTemplateCollection.InsertOne(ctx, product)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Error adding product"})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"message": "Product created successfully",
		"photo_url": product.Productphotourl,
		"preview_url": product.Productphotopreviewurl,
	})
}




// GetProduct - получение товара по ID
func GetProductTemplate(c *gin.Context) {
	productID := c.Param("id")
	objID, err := primitive.ObjectIDFromHex(productID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid product ID"})
		return
	}

	var product models.ProductTemplate
	err = config.ProductTemplateCollection.FindOne(context.TODO(), bson.M{"_id": objID}).Decode(&product)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Product not found"})
		return
	}

	c.JSON(http.StatusOK, product)
}

func EditProductTemplate(c *gin.Context) {
	productID := c.Param("id")
	objID, err := primitive.ObjectIDFromHex(productID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid product ID"})
		return
	}

	var existingProduct models.ProductTemplate
	err = config.ProductTemplateCollection.FindOne(context.TODO(), bson.M{"_id": objID}).Decode(&existingProduct)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Product not found"})
		return
	}

	updateFields := bson.M{}
	productUpdateFields := bson.M{}
	posUpdateFields := bson.M{}
	oldBarcode := existingProduct.Barcode
	shouldSyncToPOS := false

	if name := c.PostForm("name"); name != "" {
		updateFields["name"] = name
		productUpdateFields["name"] = name
		posUpdateFields["name"] = name
		shouldSyncToPOS = true
	}
	if categoryID := c.PostForm("categoryid"); categoryID != "" {
		updateFields["categoryid"] = categoryID
		productUpdateFields["categoryid"] = categoryID
		posUpdateFields["categoryid"] = categoryID
		shouldSyncToPOS = true
	}
	if unm := c.PostForm("unm"); unm != "" {
		updateFields["unm"] = unm
		productUpdateFields["unm"] = unm
		posUpdateFields["unm"] = unm
		shouldSyncToPOS = true
	}
	if minimumOrderStr := c.PostForm("minimumorder"); minimumOrderStr != "" {
		minimumOrder, err := strconv.Atoi(minimumOrderStr)
		if err != nil || minimumOrder <= 0 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Minimumorder must be a positive integer"})
			return
		}
		updateFields["minimumorder"] = minimumOrder
		productUpdateFields["minimumorder"] = minimumOrderStr
	}
	if grossWeightStr := c.PostForm("grossweight"); grossWeightStr != "" {
		grossWeight, err := strconv.ParseFloat(grossWeightStr, 64)
		if err != nil || grossWeight <= 0 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Grossweight must be a positive number"})
			return
		}
		updateFields["grossweight"] = grossWeight
		productUpdateFields["grossweight"] = grossWeight
		posUpdateFields["grossweight"] = grossWeight
		if grossWeight != existingProduct.Grossweight {
			shouldSyncToPOS = true
		}
	}
	if barcode := c.PostForm("barcode"); barcode != "" {
		updateFields["barcode"] = barcode
		productUpdateFields["barcode"] = barcode
		posUpdateFields["old_barcode"] = oldBarcode
		posUpdateFields["barcode"] = barcode

		if barcode != existingProduct.Barcode {
			shouldSyncToPOS = true
			ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			defer cancel()

			var tempProduct models.ProductTemplate
			err = config.ProductTemplateCollection.FindOne(ctx, bson.M{"barcode": barcode}).Decode(&tempProduct)
			if err == nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": "Barcode must be unique"})
				return
			}
			if err != mongo.ErrNoDocuments {
				c.JSON(http.StatusInternalServerError, gin.H{"error": "Error checking barcode uniqueness"})
				return
			}

			history := existingProduct.BarcodeHistory
			if history == nil {
				history = []models.BarcodeHistory{}
			}

			alreadyExists := false
			for _, h := range history {
				if h.Barcode == oldBarcode {
					alreadyExists = true
					break
				}
			}
			if !alreadyExists && oldBarcode != "" {
				history = append(history, models.BarcodeHistory{
					Barcode:   oldBarcode,
					ChangedAt: time.Now(),
				})
				updateFields["barcodehistory"] = history
			}
		}
	}

	file, err := c.FormFile("productphoto")
	if err == nil {
		if existingProduct.Productphotourl != "" && strings.Contains(existingProduct.Productphotourl, cdnDomain) {
			if parts := strings.Split(existingProduct.Productphotourl, "/"); len(parts) > 0 {
				oldKey := strings.Join(parts[len(parts)-2:], "/")
				s3Client.RemoveObject(context.Background(), s3Bucket, oldKey, minio.RemoveObjectOptions{})
			}
		}
		if existingProduct.Productphotopreviewurl != "" && strings.Contains(existingProduct.Productphotopreviewurl, cdnDomain) {
			if parts := strings.Split(existingProduct.Productphotopreviewurl, "/"); len(parts) > 0 {
				oldPreviewKey := strings.Join(parts[len(parts)-2:], "/")
				s3Client.RemoveObject(context.Background(), s3Bucket, oldPreviewKey, minio.RemoveObjectOptions{})
			}
		}
		photoURL, previewURL, err := SaveProductPhotoToS3(c, file, objID.Hex())
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		updateFields["productphotourl"] = photoURL
		updateFields["productphotopreviewurl"] = previewURL
		productUpdateFields["productphotourl"] = photoURL
		productUpdateFields["productphotopreviewurl"] = previewURL
	}

	if len(updateFields) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "No fields to update"})
		return
	}

	filter := bson.M{"_id": objID}
	update := bson.M{"$set": updateFields}
	_, err = config.ProductTemplateCollection.UpdateOne(context.TODO(), filter, update)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update product template"})
		return
	}

	if oldBarcode != "" && len(productUpdateFields) > 0 {
		// Update related products collection
		prodFilter := bson.M{"barcode": oldBarcode}
		prodUpdate := bson.M{"$set": productUpdateFields}
		_, err := config.ProductCollection.UpdateMany(context.TODO(), prodFilter, prodUpdate)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update products based on template changes"})
			return
		}

		// 🔁 Update supplierorders.product[].* fields
		supplierOrderFilter := bson.M{"products.barcode": oldBarcode}
		updateArray := bson.M{}

		if name, ok := productUpdateFields["name"]; ok {
			updateArray["products.$[elem].name"] = name
		}
		if barcode, ok := productUpdateFields["barcode"]; ok {
			updateArray["products.$[elem].barcode"] = barcode
		}
		if categoryid, ok := productUpdateFields["categoryid"]; ok {
			updateArray["products.$[elem].categoryid"] = categoryid
		}
		if unm, ok := productUpdateFields["unm"]; ok {
			updateArray["products.$[elem].unm"] = unm
		}
		if minOrderStr, ok := productUpdateFields["minimumorder"].(string); ok {
			if minOrder, err := strconv.Atoi(minOrderStr); err == nil {
			updateArray["products.$[elem].minimumorder"] = minOrder
		}
		}
		if grossweight, ok := productUpdateFields["grossweight"]; ok {
			updateArray["products.$[elem].grossweight"] = grossweight
		}
			
		if len(updateArray) > 0 {
			arrayUpdate := bson.M{"$set": updateArray}
			arrayOptions := options.Update().SetArrayFilters(options.ArrayFilters{
				Filters: []interface{}{bson.M{"elem.barcode": oldBarcode}},
			})

			_, err := config.SupplierOrderCollection.UpdateMany(
				context.TODO(),
				supplierOrderFilter,
				arrayUpdate,
				arrayOptions,
			)
			if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update supplier orders"})
				return
			}
		}
	}

	if shouldSyncToPOS {
		go SyncProductStructureToPOS(posUpdateFields)
	}

	c.JSON(http.StatusOK, gin.H{"message": "Product template and related products updated successfully"})
}


func SyncProductStructureToPOS(payload map[string]interface{}) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	apiKey, err := GetShopAPIKeyPOS(ctx)
	if err != nil {
		log.Printf("[ERROR] POS sync failed (GetShopAPIKey): %v", err)
		return
	}

	jsonData, _ := json.Marshal(payload)
	req, err := http.NewRequestWithContext(ctx, "POST", "https://bpos.nadim.shop/api/update-product-structure", bytes.NewBuffer(jsonData))
	if err != nil {
		log.Printf("[ERROR] POS sync request creation failed: %v", err)
		return
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-API-Key", apiKey)

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil || resp.StatusCode != http.StatusOK {
		log.Printf("[ERROR] POS sync failed. Status: %d, Err: %v", resp.StatusCode, err)
		return
	}

	log.Printf("[OK] POS product structure sync success for barcode: %v", payload["barcode"])
}





// DeleteProduct - удаление товара по ID и его фото
func DeleteProductTemplate(c *gin.Context) {
	productID := c.Param("id")
	objID, err := primitive.ObjectIDFromHex(productID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid product ID"})
		return
	}

	// Находим товар перед удалением, чтобы получить путь к фото
	var product models.ProductTemplate
	err = config.ProductTemplateCollection.FindOne(context.TODO(), bson.M{"_id": objID}).Decode(&product)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Product not found"})
		return
	}

	// Удаление файла фото, если он существует
	if product.Productphotourl != "" {
		photoPath := "./uploads/products/" + product.Productphotourl
		if _, err := os.Stat(photoPath); err == nil {
			err := os.Remove(photoPath)
			if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete product photo"})
				return
			}
		}
	}

	// Удаление товара из базы данных
	_, err = config.ProductTemplateCollection.DeleteOne(context.TODO(), bson.M{"_id": objID})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete product"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Product and its photo deleted successfully"})
}