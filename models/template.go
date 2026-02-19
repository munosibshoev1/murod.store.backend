package models

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

type UpdateProductTemplate struct {
	CategoryID     string `json:"categoryid,omitempty"`
	Name           string `json:"name,omitempty"`
	Unm            string `json:"unm,omitempty"`
	Minimumorder   int `json:"minimumorder,omitempty"`
	Barcode        string `json:"barcode,omitempty"`
	Productphotourl string `json:"productphotourl,omitempty"`
	Productphotopreviewurl string             `json:"productphotopreviewurl" binding:"required"`
}


type BarcodeHistory struct {
	Barcode   string    `bson:"barcode" json:"barcode"`
	ChangedAt time.Time `bson:"changedat" json:"changedat"`
}

type ProductTemplate struct {
	ID                     primitive.ObjectID `bson:"_id,omitempty" json:"id,omitempty"`
	CategoryID             string             `json:"categoryid" binding:"required"`
	Name                   string             `json:"name" binding:"required"`
	Unm                    string             `json:"unm" binding:"required"`
	MinimumOrder           int                `bson:"minimumorder" json:"minimumorder" binding:"required"`
	Barcode                string             `json:"barcode" binding:"required"`
	Productphotourl        string             `json:"productphotourl" binding:"required"`
	Productphotopreviewurl string             `json:"productphotopreviewurl" binding:"required"`
	BarcodeHistory         []BarcodeHistory   `bson:"barcodehistory,omitempty" json:"barcodehistory,omitempty"`
	Grossweight            float64             `json:"grossweight" binding:"required"`
}

