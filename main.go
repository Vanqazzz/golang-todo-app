package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/thedevsaddam/renderer"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"go.mongodb.org/mongo-driver/mongo/readpref"
)

var rnd *renderer.Render
var client *mongo.Client
var db *mongo.Database

const (
	dbName         string = "golang-todo"
	collectionName string = "todo"
)

type (
	TodoModel struct {
		ID        primitive.ObjectID `bson:"id,omitempty"`
		Title     string             `bson:"title"`
		Completed bool               `bson:"completed"`
		CreatedAt time.Time          `bson:"created_at"`
	}

	Todo struct {
		ID        string    `json:"id"`
		Title     string    `json:"title"`
		Completed bool      `json:"completed"`
		CreatedAt time.Time `json:"created_at"`
	}
)

func init() {

	rnd = renderer.New()
	var err error

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	client, err = mongo.Connect(ctx, options.Client().ApplyURI("mongodb://localhost:27017"))
	checkError(err)

	err = client.Ping(ctx, readpref.Primary())
	checkError(err)

	db = client.Database(dbName)
}

func checkError(err error) {

	if err != nil {
		log.Fatal(err)
	}
}

func main() {

	server := &http.Server{
		Addr:         ":8080",
		Handler:      chi.NewRouter(),
		ReadTimeout:  60 * time.Second,
		WriteTimeout: 60 * time.Second,
	}

	fmt.Println("Server started on port:", 8080)
	if err := server.ListenAndServe(); err != nil {
		log.Printf("listen:%s\n", err)
	}
}
