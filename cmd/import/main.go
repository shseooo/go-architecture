// Command import is a one-off batch job that reads a CSV of items and inserts
// them through the item service (so the same validation and category linking as
// the HTTP API apply). It is a single-run command, not a long-running queue
// worker — hence it lives under cmd/import.
//
// Usage:
//
//	go run ./cmd/import -file cmd/import/items.sample.csv
//
// CSV header (column order is flexible; required: name, price, stock_quantity, type):
//
//	name,price,stock_quantity,type,author,isbn,artist,etc,director,actor,category_ids
//
// type is BOOK|ALBUM|MOVIE. category_ids is ';'-separated (e.g. "2;3").
package main

import (
	"context"
	"encoding/csv"
	"errors"
	"flag"
	"fmt"
	"io"
	"log/slog"
	"os"
	"strconv"
	"strings"

	"github.com/joho/godotenv"

	"github.com/shseooo/go-architecture/app/domain"
	mysqlRepo "github.com/shseooo/go-architecture/app/repository/mysql"
	"github.com/shseooo/go-architecture/app/service/item"
	"github.com/shseooo/go-architecture/config"
)

func main() {
	if err := run(); err != nil {
		slog.Error("import failed", "error", err)
		os.Exit(1)
	}
}

func run() error {
	csvPath := flag.String("file", "", "path to the items CSV file")
	flag.Parse()
	if *csvPath == "" {
		return errors.New("-file is required")
	}

	_ = godotenv.Load()
	db, err := config.NewMySQLConn()
	if err != nil {
		return err
	}
	defer db.Close()

	svc := item.NewService(mysqlRepo.NewItemRepository(db))

	f, err := os.Open(*csvPath)
	if err != nil {
		return err
	}
	defer f.Close()

	reader := csv.NewReader(f)
	reader.TrimLeadingSpace = true

	header, err := reader.Read()
	if err != nil {
		return fmt.Errorf("failed to read CSV header: %w", err)
	}
	cols, err := indexColumns(header)
	if err != nil {
		return err
	}

	ctx := context.Background()
	var imported, failed int
	for line := 2; ; line++ { // line 1 was the header
		record, err := reader.Read()
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			slog.Warn("skipping malformed row", "line", line, "error", err)
			failed++
			continue
		}

		it, err := parseItem(record, cols)
		if err != nil {
			slog.Warn("skipping invalid row", "line", line, "error", err)
			failed++
			continue
		}
		if err := svc.Create(ctx, it); err != nil {
			slog.Warn("failed to import row", "line", line, "name", it.Name, "error", err)
			failed++
			continue
		}
		imported++
		slog.Info("imported item", "id", it.ID, "name", it.Name)
	}

	slog.Info("import finished", "imported", imported, "failed", failed)
	if imported == 0 && failed > 0 {
		return errors.New("no rows imported")
	}
	return nil
}

// indexColumns maps required/optional column names to their position, verifying
// that the required columns are present.
func indexColumns(header []string) (map[string]int, error) {
	cols := make(map[string]int, len(header))
	for i, name := range header {
		cols[strings.TrimSpace(strings.ToLower(name))] = i
	}
	for _, required := range []string{"name", "price", "stock_quantity", "type"} {
		if _, ok := cols[required]; !ok {
			return nil, fmt.Errorf("CSV is missing required column %q", required)
		}
	}
	return cols, nil
}

func parseItem(record []string, cols map[string]int) (*domain.Item, error) {
	get := func(name string) string {
		if i, ok := cols[name]; ok && i < len(record) {
			return strings.TrimSpace(record[i])
		}
		return ""
	}

	price, err := strconv.Atoi(get("price"))
	if err != nil {
		return nil, fmt.Errorf("invalid price %q: %w", get("price"), err)
	}
	stock, err := strconv.Atoi(get("stock_quantity"))
	if err != nil {
		return nil, fmt.Errorf("invalid stock_quantity %q: %w", get("stock_quantity"), err)
	}

	categoryIDs, err := parseCategoryIDs(get("category_ids"))
	if err != nil {
		return nil, err
	}

	return &domain.Item{
		Name:          get("name"),
		Price:         price,
		StockQuantity: stock,
		Type:          domain.ItemType(strings.ToUpper(get("type"))),
		Author:        get("author"),
		ISBN:          get("isbn"),
		Artist:        get("artist"),
		Etc:           get("etc"),
		Director:      get("director"),
		Actor:         get("actor"),
		CategoryIDs:   categoryIDs,
	}, nil
}

func parseCategoryIDs(raw string) ([]int64, error) {
	if raw == "" {
		return nil, nil
	}
	parts := strings.Split(raw, ";")
	ids := make([]int64, 0, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p == "" {
			continue
		}
		id, err := strconv.ParseInt(p, 10, 64)
		if err != nil {
			return nil, fmt.Errorf("invalid category id %q: %w", p, err)
		}
		ids = append(ids, id)
	}
	return ids, nil
}
