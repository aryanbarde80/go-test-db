package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"html/template"
	"log"
	"net/http"
	"time"

	_ "github.com/go-sql-driver/mysql"
	"github.com/gorilla/mux"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// Product struct for MongoDB
type Product struct {
	Name     string  `json:"name" bson:"name"`
	Quantity int     `json:"quantity" bson:"quantity"`
	Price    float64 `json:"price" bson:"price"`
}

// MySQL Product struct
type MySQLProduct struct {
	ID       int
	Name     string
	Quantity int
	Price    float64
}

var (
	mysqlDB *sql.DB
	mongoDB *mongo.Database
)

func main() {
	// Initialize MySQL
	initMySQL()

	// Initialize MongoDB
	initMongoDB()

	// Setup Router
	r := mux.NewRouter()

	// Routes
	r.HandleFunc("/", homeHandler)
	r.HandleFunc("/status", statusHandler)

	// Start Server
	port := ":5000"
	fmt.Printf("🚀 Server starting on http://localhost%s\n", port)
	log.Fatal(http.ListenAndServe(port, r))
}

// ============ MySQL Initialization ============
func initMySQL() {
	var err error
	dsn := "appuser:password123@tcp(localhost:3306)/inventory_db?charset=utf8mb4&parseTime=True&loc=Local"

	mysqlDB, err = sql.Open("mysql", dsn)
	if err != nil {
		log.Printf("⚠️ MySQL Connection Error: %v", err)
		return
	}

	mysqlDB.SetMaxOpenConns(10)
	mysqlDB.SetMaxIdleConns(5)
	mysqlDB.SetConnMaxLifetime(time.Minute * 5)

	err = mysqlDB.Ping()
	if err != nil {
		log.Printf("⚠️ MySQL Ping Error: %v", err)
		return
	}

	fmt.Println("✅ MySQL Connected Successfully!")

	// Create table if not exists
	createTableSQL := `
	CREATE TABLE IF NOT EXISTS products (
		id INT AUTO_INCREMENT PRIMARY KEY,
		name VARCHAR(255) NOT NULL,
		quantity INT DEFAULT 0,
		price DECIMAL(10,2) DEFAULT 0.00
	)`
	_, err = mysqlDB.Exec(createTableSQL)
	if err != nil {
		log.Printf("⚠️ MySQL Table Creation Error: %v", err)
		return
	}

	// Insert sample data if table is empty
	var count int
	mysqlDB.QueryRow("SELECT COUNT(*) FROM products").Scan(&count)
	if count == 0 {
		_, err = mysqlDB.Exec(`
			INSERT INTO products (name, quantity, price) VALUES
			('Laptop', 10, 50000),
			('Mouse', 25, 500),
			('Keyboard', 15, 1200),
			('Monitor', 8, 15000)
		`)
		if err != nil {
			log.Printf("⚠️ MySQL Sample Data Insert Error: %v", err)
		} else {
			fmt.Println("✅ MySQL Sample Data Inserted!")
		}
	}
}

// ============ MongoDB Initialization ============
func initMongoDB() {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	client, err := mongo.Connect(ctx, options.Client().ApplyURI("mongodb://localhost:27017"))
	if err != nil {
		log.Printf("⚠️ MongoDB Connection Error: %v", err)
		return
	}

	err = client.Ping(ctx, nil)
	if err != nil {
		log.Printf("⚠️ MongoDB Ping Error: %v", err)
		return
	}

	mongoDB = client.Database("inventory_db")
	fmt.Println("✅ MongoDB Connected Successfully!")

	// Create collection and insert sample data
	collection := mongoDB.Collection("products")

	// Check if collection is empty
	count, err := collection.CountDocuments(ctx, bson.M{})
	if err != nil {
		log.Printf("⚠️ MongoDB Count Error: %v", err)
		return
	}

	if count == 0 {
		products := []interface{}{
			Product{Name: "Laptop", Quantity: 10, Price: 50000},
			Product{Name: "Mouse", Quantity: 25, Price: 500},
			Product{Name: "Keyboard", Quantity: 15, Price: 1200},
			Product{Name: "Monitor", Quantity: 8, Price: 15000},
		}

		_, err = collection.InsertMany(ctx, products)
		if err != nil {
			log.Printf("⚠️ MongoDB Sample Data Insert Error: %v", err)
		} else {
			fmt.Println("✅ MongoDB Sample Data Inserted!")
		}
	}
}

// ============ Handlers ============

// Home Handler - Shows UI
func homeHandler(w http.ResponseWriter, r *http.Request) {
	// Get MySQL Data
	mysqlStatus := "❌ Not Connected"
	mysqlProducts := []MySQLProduct{}

	if mysqlDB != nil {
		err := mysqlDB.Ping()
		if err == nil {
			mysqlStatus = "✅ Connected"
			rows, err := mysqlDB.Query("SELECT id, name, quantity, price FROM products")
			if err == nil {
				defer rows.Close()
				for rows.Next() {
					var p MySQLProduct
					if err := rows.Scan(&p.ID, &p.Name, &p.Quantity, &p.Price); err == nil {
						mysqlProducts = append(mysqlProducts, p)
					}
				}
			}
		}
	}

	// Get MongoDB Data
	mongoStatus := "❌ Not Connected"
	mongoProducts := []Product{}

	if mongoDB != nil {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		collection := mongoDB.Collection("products")
		cursor, err := collection.Find(ctx, bson.M{})
		if err == nil {
			defer cursor.Close(ctx)
			mongoStatus = "✅ Connected"
			for cursor.Next(ctx) {
				var p Product
				if err := cursor.Decode(&p); err == nil {
					mongoProducts = append(mongoProducts, p)
				}
			}
		}
	}

	data := struct {
		MySQLStatus  string
		MySQLData    []MySQLProduct
		MongoStatus  string
		MongoData    []Product
	}{
		MySQLStatus:  mysqlStatus,
		MySQLData:    mysqlProducts,
		MongoStatus:  mongoStatus,
		MongoData:    mongoProducts,
	}

	tmpl, err := template.ParseFiles("templates/index.html")
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	tmpl.Execute(w, data)
}

// Status Handler - JSON Response
func statusHandler(w http.ResponseWriter, r *http.Request) {
	mysqlStatus := "❌ Not Connected"
	mongoStatus := "❌ Not Connected"

	if mysqlDB != nil {
		if err := mysqlDB.Ping(); err == nil {
			mysqlStatus = "✅ Connected"
		}
	}

	if mongoDB != nil {
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()
		if err := mongoDB.Client().Ping(ctx, nil); err == nil {
			mongoStatus = "✅ Connected"
		}
	}

	response := map[string]string{
		"mysql":  mysqlStatus,
		"mongodb": mongoStatus,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}
