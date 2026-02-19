package models

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

type User struct {
    ID                 primitive.ObjectID `bson:"_id,omitempty" json:"id,omitempty"`
    FirstName          string             `bson:"first_name" json:"first_name"`
    LastName           string             `bson:"last_name" json:"last_name"`
    BirthDate          string             `bson:"birth_date" json:"birth_date"`
    Phone              string             `bson:"phone" json:"phone"`
    ImageURL           string             `bson:"image_url,omitempty" json:"image_url,omitempty"`
    Role               string             `bson:"role" json:"role"`
    Password           string             `bson:"password,omitempty" json:"password,omitempty"`
    Permissions        string             `bson:"permissions,omitempty" json:"permissions,omitempty"`
    RecoveryCode       string             `bson:"recovery_code,omitempty" json:"recoveryCode,omitempty"`
    RecoveryExpires    time.Time          `bson:"recovery_expires,omitempty" json:"recoveryExpires,omitempty"`
    
}

type Cashier struct {
    ID        primitive.ObjectID `bson:"_id,omitempty" json:"id,omitempty"`
    FirstName string             `bson:"first_name" json:"first_name"`
    LastName  string             `bson:"last_name" json:"last_name"`
    Phone     string             `bson:"phone" json:"phone"`
    BirthDate string             `bson:"birth_date" json:"birth_date"`
    Password  string             `bson:"password" json:"password"`
    Role      string             `bson:"role" json:"role"`
    Location string              `bson:"location" json:"location"`
    RecoveryCode    string             `bson:"recovery_code,omitempty" json:"recoveryCode,omitempty"`
    RecoveryExpires time.Time          `bson:"recovery_expires,omitempty" json:"recoveryExpires,omitempty"`
}

type Storekeeper struct {
    ID        primitive.ObjectID `bson:"_id,omitempty" json:"id,omitempty"`
    FirstName string             `bson:"first_name" json:"first_name"`
    LastName  string             `bson:"last_name" json:"last_name"`
    Phone     string             `bson:"phone" json:"phone"`
    BirthDate string             `bson:"birth_date" json:"birth_date"`
    Password  string             `bson:"password" json:"password"`
    Role      string             `bson:"role" json:"role"`
    Location string              `bson:"location" json:"location"`
    RecoveryCode    string             `bson:"recovery_code,omitempty" json:"recoveryCode,omitempty"`
    RecoveryExpires time.Time          `bson:"recovery_expires,omitempty" json:"recoveryExpires,omitempty"`
}


type Operator struct {
    ID        primitive.ObjectID `bson:"_id,omitempty" json:"id,omitempty"`
    FirstName string             `bson:"first_name" json:"first_name"`
    LastName  string             `bson:"last_name" json:"last_name"`
    Phone     string             `bson:"phone" json:"phone"`
    BirthDate string             `bson:"birth_date" json:"birth_date"`
    Password  string             `bson:"password" json:"password"`
    Role      string             `bson:"role" json:"role"`
    // Location string              `bson:"location" json:"location"`
}

type Client struct {
    ID                primitive.ObjectID `bson:"_id,omitempty" json:"id"`
    FirstName          string             `bson:"first_name" json:"first_name"`
    LastName           string             `bson:"last_name" json:"last_name"`
    BirthDate          string             `bson:"birth_date" json:"birth_date"`
    Phone              string             `bson:"phone" json:"phone"`
    Email              string             `bson:"email" json:"email"`
    CardNumber         string             `bson:"cardnumber,omitempty" json:"cardnumber,omitempty"`
    Role               string             `bson:"role" json:"role"`
    Password           string             `bson:"password" json:"password"`
    Limit              float64            `json:"limit" binding:"required"`
    Gender             string             `bson:"gender" json:"gender"`
    Permissions        string             `bson:"permissions,omitempty" json:"permissions,omitempty"`
    Photo_url          string             `bson:"photo_url,omitempty"` 
    Avatarurl          string             `bson:"avatarurl,omitempty"` 
    HamrohCard         string             `bson:"hamrohcard,omitempty" json:"hamrohcard,omitempty"`
    Type         string             `bson:"type" json:"type"` 
}