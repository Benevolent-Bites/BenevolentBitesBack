package database

import (
	"context"
	"encoding/json"
	"time"

	log "github.com/sirupsen/logrus"

	"github.com/rishabh-bector/BenevolentBitesBack/auth"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
)

type User struct {
	Email string `bson:"email" json:"email"`
	Name  string `bson:"name" json:"name"`
	Token string `bson:"token" json:"token"`
	Class string `bson:"class" json:"class"`

	Address string `bson:"address" json:"address"`
	City    string `bson:"city" json:"city"`
	State   string `bson:"state" json:"state"`
	Zip     string `bson:"zip" json:"zip"`

	Square auth.SquareAuth `bson:"square" json:"square"`
}

// NilUser represents nil user
var NilUser = User{Email: "nil"}
var SuccessUser = User{Email: "success"}

// UpdateUser updates existing user details
func UpdateUser(email string, u User) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Marshal data for Mongo
	marshaled, err := bson.Marshal(u)
	if err != nil {
		log.Error(err)
		return err
	}

	// Add new user if there are none
	oldR := DoesUserExist(email)
	if oldR.Email == "nil" {
		_, err := UserCollection.InsertOne(ctx, marshaled)
		if err != nil {
			return err
		}
		return nil
	}

	// Update existing user
	merged := MergeUsers(oldR, u)
	filter := bson.D{{"email", email}}
	update := bson.D{{"$set", merged}}
	UserCollection.UpdateOne(ctx, filter, update)

	return nil
}

// ValidateUser authorizes incoming frontend requests through the user's JWT
//	- if the user is already in Mongo, update the token in Mongo
//	- if the user is not in Mongo, add user to Mongo
//	- if the token is invalid, return NilUser
//  "class" determines whether the user is a restaurant or a customer
func ValidateUser(token string) User {
	log.Info("BB: Validating user")
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	claims, err := auth.ValidateToken(token)
	if claims == nil || err != nil {
		// Invalid token
		return NilUser
	}
	email := claims["email"].(string)
	result := DoesUserExist(email)

	if result.Email == "nil" { // User does not exist in Mongo
		newUser := User{
			Email:  email,
			Token:  token,
			Square: auth.SquareAuth{},
		}
		marshaled, err := bson.Marshal(newUser)
		if err != nil {
			log.Error(err)
		}

		_, err = UserCollection.InsertOne(ctx, marshaled)
		if err != nil {
			log.Error(err)
		}
		return SuccessUser

	} else { // User already exists in Mongo
		filter := bson.D{{"email", email}}
		update := bson.D{
			{"$set", bson.D{
				{"token", token},
			}},
		}
		UserCollection.UpdateOne(ctx, filter, update)
		return SuccessUser
	}
}

// DoesUserExist searches Mongo for a User
func DoesUserExist(email string) User {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	filter := bson.D{{"email", email}}
	cur := UserCollection.FindOne(ctx, filter)

	var result User
	if cur.Err() == mongo.ErrNoDocuments {
		return NilUser
	}
	cur.Decode(&result)
	return result
}

func ConvertUserToMap(u User) map[string]interface{} {
	var out map[string]interface{}
	m, _ := json.Marshal(u)
	json.Unmarshal(m, &out)
	return out
}

func ConvertMapToUser(mIn map[string]interface{}) (User, error) {
	var out User
	m, _ := json.Marshal(mIn)
	err := json.Unmarshal(m, &out)
	if err != nil {
		return out, err
	}
	return out, nil
}

func MergeUsers(uOld, uNew User) User {
	mOut := ConvertUserToMap(uOld)
	nMap := ConvertUserToMap(uNew)
	for k, v := range nMap {
		if vc, ok := v.(string); ok {
			if vc != "" {
				mOut[k] = vc
			}
		} else {
			if k == "square" {
				mOut[k] = auth.MergeSquareAuths(uOld.Square, uNew.Square)
			}
		}
	}

	finalUser, err := ConvertMapToUser(mOut)
	if err != nil {
		log.Error(err)
	}

	return finalUser
}
