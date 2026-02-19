package models

import (
    // "time"

    "go.mongodb.org/mongo-driver/bson/primitive"
)


type Card struct {
	ID              primitive.ObjectID `bson:"_id,omitempty" json:"id,omitempty"`
	CardNumber      string             `json:"cardnumber" binding:"required"`
	Status          string             `json:"status" binding:"required"`
	// CreateDate      primitive.DateTime `json:"createdate" binding:"required"`
	Limit           float64            `json:"limit" binding:"required"`
	Limits          float64            `json:"limits" binding:"required"`
	TotalPurchase   float64            `json:"totalpurchase" binding:"required"`
	TotalLoan       float64            `json:"totalloan" binding:"required"`
	TotalOut        float64            `json:"totalout" binding:"required"`
	TotalFast       float64            `json:"totalfast" binding:"required"`
	TotalSettle     float64  		   `json:"totalsettle" binding:"required"`
	AllTotal 		float64     	   `json:"alltotal" binding:"required"`
    CreateDate      primitive.DateTime `json:"createdate" binding:"required"`
	StartDate       primitive.DateTime `json:"startdate" binding:"required"`
	Days            int64              `json:"days" binding:"required"`
	Retday          int64              `json:"retday" binding:"required"`
	AllDays         int64              `json:"alldays" binding:"required"`
}