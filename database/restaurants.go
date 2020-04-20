package database

import (
	"context"
	"encoding/json"
	"time"

	"github.com/rishabh-bector/BenevolentBitesBack/auth"
	log "github.com/sirupsen/logrus"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
)

type Restaurant struct {
	// Mutable
	ContactEmail string              `bson:"contact" json:"contact"`
	Name         string              `bson:"name" json:"name"`
	Address      string              `bson:"address" json:"address"`
	City         string              `bson:"city" json:"city"`
	State        string              `bson:"state" json:"state"`
	Zip          string              `bson:"zip" json:"zip"`
	Website      string              `bson:"website" json:"website"`
	Yelp         string              `bson:"yelp" json:"yelp"`
	Description  string              `bson:"description" json:"description"`
	Employees    []map[string]string `bson:"employees" json:"employees"`
	Verified     bool                `bson:"verified" json:"verified"`
	Published    bool                `bson:"published" json:"published"`
	Signed       bool                `bson:"signed" json:"signed"`
	Photos       []string            `bson:"photos" json:"photos"`

	// Constant
	Owner    string          `bson:"owner" json:"owner"`
	UUID     string          `bson:"uuid" json:"uuid"`
	PlaceID  string          `bson:"placeId" json:"placeId"`
	PassHash string          `bson:"passHash" json:"passHash"`
	Square   auth.SquareAuth `bson:"square" json:"square"`
}

// UpdateRestaurant adds a new restaurant into the DB if it doesn't yet exist
// and updates existing restaurant details
func UpdateRestaurant(owner string, r Restaurant) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Add new restaurant if there are none
	oldR := DoesRestaurantExist(owner)
	if oldR.Owner == "nil" {
		// Marshal data for Mongo
		r.Owner = owner
		r.UUID = auth.GenerateUUID()
		marshaled, err := bson.Marshal(r)
		if err != nil {
			log.Error(err)
			return err
		}

		_, err = RestCollection.InsertOne(ctx, marshaled)
		if err != nil {
			return err
		}

		return nil
	}

	// Update existing restaurant
	merged := MergeRestaurants(oldR, r)
	filter := bson.D{{"owner", owner}}
	update := bson.D{{"$set", merged}}
	RestCollection.UpdateOne(ctx, filter, update)

	return nil
}

var NilRestaurant = Restaurant{Owner: "nil"}

// DoesRestaurantExist searches Mongo for a restaurant
func DoesRestaurantExist(email string) Restaurant {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	filter := bson.D{{"owner", email}}
	cur := RestCollection.FindOne(ctx, filter)

	var result Restaurant
	if cur.Err() == mongo.ErrNoDocuments {
		return NilRestaurant
	}
	cur.Decode(&result)

	return result
}

// DoesRestaurantExistUUID searches Mongo for a restaurant, by GUID
func DoesRestaurantExistUUID(uuid string) Restaurant {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	filter := bson.D{{"uuid", uuid}}
	cur := RestCollection.FindOne(ctx, filter)

	var result Restaurant
	if cur.Err() == mongo.ErrNoDocuments {
		return NilRestaurant
	}
	cur.Decode(&result)
	return result
}

// DoesRestaurantExistUUID searches Mongo for a restaurant, by Place ID
func DoesRestaurantExistPlaceID(placeID string) Restaurant {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	filter := bson.D{{"placeId", placeID}}
	cur := RestCollection.FindOne(ctx, filter)

	var result Restaurant
	if cur.Err() == mongo.ErrNoDocuments {
		return NilRestaurant
	}
	cur.Decode(&result)
	return result
}

// UpdateRestaurantSquareAuth updates square details for a restaurant
func UpdateRestaurantSquareAuth(owner string, s auth.SquareAuth) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Update existing restaurant
	filter := bson.D{{"owner", owner}}
	update := bson.D{{"$set", bson.D{{"square", s}}}}
	RestCollection.UpdateOne(ctx, filter, update)

	return nil
}

func ConvertRestToMap(u Restaurant) map[string]interface{} {
	var out map[string]interface{}
	m, _ := json.Marshal(u)
	json.Unmarshal(m, &out)
	return out
}

func ConvertMapToRest(mIn map[string]interface{}) Restaurant {
	var out Restaurant
	m, _ := json.Marshal(mIn)
	json.Unmarshal(m, &out)
	return out
}

func MergeRestaurants(uOld, uNew Restaurant) Restaurant {
	mOut := ConvertRestToMap(uOld)
	nMap := ConvertRestToMap(uNew)
	for k, v := range nMap {
		if vc, ok := v.(string); ok {
			if vc != "" {
				mOut[k] = vc
			}
		} else {
			if k == "employees" {
				if vcArray, ok := v.([]interface{}); ok {
					if len(vcArray) > 0 {
						mOut[k] = []map[string]interface{}{}
						for _, v2 := range vcArray {
							var v3 = v2.(map[string]interface{})
							mOut[k] = append(mOut[k].([]map[string]interface{}), v3)
						}
					}
				}
			}
			if k == "square" {
				mOut[k] = auth.MergeSquareAuths(uOld.Square, uNew.Square)
			}
			if k == "published" || k == "signed" || k == "verified" {
				if vc, ok := v.(bool); ok {
					mOut[k] = vc
				}
			}
			if k == "photos" {
				if vc, ok := v.([]interface{}); ok {
					if len(vc) > 0 {
						mOut[k] = []string{}
						for _, v2 := range vc {
							mOut[k] = append(mOut[k].([]string), v2.(string))
						}
					}
				}
			}
		}
	}

	return ConvertMapToRest(mOut)
}

func GetAllPublishedRestaurants() []Restaurant {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	filter := bson.D{{"published", true}}
	cur, err := RestCollection.Find(ctx, filter)
	if err != nil {
		log.Info(err)
	}

	var result []Restaurant
	if cur.Err() == mongo.ErrNoDocuments {
		return nil
	}
	cur.All(ctx, &result)

	return result
}
