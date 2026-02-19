package controllers

import (
    "backend/config"
    "backend/models"
    "backend/utils"
    "context"
    "crypto/rand"
    "encoding/base64"
    "log"
    "net/http"
    "time"

    "github.com/gin-gonic/gin"
    "go.mongodb.org/mongo-driver/bson"
    
)

// Temporary storage for verification codes
var verificationCodes = make(map[string]string)
var codeExpiry = make(map[string]time.Time)

// Generate random verification code
func generateVerificationCode() string {
    b := make([]byte, 6)
    rand.Read(b)
    return base64.StdEncoding.EncodeToString(b)
}

// RequestPasswordReset handles password reset requests
func RequestPasswordReset(c *gin.Context) {
    var input struct {
        Phone string `json:"phone" binding:"required"`
    }
    if err := c.ShouldBindJSON(&input); err != nil {
        c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
        return
    }

    ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
    defer cancel()

    var user models.User
    var cashier models.Cashier
    var client models.Client

    err := config.UserCollection.FindOne(ctx, bson.M{"phone": input.Phone}).Decode(&user)
    if err != nil {
        err = config.CashierCollection.FindOne(ctx, bson.M{"phone": input.Phone}).Decode(&cashier)
        if err != nil {
            err = config.ClientCollection.FindOne(ctx, bson.M{"phone": input.Phone}).Decode(&client)
            if err != nil {
                c.JSON(http.StatusNotFound, gin.H{"error": "User not found"})
                return
            }
        }
        
    }

    // Generate and store verification code
    code := generateVerificationCode()
    verificationCodes[input.Phone] = code
    codeExpiry[input.Phone] = time.Now().Add(2 * time.Minute)

    // Send email with verification code
    err = utils.SendEmail("recoverycashback@nadim.shop", "notification@nadim.shop", "Password Reset Code", "Your verification code is: "+code)
    if err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"error": "Error sending email"})
        return
    }

    c.JSON(http.StatusOK, gin.H{"message": "Verification code sent"})
}

// VerifyCode handles code verification
func VerifyCode(c *gin.Context) {
    var input struct {
        Phone string `json:"phone" binding:"required"`
        Code  string `json:"code" binding:"required"`
    }
    if err := c.ShouldBindJSON(&input); err != nil {
        c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
        return
    }

    storedCode, exists := verificationCodes[input.Phone]
    if !exists || storedCode != input.Code || time.Now().After(codeExpiry[input.Phone]) {
        c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid or expired code"})
        return
    }

    c.JSON(http.StatusOK, gin.H{"message": "Code verified"})
}

// ResetPassword handles password resetting
func ResetPassword(c *gin.Context) {
    var input struct {
        Phone       string `json:"phone" binding:"required"`
        Code        string `json:"code" binding:"required"`
        NewPassword string `json:"newpassword" binding:"required"`
    }
    if err := c.ShouldBindJSON(&input); err != nil {
        c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
        return
    }

    // Verify the code
    storedCode, exists := verificationCodes[input.Phone]
    if !exists || storedCode != input.Code || time.Now().After(codeExpiry[input.Phone]) {
        c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid or expired code"})
        return
    }

    // Hash the new password
    hashedPassword, err := utils.HashPassword(input.NewPassword)
    if err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"error": "Error hashing password"})
        return
    }

    // Update the password in the database
    ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
    defer cancel()

    // Attempt to update user
    updateResult, err := config.UserCollection.UpdateOne(ctx, bson.M{"phone": input.Phone}, bson.M{"$set": bson.M{"password": hashedPassword}})
    if err != nil {
        log.Println("Error updating user password:", err)
        c.JSON(http.StatusInternalServerError, gin.H{"error": "Error updating password"})
        return
    }

    // If no user updated, attempt to update cashier
    if updateResult.MatchedCount == 0 {
        updateResult, err = config.CashierCollection.UpdateOne(ctx, bson.M{"phone": input.Phone}, bson.M{"$set": bson.M{"password": hashedPassword}})
        if err != nil {
            log.Println("Error updating cashier password:", err)
            c.JSON(http.StatusInternalServerError, gin.H{"error": "Error updating password"})
            return
        }
    }
    

    // Log the update result for debugging
    log.Printf("Password update result: MatchedCount=%d, ModifiedCount=%d", updateResult.MatchedCount, updateResult.ModifiedCount)

    // Remove the code after successful reset
    delete(verificationCodes, input.Phone)
    delete(codeExpiry, input.Phone)

    c.JSON(http.StatusOK, gin.H{"message": "Password reset successful"})
}
