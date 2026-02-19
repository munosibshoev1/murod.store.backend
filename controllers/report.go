package controllers

import (
	"context"
	"math"
	"net/http"
	"sort"
	"time"

	"github.com/gin-gonic/gin"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"

	// "go.mongodb.org/mongo-driver/mongo"

	"backend/config"
	"backend/models"
)

func GetProductSalesReport(c *gin.Context) {
	barcode := c.Param("barcode")
	if barcode == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Barcode is required"})
		return
	}

	var product struct {
		Name string `bson:"name"`
	}
	err := config.ProductCollection.FindOne(context.TODO(), bson.M{"barcode": barcode}).Decode(&product)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Product not found"})
		return
	}

	filter := bson.M{"status": "Заказ собран, можете забрать со склада."}
	cursor, err := config.OrderCollection.Find(context.TODO(), filter)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to retrieve orders"})
		return
	}
	defer cursor.Close(context.TODO())

	type SaleRow struct {
		OrderID    primitive.ObjectID `json:"order_id"`
		ClientName string             `json:"client_name"`
		CreatedAt  time.Time          `json:"created_at"`
		Quantity   float64            `json:"quantity"`
		StockRemaining   float64      `json:"stock_remaining"`
		UnitPrice  float64            `json:"unit_price"`
		TotalPrice float64            `json:"total_price"`
	}

	var table []SaleRow
	var totalQuantity float64

	for cursor.Next(context.TODO()) {
		var order models.CustomerOrder
		err := cursor.Decode(&order)
		if err != nil {
			continue
		}

		for _, p := range order.Products {
			if p.Barcode == barcode {
				clientID, err := primitive.ObjectIDFromHex(order.Clientid)
				if err != nil {
					continue
				}

				var client struct {
					FirstName string `bson:"first_name"`
					LastName  string `bson:"last_name"`
				}

				err = config.ClientCollection.FindOne(context.TODO(), bson.M{"_id": clientID}).Decode(&client)
				if err != nil {
					continue
				}

				row := SaleRow{
					OrderID:    order.ID,
					ClientName: client.FirstName + " " + client.LastName,
					CreatedAt:  order.CreatedAt,
					Quantity:   p.Quantity,
					StockRemaining:  p.StockRemaining,
					UnitPrice:  p.UnitPrice,
					TotalPrice: p.TotalPrice,
				}
				table = append(table, row)
				totalQuantity += p.Quantity
			}
		}
	}

	if err := cursor.Err(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Error while processing orders"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"name":           product.Name,
		"barcode":        barcode,
		"sales_table":    table,
		"total_entries":  len(table),
		"totalquantity": totalQuantity,
	})
}

func GetProductSalesReportNEW(c *gin.Context) {
	barcode := c.Param("barcode")
	if barcode == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Barcode is required"})
		return
	}

	var product struct {
		Name string `bson:"name"`
	}
	if err := config.ProductCollection.FindOne(context.TODO(), bson.M{"barcode": barcode}).Decode(&product); err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Product not found"})
		return
	}

	type TransactionRow struct {
		ID             primitive.ObjectID `json:"id"`
		Transaction    string             `json:"transaction"`
		CreatedAt      time.Time          `json:"created_at"`
		Quantity       float64            `json:"quantity"`
		StockRemaining float64            `json:"stock_remaining,omitempty"`
		UnitPrice      float64            `json:"unit_price"`
		TotalPrice     float64            `json:"total_price"`
		Comment        string             `json:"comment,omitempty"`
	}

	var (
		rows                   []TransactionRow
		totalWhosaleQty        float64
		totalWriteOffQty       float64
		totalPurchaseQty       float64
		totalWhosalePrice      float64
		totalWriteOffPrice     float64
		totalPurchasePrice     float64
		lastPurchaseUnitPrice  float64
	)

	round2 := func(val float64) float64 {
		return math.Round(val*100) / 100
	}

	orderFilter := bson.M{"status": bson.M{"$in": []string{"Заказ собран, можете забрать со склада.", "Подтвержден менеджером"}}}
	orderCur, err := config.OrderCollection.Find(context.TODO(), orderFilter)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to retrieve orders"})
		return
	}
	defer orderCur.Close(context.TODO())

	for orderCur.Next(context.TODO()) {
		var order models.CustomerOrder
		if err := orderCur.Decode(&order); err != nil {
			continue
		}
		for _, p := range order.Products {
			if p.Barcode != barcode {
				continue
			}

			clientID, err := primitive.ObjectIDFromHex(order.Clientid)
			if err != nil {
				continue
			}

			var client struct {
				FirstName string `bson:"first_name"`
				LastName  string `bson:"last_name"`
			}
			if err := config.ClientCollection.FindOne(context.TODO(), bson.M{"_id": clientID}).Decode(&client); err != nil {
				continue
			}

			totalWhosaleQty += p.Quantity
			totalWhosalePrice += p.TotalPrice

			rows = append(rows, TransactionRow{
				ID:             order.ID,
				Transaction:    "Продажа",
				CreatedAt:      order.CreatedAt,
				Quantity:       p.Quantity,
				StockRemaining: p.StockRemaining,
				UnitPrice:      p.UnitPrice,
				TotalPrice:     p.TotalPrice,
				Comment:        client.FirstName + " " + client.LastName,
			})
		}
	}

	writeOffFilter := bson.M{"status": "Списан"}
	writeOffCur, err := config.WriteOffCollection.Find(context.TODO(), writeOffFilter)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to retrieve write-offs"})
		return
	}
	defer writeOffCur.Close(context.TODO())

	for writeOffCur.Next(context.TODO()) {
		var doc struct {
			ID        primitive.ObjectID `bson:"_id"`
			Products  []struct {
				Barcode         string  `bson:"barcode"`
				Quantity        float64 `bson:"quantity"`
				RemainingStock  float64 `bson:"remainingstock"`
				PurchasePrice   float64 `bson:"purchaseprice"`
				WriteOffValue   float64 `bson:"write_off_value"`
				Status          string  `bson:"status"`
				Comment  		string  `bson:"comment"`
			} `bson:"products"`
			CreatedAt time.Time `bson:"created_at"`
		}
		if err := writeOffCur.Decode(&doc); err != nil {
			continue
		}

		for _, p := range doc.Products {
			if p.Barcode != barcode || p.Status != "Списан" {
				continue
			}

			totalWriteOffQty += p.Quantity
			totalWriteOffPrice += p.WriteOffValue

			rows = append(rows, TransactionRow{
				ID:             doc.ID,
				Transaction:    "Списание",
				CreatedAt:      doc.CreatedAt,
				Quantity:       p.Quantity,
				StockRemaining: p.RemainingStock,
				UnitPrice:      p.PurchasePrice,
				TotalPrice:     p.WriteOffValue,
				Comment: 		p.Comment,
			})
		}
	}

	supplierFilter := bson.M{"status": "Confirmed price for store"}
	supplierCur, err := config.SupplierOrderCollection.Find(context.TODO(), supplierFilter)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to retrieve supplier orders"})
		return
	}
	defer supplierCur.Close(context.TODO())

	for supplierCur.Next(context.TODO()) {
		var doc models.SupplierOrder
		if err := supplierCur.Decode(&doc); err != nil {
			continue
		}
		for _, p := range doc.Products {
			if p.Barcode != barcode {
				continue
			}

			var qty float64
			for _, q := range p.Quantities {
				qty += q
			}

			totalPurchaseQty += qty
			totalPurchasePrice += p.TotalPurchasePrice
			lastPurchaseUnitPrice = p.PurchasePrice

			rows = append(rows, TransactionRow{
				ID:             doc.ID,
				Transaction:    "Покупка",
				CreatedAt:      doc.CreatedAt,
				Quantity:       qty,
				StockRemaining: p.Remainstock,
				UnitPrice:      p.PurchasePrice,
				TotalPrice:     p.TotalPurchasePrice,
				Comment:        doc.SupplierName,
			})
		}
	}

	sort.Slice(rows, func(i, j int) bool {
		return rows[i].CreatedAt.After(rows[j].CreatedAt)
	})

	totalQuantityInStock := totalPurchaseQty - totalWhosaleQty - totalWriteOffQty
	totalPriceInStock := totalQuantityInStock * lastPurchaseUnitPrice

	c.JSON(http.StatusOK, gin.H{
		"name":                  product.Name,
		"barcode":               barcode,
		"transactions":          rows,
		"total_entries":         len(rows),
		"totalpurchase":         round2(totalPurchaseQty),
		"totalwhosale":          round2(totalWhosaleQty),
		"totalwriteoff":         round2(totalWriteOffQty),
		"totalquantityinstock": round2(totalQuantityInStock),
		"totalpurchaseprice":    round2(totalPurchasePrice),
		"totalwhosaleprice":     round2(totalWhosalePrice),
		"totalwriteoffprice":    round2(totalWriteOffPrice),
		"totalsummainstock":     round2(totalPriceInStock),
	})
}