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

type Product struct {
	Name     string  `json:"name" bson:"name"`
	Quantity int     `json:"quantity" bson:"quantity"`
	Price    float64 `json:"price" bson:"price"`
}

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
	log.Println("🚀 Starting application...")

	log.Println("📦 Initializing MySQL...")
	initMySQL()

	log.Println("🍃 Initializing MongoDB...")
	initMongoDB()

	r := mux.NewRouter()
	r.HandleFunc("/", homeHandler)
	r.HandleFunc("/status", statusHandler)

	port := ":5000"
	log.Printf("🚀 Server starting on http://0.0.0.0%s\n", port)
	if err := http.ListenAndServe("0.0.0.0:5000", r); err != nil {
		log.Fatalf("❌ Server failed: %v", err)
	}
}

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

	log.Println("✅ MySQL Connected Successfully!")

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
			log.Println("✅ MySQL Sample Data Inserted!")
		}
	}
}

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
	log.Println("✅ MongoDB Connected Successfully!")

	collection := mongoDB.Collection("products")

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
			log.Println("✅ MongoDB Sample Data Inserted!")
		}
	}
}

func homeHandler(w http.ResponseWriter, r *http.Request) {
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
		MySQLStatus string
		MySQLData   []MySQLProduct
		MongoStatus string
		MongoData   []Product
	}{
		MySQLStatus: mysqlStatus,
		MySQLData:   mysqlProducts,
		MongoStatus: mongoStatus,
		MongoData:   mongoProducts,
	}

	tmpl, err := template.ParseFiles("templates/index.html")
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	tmpl.Execute(w, data)
}

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
