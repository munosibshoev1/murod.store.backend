package controllers

import (
	"backend/config"
	"backend/models"
	"context"
	"strings"

	// "encoding/json"
	"fmt"

	"math/rand"
	"net/http"
	"os"

	// "github.com/shopspring/decimal"
	"github.com/gin-gonic/gin"
	"github.com/minio/minio-go/v7"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// CreateCategory добавляет новую категорию в базу данных
func CreateCategory(c *gin.Context) {
	category := models.Category{
		ID: primitive.NewObjectID(),
	}

	categoryID, err := generateCategoryID()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to generate category ID"})
		return
	}
	category.CategoryID = categoryID

	category.Name = c.PostForm("name")
	category.TopCategoryID = c.PostForm("topcategoryid")

	if category.Name == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Name is required"})
		return
	}

	file, err := c.FormFile("categoryphoto")
	if err == nil {
		photoURL, err := UploadCategoryPhotoToS3(file, category.ID.Hex())
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		category.PhotoURL = photoURL
	}

	_, err = config.CategoryCollection.InsertOne(context.TODO(), category)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create category"})
		return
	}

	c.JSON(http.StatusCreated, gin.H{"message": "Category created successfully", "categoryid": category.CategoryID, "photo_url": category.PhotoURL})
}


// EditCategory - редактирование категории по ID
func EditCategory(c *gin.Context) {
	categoryID := c.Param("id")
	objID, err := primitive.ObjectIDFromHex(categoryID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid category ID"})
		return
	}

	var existingCategory models.Category
	err = config.CategoryCollection.FindOne(context.TODO(), bson.M{"_id": objID}).Decode(&existingCategory)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Category not found"})
		return
	}

	updateFields := bson.M{}
	if name := c.PostForm("name"); name != "" {
		updateFields["name"] = name
	}
	if topCategoryID := c.PostForm("topcategoryid"); topCategoryID != "" {
		updateFields["topcategoryid"] = topCategoryID
	}

	file, err := c.FormFile("categoryphoto")
	if err == nil {
		// Удаление старого изображения из S3
		if existingCategory.PhotoURL != "" && strings.Contains(existingCategory.PhotoURL, cdnDomain) {
			oldKey := strings.TrimPrefix(existingCategory.PhotoURL, fmt.Sprintf("https://%s/", cdnDomain))
			s3Client.RemoveObject(context.Background(), s3Bucket, oldKey, minio.RemoveObjectOptions{})
		}

		photoURL, err := UploadCategoryPhotoToS3(file, objID.Hex())
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		updateFields["photourl"] = photoURL
	}

	if len(updateFields) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "No fields to update"})
		return
	}

	filter := bson.M{"_id": objID}
	update := bson.M{"$set": updateFields}

	_, err = config.CategoryCollection.UpdateOne(context.TODO(), filter, update)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update category"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Category updated successfully"})
}

// DeleteCategory - удаление категории по ID и её фото
func DeleteCategory(c *gin.Context) {
	categoryID := c.Param("id")
	objID, err := primitive.ObjectIDFromHex(categoryID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid category ID"})
		return
	}

	// Находим категорию перед удалением, чтобы получить путь к фото
	var category models.Category
	err = config.CategoryCollection.FindOne(context.TODO(), bson.M{"_id": objID}).Decode(&category)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Category not found"})
		return
	}

	// Удаление файла фото, если он существует
	if category.PhotoURL != "" {
		photoPath := "./uploads/categories/" + category.PhotoURL
		if _, err := os.Stat(photoPath); err == nil {
			os.Remove(photoPath)
		}
	}

	// Удаление категории из базы данных
	_, err = config.CategoryCollection.DeleteOne(context.TODO(), bson.M{"_id": objID})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete category"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Category and its photo deleted successfully"})
}

// GetCategory - получение категории по ID
func GetCategory(c *gin.Context) {
	categoryID := c.Param("id")
	objID, err := primitive.ObjectIDFromHex(categoryID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid category ID"})
		return
	}

	var category models.Category
	err = config.CategoryCollection.FindOne(context.TODO(), bson.M{"_id": objID}).Decode(&category)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Category not found"})
		return
	}

	// Возвращаем категорию с URL фото, если оно есть
	c.JSON(http.StatusOK, gin.H{
		"id":            category.ID.Hex(),
		"categoryid":    category.CategoryID,
		"name":          category.Name,
		"topcategoryid": category.TopCategoryID,
		"photourl":      category.PhotoURL,
	})
}

// GetAllCategories - получение всех категорий
func GetAllCategories(c *gin.Context) {
	cursor, err := config.CategoryCollection.Find(context.TODO(), bson.M{})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch categories"})
		return
	}
	defer cursor.Close(context.TODO())

	var categories []models.Category
	if err = cursor.All(context.TODO(), &categories); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to decode categories"})
		return
	}

	// Формируем список категорий с их фото
	response := []gin.H{}
	for _, category := range categories {
		response = append(response, gin.H{
			"id":            category.ID.Hex(),
			"categoryid":    category.CategoryID,
			"name":          category.Name,
			"topcategoryid": category.TopCategoryID,
			"photourl":      category.PhotoURL,
		})
	}

	c.JSON(http.StatusOK, response)
}

// GetAllCategories - получение всех основных категорий
func GetAllCategories1(c *gin.Context) {
	// Фильтр для поиска только основных категорий
	filter := bson.M{"$or": []bson.M{
		{"topcategoryid": bson.M{"$eq": "000"}}, // TopCategoryID равно "000"
		{"topcategoryid": bson.M{"$eq": ""}},    // TopCategoryID пустое
	}}

	cursor, err := config.CategoryCollection.Find(context.TODO(), filter)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch categories"})
		return
	}
	defer cursor.Close(context.TODO())

	var categories []models.Category
	if err = cursor.All(context.TODO(), &categories); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to decode categories"})
		return
	}

	// Формируем список основных категорий
	response := []gin.H{}
	for _, category := range categories {
		response = append(response, gin.H{
			"id":            category.ID.Hex(),
			"categoryid":    category.CategoryID,
			"name":          category.Name,
			"topcategoryid": category.TopCategoryID,
			"photourl":      category.PhotoURL,
		})
	}

	c.JSON(http.StatusOK, response)
}

func generateCategoryID() (string, error) {
	const idLength = 3

	// Максимальная попытка для генерации уникального ID
	const maxAttempts = 10

	for i := 0; i < maxAttempts; i++ {
		// Генерация случайного 3-значного числа
		randomID := fmt.Sprintf("%03d", rand.Intn(1000))

		// Проверяем, существует ли уже категория с таким ID
		count, err := config.CategoryCollection.CountDocuments(context.TODO(), bson.M{"categoryid": randomID})
		if err != nil {
			return "", fmt.Errorf("failed to check category ID uniqueness: %v", err)
		}

		// Если такой ID не найден, возвращаем его
		if count == 0 {
			return randomID, nil
		}
	}
	return "", fmt.Errorf("failed to generate unique category ID after %d attempts", maxAttempts)
}
