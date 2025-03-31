package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"time"

	"github.com/go-chi/chi/middleware"
	"github.com/go-chi/chi/v5"
	"github.com/thedevsaddam/renderer"
	"go.mongodb.org/mongo-driver/bson"
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
	GetTodoResponse struct {
		Message string `json:"message"`
		Data    []Todo `json:"data"`
	}
	CreateTodo struct {
		Title string `json:"title"`
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

func homeHandler(rw http.ResponseWriter, r *http.Request) {
	filepath := "./README.md"
	err := rnd.FileView(rw, http.StatusOK, filepath, "readme.md")
	checkError(err)
}

func main() {
	router := chi.NewRouter()
	router.Use(middleware.Logger)
	router.Get("/", homeHandler)
	router.Mount("/todo", todoHandlers())

	server := &http.Server{
		Addr:         ":8080",
		Handler:      router,
		ReadTimeout:  60 * time.Second,
		WriteTimeout: 60 * time.Second,
	}

	// channel for receive signal
	stopChan := make(chan os.Signal, 1)
	signal.Notify(stopChan, os.Interrupt)

	go func() {
		fmt.Println("Server started on port:", 8080)
		if err := server.ListenAndServe(); err != nil {
			log.Printf("listen:%s\n", err)
		}
	}()

	// wait for a signal to shut down the server
	sig := <-stopChan
	log.Printf("signal recevied:%v", sig)

	// disconnect mongo client from db
	if err := client.Disconnect(context.Background()); err != nil {
		panic(err)
	}

	// create a context with a timeout for safely shutdown
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// shutdown  the server
	if err := server.Shutdown(ctx); err != nil {
		log.Fatalf("Server shutdown failed: %v\n", err)
	}
	log.Println("Server shutdown")
}

func todoHandlers() http.Handler {
	router := chi.NewRouter()
	router.Group(func(r chi.Router) {
		r.Get("/", getTodos)
		r.Post("/", createTodo)
		r.Put("/{id}", updateTodo)
		r.Delete("/{id}", deleteTodo)
	})
	return router
}

func getTodos(rw http.ResponseWriter, r *http.Request) {
	var todoListFromDB = []TodoModel{}
	filter := bson.D{}

	cursor, err := db.Collection(collectionName).Find(context.Background(), filter)
	if err != nil {
		log.Printf("failed to fetch todo records from the db: %v\n", err.Error())
		rnd.JSON(rw, http.StatusBadRequest, renderer.M{
			"message": "Could not  fetch the todo collection",
			"error":   err.Error(),
		})
		return
	}

	todoList := []Todo{}
	if err = cursor.All(context.Background(), &todoListFromDB); err != nil {
		checkError(err)
	}

	for _, td := range todoListFromDB {
		todoList = append(todoList, Todo{
			ID:        td.ID.Hex(),
			Title:     td.Title,
			Completed: td.Completed,
			CreatedAt: td.CreatedAt,
		})
	}
	rnd.JSON(rw, http.StatusOK, GetTodoResponse{
		Message: "All todos retrieved",
		Data:    todoList,
	})
}

func createTodo(rw http.ResponseWriter, r *http.Request) {
	var todoReq CreateTodo
	if err := json.NewDecoder(r.Body).Decode(&todoReq); err != nil {
		log.Printf("failed to decode json data:%v", err.Error())
		rnd.JSON(rw, http.StatusBadRequest, renderer.M{
			"message": "could not decode data",
		})
		return
	}

	if todoReq.Title == "" {
		log.Println("no title added to response body")
		rnd.JSON(rw, http.StatusBadRequest, renderer.M{
			"message": "please add a title",
		})
		return
	}

	// create  a todoModel for adding a todo to the  database
	todoModel := TodoModel{
		ID:        primitive.NewObjectID(),
		Title:     todoReq.Title,
		Completed: false,
		CreatedAt: time.Now(),
	}

	// add the todo to database
	data, err := db.Collection(collectionName).InsertOne(r.Context(), todoModel)
	if err != nil {
		log.Printf("failed to insert data into the database: %v\n", err.Error())

		rnd.JSON(rw, http.StatusBadRequest, renderer.M{
			"message": "Failed to insert data into database",
			"error":   err.Error(),
		})
		return
	}
	rnd.JSON(rw, http.StatusCreated, renderer.M{

		"message": "Todo created succesfully",
		"ID":      data.InsertedID,
	})

}

func checkError(err error) {

	if err != nil {
		log.Fatal(err)
	}
}
