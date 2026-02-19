package controllers

import (
	"backend/config"
	"backend/models"
	"backend/utils"
	"context"

	"net/http"

	// "strings"
	"time"

	"github.com/gin-gonic/gin"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
)

func ListStorekeepers(c *gin.Context) {
    ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
    defer cancel()

    cursor, err := config.StorekeeperCollection.Find(ctx, bson.M{"role": "storekeeper"})
    if err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"error": "Error retrieving storekeepers"})
        return
    }
    defer cursor.Close(ctx)

    var storekeeperReports []map[string]interface{}

    for cursor.Next(ctx) {
        var storekeeper models.Storekeeper
        if err := cursor.Decode(&storekeeper); err != nil {
            c.JSON(http.StatusInternalServerError, gin.H{"error": "Error decoding storekeeper"})
            return
        }

        // Custom logic can be added here if needed for storekeepers

        // Construct the storekeeper report
        storekeeperReport := map[string]interface{}{
            "fullname":   storekeeper.FirstName + " " + storekeeper.LastName,
            "phone":      storekeeper.Phone,
            "location":   storekeeper.Location,
            "birth_date": storekeeper.BirthDate,
            "id":         storekeeper.ID,
        }
        storekeeperReports = append(storekeeperReports, storekeeperReport)
    }

    if err := cursor.Err(); err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"error": "Error processing storekeepers"})
        return
    }

    c.JSON(http.StatusOK, storekeeperReports)
}

func AddStorekeeper(c *gin.Context) {
    var storekeeper models.Storekeeper
    if err := c.ShouldBindJSON(&storekeeper); err != nil {
        c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
        return
    }

    isUsed, err := isPhoneNumberInUse(storekeeper.Phone)
    if err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"error": "Error checking phone number"})
        return
    }
    if isUsed {
        c.JSON(http.StatusBadRequest, gin.H{"error": "Phone number already in use"})
        return
    }

    hashedPassword, err := utils.HashPassword(storekeeper.Password)
    if err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"error": "Error hashing password"})
        return
    }
    storekeeper.Password = hashedPassword

    storekeeper.ID = primitive.NewObjectID()
    storekeeper.Role = "storekeeper"

    ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
    defer cancel()

    _, err = config.StorekeeperCollection.InsertOne(ctx, storekeeper)
    if err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"error": "Error adding storekeeper"})
        return
    }

    c.JSON(http.StatusCreated, storekeeper)
}

func UpdateStorekeeper(c *gin.Context) {
    storekeeperID := c.Param("id")
    var updateData models.Storekeeper

    if err := c.ShouldBindJSON(&updateData); err != nil {
        c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
        return
    }

    update := bson.M{}

    if updateData.Phone != "" {
        isUsed, err := isPhoneNumberInUse(updateData.Phone)
        if err != nil {
            c.JSON(http.StatusInternalServerError, gin.H{"error": "Error checking phone number"})
            return
        }
        if isUsed {
            var existingStorekeeper models.Storekeeper
            err = config.StorekeeperCollection.FindOne(context.TODO(), bson.M{"phone": updateData.Phone}).Decode(&existingStorekeeper)
            if err == nil && existingStorekeeper.ID.Hex() != storekeeperID {
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

    oid, err := primitive.ObjectIDFromHex(storekeeperID)
    if err != nil {
        c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid storekeeper ID"})
        return
    }

    _, err = config.StorekeeperCollection.UpdateOne(
        ctx,
        bson.M{"_id": oid},
        bson.M{"$set": update},
    )
    if err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"error": "Error updating storekeeper"})
        return
    }

    c.JSON(http.StatusOK, gin.H{"message": "Storekeeper updated successfully"})
}

func GetStorekeeper(c *gin.Context) {
    storekeeperID := c.Param("id")

    oid, err := primitive.ObjectIDFromHex(storekeeperID)
    if err != nil {
        c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid storekeeper ID"})
        return
    }

    var storekeeper models.Storekeeper
    ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
    defer cancel()

    err = config.StorekeeperCollection.FindOne(ctx, bson.M{"_id": oid}).Decode(&storekeeper)
    if err != nil {
        c.JSON(http.StatusNotFound, gin.H{"error": "Storekeeper not found"})
        return
    }

    c.JSON(http.StatusOK, gin.H{
        "first_name": storekeeper.FirstName,
        "last_name":  storekeeper.LastName,
        "birth_date": storekeeper.BirthDate,
        "phone":      storekeeper.Phone,
        "location":   storekeeper.Location,
    })
}

func DeleteStorekeeper(c *gin.Context) {
    storekeeperID := c.Param("id")

    oid, err := primitive.ObjectIDFromHex(storekeeperID)
    if err != nil {
        c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid storekeeper ID"})
        return
    }

    ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
    defer cancel()

    _, err = config.StorekeeperCollection.DeleteOne(ctx, bson.M{"_id": oid})
    if err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"error": "Error deleting storekeeper"})
        return
    }

    c.JSON(http.StatusOK, gin.H{"message": "Storekeeper deleted successfully"})
}


func GetStorekeeperLocations(c *gin.Context) {
    ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
    defer cancel()

    pipeline := mongo.Pipeline{
        bson.D{{"$group", bson.D{{"_id", "$location"}}}},
        bson.D{{"$project", bson.D{{"location", "$_id"}, {"_id", 0}}}},
        bson.D{{"$sort", bson.D{{"location", -1}}}}, // Сортировка по убыванию
    }

    cursor, err := config.StorekeeperCollection.Aggregate(ctx, pipeline)
    if err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch storekeeper locations"})
        return
    }
    defer cursor.Close(ctx)

    var locationDocs []struct {
        Location string `bson:"location"`
    }
    if err = cursor.All(ctx, &locationDocs); err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to decode locations"})
        return
    }

    var locations []string
    for _, doc := range locationDocs {
        locations = append(locations, doc.Location)
    }

    c.JSON(http.StatusOK, locations)
}

func GetDeliveryLocations(c *gin.Context) {
    deliveryLocations := []gin.H{
        {"location": "Вамар", "cost": 10},
        {"location": "Дерзуд", "cost": 15},
        {"location": "Барушан", "cost": 20},
        {"location": "Пастхуф", "cost": 20},
        {"location": "Шучанд", "cost": 15},
    }

    c.JSON(http.StatusOK, deliveryLocations)
}

func AddOperator(c *gin.Context) {
	var operator models.Operator
	if err := c.ShouldBindJSON(&operator); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	isUsed, err := isPhoneNumberInUse(operator.Phone)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Error checking phone number"})
		return
	}
	if isUsed {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Phone number already in use"})
		return
	}

	hashedPassword, err := utils.HashPassword(operator.Password)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Error hashing password"})
		return
	}
	operator.Password = hashedPassword

	operator.ID = primitive.NewObjectID()
	operator.Role = "operator"

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	_, err = config.OperatorCollection.InsertOne(ctx, operator)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Error adding operator"})
		return
	}

	c.JSON(http.StatusCreated, operator)
}


func ListOperators(c *gin.Context) {
    ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
    defer cancel()

    cursor, err := config.OperatorCollection.Find(ctx, bson.M{"role": "operator"})
    if err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"error": "Error retrieving storekeepers"})
        return
    }
    defer cursor.Close(ctx)

    var operatorReports []map[string]interface{}

    for cursor.Next(ctx) {
        var operator models.Operator
        if err := cursor.Decode(&operator); err != nil {
            c.JSON(http.StatusInternalServerError, gin.H{"error": "Error decoding storekeeper"})
            return
        }

        // Custom logic can be added here if needed for storekeepers

        // Construct the storekeeper report
        operatorReport := map[string]interface{}{
            "fullname":   operator.FirstName + " " + operator.LastName,
            "phone":      operator.Phone,
            "birth_date": operator.BirthDate,
            "id":         operator.ID,
        }
        operatorReports = append(operatorReports, operatorReport)
    }

    if err := cursor.Err(); err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"error": "Error processing storekeepers"})
        return
    }

    c.JSON(http.StatusOK, operatorReports)
}