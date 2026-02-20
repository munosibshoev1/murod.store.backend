package controllers

import (
	"backend/config"
	"backend/utils"
	"errors"
	"strings"

	"backend/models"
	"bytes"
	"context"
	"encoding/json"
	"io"
	"log"

	// "encoding/json"
	"fmt"
	"math"
	"math/rand"
	"net/http"
	"os"
	"strconv"
	"time"

	// "github.com/shopspring/decimal"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// ValidateAmount ensures the amount has at most two decimal places and is positive
func ValidateAmount(amount float64) (float64, error) {
	formattedAmount, err := strconv.ParseFloat(fmt.Sprintf("%.2f", amount), 64)
	if err != nil {
		return 0, fmt.Errorf("invalid amount format")
	}

	if formattedAmount <= 0 {
		return 0, fmt.Errorf("amount must be a positive number")
	}

	return formattedAmount, nil
}

// // CheckCard checks the details of a card based on the card number
// func CheckCard(c *gin.Context) {
// 	cardNumber := c.Param("cardNumber")
// 	var card models.Card
// 	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
// 	defer cancel()

// 	// Find the card details
// 	err := config.CardCollection.FindOne(ctx, bson.M{"cardnumber": cardNumber}).Decode(&card)
// 	if err != nil {
// 		if err == mongo.ErrNoDocuments {
// 			c.JSON(http.StatusNotFound, gin.H{"error": "Card not found"})
// 		} else {
// 			c.JSON(http.StatusInternalServerError, gin.H{"error": "Error retrieving card"})
// 		}
// 		return
// 	}

// 	// Find the client who owns the card
// 	var client models.Client
// 	err = config.ClientCollection.FindOne(ctx, bson.M{"cardnumber": cardNumber}).Decode(&client)
// 	if err != nil {
// 		if err == mongo.ErrNoDocuments {
// 			// Return card details with a message that there is no owner
// 			cardMap := map[string]interface{}{
// 				"id":            card.ID,
// 				"cardnumber":    card.CardNumber,
// 				"status":        card.Status,
// 				"limit":         card.Limit,
// 				"totalpurchase": card.TotalPurchase,
// 				"totalloan":     card.TotalLoan,
// 				"totalout":      card.TotalOut,
// 				"totalfast":     card.TotalFast,
// 				"totalsettle":   card.TotalSettle,
// 				"days":          card.Days,
// 				"alldays":       card.AllDays,
// 				"fullname":      "No owner found",
// 			}
// 			c.JSON(http.StatusOK, cardMap)
// 		} else {
// 			c.JSON(http.StatusInternalServerError, gin.H{"error": "Error retrieving client"})
// 		}
// 		return
// 	}

// 	// Add the fullname to the card data
// 	cardMap := map[string]interface{}{
// 		"id":            card.ID,
// 		"cardnumber":    card.CardNumber,
// 		"status":        card.Status,
// 		"limit":         card.Limit,
// 		"totalpurchase": card.TotalPurchase,
// 		"totalloan":     card.TotalLoan,
// 		"totalout":      card.TotalOut,
// 		"totalsettle":   card.TotalSettle,
// 		"totalfast":     card.TotalFast,
// 		"days":          card.Days,
// 		"alldays":       card.AllDays,
// 		"fullname":      client.FirstName + " " + client.LastName,
// 		"photo_url":     client.Photo_url,
// 	}

// 	c.JSON(http.StatusOK, cardMap)
// }

func GetCashierByID(ctx context.Context, cashierID string) (*models.Cashier, error) {
	var cashier models.Cashier
	objectID, err := primitive.ObjectIDFromHex(cashierID)
	if err != nil {
		return nil, err
	}
	err = config.CashierCollection.FindOne(ctx, bson.M{"_id": objectID}).Decode(&cashier)
	if err != nil {
		return nil, err
	}
	return &cashier, nil
}

// RecordTransaction saves the transaction details in the transactions collection
func RecordTransaction(ctx context.Context, transaction models.Transaction) error {
	_, err := config.TransactionCollection.InsertOne(ctx, transaction)
	if err != nil {
		return fmt.Errorf("error recording transaction: %v", err)
	}
	return nil
}

// ValidateCashierID ensures the cashier ID exists in the CashierCollection
func ValidateCashierID(ctx context.Context, cashierID string) error {
	objID, err := primitive.ObjectIDFromHex(cashierID)
	if err != nil {
		return fmt.Errorf("invalid cashier ID format")
	}

	var cashier models.Cashier
	err = config.CashierCollection.FindOne(ctx, bson.M{"_id": objID}).Decode(&cashier)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return fmt.Errorf("cashier not found")
		}
		return fmt.Errorf("error retrieving cashier")
	}

	return nil
}

func TruncateToTwoDecimals(value float64) float64 {
	factor := 100.0
	return math.Round(value*factor) / factor
}

// func AddPurchaseTransaction(c *gin.Context) {
// 	cardNumber := c.Param("cardNumber")
// 	var requestBody struct {
// 		Amount float64 `json:"amount" binding:"required"`
// 	}
// 	if err := c.ShouldBindJSON(&requestBody); err != nil {
// 		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
// 		return
// 	}

// 	cashierID := c.GetHeader("Cashier-ID")
// 	if cashierID == "" {
// 		c.JSON(http.StatusBadRequest, gin.H{"error": "Cashier ID not provided"})
// 		return
// 	}

// 	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
// 	defer cancel()

// 	if err := ValidateCashierID(ctx, cashierID); err != nil {
// 		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
// 		return
// 	}
// 	// Fetch the cashier's details by cashierID (assuming you have a function GetCashierByID)

// 	var card models.Card
// 	err := config.CardCollection.FindOne(ctx, bson.M{"cardnumber": cardNumber}).Decode(&card)
// 	if err != nil {
// 		if err == mongo.ErrNoDocuments {
// 			c.JSON(http.StatusNotFound, gin.H{"error": "Card not found"})
// 		} else {
// 			c.JSON(http.StatusInternalServerError, gin.H{"error": "Error retrieving card"})
// 		}
// 		return
// 	}

// 	// Check if the card has sufficient limit for the purchase
// 	if card.Limit < requestBody.Amount {
// 		c.JSON(http.StatusBadRequest, gin.H{"error": "Insufficient limit on the card"})
// 		return
// 	}

// 	cashier, err := GetCashierByID(ctx, cashierID)
// 	if err != nil {
// 		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch cashier details"})
// 		return
// 	}

// 	// Assign cashier location to the location variable
// 	without := requestBody.Amount * 1.06
// 	without = TruncateToTwoDecimals(without)
// 	// Create and record the transaction
// 	transaction := models.Transaction{
// 		CardNumber: cardNumber,
// 		Type:       "purchase",
// 		Purchase:   requestBody.Amount,
// 		Without:    without,
// 		Date:       time.Now(),
// 		Location:   cashier.Location,
// 		CashierID:  cashierID,
// 	}

// 	if err := RecordTransaction(ctx, transaction); err != nil {
// 		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
// 		return
// 	}

// 	// Update card details after recording the transaction
// 	newTotalPurchase := card.TotalPurchase + transaction.Purchase
// 	newTotalPurchase = TruncateToTwoDecimals(newTotalPurchase)

// 	newTotalLoan := card.TotalLoan + transaction.Purchase
// 	newTotalLoan = TruncateToTwoDecimals(newTotalLoan)

// 	newTotalFast := newTotalLoan - (newTotalLoan * 0.005)
// 	newTotalFast = TruncateToTwoDecimals(newTotalFast)

// 	newTotalOut := newTotalPurchase + ((newTotalPurchase * 6) / 100)
// 	newTotalOut = TruncateToTwoDecimals(newTotalOut)

// 	newLimit := card.Limit - transaction.Purchase
// 	newLimit = TruncateToTwoDecimals(newLimit)
// 	//newLimit, _ = strconv.ParseFloat(fmt.Sprintf("%.2f", newLimit), 64)

// 	newAllTotal := card.AllTotal + transaction.Purchase
// 	update := bson.M{
// 		"$set": bson.M{
// 			"totalpurchase": newTotalPurchase,
// 			"limit":         newLimit,
// 			"alltotal":      newAllTotal,
// 			"totalfast":     newTotalFast,
// 			"totalout":      newTotalOut,
// 			"totalloan":     newTotalLoan,
// 		},
// 	}

// 	// Check if StartDate is not set (equals to the zero value of primitive.DateTime)
// 	if card.StartDate == primitive.DateTime(0) {
// 		update["$set"].(bson.M)["startdate"] = transaction.Date
// 	}

// 	opts := options.FindOneAndUpdate().SetReturnDocument(options.After)
// 	err = config.CardCollection.FindOneAndUpdate(ctx, bson.M{"cardnumber": cardNumber}, update, opts).Decode(&card)
// 	if err != nil {
// 		c.JSON(http.StatusInternalServerError, gin.H{"error": "Error updating card details"})
// 		return
// 	}

// 	c.JSON(http.StatusOK, card)
// }

// // PurchaseWithCard processes a purchase using the cashback balance on a card
// func PurchaseWithCard(c *gin.Context) {
// 	cardNumber := c.Param("cardNumber")
// 	var requestBody struct {
// 		Amount float64 `json:"amount" binding:"required"`
// 	}
// 	if err := c.ShouldBindJSON(&requestBody); err != nil {
// 		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
// 		return
// 	}

// 	cashierID := c.GetHeader("Cashier-ID")
// 	if cashierID == "" {
// 		c.JSON(http.StatusBadRequest, gin.H{"error": "Cashier ID not provided"})
// 		return
// 	}

// 	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
// 	defer cancel()

// 	// Validate CashierID
// 	if err := ValidateCashierID(ctx, cashierID); err != nil {
// 		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
// 		return
// 	}

// 	amount, err := ValidateAmount(requestBody.Amount)
// 	if err != nil {
// 		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
// 		return
// 	}

// 	var card models.Card
// 	err = config.CardCollection.FindOne(ctx, bson.M{"cardnumber": cardNumber}).Decode(&card)
// 	if err != nil {
// 		if err == mongo.ErrNoDocuments {
// 			c.JSON(http.StatusNotFound, gin.H{"error": "Card not found"})
// 		} else {
// 			c.JSON(http.StatusInternalServerError, gin.H{"error": "Error retrieving card"})
// 		}
// 		return
// 	}

// 	if card.TotalLoan == 0 {
// 		c.JSON(http.StatusBadRequest, gin.H{"error": "Cashback balance is less than the minimum threshold of 0"})
// 		return
// 	}

// 	if card.TotalFast < amount {
// 		c.JSON(http.StatusBadRequest, gin.H{"error": "Insufficient cashback balance"})
// 		return
// 	}
// 	cashier, err := GetCashierByID(ctx, cashierID)
// 	if err != nil {
// 		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch cashier details"})
// 		return
// 	}
// 	// Create and record the transaction
// 	transaction := models.Transaction{
// 		CardNumber: cardNumber,
// 		Type:       "settle",
// 		Sumsettle:  amount,
// 		Date:       time.Now(),
// 		Location:   cashier.Location,
// 		CashierID:  cashierID,
// 	}

// 	if err := RecordTransaction(ctx, transaction); err != nil {
// 		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
// 		return
// 	}

// 	newTotalSettle := card.TotalSettle + transaction.Sumsettle
// 	newTotalSettle = TruncateToTwoDecimals(newTotalSettle)

// 	newTotalPurchase := card.TotalPurchase - (transaction.Sumsettle / 0.995)
// 	newTotalPurchase = TruncateToTwoDecimals(newTotalPurchase)

// 	newLimit := card.Limit + (transaction.Sumsettle / 0.995)
// 	newLimit = TruncateToTwoDecimals(newLimit)

// 	if newTotalPurchase < 0 {
// 		newtest := math.Abs(newTotalPurchase)
// 		fmt.Printf("%.2f\n", newtest)
// 		newTotalPurchase = newTotalPurchase + newtest
// 		newLimit = newLimit - newtest
// 		newLimit = TruncateToTwoDecimals(newLimit)
// 		fmt.Printf("%.2f\n", newLimit)
// 	}
// 	if newLimit > card.Limits {

// 		newLimit = card.Limits
// 	}
// 	transaction.Sumsettle = (transaction.Sumsettle / 0.995)
// 	transaction.Sumsettle = TruncateToTwoDecimals(transaction.Sumsettle)

// 	newTotalLoan := card.TotalLoan - transaction.Sumsettle
// 	newTotalLoan = TruncateToTwoDecimals(newTotalLoan)

// 	newTotalFast := newTotalLoan - ((newTotalLoan * 0.5) / 100)
// 	newTotalFast = TruncateToTwoDecimals(newTotalFast)

// 	newTotalOut := newTotalLoan + ((newTotalLoan * 6) / 100)
// 	newTotalOut = TruncateToTwoDecimals(newTotalOut)

// 	if newTotalFast < 0 {
// 		newtest := math.Abs(newTotalFast)
// 		fmt.Printf("%.2f\n", newtest)
// 		newTotalFast = newTotalFast + newtest
// 		newTotalOut = newTotalFast
// 		newTotalLoan = newTotalFast
// 		// newLimit=newLimit-newtest
// 		// newLimit = TruncateToTwoDecimals(newLimit)
// 		// fmt.Printf("%.2f\n", newLimit)
// 	}
// 	days := card.Days
// 	card.Retday = card.Days
// 	if newTotalLoan == 0 {
// 		days = 0
// 	}
// 	newTotalPurchase = TruncateToTwoDecimals(newTotalPurchase)
// 	update := bson.M{
// 		"$set": bson.M{
// 			"totalloan":     newTotalLoan,
// 			"totalfast":     newTotalFast,
// 			"totalout":      newTotalOut,
// 			"totalsettle":   newTotalSettle,
// 			"limit":         newLimit,
// 			"totalpurchase": newTotalPurchase,
// 			"days":          days,
// 			"retday":        card.Retday,
// 		},
// 	}
// 	opts := options.FindOneAndUpdate().SetReturnDocument(options.After)
// 	err = config.CardCollection.FindOneAndUpdate(ctx, bson.M{"cardnumber": cardNumber}, update, opts).Decode(&card)
// 	if err != nil {
// 		c.JSON(http.StatusInternalServerError, gin.H{"error": "Error updating card details"})
// 		return
// 	}

// 	c.JSON(http.StatusOK, card)
// }

// // Return Transaction
// func ReturnTransaction(c *gin.Context) {
// 	transactionID := c.Param("transactionID")
// 	var requestBody struct {
// 		Amount float64 `json:"amount" binding:"required"`
// 	}
// 	if err := c.ShouldBindJSON(&requestBody); err != nil {
// 		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
// 		return
// 	}

// 	cashierID := c.GetHeader("Cashier-ID")
// 	if cashierID == "" {
// 		c.JSON(http.StatusBadRequest, gin.H{"error": "Cashier ID not provided"})
// 		return
// 	}

// 	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
// 	defer cancel()

// 	// Validate CashierID
// 	if err := ValidateCashierID(ctx, cashierID); err != nil {
// 		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
// 		return
// 	}

// 	objID, err := primitive.ObjectIDFromHex(transactionID)
// 	if err != nil {
// 		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid transaction ID format"})
// 		return
// 	}

// 	var transaction models.Transaction
// 	err = config.TransactionCollection.FindOne(ctx, bson.M{"_id": objID}).Decode(&transaction)
// 	if err != nil {
// 		if err == mongo.ErrNoDocuments {
// 			c.JSON(http.StatusNotFound, gin.H{"error": "Transaction not found"})
// 		} else {
// 			c.JSON(http.StatusInternalServerError, gin.H{"error": "Error retrieving transaction"})
// 		}
// 		return
// 	}

// 	// Check if transaction is within 7 days
// 	if time.Since(transaction.Date) > 10*24*time.Hour {
// 		c.JSON(http.StatusBadRequest, gin.H{"error": "Transaction is older than 7 days and cannot be returned or canceled"})
// 		return
// 	}

// 	var card models.Card
// 	err = config.CardCollection.FindOne(ctx, bson.M{"cardnumber": transaction.CardNumber}).Decode(&card)
// 	if err != nil {
// 		if err == mongo.ErrNoDocuments {
// 			c.JSON(http.StatusNotFound, gin.H{"error": "Card not found"})
// 		} else {
// 			c.JSON(http.StatusInternalServerError, gin.H{"error": "Error retrieving card"})
// 		}
// 		return
// 	}
// 	// Process return based on transaction type
// 	if transaction.Type == "purchase" {
// 		if requestBody.Amount > transaction.Purchase {
// 			c.JSON(http.StatusBadRequest, gin.H{"error": "Return amount exceeds the original transaction amount"})
// 			return
// 		} else if requestBody.Amount > card.TotalPurchase {
// 			c.JSON(http.StatusBadRequest, gin.H{"error": "Return amount exceeds the Card balance amount"})
// 			return
// 		}

// 		// Reverse the transaction effects on the card
// 		newTotalPurchase := card.TotalPurchase - requestBody.Amount

// 		daysSinceTransaction := int(time.Since(transaction.Date).Hours() / 24)
// 		// Вычисляем оставшиеся дни

// 		remainingDays := card.AllDays - int64(daysSinceTransaction)
// 		newLimit := card.Limit + requestBody.Amount
// 		newLimit = TruncateToTwoDecimals(newLimit)
// 		tran := requestBody.Amount
// 		if ((card.AllDays / 40) - 1) == (remainingDays / 40) {
// 			tran = requestBody.Amount
// 			requestBody.Amount = requestBody.Amount * 1.06
// 			fmt.Printf("HIIIIIII")
// 		}
// 		newTotalLoan := card.TotalLoan - requestBody.Amount

// 		newTotalFast := newTotalLoan - (newTotalLoan * 0.005)
// 		newTotalFast = TruncateToTwoDecimals(newTotalFast)

// 		newTotalOut := newTotalPurchase + ((newTotalPurchase * 6) / 100)
// 		newTotalOut = TruncateToTwoDecimals(newTotalOut)

// 		if newLimit < 0 {
// 			newLimit = 0
// 		}
// 		newAllTotal := card.AllTotal - tran
// 		transaction.Purchase -= tran
// 		days := card.Days
// 		if newTotalLoan == 0 {
// 			days = card.Retday
// 		}
// 		update := bson.M{
// 			"$set": bson.M{
// 				"totalpurchase": newTotalPurchase,
// 				"limit":         newLimit,
// 				"alltotal":      newAllTotal,
// 				"totalfast":     newTotalFast,
// 				"totalout":      newTotalOut,
// 				"totalloan":     newTotalLoan,
// 				"days":          days,
// 			},
// 		}
// 		opts := options.FindOneAndUpdate().SetReturnDocument(options.After)
// 		err = config.CardCollection.FindOneAndUpdate(ctx, bson.M{"cardnumber": card.CardNumber}, update, opts).Decode(&card)
// 		if err != nil {
// 			c.JSON(http.StatusInternalServerError, gin.H{"error": "Error updating card details"})
// 			return
// 		}

// 	} else if transaction.Type == "settle" {
// 		// Check if transaction is within 1 days
// 		if time.Since(transaction.Date) > 1*24*time.Hour {

// 			c.JSON(http.StatusBadRequest, gin.H{"error": "Transaction is older than 1 days for settle and cannot be returned or canceled"})
// 			return
// 		}
// 		if requestBody.Amount > transaction.Sumsettle {
// 			c.JSON(http.StatusBadRequest, gin.H{"error": "Return amount cannot exceed transaction amount"})
// 			return
// 		}

// 		// Adjust card balances
// 		newTotalSettle := card.TotalSettle - requestBody.Amount
// 		newTotalSettle, _ = strconv.ParseFloat(fmt.Sprintf("%.2f", newTotalSettle), 64)
// 		newTotalSettle = TruncateToTwoDecimals(newTotalSettle)

// 		newTotalPurchase := card.TotalPurchase + (requestBody.Amount / 0.995)
// 		newTotalPurchase = TruncateToTwoDecimals(newTotalPurchase)

// 		newLimit := card.Limit - (requestBody.Amount / 0.995)
// 		newLimit = TruncateToTwoDecimals(newLimit)

// 		// Ensure we properly truncate the values
// 		newTotalSettle = TruncateToTwoDecimals(newTotalSettle)

// 		newLimit = TruncateToTwoDecimals(newLimit)
// 		newTotalFast := newTotalPurchase * 0.995
// 		newTotalFast = TruncateToTwoDecimals(newTotalFast)

// 		daysSinceTransaction := int(time.Since(transaction.Date).Hours() / 24)
// 		remainingDays := card.AllDays - int64(daysSinceTransaction)
// 		tran := newTotalPurchase
// 		if ((card.AllDays / 40) - 1) == (remainingDays / 40) {
// 			tran = newTotalPurchase * 1.06
// 			fmt.Printf("HIIIIIII")
// 		}
// 		newTotalLoan := tran
// 		// newTotalLoan := newTotalPurchase * 1.06
// 		fmt.Printf("%.2f\n", newTotalLoan)
// 		newTotalLoan = TruncateToTwoDecimals(newTotalLoan)

// 		newTotalOut := newTotalPurchase * 1.06
// 		newTotalOut = TruncateToTwoDecimals(newTotalOut)
// 		// Ensure that the new values do not exceed or drop below certain limits
// 		if newLimit < 0 {
// 			newLimit = 0
// 		}

// 		if newTotalPurchase > card.Limits {
// 			newTotalPurchase = card.Limits
// 		}
// 		transaction.Sumsettle -= requestBody.Amount
// 		// Update the card information in the database
// 		days := card.Days
// 		if newTotalLoan != 0 {
// 			days = card.Retday
// 		}
// 		update := bson.M{
// 			"$set": bson.M{
// 				"totalout":      newTotalOut,
// 				"totalloan":     newTotalLoan,
// 				"totalfast":     newTotalFast,
// 				"totalsettle":   newTotalSettle,
// 				"totalpurchase": newTotalPurchase,
// 				"limit":         newLimit,
// 				"days":          days,
// 			},
// 		}
// 		opts := options.FindOneAndUpdate().SetReturnDocument(options.After)
// 		err = config.CardCollection.FindOneAndUpdate(ctx, bson.M{"cardnumber": transaction.CardNumber}, update, opts).Decode(&card)
// 		if err != nil {
// 			c.JSON(http.StatusInternalServerError, gin.H{"error": "Error updating card details"})
// 			return
// 		}

// 	} else {
// 		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid transaction type"})
// 		return
// 	}

// 	// Update transaction in the database
// 	updateTransaction := bson.M{
// 		"$set": bson.M{
// 			"purchase":  transaction.Purchase,
// 			"without":   (transaction.Purchase * 1.06),
// 			"sumsettle": transaction.Sumsettle,
// 		},
// 	}
// 	_, err = config.TransactionCollection.UpdateOne(ctx, bson.M{"_id": objID}, updateTransaction)
// 	if err != nil {
// 		c.JSON(http.StatusInternalServerError, gin.H{"error": "Error updating transaction"})
// 		return
// 	}

// 	// Ensure card values are correctly formatted
// 	card.TotalPurchase, _ = strconv.ParseFloat(fmt.Sprintf("%.2f", card.TotalPurchase), 64)

// 	// Update card in the database
// 	updateCard := bson.M{
// 		"$set": bson.M{
// 			"totalpurchase": card.TotalPurchase,
// 		},
// 	}
// 	_, err = config.CardCollection.UpdateOne(ctx, bson.M{"cardnumber": transaction.CardNumber}, updateCard)
// 	if err != nil {
// 		c.JSON(http.StatusInternalServerError, gin.H{"error": "Error updating card balances"})
// 		return
// 	}

// 	c.JSON(http.StatusOK, gin.H{"message": "Transaction returned successfully"})
// }

func GetAllProducts(c *gin.Context) {
	projection := bson.M{
		"expirationdate": 0,
		"supplierid":     0,
		"updated_at":     0,
		"purchaseprice":  0,
	}

	cursor, err := config.ProductCollection.Find(context.TODO(), bson.M{}, options.Find().SetProjection(projection))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch products"})
		return
	}
	defer cursor.Close(context.TODO())

	var rawProducts []models.Product
	if err = cursor.All(context.TODO(), &rawProducts); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to decode products"})
		return
	}

	var products []map[string]interface{}
	for _, product := range rawProducts {
		totalQuantity := sumQuantitiesFloat(product.Quantities)

		if totalQuantity == 0 || product.Sellingprice == 0 || product.Whosaleprice == 0 || product.Retailprice == 0 {
			continue
		}

		productData := map[string]interface{}{
			"id":                     product.ID,
			"categoryid":             product.CategoryID,
			"name":                   product.Name,
			"unm":                    product.Unm,
			"quantity":               totalQuantity,
			"minimumorder":           product.Minimumorder,
			"barcode":                product.Barcode,
			"sellingprice":           product.Sellingprice,
			"whosaleprice":           product.Whosaleprice,
			"retailprice":            product.Retailprice,
			"productphotourl":        product.Productphotourl,
			"productphotopreviewurl": product.Productphotopreviewurl,
		}
		products = append(products, productData)
	}

	c.JSON(http.StatusOK, products)
}

func GetAllProductsAdmin(c *gin.Context) {
	projection := bson.M{
		"expirationdate": 0,
		"supplierid":     0,
		"updated_at":     0,
		"purchaseprice":  0,
	}

	cursor, err := config.ProductCollection.Find(context.TODO(), bson.M{}, options.Find().SetProjection(projection))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch products"})
		return
	}
	defer cursor.Close(context.TODO())

	var rawProducts []models.Product
	if err = cursor.All(context.TODO(), &rawProducts); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to decode products"})
		return
	}

	var products []map[string]interface{}
	for _, product := range rawProducts {
		totalQuantity := sumQuantitiesFloat(product.Quantities)

		productData := map[string]interface{}{
			"id":                     product.ID,
			"categoryid":             product.CategoryID,
			"name":                   product.Name,
			"unm":                    product.Unm,
			"quantity":               totalQuantity,
			"minimumorder":           product.Minimumorder,
			"barcode":                product.Barcode,
			"sellingprice":           product.Sellingprice,
			"whosaleprice":           product.Whosaleprice,
			"retailprice":            product.Retailprice,
			"productphotourl":        product.Productphotourl,
			"productphotopreviewurl": product.Productphotopreviewurl,
		}
		products = append(products, productData)
	}

	c.JSON(http.StatusOK, products)
}

func GetProductsWithSelectedFields(c *gin.Context) {
	projection := bson.M{
		"_id":           1,
		"categoryid":    1,
		"name":          1,
		"unm":           1,
		"quantities":    1,
		"barcode":       1,
		"purchaseprice": 1,
	}

	cursor, err := config.ProductCollection.Find(context.TODO(), bson.M{}, options.Find().SetProjection(projection))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch products"})
		return
	}
	defer cursor.Close(context.TODO())

	var rawProducts []models.Product
	if err = cursor.All(context.TODO(), &rawProducts); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to decode products"})
		return
	}

	var products []map[string]interface{}
	for _, product := range rawProducts {
		totalQuantity := sumQuantitiesFloat(product.Quantities)

		productData := map[string]interface{}{
			"id":            product.ID,
			"categoryid":    product.CategoryID,
			"name":          product.Name,
			"unm":           product.Unm,
			"quantity":      totalQuantity,
			"barcode":       product.Barcode,
			"purchaseprice": product.Purchaseprice,
		}
		products = append(products, productData)
	}

	c.JSON(http.StatusOK, products)
}

// sumQuantitiesFloat - суммирует массив количеств типа float64
func sumQuantitiesFloat(quantities []float64) float64 {
	var total float64
	for _, quantity := range quantities {
		total += quantity
	}
	return total
}

// GetProduct - получение товара по ID
func GetProduct(c *gin.Context) {
	productID := c.Param("id")
	objID, err := primitive.ObjectIDFromHex(productID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid product ID"})
		return
	}

	var product models.Product
	err = config.ProductCollection.FindOne(context.TODO(), bson.M{"_id": objID}).Decode(&product)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Product not found"})
		return
	}

	c.JSON(http.StatusOK, product)
}

func EditProduct(c *gin.Context) {
	productID := c.Param("id")
	objID, err := primitive.ObjectIDFromHex(productID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid product ID"})
		return
	}

	var existingProduct models.Product
	err = config.ProductCollection.FindOne(context.TODO(), bson.M{"_id": objID}).Decode(&existingProduct)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Product not found"})
		return
	}

	updateFields := bson.M{}
	retailPriceStr := c.PostForm("retailprice")
	whosalePriceStr := c.PostForm("whosaleprice")

	var retailPrice, whosalePrice float64
	if retailPriceStr != "" {
		rp, err := strconv.ParseFloat(retailPriceStr, 64)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid retail price format"})
			return
		}
		retailPrice = rp
		updateFields["retailprice"] = retailPrice
	}

	if whosalePriceStr != "" {
		wp, err := strconv.ParseFloat(whosalePriceStr, 64)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid wholesale price format"})
			return
		}
		whosalePrice = wp
		updateFields["whosaleprice"] = whosalePrice
	}

	if len(updateFields) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "No fields to update"})
		return
	}

	if retailPriceStr != "" && whosalePriceStr != "" && retailPrice < whosalePrice {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Retail price must be greater than or equal to wholesale price"})
		return
	}

	filter := bson.M{"_id": objID}
	update := bson.M{"$set": updateFields}

	_, err = config.ProductCollection.UpdateOne(context.TODO(), filter, update)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update product"})
		return
	}

	if retailPriceStr != "" {
		success, response, err := SyncRetailPriceWithPOS(existingProduct.Barcode, retailPrice)
		if err != nil || !success {
			log.Printf("[ERROR] POS sync failed: %v — %s", err, response)
			// Можно не прерывать, а просто логировать
		}
	}

	c.JSON(http.StatusOK, gin.H{"message": "Product updated successfully"})
}

func SyncRetailPriceWithPOS(barcode string, price float64) (bool, string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	apiKey, err := GetShopAPIKeyPOS(ctx)
	if err != nil {
		return false, "", fmt.Errorf("failed to get POS API key: %w", err)
	}

	url := "https://bpos.nadim.shop/api/update-price"

	body := map[string]interface{}{
		"barcode":     barcode,
		"retailprice": price,
	}
	jsonData, err := json.Marshal(body)
	if err != nil {
		return false, "", fmt.Errorf("marshal error: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewBuffer(jsonData))
	if err != nil {
		return false, "", fmt.Errorf("request creation error: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-API-Key", apiKey)

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return false, "", fmt.Errorf("request send error: %w", err)
	}
	defer resp.Body.Close()

	bodyResp, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK {
		return false, string(bodyResp), fmt.Errorf("POS error %d: %s", resp.StatusCode, bodyResp)
	}

	return true, string(bodyResp), nil
}

func GetShopAPIKeyPOS(ctx context.Context) (string, error) {
	var apiKey models.ShopAPIKey

	err := config.ShopAPIKeyCollection.FindOne(ctx, bson.M{
		"is_active":   true,
		"description": "POS",
		"expires_at": bson.M{
			"$gt": time.Now(),
		},
	}).Decode(&apiKey)

	if err != nil {
		if err == mongo.ErrNoDocuments {
			return "", fmt.Errorf("no active POS API key found")
		}
		return "", fmt.Errorf("failed to retrieve POS API key: %w", err)
	}

	return apiKey.Key, nil
}

// DeleteProduct - удаление товара по ID и его фото
func DeleteProduct(c *gin.Context) {
	productID := c.Param("id")
	objID, err := primitive.ObjectIDFromHex(productID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid product ID"})
		return
	}

	// Находим товар перед удалением, чтобы получить путь к фото
	var product models.Product
	err = config.ProductCollection.FindOne(context.TODO(), bson.M{"_id": objID}).Decode(&product)
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
	_, err = config.ProductCollection.DeleteOne(context.TODO(), bson.M{"_id": objID})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete product"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Product and its photo deleted successfully"})
}

// CreateSupplier добавляет нового поставщика в базу данных
func CreateSupplier(c *gin.Context) {
	// Создаём объект поставщика с новым `ObjectID`
	supplier := models.Supplier{
		ID: primitive.NewObjectID(),
	}

	// Генерация уникального `SupplierID`
	supplierID, err := GenerateSupplierID()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to generate supplier ID"})
		return
	}
	supplier.SupplierID = supplierID

	// Заполняем оставшиеся поля из JSON-запроса
	if err := c.ShouldBindJSON(&supplier); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid input data"})
		return
	}

	// Проверка обязательных полей
	if supplier.Name == "" || supplier.Phone == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Name and Phone are required"})
		return
	}

	// Устанавливаем значения по умолчанию
	supplier.Status = "Active"
	// supplier.CreatedAt = time.Now()
	// supplier.UpdatedAt = time.Now()

	// Вставка поставщика в базу данных
	_, err = config.SupplierCollection.InsertOne(context.TODO(), supplier)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create supplier"})
		return
	}

	// Возвращаем успешный ответ с созданным поставщиком
	c.JSON(http.StatusCreated, gin.H{"message": "Supplier created successfully", "supplier": supplier})
}

// GetAllSuppliers - получение всех поставщиков
func GetAllSuppliers(c *gin.Context) {
	cursor, err := config.SupplierCollection.Find(context.TODO(), bson.M{})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch suppliers"})
		return
	}
	defer cursor.Close(context.TODO())

	var suppliers []models.Supplier
	if err = cursor.All(context.TODO(), &suppliers); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to decode suppliers"})
		return
	}

	c.JSON(http.StatusOK, suppliers)
}

func GetAllSuppliersNEW(c *gin.Context) {
	roleRaw, hasRole := c.Get("role")
	if !hasRole {
		c.JSON(http.StatusForbidden, gin.H{"error": "role not found"})
		return
	}

	cursor, err := config.SupplierCollection.Find(context.TODO(), bson.M{})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch suppliers"})
		return
	}
	defer cursor.Close(context.TODO())

	var suppliers []models.Supplier
	if err = cursor.All(context.TODO(), &suppliers); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to decode suppliers"})
		return
	}

	role := roleRaw.(string)

	if role == "admin" {
		c.JSON(http.StatusOK, suppliers)
		return
	}

	if role == "operator" {
		var filtered []map[string]interface{}
		for _, s := range suppliers {
			filtered = append(filtered, map[string]interface{}{
				"id":            s.ID,
				"supplierid":    s.SupplierID,
				"name":          s.Name,
				"payment_terms": s.PaymentTerms,
				"delivery_time": s.DeliveryTime,
				"status":        s.Status,
			})
		}
		c.JSON(http.StatusOK, filtered)
		return
	}

	c.JSON(http.StatusForbidden, gin.H{"error": "access denied"})
}

// GetSupplier - получение информации о поставщике по ID
func GetSupplier(c *gin.Context) {
	supplierID := c.Param("id")
	objID, err := primitive.ObjectIDFromHex(supplierID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid supplier ID"})
		return
	}

	var supplier models.Supplier
	err = config.SupplierCollection.FindOne(context.TODO(), bson.M{"_id": objID}).Decode(&supplier)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Supplier not found"})
		return
	}

	c.JSON(http.StatusOK, supplier)
}

// EditSupplier - редактирование поставщика по ID
func EditSupplier(c *gin.Context) {
	supplierID := c.Param("id")
	objID, err := primitive.ObjectIDFromHex(supplierID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid supplier ID"})
		return
	}

	var updatedSupplier models.UpdateSupplier
	if err := c.ShouldBindJSON(&updatedSupplier); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid input data"})
		return
	}

	// Создаём динамическое обновление на основе переданных полей
	updateFields := bson.M{}
	if updatedSupplier.Name != "" {
		updateFields["name"] = updatedSupplier.Name
	}
	if updatedSupplier.ContactPerson != "" {
		updateFields["contact_person"] = updatedSupplier.ContactPerson
	}
	if updatedSupplier.Phone != "" {
		updateFields["phone"] = updatedSupplier.Phone
	}
	if updatedSupplier.Email != "" {
		updateFields["email"] = updatedSupplier.Email
	}
	if updatedSupplier.Address != "" {
		updateFields["address"] = updatedSupplier.Address
	}
	if updatedSupplier.PaymentTerms != "" {
		updateFields["payment_terms"] = updatedSupplier.PaymentTerms
	}
	if updatedSupplier.DeliveryTime != 0 {
		updateFields["delivery_time"] = updatedSupplier.DeliveryTime
	}
	if updatedSupplier.Status != "" {
		updateFields["status"] = updatedSupplier.Status
	}

	// Если нет полей для обновления, возвращаем ошибку
	if len(updateFields) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "No fields to update"})
		return
	}

	updateFields["updated_at"] = time.Now()

	filter := bson.M{"_id": objID}
	update := bson.M{"$set": updateFields}

	_, err = config.SupplierCollection.UpdateOne(context.TODO(), filter, update)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update supplier"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Supplier updated successfully"})
}

// DeleteSupplier - удаление поставщика по ID
func DeleteSupplier(c *gin.Context) {
	supplierID := c.Param("id")
	objID, err := primitive.ObjectIDFromHex(supplierID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid supplier ID"})
		return
	}

	_, err = config.SupplierCollection.DeleteOne(context.TODO(), bson.M{"_id": objID})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete supplier"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Supplier deleted successfully"})
}

// GenerateSupplierID генерирует уникальный трёхзначный идентификатор для поставщика
func GenerateSupplierID() (string, error) {
	for {
		// Генерируем случайный трёхзначный код
		supplierID := fmt.Sprintf("%03d", rand.Intn(1000))

		// Проверяем, что такой SupplierID ещё не существует
		var existingSupplier models.Supplier
		err := config.SupplierCollection.FindOne(context.TODO(), bson.M{"supplierid": supplierID}).Decode(&existingSupplier)
		if err == mongo.ErrNoDocuments {
			// Если поставщик с таким ID не найден, возвращаем его
			return supplierID, nil
		} else if err != nil {
			return "", err
		}
	}
}

// GetSuppliersForSelect - получение всех поставщиков с ограниченной информацией
func GetSuppliersForSelect(c *gin.Context) {
	// Определяем, какие поля нужно вернуть
	projection := bson.M{
		"supplierid": 1,
		"name":       1,
	}

	cursor, err := config.SupplierCollection.Find(context.TODO(), bson.M{}, options.Find().SetProjection(projection))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch suppliers for select"})
		return
	}
	defer cursor.Close(context.TODO())

	var suppliers []struct {
		SupplierID string `json:"supplierid"`
		Name       string `json:"name"`
	}

	if err = cursor.All(context.TODO(), &suppliers); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to decode suppliers"})
		return
	}

	c.JSON(http.StatusOK, suppliers)
}

// AddSupplierOrder добавляет новый заказ для поставщика
// AddSupplierOrder добавляет новый заказ для поставщика
func AddSupplierOrder(c *gin.Context) {
	var order models.SupplierOrder

	if err := c.ShouldBindJSON(&order); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid input data"})
		return
	}

	if order.SupplierID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "SupplierID is required"})
		return
	}

	if len(order.Products) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "At least one product is required"})
		return
	}

	// Duplicate barcode check
	barcodeMap := make(map[string]bool)
	for _, product := range order.Products {
		if barcodeMap[product.Barcode] {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Duplicate product barcodes are not allowed"})
			return
		}
		barcodeMap[product.Barcode] = true
	}

	var supplier models.Supplier
	err := config.SupplierCollection.FindOne(context.TODO(), bson.M{"supplierid": order.SupplierID}).Decode(&supplier)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Supplier not found"})
		return
	}
	order.SupplierName = supplier.Name
	switch v := order.FClientID.(type) {
	case string:
		order.ClientID = v
	case map[string]interface{}:
		if val, ok := v["clientid"].(string); ok {
			order.ClientID = val
		}
	}

	order.ID = primitive.NewObjectID()
	order.CreatedAt = time.Now()
	order.UpdatedAt = time.Now()

	allPurchaseFound := true
	allExpirationFilled := true
	productMatched := false

	for i, product := range order.Products {
		matched := false
		order.Products[i].Confirmed = false

		var lastOrder models.SupplierOrder
		err := config.SupplierOrderCollection.FindOne(
			context.TODO(),
			bson.M{"supplierid": order.SupplierID, "products.barcode": product.Barcode},
			options.FindOne().SetSort(bson.D{{Key: "created_at", Value: -1}}),
		).Decode(&lastOrder)

		if err == nil {
			for _, prevProduct := range lastOrder.Products {
				if prevProduct.Barcode == product.Barcode {
					if product.PurchasePrice == 0 {
						order.Products[i].PurchasePrice = prevProduct.PurchasePrice
					}
					if len(product.ExpirationDate) == 0 {
						order.Products[i].ExpirationDate = prevProduct.ExpirationDate
					}
					if product.Whosaleprice == 0 {
						order.Products[i].Whosaleprice = prevProduct.Whosaleprice
					}
					if product.Retailprice == 0 {
						order.Products[i].Retailprice = prevProduct.Retailprice
					}
					matched = true
					productMatched = true
					break
				}
			}
		}

		if !matched {
			var altOrder models.SupplierOrder
			err := config.SupplierOrderCollection.FindOne(
				context.TODO(),
				bson.M{"products.barcode": product.Barcode},
				options.FindOne().SetSort(bson.D{{Key: "created_at", Value: -1}}),
			).Decode(&altOrder)

			if err == nil {
				for _, altProduct := range altOrder.Products {
					if altProduct.Barcode == product.Barcode {
						if product.PurchasePrice == 0 {
							order.Products[i].PurchasePrice = altProduct.PurchasePrice
						}
						if len(product.ExpirationDate) == 0 {
							order.Products[i].ExpirationDate = altProduct.ExpirationDate
						}
						if product.Whosaleprice == 0 {
							order.Products[i].Whosaleprice = altProduct.Whosaleprice
						}
						if product.Retailprice == 0 {
							order.Products[i].Retailprice = altProduct.Retailprice
						}
						matched = true
						productMatched = true
						break
					}
				}
			}
		}

		if order.Products[i].PurchasePrice == 0 {
			allPurchaseFound = false
		}
		if len(order.Products[i].ExpirationDate) == 0 {
			allExpirationFilled = false
		}
	}

	order.Status = "Pending Approval"

	if productMatched && allPurchaseFound && allExpirationFilled {
		order.Status = "Confirmed in stock"
	}

	_, err = config.SupplierOrderCollection.InsertOne(context.TODO(), order)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create supplier order"})
		return
	}

	c.JSON(http.StatusCreated, gin.H{"message": "Supplier order created successfully", "order": order})
}

// AddSupplierOrder добавляет новый заказ для поставщика
// AddSupplierOrderNEW добавляет новый заказ для поставщика с авто-заполнением полей и статусом
func AddSupplierOrderNEW(c *gin.Context) {
	roleRaw, hasRole := c.Get("role")
	if !hasRole {
		c.JSON(http.StatusForbidden, gin.H{"error": "role not found"})
		return
	}

	clientIDRaw, hasClient := c.Get("clientID")
	if !hasClient {
		c.JSON(http.StatusForbidden, gin.H{"error": "clientID not found"})
		return
	}

	var order models.SupplierOrder

	if err := c.ShouldBindJSON(&order); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid input data"})
		return
	}

	if order.SupplierID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "SupplierID is required"})
		return
	}

	if len(order.Products) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "At least one product is required"})
		return
	}

	// Duplicate barcode check
	barcodeMap := make(map[string]bool)
	for _, product := range order.Products {
		if barcodeMap[product.Barcode] {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Duplicate product barcodes are not allowed"})
			return
		}
		barcodeMap[product.Barcode] = true
	}

	var supplier models.Supplier
	err := config.SupplierCollection.FindOne(context.TODO(), bson.M{"supplierid": order.SupplierID}).Decode(&supplier)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Supplier not found"})
		return
	}
	order.SupplierName = supplier.Name

	order.ID = primitive.NewObjectID()
	order.Status = "Pending Approval"
	order.CreatedAt = time.Now()
	order.UpdatedAt = time.Now()

	if roleRaw.(string) == "operator" {
		order.CreatedBy = clientIDRaw.(string)
	} else {
		order.CreatedBy = ""
	}

	allPurchaseFound := true
	allExpirationFilled := true
	productMatched := false

	for i, product := range order.Products {
		matched := false
		order.Products[i].Confirmed = false

		var lastOrder models.SupplierOrder
		err := config.SupplierOrderCollection.FindOne(
			context.TODO(),
			bson.M{"supplierid": order.SupplierID, "products.barcode": product.Barcode},
			options.FindOne().SetSort(bson.D{{Key: "created_at", Value: -1}}),
		).Decode(&lastOrder)

		if err == nil {
			for _, prevProduct := range lastOrder.Products {
				if prevProduct.Barcode == product.Barcode {
					if product.PurchasePrice == 0 {
						order.Products[i].PurchasePrice = prevProduct.PurchasePrice
					}
					if len(product.ExpirationDate) == 0 {
						order.Products[i].ExpirationDate = prevProduct.ExpirationDate
					}
					if product.Whosaleprice == 0 {
						order.Products[i].Whosaleprice = prevProduct.Whosaleprice
					}
					if product.Retailprice == 0 {
						order.Products[i].Retailprice = prevProduct.Retailprice
					}
					matched = true
					productMatched = true
					break
				}
			}
		}

		if !matched {
			var altOrder models.SupplierOrder
			err := config.SupplierOrderCollection.FindOne(
				context.TODO(),
				bson.M{"products.barcode": product.Barcode},
				options.FindOne().SetSort(bson.D{{Key: "created_at", Value: -1}}),
			).Decode(&altOrder)

			if err == nil {
				for _, altProduct := range altOrder.Products {
					if altProduct.Barcode == product.Barcode {
						if product.PurchasePrice == 0 {
							order.Products[i].PurchasePrice = altProduct.PurchasePrice
						}
						if len(product.ExpirationDate) == 0 {
							order.Products[i].ExpirationDate = altProduct.ExpirationDate
						}
						if product.Whosaleprice == 0 {
							order.Products[i].Whosaleprice = altProduct.Whosaleprice
						}
						if product.Retailprice == 0 {
							order.Products[i].Retailprice = altProduct.Retailprice
						}
						matched = true
						productMatched = true
						break
					}
				}
			}
		}

		if order.Products[i].PurchasePrice == 0 {
			allPurchaseFound = false
		}
		if len(order.Products[i].ExpirationDate) == 0 {
			allExpirationFilled = false
		}
	}

	if productMatched && allPurchaseFound && allExpirationFilled {
		order.Status = "Confirmed in stock"
	}

	_, err = config.SupplierOrderCollection.InsertOne(context.TODO(), order)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create supplier order"})
		return
	}

	c.JSON(http.StatusCreated, gin.H{"message": "Supplier order created successfully", "order": order})
}

// GetSupplierOrder - получение заказа поставщика по ID
func GetSupplierOrder(c *gin.Context) {
	orderID := c.Param("id")
	objID, err := primitive.ObjectIDFromHex(orderID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid order ID"})
		return
	}

	var order models.SupplierOrder
	err = config.SupplierOrderCollection.FindOne(context.TODO(), bson.M{"_id": objID}).Decode(&order)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Order not found"})
		return
	}

	totalPurchase := 0.0
	totalWholesale := 0.0
	totalOrderGrossWeight := 0.0

	for i, product := range order.Products {
		if len(product.Quantities) == 0 {
			continue
		}
		qty := product.Quantities[0]

		totalWholesale += product.Whosaleprice * float64(qty)
		totalPurchase += product.PurchasePrice * float64(qty)

		var template models.ProductTemplate
		err := config.ProductTemplateCollection.FindOne(context.TODO(), bson.M{"barcode": product.Barcode}).Decode(&template)
		if err == nil {
			grossPerUnit := template.Grossweight
			unitLower := strings.ToLower(product.UNM)
			if unitLower == "кг" {
				grossPerUnit = 1.05
			}
			totalGross := grossPerUnit * float64(qty)
			totalOrderGrossWeight += totalGross

			order.Products[i].TotalGrossWeight = TruncateToTwoDecimals(totalGross)
		}
	}

	profit := totalWholesale - totalPurchase

	orderMap := make(map[string]interface{})
	orderJSON, _ := json.Marshal(order)
	_ = json.Unmarshal(orderJSON, &orderMap)

	orderMap["totalpurchaseprice"] = totalPurchase
	orderMap["totalwhosaleprice"] = totalWholesale
	orderMap["profit"] = profit
	orderMap["totalorgrowght"] = totalOrderGrossWeight
	orderMap["clientid"] = order.ClientID
	c.JSON(http.StatusOK, orderMap)
}

// GetSupplierOrder - получение заказа поставщика по ID
func GetSupplierOrderStorekeeper(c *gin.Context) {
	orderID := c.Param("id")
	objID, err := primitive.ObjectIDFromHex(orderID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid order ID"})
		return
	}

	var order models.SupplierOrder
	err = config.SupplierOrderCollection.FindOne(context.TODO(), bson.M{"_id": objID}).Decode(&order)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Order not found"})
		return
	}

	totalPurchase := 0.0
	totalWholesale := 0.0
	totalOrderGrossWeight := 0.0

	for i, product := range order.Products {
		if len(product.Quantities) == 0 {
			continue
		}
		qty := product.Quantities[0]

		totalWholesale += product.Whosaleprice * float64(qty)
		totalPurchase += product.PurchasePrice * float64(qty)

		var template models.ProductTemplate
		err := config.ProductTemplateCollection.FindOne(context.TODO(), bson.M{"barcode": product.Barcode}).Decode(&template)
		if err == nil {
			grossPerUnit := template.Grossweight
			unitLower := strings.ToLower(product.UNM)
			if unitLower == "кг" {
				grossPerUnit = 1.05
			}
			totalGross := grossPerUnit * float64(qty)
			totalOrderGrossWeight += totalGross
			order.Products[i].PurchasePrice = 0
			order.Products[i].TotalGrossWeight = TruncateToTwoDecimals(totalGross)
		}
	}

	profit := totalWholesale - totalPurchase

	orderMap := make(map[string]interface{})
	orderJSON, _ := json.Marshal(order)
	_ = json.Unmarshal(orderJSON, &orderMap)

	//orderMap["totalpurchaseprice"] = totalPurchase
	orderMap["totalwhosaleprice"] = totalWholesale
	orderMap["profit"] = profit
	orderMap["totalorgrowght"] = totalOrderGrossWeight
	orderMap["clientid"] = order.ClientID
	c.JSON(http.StatusOK, orderMap)
}

func GetSupplierOrderNEW(c *gin.Context) {
	roleRaw, hasRole := c.Get("role")
	if !hasRole {
		c.JSON(http.StatusForbidden, gin.H{"error": "role not found"})
		return
	}

	orderID := c.Param("id")
	objID, err := primitive.ObjectIDFromHex(orderID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid order ID"})
		return
	}

	var order models.SupplierOrder
	err = config.SupplierOrderCollection.FindOne(context.TODO(), bson.M{"_id": objID}).Decode(&order)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Order not found"})
		return
	}

	role := roleRaw.(string)

	if role == "admin" {
		c.JSON(http.StatusOK, order)
		return
	}

	if role == "operator" {
		var filteredProducts []map[string]interface{}
		for _, p := range order.Products {
			filteredProducts = append(filteredProducts, map[string]interface{}{
				"name":         p.Name,
				"unm":          p.UNM,
				"minimumorder": p.MinimumOrder,
				"quantities":   p.Quantities,
				"categoryid":   p.CategoryID,
			})
		}

		filteredOrder := map[string]interface{}{
			"id":            order.ID,
			"supplierid":    order.SupplierID,
			"supplier_name": order.SupplierName,
			"products":      filteredProducts,
			"status":        order.Status,
			"deliverytime":  order.DeliveryTime,
		}

		c.JSON(http.StatusOK, filteredOrder)
		return
	}

	c.JSON(http.StatusForbidden, gin.H{"error": "access denied"})
}

func GetAllSupplierOrders(c *gin.Context) {
	roleRaw, hasRole := c.Get("role")
	if !hasRole {
		c.JSON(http.StatusForbidden, gin.H{"error": "role not found"})
		return
	}

	cursor, err := config.SupplierOrderCollection.Find(context.TODO(), bson.M{})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch supplier orders"})
		return
	}
	defer cursor.Close(context.TODO())

	var orders []models.SupplierOrder
	if err := cursor.All(context.TODO(), &orders); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to decode supplier orders"})
		return
	}

	role := roleRaw.(string)

	if role == "admin" {
		c.JSON(http.StatusOK, orders)
		return
	}

	if role == "operator" {
		var filtered []map[string]interface{}
		for _, o := range orders {
			var filteredProducts []map[string]interface{}
			for _, p := range o.Products {
				filteredProducts = append(filteredProducts, map[string]interface{}{
					"name":         p.Name,
					"unm":          p.UNM,
					"minimumorder": p.MinimumOrder,
				})
			}

			filtered = append(filtered, map[string]interface{}{
				"id":            o.ID,
				"supplierid":    o.SupplierID,
				"supplier_name": o.SupplierName,
				"products":      filteredProducts,
				"status":        o.Status,
				"deliverytime":  o.DeliveryTime,
			})
		}
		c.JSON(http.StatusOK, filtered)
		return
	}

	c.JSON(http.StatusForbidden, gin.H{"error": "access denied"})
}

// GetSupplierOrders - получение всех заказов без products, с фильтрацией по SupplierID и сортировкой по created_at
func GetSupplierOrders(c *gin.Context) {
	supplierID := c.Query("supplierid")

	filter := bson.M{}
	if supplierID != "" {
		filter["supplierid"] = supplierID
	}

	findOptions := options.Find().SetSort(bson.D{{Key: "created_at", Value: -1}})
	findOptions.SetProjection(bson.M{"products": 0})

	cursor, err := config.SupplierOrderCollection.Find(context.TODO(), filter, findOptions)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch orders"})
		return
	}
	defer cursor.Close(context.TODO())

	var orders []models.SupplierOrder
	if err = cursor.All(context.TODO(), &orders); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to decode orders"})
		return
	}

	c.JSON(http.StatusOK, orders)
}

// GetSupplierOrders - получение всех заказов
func GetSupplierOrders1(c *gin.Context) {
	// Устанавливаем пустой фильтр для получения всех заказов
	filter := bson.M{}

	// Получение данных из базы данных
	cursor, err := config.SupplierOrderCollection.Find(context.TODO(), filter)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch orders"})
		return
	}
	defer cursor.Close(context.TODO())

	var orders []models.SupplierOrder
	if err = cursor.All(context.TODO(), &orders); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to decode orders"})
		return
	}

	c.JSON(http.StatusOK, orders)
}

func EditSupplierOrder(c *gin.Context) {
	orderID := c.Param("id")
	objID, err := primitive.ObjectIDFromHex(orderID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid order ID"})
		return
	}

	var updatedOrder struct {
		Products []struct {
			Barcode       string    `json:"barcode"`
			PurchasePrice float64   `json:"purchaseprice"`
			Quantities    []float64 `json:"quantities"`
		} `json:"products"`
		Payment      *string    `json:"payment"`
		DeliveryTime *time.Time `json:"deliverytime"`
	}

	if err := c.ShouldBindJSON(&updatedOrder); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid input data"})
		return
	}

	for _, p := range updatedOrder.Products {
		if p.PurchasePrice <= 0 {
			c.JSON(http.StatusBadRequest, gin.H{
				"error": fmt.Sprintf("Invalid purchase price for product with barcode %s", p.Barcode),
			})
			return
		}
	}

	var existingOrder struct {
		Products []struct {
			CategoryID         string    `json:"categoryid"`
			ExpirationDate     []string  `json:"expirationdate"`
			Name               string    `json:"name"`
			UNM                string    `json:"unm"`
			Quantities         []float64 `json:"quantities"`
			MinimumOrder       int       `json:"minimumorder"`
			Barcode            string    `json:"barcode"`
			PurchasePrice      float64   `json:"purchaseprice"`
			TotalPurchasePrice float64   `json:"totalpurchaseprice,omitempty"`
			RetailPrice        float64   `json:"retailprice"`
			WhosalePrice       float64   `json:"whosaleprice"`
			Profit             float64   `json:"profit"`
			RemainStock        float64   `json:"remainstock"`
			GrossWeight        float64   `json:"grossweight"`
			TotalOrgRowght     float64   `json:"totalorgrowght"`
			TotalWhosalePrice  float64   `json:"totalwhosaleprice"`
		} `json:"products"`
		Status       string     `json:"status"`
		Payment      string     `json:"payment"`
		DeliveryTime *time.Time `json:"deliverytime"`
	}

	err = config.SupplierOrderCollection.FindOne(context.TODO(), bson.M{"_id": objID}).Decode(&existingOrder)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Order not found"})
		return
	}

	var orderTotal float64
	productsUpdated := false

	for _, updatedProduct := range updatedOrder.Products {
		found := false
		for i, existingProduct := range existingOrder.Products {
			if existingProduct.Barcode == updatedProduct.Barcode {
				found = true
				existingOrder.Products[i].PurchasePrice = updatedProduct.PurchasePrice

				if len(updatedProduct.Quantities) > 0 && !equalFloatSlices(existingProduct.Quantities, updatedProduct.Quantities) {
					existingOrder.Products[i].Quantities = updatedProduct.Quantities
					productsUpdated = true
				}

				var totalProductPrice float64
				for _, quantity := range existingOrder.Products[i].Quantities {
					totalProductPrice += updatedProduct.PurchasePrice * quantity
				}
				existingOrder.Products[i].TotalPurchasePrice = totalProductPrice
				orderTotal += totalProductPrice
				break
			}
		}
		if !found {
			c.JSON(http.StatusBadRequest, gin.H{
				"error": fmt.Sprintf("Product with barcode %s not found in existing order", updatedProduct.Barcode),
			})
			return
		}
	}

	updateFields := bson.M{
		"products":   existingOrder.Products,
		"ordertotal": math.Round(orderTotal),
		"updated_at": time.Now(),
	}

	if productsUpdated {
		updateFields["status"] = "Confirmed in Delivery"
	}

	if updatedOrder.Payment != nil {
		updateFields["payment"] = *updatedOrder.Payment
	}
	if updatedOrder.DeliveryTime != nil {
		updateFields["deliverytime"] = *updatedOrder.DeliveryTime
	}

	_, err = config.SupplierOrderCollection.UpdateOne(context.TODO(), bson.M{"_id": objID}, bson.M{"$set": updateFields})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update supplier order"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message":     "Supplier order updated successfully",
		"order_total": orderTotal,
	})
}

func equalFloatSlices(a, b []float64) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

// EditSupplierOrder - редактирование заказа поставщика
func EditSupplierOrderS(c *gin.Context) {
	orderID := c.Param("id")
	objID, err := primitive.ObjectIDFromHex(orderID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid order ID"})
		return
	}

	var updatedOrder struct {
		Products []struct {
			CategoryID         string    `json:"categoryid"`
			ExpirationDate     []string  `json:"expirationdate"`
			Name               string    `json:"name"`
			UNM                string    `json:"unm"`
			Quantities         []float64 `json:"quantities"`
			MinimumOrder       int       `bson:"minimumorder" json:"minimumorder" binding:"required"`
			Barcode            string    `json:"barcode"`
			PurchasePrice      float64   `json:"purchaseprice"`
			TotalPurchasePrice float64   `json:"totalpurchaseprice,omitempty"`
			Retailprice        float64   `json:"retailprice"`
			Whosaleprice       float64   `json:"whosaleprice"`
		} `json:"products"`
		Status       string    `json:"status"`
		Payment      string    `json:"payment"`
		DeliveryTime time.Time `json:"deliverytime"`
	}

	if err := c.ShouldBindJSON(&updatedOrder); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid input data"})
		return
	}

	// Получаем текущие данные заказа из базы
	var existingOrder struct {
		Products []struct {
			Barcode      string  `bson:"barcode"`
			Retailprice  float64 `bson:"retailprice"`
			Whosaleprice float64 `bson:"whosaleprice"`
		} `bson:"products"`
	}
	_ = config.SupplierOrderCollection.FindOne(context.TODO(), bson.M{"_id": objID}).Decode(&existingOrder)

	var orderTotal float64
	valid := true
	for _, product := range updatedOrder.Products {
		if product.ExpirationDate == nil || len(product.Quantities) != len(product.ExpirationDate) {
			valid = false
			c.JSON(http.StatusBadRequest, gin.H{
				"error":            "Each quantity must have a corresponding expiration date",
				"barcode":          product.Barcode,
				"quantities_count": len(product.Quantities),
				"expiration_count": len(product.ExpirationDate),
			})
			return
		}
		if product.PurchasePrice <= 0 {
			c.JSON(http.StatusBadRequest, gin.H{
				"error":   "Purchase price must be greater than zero for all products",
				"barcode": product.Barcode,
			})
			return
		}
	}

	// Подставляем retail/whosale цены из базы, если они были сброшены (равны 0)
	for i, p := range updatedOrder.Products {
		for _, existing := range existingOrder.Products {
			if p.Barcode == existing.Barcode {
				if p.Retailprice == 0 {
					updatedOrder.Products[i].Retailprice = existing.Retailprice
				}
				if p.Whosaleprice == 0 {
					updatedOrder.Products[i].Whosaleprice = existing.Whosaleprice
				}
				break
			}
		}
	}

	for i, product := range updatedOrder.Products {
		var totalProductPrice float64
		for _, quantity := range product.Quantities {
			totalProductPrice += product.PurchasePrice * quantity
		}
		updatedOrder.Products[i].TotalPurchasePrice = totalProductPrice
		orderTotal += totalProductPrice
	}

	updateFields := bson.M{
		"products":     updatedOrder.Products,
		"ordertotal":   math.Round(orderTotal),
		"payment":      updatedOrder.Payment,
		"deliverytime": updatedOrder.DeliveryTime,
		"updated_at":   time.Now(),
	}
	if valid {
		updateFields["status"] = "Confirmed price for store"
	}

	filter := bson.M{"_id": objID}
	update := bson.M{"$set": updateFields}

	_, err = config.SupplierOrderCollection.UpdateOne(context.TODO(), filter, update)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update supplier order"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Supplier order updated successfully",
	})
}

// EditSupplierOrderSelling обновлён с логикой Confirmed
func EditSupplierOrderSelling(c *gin.Context) {
	orderID := c.Param("id")
	objID, err := primitive.ObjectIDFromHex(orderID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid order ID"})
		return
	}

	var existingOrder struct {
		Status string `bson:"status"`
	}
	if err := config.SupplierOrderCollection.FindOne(context.TODO(), bson.M{"_id": objID}).Decode(&existingOrder); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to find existing supplier order"})
		return
	}

	if existingOrder.Status != "Confirmed in stock" && existingOrder.Status != "Confirmed price for store" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":  "Cannot set selling prices until expiration dates are added",
			"status": existingOrder.Status,
		})
		return
	}

	var updatedOrder struct {
		Products []struct {
			CategoryID         string    `json:"categoryid"`
			ExpirationDate     []string  `json:"expirationdate"`
			Name               string    `json:"name"`
			UNM                string    `json:"unm"`
			Quantities         []float64 `json:"quantities"`
			MinimumOrder       int       `json:"minimumorder"`
			Barcode            string    `json:"barcode"`
			PurchasePrice      float64   `json:"purchaseprice"`
			SellingPrice       float64   `json:"sellingprice"`
			Whosaleprice       float64   `json:"whosaleprice"`
			Retailprice        float64   `json:"retailprice"`
			TotalPurchasePrice float64   `json:"totalpurchaseprice,omitempty"`
			RemainStock        float64   `json:"remainstock,omitempty"`
			Confirmed          bool      `json:"confirmed"`
		} `json:"products"`
		Status       string    `json:"status"`
		Payment      string    `json:"payment"`
		DeliveryTime time.Time `json:"deliverytime"`
	}

	if err := c.ShouldBindJSON(&updatedOrder); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid input data"})
		return
	}

	var orderTotal float64

	for i, product := range updatedOrder.Products {
		var totalProductPrice float64
		for _, quantity := range product.Quantities {
			rounded := math.Round(product.PurchasePrice*quantity*100) / 100
			totalProductPrice += rounded
		}
		updatedOrder.Products[i].TotalPurchasePrice = math.Round(totalProductPrice*100) / 100
		orderTotal += totalProductPrice

		var existingProduct models.Product
		filter := bson.M{"barcode": product.Barcode}
		err := config.ProductCollection.FindOne(context.TODO(), filter).Decode(&existingProduct)

		if err == nil {
			existingQuantities := existingProduct.Quantities
			existingExpirationDates := existingProduct.ExpirationDate

			if existingOrder.Status == "Confirmed in stock" {
				if !product.Confirmed {
					for j, newDate := range product.ExpirationDate {
						if index := findIndex(existingExpirationDates, newDate); index != -1 {
							existingQuantities[index] += product.Quantities[j]
						} else {
							existingExpirationDates = append(existingExpirationDates, newDate)
							existingQuantities = append(existingQuantities, product.Quantities[j])
						}
					}
					totalRemain := 0.0
					for _, qty := range existingQuantities {
						totalRemain += qty
					}
					updatedOrder.Products[i].RemainStock = totalRemain
					updatedOrder.Products[i].Confirmed = true

					updateProductFields := bson.M{
						"quantities":     existingQuantities,
						"expirationdate": existingExpirationDates,
						"sellingprice":   product.Whosaleprice,
						"whosaleprice":   product.Whosaleprice,
						"retailprice":    product.Retailprice,
						"purchaseprice":  product.PurchasePrice,
						"remainstock":    totalRemain,
						"updated_at":     time.Now(),
					}
					_, err := config.ProductCollection.UpdateOne(context.TODO(), filter, bson.M{"$set": updateProductFields})
					if err != nil {
						c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update product in the database"})
						return
					}
				}
			} else if existingOrder.Status == "Confirmed price for store" {
				updateProductFields := bson.M{
					"sellingprice": product.Whosaleprice,
					"whosaleprice": product.Whosaleprice,
					"retailprice":  product.Retailprice,
					"updated_at":   time.Now(),
				}
				_, err := config.ProductCollection.UpdateOne(context.TODO(), filter, bson.M{"$set": updateProductFields})
				if err != nil {
					c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update prices only"})
					return
				}
			}
		} else if err == mongo.ErrNoDocuments {
			var templateProduct struct {
				ProductPhotoURL        string `bson:"productphotourl"`
				ProductPhotoPreviewURL string `bson:"productphotopreviewurl"`
			}
			templateFilter := bson.M{"barcode": product.Barcode}
			tmplErr := config.ProductTemplateCollection.FindOne(context.TODO(), templateFilter).Decode(&templateProduct)

			var productPhotoURL, productPhotoPreviewURL string
			if tmplErr == nil {
				productPhotoURL = templateProduct.ProductPhotoURL
				productPhotoPreviewURL = templateProduct.ProductPhotoPreviewURL
			}

			totalRemain := 0.0
			for _, q := range product.Quantities {
				totalRemain += q
			}

			productConfirmed := existingOrder.Status == "Confirmed in stock"
			updatedOrder.Products[i].RemainStock = totalRemain
			updatedOrder.Products[i].Confirmed = productConfirmed

			newProduct := models.Product{
				ID:                     primitive.NewObjectID(),
				CategoryID:             product.CategoryID,
				Name:                   product.Name,
				Unm:                    product.UNM,
				Quantities:             product.Quantities,
				ExpirationDate:         product.ExpirationDate,
				SupplierID:             orderID,
				Purchaseprice:          product.PurchasePrice,
				Whosaleprice:           product.Whosaleprice,
				Retailprice:            product.Retailprice,
				Sellingprice:           product.Whosaleprice,
				Minimumorder:           fmt.Sprintf("%d", product.MinimumOrder),
				Barcode:                product.Barcode,
				Productphotourl:        productPhotoURL,
				Productphotopreviewurl: productPhotoPreviewURL,
				Remainstock:            totalRemain,
				CreatedAt:              time.Now(),
				UpdatedAt:              time.Now(),
			}

			_, err = config.ProductCollection.InsertOne(context.TODO(), newProduct)
			if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to insert new product into the database"})
				return
			}
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Error checking product in the database"})
			return
		}
	}

	updatedOrder.Status = "Confirmed price for store"
	updateFields := bson.M{
		"products":     updatedOrder.Products,
		"ordertotal":   orderTotal,
		"status":       updatedOrder.Status,
		"payment":      updatedOrder.Payment,
		"deliverytime": updatedOrder.DeliveryTime,
		"updated_at":   time.Now(),
	}

	filter := bson.M{"_id": objID}
	update := bson.M{"$set": updateFields}

	_, err = config.SupplierOrderCollection.UpdateOne(context.TODO(), filter, update)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update supplier order"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Supplier order successfully setting selling price"})
}

func findIndex(slice []string, val string) int {
	for i, v := range slice {
		if v == val {
			return i
		}
	}
	return -1
}

// // findIndex returns index of the target date or -1.
// func findIndex(dates []string, target string) int {
// 	for i, d := range dates {
// 		if d == target {
// 			return i
// 		}
// 	}
// 	return -1
// }

func round2(v float64) float64 { return math.Round(v*100) / 100 }

// EditSupplierOrderSelling updates prices; on the first confirmation also moves stock quantities into Product.
// Decision: stock move is controlled ONLY by per-product priorConfirmed=false, not by order status.
func EditSupplierOrderSelling1(c *gin.Context) {
	ctx := c.Request.Context()

	orderID := c.Param("id")
	objID, err := primitive.ObjectIDFromHex(orderID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid order ID"})
		return
	}

	// Load order status + per-product confirmation from DB.
	type dbSupplierOrder struct {
		Status   string `bson:"status"`
		Products []struct {
			Barcode   string `bson:"barcode"`
			Confirmed bool   `bson:"confirmed"`
		} `bson:"products"`
	}

	var dbOrder dbSupplierOrder
	if err := config.SupplierOrderCollection.FindOne(ctx, bson.M{"_id": objID}).Decode(&dbOrder); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to load supplier order"})
		return
	}

	// Fast lookup of previous confirmation.
	priorConfirmedByBarcode := make(map[string]bool, len(dbOrder.Products))
	for _, p := range dbOrder.Products {
		priorConfirmedByBarcode[p.Barcode] = p.Confirmed
	}

	// Parse request payload (supports either quantities[] or a single quantity).
	var req struct {
		Products []struct {
			CategoryID         string    `json:"categoryid"`
			ExpirationDate     []string  `json:"expirationdate"`
			Name               string    `json:"name"`
			UNM                string    `json:"unm"`
			Quantities         []float64 `json:"quantities"`
			Quantity           *float64  `json:"quantity"`
			MinimumOrder       int       `json:"minimumorder"`
			Barcode            string    `json:"barcode"`
			PurchasePrice      float64   `json:"purchaseprice"`
			SellingPrice       float64   `json:"sellingprice"`
			Whosaleprice       float64   `json:"whosaleprice"`
			Retailprice        float64   `json:"retailprice"`
			TotalPurchasePrice float64   `json:"totalpurchaseprice,omitempty"`
			RemainStock        float64   `json:"remainstock,omitempty"`
			Confirmed          bool      `json:"confirmed"`
		} `json:"products"`
		Status       string    `json:"status"`
		Payment      string    `json:"payment"`
		DeliveryTime time.Time `json:"deliverytime"`
		Clientid     string    `json:"clientid"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid input data"})
		return
	}

	// Lightweight Product view for reads.
	type dbProduct struct {
		Quantities     []float64 `bson:"quantities"`
		ExpirationDate []string  `bson:"expirationdate"`
		Remainstock    float64   `bson:"remainstock"`
	}

	// Helper to compute sum.
	sum := func(a []float64) (r float64) {
		for _, v := range a {
			r += v
		}
		return
	}

	var orderTotal float64

	for i := range req.Products {
		p := &req.Products[i]

		// Normalize quantities: accept either quantities[] or single quantity.
		if len(p.Quantities) == 0 && p.Quantity != nil {
			p.Quantities = []float64{*p.Quantity}
		}

		// Compute total purchase price for the line.
		var totalProductPrice float64
		for _, q := range p.Quantities {
			totalProductPrice += round2(p.PurchasePrice * q)
		}
		p.TotalPurchasePrice = round2(totalProductPrice)
		orderTotal += totalProductPrice

		priorConfirmed := priorConfirmedByBarcode[p.Barcode]

		filter := bson.M{"barcode": p.Barcode}
		var existing dbProduct
		err = config.ProductCollection.FindOne(ctx, filter).Decode(&existing)

		switch {
		case err == nil:
			// If FE omitted expirationdate but we have exactly one in DB, reuse it (common FE case with single quantity).
			if len(p.ExpirationDate) == 0 && len(existing.ExpirationDate) == 1 && len(p.Quantities) == 1 {
				p.ExpirationDate = []string{existing.ExpirationDate[0]}
			}

			if !priorConfirmed {
				// Merge by expiration date and move stock in.
				exDates := append([]string{}, existing.ExpirationDate...)
				exQtys := append([]float64{}, existing.Quantities...)
				if exDates == nil {
					exDates = []string{}
				}
				if exQtys == nil {
					exQtys = []float64{}
				}
				for j, newDate := range p.ExpirationDate {
					q := 0.0
					if j < len(p.Quantities) {
						q = p.Quantities[j]
					}
					if idx := findIndex(exDates, newDate); idx != -1 {
						exQtys[idx] += q
					} else {
						exDates = append(exDates, newDate)
						exQtys = append(exQtys, q)
					}
				}

				totalRemain := sum(exQtys)
				p.RemainStock = totalRemain
				p.Confirmed = true

				update := bson.M{
					"quantities":     exQtys,
					"expirationdate": exDates,
					"sellingprice":   p.Whosaleprice,
					"whosaleprice":   p.Whosaleprice,
					"retailprice":    p.Retailprice,
					"purchaseprice":  p.PurchasePrice,
					"remainstock":    totalRemain,
					"updated_at":     time.Now(),
				}
				if _, uerr := config.ProductCollection.UpdateOne(ctx, filter, bson.M{"$set": update}); uerr != nil {
					c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update product in the database"})
					return
				}
				autoOrder := models.CustomerOrderInput{
					Products: []models.ProductQuantity{ // single product
						{
							Barcode:  p.Barcode,
							Quantity: sum(p.Quantities),
						},
					},
					DeliveryMethod:       "pickup",
					DeliveryAddress:      "Рушон Вамар",
					PaymentMethod:        "Peshraft",
					CardNumber:           "", // will be populated from client
					Clientid:             "",
					CashierID:            "68bece473ca36931985548b2",
					AutoCreatedFromStock: true,
				}

				if req.Clientid != "" {
					clientObjID, err := primitive.ObjectIDFromHex(req.Clientid)
					if err != nil {
						c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid client ID format"})
						return
					}
					autoOrder.Clientid = req.Clientid

					var client models.Client
					if err := config.ClientCollection.FindOne(ctx, bson.M{"_id": clientObjID}).Decode(&client); err != nil {
						c.JSON(http.StatusBadRequest, gin.H{"error": "Client not found for auto customer order"})
						return
					}
					if client.HamrohCard == "" {
						c.JSON(http.StatusBadRequest, gin.H{"error": "HamrohCard is required for Peshraft client order"})
						return
					}
					autoOrder.CardNumber = client.HamrohCard

					if err := CreateCustomerOrderInternal(ctx, autoOrder); err != nil {
						log.Printf("Auto-customer-order creation failed for barcode %s: %v", p.Barcode, err)
					}
				}

			} else {
				// Already confirmed: prices only.
				upd := bson.M{
					"sellingprice": p.Whosaleprice,
					"whosaleprice": p.Whosaleprice,
					"retailprice":  p.Retailprice,
					"updated_at":   time.Now(),
				}
				if _, uerr := config.ProductCollection.UpdateOne(ctx, filter, bson.M{"$set": upd}); uerr != nil {
					c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update prices only"})
					return
				}
				p.Confirmed = true
			}

		case errors.Is(err, mongo.ErrNoDocuments):
			// Product absent in catalog. Allow insert only if this is the first confirmation (priorConfirmed=false).
			if priorConfirmed {
				c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("Product with barcode %s not found for prices-only update", p.Barcode)})
				return
			}

			var templateProduct struct {
				ProductPhotoURL        string `bson:"productphotourl"`
				ProductPhotoPreviewURL string `bson:"productphotopreviewurl"`
			}
			_ = config.ProductTemplateCollection.FindOne(ctx, bson.M{"barcode": p.Barcode}).Decode(&templateProduct)

			if len(p.ExpirationDate) == 0 {
				c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("Missing expirationdate for barcode %s", p.Barcode)})
				return
			}
			if len(p.Quantities) == 0 && p.Quantity != nil {
				p.Quantities = []float64{*p.Quantity}
			}
			totalRemain := sum(p.Quantities)
			p.RemainStock = totalRemain
			p.Confirmed = true

			newProduct := models.Product{
				ID:                     primitive.NewObjectID(),
				CategoryID:             p.CategoryID,
				Name:                   p.Name,
				Unm:                    p.UNM,
				Quantities:             p.Quantities,
				ExpirationDate:         p.ExpirationDate,
				SupplierID:             orderID,
				Purchaseprice:          p.PurchasePrice,
				Whosaleprice:           p.Whosaleprice,
				Retailprice:            p.Retailprice,
				Sellingprice:           p.Whosaleprice,
				Minimumorder:           fmt.Sprintf("%d", p.MinimumOrder),
				Barcode:                p.Barcode,
				Productphotourl:        templateProduct.ProductPhotoURL,
				Productphotopreviewurl: templateProduct.ProductPhotoPreviewURL,
				Remainstock:            totalRemain,
				CreatedAt:              time.Now(),
				UpdatedAt:              time.Now(),
			}

			if _, ierr := config.ProductCollection.InsertOne(ctx, newProduct); ierr != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to insert new product into the database"})
				return
			}

		default:
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Error checking product in the database"})
			return
		}
	}

	// Finalize order fields.
	orderTotal = round2(orderTotal)

	// Keep legacy transition, but decision-making no longer depends on it.
	newStatus := "Confirmed price for store"
	updateFields := bson.M{
		"products":     req.Products,
		"ordertotal":   orderTotal,
		"status":       newStatus,
		"payment":      req.Payment,
		"deliverytime": req.DeliveryTime,
		"updated_at":   time.Now(),
	}

	if _, err := config.SupplierOrderCollection.UpdateOne(ctx, bson.M{"_id": objID}, bson.M{"$set": updateFields}); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update supplier order"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Supplier order successfully setting selling price"})
}

// findIndex ищет индекс срока годности в массиве
// func findIndex(slice []string, value string) int {
// 	for i, v := range slice {
// 		if v == value {
// 			return i
// 		}
// 	}
// 	return -1
// }

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

// CreateCustomerOrderInternal creates order without HTTP
func CreateCustomerOrderInternal(ctx context.Context, input models.CustomerOrderInput) error {
	aggregatedProducts := aggregateProducts(input.Products)
	for i := range aggregatedProducts {
		aggregatedProducts[i].Quantity = math.Round(aggregatedProducts[i].Quantity*100) / 100
	}

	clientName := "Гость"
	isRetailClient := false
	if input.Clientid != "" && len(input.Clientid) == 24 {
		objID, err := primitive.ObjectIDFromHex(input.Clientid)
		if err == nil {
			var client models.Client
			err = config.ClientCollection.FindOne(ctx, bson.M{"_id": objID}).Decode(&client)
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
		return err
	}

	var peshraftTransactionID string
	var initialTxns []models.PeshraftTxn
	totalAmount := math.Round((total+0)*100) / 100

	if input.PaymentMethod == "Peshraft" {
		if input.CardNumber == "" {
			return fmt.Errorf("Card number is required for Peshraft payment")
		}
		success, response, err := ProcessPeshraftTransaction(input.CardNumber, totalAmount, input.CashierID)
		if err != nil || !success {
			respText := response
			if err != nil {
				respText = err.Error()
			}
			return fmt.Errorf("Failed to process Peshraft payment: %s", respText)
		}
		var peshraftResp struct {
			Transaction struct {
				ID string `json:"id"`
			} `json:"transaction"`
		}
		if json.Unmarshal([]byte(response), &peshraftResp) != nil || peshraftResp.Transaction.ID == "" {
			return fmt.Errorf("Failed to parse Peshraft response")
		}
		peshraftTransactionID = peshraftResp.Transaction.ID
		initialTxns = []models.PeshraftTxn{{ID: peshraftTransactionID, Amount: totalAmount}}
	}

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
			return fmt.Errorf("Insufficient stock for: %s", product.Barcode)
		}

		_, err := config.ProductCollection.UpdateOne(ctx,
			bson.M{"barcode": product.Barcode},
			bson.M{"$set": bson.M{
				"quantities":     update.Quantities,
				"expirationdate": update.ExpirationDates,
				"updated_at":     time.Now(),
			}})
		if err != nil {
			return err
		}

		for i := range orderedProducts {
			if orderedProducts[i].Barcode == product.Barcode {
				orderedProducts[i].Batches = usedBatches
				orderedProducts[i].StockRemaining = sumQuantitiesFloat(update.Quantities)
				break
			}
		}
	}

	newOrder := models.CustomerOrder{
		ID:                    primitive.NewObjectID(),
		Products:              orderedProducts,
		DeliveryMethod:        input.DeliveryMethod,
		DeliveryAddress:       input.DeliveryAddress,
		DeliveryCost:          0,
		PaymentMethod:         input.PaymentMethod,
		PeshraftTransactionID: peshraftTransactionID,
		PeshraftTransactions:  initialTxns,
		Tranid:                input.Tranid,
		Clientid:              input.Clientid,
		Status:                "Order confirm, in process in stock!",
		Total:                 math.Round(total*100) / 100,
		TotalAmount:           totalAmount,
		CreatedAt:             time.Now(),
		ViewToken:             uuid.NewString(),
		AutoCreatedFromStock:  input.AutoCreatedFromStock,
	}

	_, err = config.OrderCollection.InsertOne(ctx, newOrder)
	if err != nil {
		return err
	}

	utils.SendSMS(removePlusFromPhone("+992111143040"), fmt.Sprintf("Клиент %s оформил заказ на сумму %.2f сомоні", clientName, newOrder.TotalAmount))
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
