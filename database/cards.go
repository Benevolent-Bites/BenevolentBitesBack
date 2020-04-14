package database

import (
	"context"
	"errors"
	"time"

	"github.com/rishabh-bector/BenevolentBitesBack/auth"
	"github.com/rishabh-bector/BenevolentBitesBack/crypto"

	log "github.com/sirupsen/logrus"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
)

type Card struct {
	UUID         string        `bson:"uuid" json:"uuid"`
	RestUUID     string        `bson:"restaurant" json:"restaurant"`
	User         string        `bson:"user" json:"user"`
	Balance      int           `bson:"balance" json:"balance"`
	Transactions []Transaction `bson:"transactions" json:"transactions"`
}

type Transaction struct {
	Timestamp string `bson:"timestamp" json:"timestamp"`
	Amount    int    `bson:"amount" json:"amount"`
	ID        string `bson:"id" json:"id"`
	Signature string `bson:"signature" json:"signature"`
}

var NilCard = Card{UUID: "nil"}

// CreateCard makes a new card with empty balance
func CreateCard(user string, restaurant string, trans Transaction) (Card, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	c := Card{
		UUID:         auth.GenerateUUID(),
		RestUUID:     restaurant,
		User:         user,
		Balance:      trans.Amount,
		Transactions: []Transaction{trans},
	}

	// Marshal data for Mongo
	marshaled, err := bson.Marshal(c)
	if err != nil {
		log.Error(err)
		return NilCard, err
	}

	// Insert into database
	_, err = CardCollection.InsertOne(ctx, marshaled)
	if err != nil {
		return NilCard, err
	}

	return NilCard, nil
}

// AddCredit adds credit to an already existing card
func AddCredit(id, transaction_id string, amount int) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Check whether card exists
	c := DoesCardExist(id)
	if c.UUID == "nil" {
		return errors.New("sorry bro, that card doesn't exist")
	}

	// Update balance
	filter := bson.D{{"uuid", id}}
	c.Balance = c.Balance + amount

	// Add transaction
	trans := Transaction{
		ID:        transaction_id,
		Timestamp: time.Now().Format(time.RFC3339),
		Amount:    amount,
	}
	c.Transactions = append(c.Transactions, trans)

	CardCollection.UpdateOne(ctx, filter, c)

	return nil
}

// SubtractCredit removes credit from an already existing card
func SubtractCredit(id string, amount int, signature string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Check whether card exists
	c := DoesCardExist(id)
	if c.UUID == "nil" {
		return errors.New("sorry bro, that card doesn't exist")
	}

	// Update balance
	filter := bson.D{{"uuid", id}}
	c.Balance = c.Balance - amount

	if c.Balance < 0 {
		return errors.New("sorry bro, not enough balance in card")
	}

	// Add transaction
	trans := Transaction{
		ID:        "",
		Timestamp: time.Now().Format(time.RFC3339),
		Amount:    -1 * amount,
		Signature: signature,
	}
	c.Transactions = append(c.Transactions, trans)

	update := bson.D{{"$set", c}}
	CardCollection.UpdateOne(ctx, filter, update)

	return nil
}

// GetUserCards retrieves all the cards which belong to a given user
func GetUserCards(user string) ([]Card, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	filter := bson.D{{"user", user}}
	cur, err := CardCollection.Find(ctx, filter)
	if err != nil {
		return nil, err
	}

	result := []Card{}

	if cur.Err() == mongo.ErrNoDocuments {
		return nil, errors.New("sorry bro, couldn't find any cards for that user")
	}

	for cur.Next(context.TODO()) {
		var card Card
		cur.Decode(&card)
		result = append(result, card)
	}

	a := result[:0]
	for _, card := range result {
		card.UUID, err = crypto.SignString(card.UUID)
		if err != nil {
			return nil, err
		}
		a = append(a, card)
	}

	return a, nil
}

// DoesCardExist searches Mongo for a Card
func DoesCardExist(uuid string) Card {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	filter := bson.D{{"uuid", uuid}}
	cur := CardCollection.FindOne(ctx, filter)

	var result Card
	if cur.Err() == mongo.ErrNoDocuments {
		return NilCard
	}
	cur.Decode(&result)
	return result
}
