package models

import (
	// "time"

	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

type UpdateProduct struct {
	CategoryID      string   `json:"categoryid,omitempty"`
	Name            string   `json:"name,omitempty"`
	Unm             string   `json:"unm,omitempty"`
	SupplierID      string   `json:"supplierid,omitempty"`
	Purchaseprice   float64  `json:"purchaseprice,omitempty"`
	ExpirationDate  []string `json:"expirationdate"`
	Quantity        string   `json:"quantity"`
	Quantities      []float64  `json:"quantities"`
	Sellingprice    float64  `json:"sellingprice" binding:"required"`
	Whosaleprice    float64  `json:"whosaleprice" binding:"required"`
	Retailprice     float64  `json:"retailprice" binding:"required"`
	Minimumorder    string   `json:"minimumorder,omitempty"`
	Barcode         string   `json:"barcode,omitempty"`
	Productphotourl string   `json:"productphotourl,omitempty"`
}

type UpdateCategory struct {
	Name          string `json:"name,omitempty"`
	TopCategoryID string `json:"topcategoryid,omitempty"`
}

type Product struct {
	ID              primitive.ObjectID `bson:"_id,omitempty" json:"id,omitempty"`
	CategoryID      string             `json:"categoryid" binding:"required"`
	Name            string             `json:"name" binding:"required"`
	Unm             string             `json:"unm" binding:"required"`
	ExpirationDate  []string           `json:"expirationdate"`
	Quantity        string             `json:"quantity"`
	Quantities      []float64            `json:"quantities"`
	SupplierID      string             `json:"supplierid" binding:"required"`
	Purchaseprice   float64            `json:"purchaseprice" binding:"required"`
	Sellingprice    float64            `json:"sellingprice" binding:"required"`
	Whosaleprice    float64            `json:"whosaleprice" binding:"required"`
	Retailprice     float64            `json:"retailprice" binding:"required"`
	Minimumorder    string             `json:"minimumorder" binding:"required"`
	Barcode         string             `json:"barcode" binding:"required"`
	Productphotourl string             `json:"productphotourl" binding:"required"`
	Productphotopreviewurl string             `json:"productphotopreviewurl" binding:"required"`
	Remainstock float64 `bson:"remainstock,omitempty" json:"remainstock,omitempty"`
	CreatedAt time.Time `bson:"created_at,omitempty"`
	UpdatedAt time.Time `bson:"updated_at,omitempty"`
}

type Category struct {
	ID            primitive.ObjectID `bson:"_id,omitempty" json:"id,omitempty"`
	CategoryID    string             `json:"categoryid" binding:"required"`
	Name          string             `json:"name" binding:"required"`
	TopCategoryID string             `json:"topcategoryid" binding:"required"`
	PhotoURL      string             `json:"photourl" binding:"required"`
}

type OrderItem struct {
	ProductID   string  `bson:"product_id" json:"product_id"`
	ProductName string  `bson:"product_name" json:"product_name"`
	Quantity    int     `bson:"quantity" json:"quantity"`
	Price       float64 `bson:"price" json:"price"`
	TotalPrice  float64 `bson:"total_price" json:"total_price"`
}

type Order struct {
	ID          primitive.ObjectID `bson:"_id,omitempty" json:"id,omitempty"`
	ClientID    string             `bson:"clientid" json:"clientid" binding:"required"`
	Items       []OrderItem        `bson:"items" json:"items"`
	TotalAmount float64            `bson:"total_amount" json:"total_amount"`
	Status      string             `bson:"status" json:"status"` // Например: "Pending", "Completed", "Cancelled"

	PaymentMethod string `bson:"payment_method" json:"payment_method"` // "Credit Card", "PayPal", "Cash", и т.д.
	DeliveryType  string `bson:"delivery_type" json:"delivery_type"`   // "Courier", "Pickup", "Postal Service", и т.д.

	CreatedAt time.Time `bson:"created_at" json:"created_at"`
	UpdatedAt time.Time `bson:"updated_at" json:"updated_at"`
}

type Supplier struct {
	ID            primitive.ObjectID `bson:"_id,omitempty" json:"id,omitempty"`
	SupplierID    string             `json:"supplierid" binding:"required"`
	Name          string             `bson:"name" json:"name" binding:"required"`
	ContactPerson string             `bson:"contact_person" json:"contact_person"`
	Phone         string             `bson:"phone" json:"phone" binding:"required"`
	Email         string             `bson:"email" json:"email"`
	Address       string             `bson:"address" json:"address"`
	PaymentTerms  string             `bson:"payment_terms" json:"payment_terms"`
	DeliveryTime  int                `bson:"delivery_time" json:"delivery_time"` // в днях
	Status        string             `bson:"status" json:"status"`               // "Active", "Inactive"
}

type UpdateSupplier struct {
	Name          string `json:"name"`
	ContactPerson string `json:"contact_person"`
	Phone         string `json:"phone"`
	Email         string `json:"email"`
	Address       string `json:"address"`
	PaymentTerms  string `json:"payment_terms"`
	DeliveryTime  int    `json:"delivery_time"`
	Status        string `json:"status"`
}
type CustomerOrderInput struct {
    Products              []ProductQuantity `json:"products"`
    DeliveryMethod        string            `json:"delivery_method"`
    DeliveryAddress       string            `json:"delivery_address"`
    PaymentMethod         string            `json:"payment_method"`
    CardNumber            string            `json:"card_number"`
    Tranid                string            `json:"tranid"`
    Clientid              string            `json:"clientid"`
    CashierID             string            `json:"cashierid"`
    AutoCreatedFromStock  bool              `json:"auto_created_from_stock"`

}	
type SupplierOrder struct {
	ID           primitive.ObjectID `bson:"_id,omitempty" json:"id,omitempty"`
	SupplierID   string             `bson:"supplierid" json:"supplierid" `
	FClientID     interface{}        `json:"clientid" bson:"-"`
	ClientID string            `bson:"clientid" json:"-"`
	SupplierName string             `bson:"supplier_name" json:"supplier_name"`
	Products     []SupplierProduct  `bson:"products" json:"products" `
	Status       string             `bson:"status" json:"status"` // Например: "Pending Approval", "Approved", "Rejected"
	CreatedAt    time.Time          `bson:"created_at" json:"created_at"`
	UpdatedAt    time.Time          `bson:"updated_at" json:"updated_at"`
	OrderTotal   float64            `bson:"ordertotal" json:"ordertotal"`
	Payment      string             `bson:"payment" json:"payment"`
	DeliveryTime time.Time          `bson:"deliverytime" json:"deliverytime"`
	CreatedBy  string             `bson:"cretedby" json:"cretedby" `
}


type SupplierProduct struct {
	CategoryID         string   `bson:"categoryid" json:"categoryid" binding:"required"`
	Name               string   `bson:"name" json:"name" binding:"required"`
	UNM                string   `bson:"unm" json:"unm" binding:"required"`
	ExpirationDate     []string `json:"expirationdate"`
	Quantities         []float64  `json:"quantities"`
	MinimumOrder       int      `bson:"minimumorder" json:"minimumorder" binding:"required"`
	Grossweight        float64  `json:"grossweight"`
	TotalGrossWeight	 float64  `bson:"totalgrossweight,omitempty" json:"totalgrossweight,omitempty"`
	Barcode            string   `bson:"barcode" json:"barcode" binding:"required"`
	PurchasePrice      float64  `bson:"purchaseprice,omitempty" json:"purchaseprice,omitempty"`
	Whosaleprice       float64  `json:"whosaleprice" binding:"required"`
	Retailprice        float64  `json:"retailprice" binding:"required"`
	TotalPurchasePrice float64  `bson:"totalpurchaseprice,omitempty" json:"totalpurchaseprice,omitempty"`
	SellingPrice       float64  `bson:"sellingprice,omitempty" json:"sellingprice,omitempty"`
	Remainstock        float64 `bson:"remainstock,omitempty" json:"remainstock,omitempty"`
	Confirmed          bool      `bson:"confirmed" json:"confirmed"` // 
}


type CustomerOrder struct {
	ID                    primitive.ObjectID `bson:"_id,omitempty" json:"id,omitempty"`
	Products              []OrderedProduct   `bson:"products" json:"products"`
	DeliveryMethod        string             `bson:"deliverymethod" json:"deliverymethod"`
	DeliveryAddress       string             `bson:"deliveryaddress,omitempty" json:"deliveryaddress,omitempty"`
	DeliveryCost          float64            `bson:"deliverycost,omitempty" json:"deliverycost,omitempty"`
	PaymentMethod         string             `bson:"paymentmethod" json:"paymentmethod"`
	CardNumber            string             `bson:"cardnumber,omitempty" json:"card_number,omitempty"`
	Status                string             `bson:"status" json:"status"`
	Total                 float64            `bson:"total" json:"total"`
	TotalAmount           float64            `bson:"total_amount" json:"total_amount"`
	PeshraftTransactionID string             `bson:"peshraft_transaction_id,omitempty" json:"peshraft_transaction_id,omitempty"` // для старых заказов
	PeshraftTransactions  []PeshraftTxn      `bson:"peshraft_transactions,omitempty" json:"peshraft_transactions,omitempty"`    // новые транзакции
	Qrlink                string             `bson:"qrlink" json:"qrlink"`
	Tranid                string             `bson:"tranid" json:"tranid"`
	ViewToken             string             `bson:"view_token" json:"view_token"`
	Clientid              string             `bson:"clientid" json:"clientid"`
	CreatedAt             time.Time          `bson:"created_at" json:"created_at"`
	UpdatedAt             time.Time          `bson:"updated_at" json:"updated_at"`
	EditLogs 			  []OrderEditLog 	 `bson:"edit_logs,omitempty" json:"edit_logs,omitempty"`
	AutoCreatedFromStock    bool      		 `bson:"autocreatedfromstock" json:"autocreatedfromstock"` // 
}


// CustomerOrder - структура заказа клиента
type CustomerOrderReturn struct {
	ID                   primitive.ObjectID `bson:"_id,omitempty" json:"id,omitempty"`
	OriginalOrderID    primitive.ObjectID `bson:"originalorderid,omitempty" json:"originalorderid"`
	Products        []OrderedProduct   `bson:"products" json:"products"`                                   // Список заказанных продуктов
	DeliveryMethod  string             `bson:"deliverymethod" json:"deliverymethod"`                       // Метод доставки ("warehouse" или "delivery")
	DeliveryAddress string             `bson:"deliveryaddress,omitempty" json:"deliveryaddress,omitempty"` // Адрес доставки (если выбран delivery)
	PaymentMethod   string             `bson:"paymentmethod" json:"paymentmethod"`                         // Метод оплаты ("Peshraft", "Cash", "DS")
	CardNumber      string             `bson:"cardnumber,omitempty" json:"card_number,omitempty"`          // Номер карты (если выбран Peshraft)
	Status          string             `bson:"status" json:"status"`                                       // Статус заказа ("In Process", "Pending Confirmation", и т.д.)
	RefundAmount	float64             `bson:"refundamount" json:"refundamount"`
	TotalAmount     float64            `bson:"total_amount" json:"total_amount"` 
	TransactionID	string			   `bson:"transactionid,omitempty" json:"transactionid,omitempty"` 
	// PeshraftTransactionID  string      `bson:"peshraft_transaction_id,omitempty" json:"peshraft_transaction_id,omitempty"`                          // Итоговая сумма заказа
	Qrlink			string             `bson:"qrlink" json:"qrlink"`
	ViewToken 		string 				`bson:"view_token"`
	Tranid          string             `bson:"tranid" json:"tranid"`
	Clientid        string             `bson:"clientid" json:"clientid"`
	CreatedAt       time.Time          `bson:"created_at" json:"created_at"` // Дата создания заказа
}

type PeshraftTxn struct {
	ID     string  `bson:"id" json:"id"`
	Amount float64 `bson:"amount" json:"amount"`
}

// OrderedProduct - структура продукта в заказе
type OrderedProduct struct {
	Barcode    string       `bson:"barcode" json:"barcode"`         // Штрихкод продукта
	Unm        string   `json:"unm" binding:"required"`			
	Quantity   float64        `bson:"quantity" json:"quantity"`       // Количество заказанного продукта
	UnitPrice  float64      `bson:"unit_price" json:"unit_price"`   // Цена за единицу
	Retailprice float64  `bson:"retailprice" json:"retailprice"`
	TotalRetailprice float64      `bson:"totalretailprice" json:"totalretailprice"`
	TotalPrice float64      `bson:"total_price" json:"total_price"` // Общая стоимость для продукта
	Batches    []BatchUsage `bson:"batches" json:"batches"`         // Использованные партии
	Status  string       `bson:"status" json:"status"`
	StockRemaining    float64      `bson:"stock_remaining" json:"stock_remaining"`
	MinimumOrder float64 `bson:"minimumorder,omitempty" json:"minimumorder,omitempty"`

}

type BatchUsage struct {
	ExpirationDate string `bson:"expiration_date" json:"expiration_date"`
	UsedQuantity   float64  `bson:"used_quantity" json:"used_quantity"`
}
// OrderedProduct - структура продукта в заказе
type ProductReturn struct {
	
	Barcode    string       `bson:"barcode" json:"barcode"`         // Штрихкод продукта
	Unm        string   `json:"unm" binding:"required"`			
	Quantity   int64        `bson:"quantity" json:"quantity"`       // Количество заказанного продукта
	UnitPrice  float64      `bson:"unit_price" json:"unit_price"`   // Цена за единицу
	Retailprice float64  `bson:"retailprice" json:"retailprice"`
	TotalRetailprice float64      `bson:"totalretailprice" json:"totalretailprice"`
	TotalPrice float64      `bson:"total_price" json:"total_price"` // Общая стоимость для продукта
	Batches    []BatchUsage `bson:"batches" json:"batches"`         // Использованные партии
}




type WriteOffDocument struct {
	ID         primitive.ObjectID `bson:"_id,omitempty" json:"id"`
	Products   []WriteOffItem     `bson:"products" json:"products"`
	TotalValue float64            `bson:"total_value" json:"total_value"`
	CreatedAt  time.Time          `bson:"created_at" json:"created_at"`
	Status        string       `bson:"status" json:"status"` // \"Списан\"
}

type WriteOffItem struct {
	ID		string `bson:"id,omitempty" json:"id"`
	Barcode       string       `bson:"barcode" json:"barcode"`
	Quantity      float64      `bson:"quantity" json:"quantity"`
	RemainingStock  float64      `bson:"remainingstock" json:"remainingstock"`
	PurchasePrice float64      `bson:"purchaseprice" json:"purchaseprice"`
	WriteOffValue float64      `bson:"write_off_value" json:"write_off_value"`
	Comment       string       `bson:"comment" json:"comment"`
	Batches       []BatchUsage `bson:"batches" json:"batches"`
	Status        string       `bson:"status" json:"status"` // \"Списан\"
}