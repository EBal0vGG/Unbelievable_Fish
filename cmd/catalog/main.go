package main

import (
	"database/sql"
	"encoding/json"
	"log"
	"net/http"
	"os"
	"strconv"
	"time"

	catalogapp "github.com/EBal0vGG/Unbelievable_Fish/internal/catalog/app"
	catalog "github.com/EBal0vGG/Unbelievable_Fish/internal/catalog/domain"
	catalogpg "github.com/EBal0vGG/Unbelievable_Fish/internal/catalog/postgres"
	identityauth "github.com/EBal0vGG/Unbelievable_Fish/internal/identity/auth"
	identity "github.com/EBal0vGG/Unbelievable_Fish/internal/identity/domain"
	"github.com/go-chi/chi/v5"
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
	tokenProvider := identityauth.NewTokenProvider(
		envOrDefault("IDENTITY_TOKEN_SECRET", "dev-secret"),
		time.Duration(envDurationMinutes("IDENTITY_TOKEN_TTL_MINUTES", 24*60))*time.Minute,
	)
	authMiddleware := identityauth.NewMiddleware(tokenProvider, writeCatalogAuthError)

	router := chi.NewRouter()
	router.MethodFunc(http.MethodGet, "/fish", listFishHandler(service))
	router.MethodFunc(http.MethodPost, "/fish", createFishHandler(service))
	router.MethodFunc(http.MethodPost, "/products", createProductHandler(service))
	router.MethodFunc(http.MethodPost, "/products/{productID}/publish", publishProductHandler(service))
	router.Method(http.MethodPost, "/lots", authMiddleware.RequireRole(identity.RoleSeller, createLotHandler(service)))
	router.Method(http.MethodPost, "/lots/{lotID}/publish", authMiddleware.RequireRole(identity.RoleSeller, lotCommandsHandler(service)))
	router.Get("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	port := envOrDefault("CATALOG_PORT", "8081")
	log.Printf("catalog http listening on :%s", port)
	log.Fatal(http.ListenAndServe(":"+port, router))
}

func listFishHandler(service *catalogapp.CatalogService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		list, err := service.ListFish(r.Context())
		if err != nil {
			log.Printf("catalog_list_fish_error err=%v", err)
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		type fishItem struct {
			ID          string `json:"id"`
			FishID      string `json:"fish_id"`
			Name        string `json:"name"`
			Description string `json:"description"`
		}
		out := make([]fishItem, 0, len(list))
		for _, fish := range list {
			out = append(out, fishItem{
				ID:          fish.ID(),
				FishID:      fish.ID(),
				Name:        fish.Name(),
				Description: fish.Description(),
			})
		}
		writeJSON(w, http.StatusOK, out)
	}
}

func createFishHandler(service *catalogapp.CatalogService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			Name        string `json:"name"`
			Description string `json:"description"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			log.Printf("catalog_create_fish_invalid_body err=%v", err)
			http.Error(w, "invalid body", http.StatusBadRequest)
			return
		}
		id, err := service.CreateFish(r.Context(), catalogapp.CreateFishCommand{
			Name:        req.Name,
			Description: req.Description,
		})
		if err != nil {
			log.Printf("catalog_create_fish_error err=%v", err)
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		writeJSON(w, http.StatusAccepted, map[string]string{"fish_id": id})
	}
}

func createProductHandler(service *catalogapp.CatalogService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			FishID         string  `json:"fish_id"`
			Weight         float64 `json:"weight"`
			Unit           string  `json:"unit"`
			Size           string  `json:"size"`
			ProcessingType string  `json:"processing_type"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			log.Printf("catalog_create_product_invalid_body err=%v", err)
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
			log.Printf("catalog_create_product_error fish_id=%s unit=%s processing_type=%s err=%v", req.FishID, req.Unit, req.ProcessingType, err)
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		writeJSON(w, http.StatusAccepted, map[string]string{"product_id": id})
	}
}

func publishProductHandler(service *catalogapp.CatalogService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		productID, err := productIDFromRequest(r)
		if err != nil {
			http.Error(w, "invalid path", http.StatusBadRequest)
			return
		}
		if err := service.PublishProduct(r.Context(), productID); err != nil {
			log.Printf("catalog_publish_product_error product_id=%s err=%v", productID, err)
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		w.WriteHeader(http.StatusAccepted)
	}
}

func createLotHandler(service *catalogapp.CatalogService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			ProductID              string    `json:"product_id"`
			Photo                  string    `json:"photo"`
			Quantity               float64   `json:"quantity"`
			StartPrice             int64     `json:"start_price"`
			MinBidStep             int64     `json:"min_bid_step"`
			AuctionStartsAt        time.Time `json:"auction_starts_at"`
			AuctionDurationMinutes int64     `json:"auction_duration_minutes"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			log.Printf("catalog_create_lot_invalid_body err=%v", err)
			http.Error(w, "invalid body", http.StatusBadRequest)
			return
		}
		durationMinutes := req.AuctionDurationMinutes
		if durationMinutes <= 0 {
			durationMinutes = envDurationMinutes("CATALOG_AUCTION_DURATION_MINUTES", 60)
		}
		companyID := companyIDFromRequest(r)
		ctx := catalogapp.WithCompanyID(r.Context(), companyID)
		id, _, err := service.CreateLot(ctx, catalogapp.CreateLotCommand{
			ProductID:              req.ProductID,
			Photo:                  req.Photo,
			Quantity:               req.Quantity,
			StartPrice:             req.StartPrice,
			MinBidStep:             req.MinBidStep,
			AuctionStartsAt:        req.AuctionStartsAt,
			AuctionDurationMinutes: durationMinutes,
		})
		if err != nil {
			log.Printf("catalog_create_lot_error product_id=%s company_id=%s err=%v", req.ProductID, companyID, err)
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		writeJSON(w, http.StatusAccepted, map[string]string{"lot_id": id})
	}
}

func lotCommandsHandler(service *catalogapp.CatalogService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		lotID, err := lotIDFromRequest(r)
		if err != nil {
			http.Error(w, "invalid path", http.StatusBadRequest)
			return
		}
		if err := service.PublishLot(r.Context(), lotID); err != nil {
			log.Printf("catalog_publish_lot_error lot_id=%s err=%v", lotID, err)
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		w.WriteHeader(http.StatusAccepted)
	}
}

func productIDFromRequest(r *http.Request) (string, error) {
	if productID := chi.URLParam(r, "productID"); productID != "" {
		return productID, nil
	}
	return "", catalog.ErrInvalidIdentifier
}

func lotIDFromRequest(r *http.Request) (string, error) {
	if lotID := chi.URLParam(r, "lotID"); lotID != "" {
		return lotID, nil
	}
	return "", catalog.ErrInvalidIdentifier
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

func companyIDFromRequest(r *http.Request) string {
	if companyID, ok := identityauth.CompanyIDFromContext(r.Context()); ok {
		return companyID
	}
	return r.Header.Get("X-Company-ID")
}

func writeCatalogAuthError(w http.ResponseWriter, r *http.Request, err error) {
	switch {
	case err == identityauth.ErrMissingAuthorizationHeader:
		http.Error(w, "missing Authorization header", http.StatusUnauthorized)
	case err == identityauth.ErrInvalidAuthorizationHeader:
		http.Error(w, "invalid Authorization header", http.StatusUnauthorized)
	case err == identityauth.ErrInvalidToken || err == identityauth.ErrExpiredToken:
		http.Error(w, "invalid token", http.StatusUnauthorized)
	case err == identityauth.ErrForbidden:
		http.Error(w, "forbidden", http.StatusForbidden)
	default:
		http.Error(w, "internal error", http.StatusInternalServerError)
	}
}
