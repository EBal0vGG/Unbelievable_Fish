package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"strconv"
	"strings"

	catalogbootstrap "github.com/EBal0vGG/Unbelievable_Fish/internal/catalog/bootstrap"
	catalog "github.com/EBal0vGG/Unbelievable_Fish/internal/catalog/domain"
	catalogpg "github.com/EBal0vGG/Unbelievable_Fish/internal/catalog/postgres"
)

type fishSeedItem struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description"`
}

func ensureBootstrapFish(ctx context.Context, fishRepo *catalogpg.FishRepository) error {
	enabled, err := envBoolDefaultTrue("CATALOG_BOOTSTRAP_FISH_ENABLED")
	if err != nil {
		return err
	}
	if !enabled {
		log.Printf("bootstrap_fish_disabled")
		return nil
	}

	source := "embedded"
	seedPath := strings.TrimSpace(os.Getenv("CATALOG_BOOTSTRAP_FISH_PATH"))
	seedData := catalogbootstrap.DefaultFishSeedJSON()
	if seedPath != "" {
		source = seedPath
		seedData, err = os.ReadFile(seedPath)
		if err != nil {
			return fmt.Errorf("read fish seed %q: %w", seedPath, err)
		}
	}

	log.Printf("bootstrap_fish_started source=%s seed_bytes=%d", source, len(seedData))

	var items []fishSeedItem
	if err := json.Unmarshal(seedData, &items); err != nil {
		return fmt.Errorf("parse fish seed: %w", err)
	}
	if len(items) == 0 {
		log.Printf("bootstrap_fish_completed count=0")
		return nil
	}

	saved := 0
	for _, item := range items {
		fish, err := catalog.NewFish(item.ID, item.Name, item.Description)
		if err != nil {
			return fmt.Errorf("build fish id=%q name=%q: %w", item.ID, item.Name, err)
		}
		if err := fishRepo.Save(ctx, fish); err != nil {
			return fmt.Errorf("save fish id=%q: %w", item.ID, err)
		}
		saved++
		if saved <= 3 || saved%100 == 0 || saved == len(items) {
			log.Printf("bootstrap_fish_item_saved fish_id=%s name=%q", item.ID, item.Name)
		}
	}

	log.Printf("bootstrap_fish_completed count=%d", saved)
	return nil
}

func envBoolDefaultTrue(key string) (bool, error) {
	value := strings.TrimSpace(os.Getenv(key))
	if value == "" {
		return true, nil
	}
	enabled, err := strconv.ParseBool(value)
	if err != nil {
		return false, fmt.Errorf("invalid bool env %s=%q", key, value)
	}
	return enabled, nil
}
