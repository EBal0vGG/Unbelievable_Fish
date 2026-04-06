package main

import (
	"database/sql"
	"encoding/json"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	catalogapp "github.com/EBal0vGG/Unbelievable_Fish/internal/catalog/app"
	catalog "github.com/EBal0vGG/Unbelievable_Fish/internal/catalog/domain"
	catalogpg "github.com/EBal0vGG/Unbelievable_Fish/internal/catalog/postgres"
	_ "github.com/jackc/pgx/v5/stdlib"
)

func main() {
	db, ok := openDB()
	if !ok {
		log.Fatal("PGHOST/PGUSER/PGDATABASE are required")
	}
	defer db.Close()

	service := catalogapp.NewCatalogService(
		catalogpg.NewFishRepository(db),
		catalogpg.NewUnitRepository(db),
		catalogpg.NewProcessingTypeRepository(db),
		catalogpg.NewProductRepository(db),
		catalogpg.NewLotRepository(db),
		catalogpg.NewOutboxRepository(db),
		catalogapp.NewRandomIDGenerator(),
		catalogpg.NewTransactionManager(db, nil),
	)

	mux := http.NewServeMux()
	mux.HandleFunc("/fish", createFishHandler(service))
	mux.HandleFunc("/products", createProductHandler(service))
	mux.HandleFunc("/products/", publishProductHandler(service))
	mux.HandleFunc("/lots", createLotHandler(service))
	mux.HandleFunc("/lots/", lotCommandsHandler(service))
	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	port := envOrDefault("CATALOG_PORT", "8081")
	log.Printf("catalog http listening on :%s", port)
	log.Fatal(http.ListenAndServe(":"+port, mux))
}

func createFishHandler(service *catalogapp.CatalogService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}
		var req struct {
			Name        string `json:"name"`
			Description string `json:"description"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "invalid body", http.StatusBadRequest)
			return
		}
		id, err := service.CreateFish(r.Context(), catalogapp.CreateFishCommand{
			Name:        req.Name,
			Description: req.Description,
		})
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		writeJSON(w, http.StatusAccepted, map[string]string{"fish_id": id})
	}
}

func createProductHandler(service *catalogapp.CatalogService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}
		var req struct {
			FishID         string  `json:"fish_id"`
			Weight         float64 `json:"weight"`
			Unit           string  `json:"unit"`
			Size           string  `json:"size"`
			ProcessingType string  `json:"processing_type"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "invalid body", http.StatusBadRequest)
			return
		}
		id, _, err := service.CreateProduct(r.Context(), catalogapp.CreateProductCommand{
			FishID:         req.FishID,
			Weight:         req.Weight,
			Unit:           req.Unit,
			Size:           req.Size,
			ProcessingType: catalog.ProcessingType(req.ProcessingType),
		})
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		writeJSON(w, http.StatusAccepted, map[string]string{"product_id": id})
	}
}

func publishProductHandler(service *catalogapp.CatalogService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || !strings.HasSuffix(r.URL.Path, "/publish") {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}
		parts := strings.Split(strings.Trim(r.URL.Path, "/"), "/")
		if len(parts) != 3 {
			http.Error(w, "invalid path", http.StatusBadRequest)
			return
		}
		productID := parts[1]
		if err := service.PublishProduct(r.Context(), productID); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		w.WriteHeader(http.StatusAccepted)
	}
}

func createLotHandler(service *catalogapp.CatalogService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}
		var req struct {
			ProductID              string    `json:"product_id"`
			Photo                  string    `json:"photo"`
			Quantity               float64   `json:"quantity"`
			StartPrice             int64     `json:"start_price"`
			AuctionStartsAt        time.Time `json:"auction_starts_at"`
			AuctionDurationMinutes int64     `json:"auction_duration_minutes"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "invalid body", http.StatusBadRequest)
			return
		}
		durationMinutes := req.AuctionDurationMinutes
		if durationMinutes <= 0 {
			durationMinutes = envDurationMinutes("CATALOG_AUCTION_DURATION_MINUTES", 60)
		}
		companyID := r.Header.Get("X-Company-ID")
		ctx := catalogapp.WithCompanyID(r.Context(), companyID)
		id, _, err := service.CreateLot(ctx, catalogapp.CreateLotCommand{
			ProductID:              req.ProductID,
			Photo:                  req.Photo,
			Quantity:               req.Quantity,
			StartPrice:             req.StartPrice,
			AuctionStartsAt:        req.AuctionStartsAt,
			AuctionDurationMinutes: durationMinutes,
		})
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		writeJSON(w, http.StatusAccepted, map[string]string{"lot_id": id})
	}
}

func lotCommandsHandler(service *catalogapp.CatalogService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		parts := strings.Split(strings.Trim(r.URL.Path, "/"), "/")
		if len(parts) < 2 {
			http.Error(w, "invalid path", http.StatusBadRequest)
			return
		}
		lotID := parts[1]
		if len(parts) == 3 && parts[2] == "publish" && r.Method == http.MethodPost {
			if err := service.PublishLot(r.Context(), lotID); err != nil {
				http.Error(w, err.Error(), http.StatusBadRequest)
				return
			}
			w.WriteHeader(http.StatusAccepted)
			return
		}
		http.Error(w, "invalid path", http.StatusBadRequest)
	}
}

func writeJSON(w http.ResponseWriter, status int, payload any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(payload)
}

func openDB() (*sql.DB, bool) {
	host := os.Getenv("PGHOST")
	user := os.Getenv("PGUSER")
	password := os.Getenv("PGPASSWORD")
	database := os.Getenv("PGDATABASE")
	port := os.Getenv("PGPORT")
	sslmode := os.Getenv("PGSSLMODE")

	if host == "" || user == "" || database == "" {
		return nil, false
	}
	if port == "" {
		port = "5432"
	}
	if sslmode == "" {
		sslmode = "disable"
	}

	dsn := "host=" + host + " user=" + user + " dbname=" + database + " port=" + port + " sslmode=" + sslmode
	if password != "" {
		dsn += " password=" + password
	}
	db, err := sql.Open("pgx", dsn)
	if err != nil {
		return nil, false
	}
	db.SetMaxOpenConns(5)
	return db, true
}

func envOrDefault(key, def string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return def
}

func envDurationMinutes(key string, def int64) int64 {
	if value := os.Getenv(key); value != "" {
		if minutes, err := strconv.ParseInt(value, 10, 64); err == nil && minutes > 0 {
			return minutes
		}
	}
	return def
}
