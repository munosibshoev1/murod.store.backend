package middleware

import (
	"backend/config"
	"backend/models"
	"backend/utils"
	"context"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
)
func AuthMiddleware(role string) gin.HandlerFunc {
    return func(c *gin.Context) {
        token, err := c.Cookie("token")
        if err != nil {
            authHeader := c.GetHeader("Authorization")
            if authHeader == "" {
                c.JSON(http.StatusUnauthorized, gin.H{"error": "Authorization token not provided"})
                c.Abort()
                return
            }
            parts := strings.Split(authHeader, " ")
            if len(parts) != 2 || parts[0] != "Bearer" {
                c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid Authorization header format"})
                c.Abort()
                return
            }
            token = parts[1]
        }

        claims, err := utils.ValidateToken(token)
        if err != nil || claims.Role != role {
            c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid authorization token"})
            c.Abort()
            return
        }

        c.Set("clientID", claims.ID)
        c.Set("role", claims.Role) // 🔥 это нужно, иначе c.Get(\"role\") не сработает

        c.Next()
    }
}


func DynamicAPIKeyMiddleware() gin.HandlerFunc {
    return func(c *gin.Context) {
        apiKey := c.GetHeader("X-API-Key")
        if apiKey == "" {
            c.JSON(http.StatusUnauthorized, gin.H{"error": "API key is required"})
            c.Abort()
            return
        }

        var key models.ShopAPIKey
        err := config.ShopAPIKeyCollection.FindOne(context.TODO(), bson.M{
            "key":       apiKey,
            "is_active": true,
            "expires_at": bson.M{"$gt": time.Now()}, // Проверяем, что ключ не истек
        }).Decode(&key)

        if err != nil {
            if err == mongo.ErrNoDocuments {
                c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid or expired API key"})
            } else {
                c.JSON(http.StatusInternalServerError, gin.H{"error": "Error validating API key"})
            }
            c.Abort()
            return
        }

        c.Next()
    }
}
