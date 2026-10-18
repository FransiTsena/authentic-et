package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"log/slog"
	"os"
	"time"

	"github.com/authentic-et/authentic-et/internal/config"
	"github.com/authentic-et/authentic-et/internal/database"
	"github.com/authentic-et/authentic-et/internal/domain"
	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
)

type DatabaseDump struct {
	ExportedAt time.Time                 `json:"exportedAt"`
	Driver     string                    `json:"driver"`
	Users      []*domain.User            `json:"users"`
	APIKeys    []*domain.APIKey          `json:"apiKeys"`
	Payments   []*domain.Payment         `json:"payments"`
	Orders     []*domain.Order           `json:"orders"`
	Jobs       []*domain.ProvisioningJob `json:"jobs"`
}

func main() {
	if len(os.Args) < 2 {
		printUsage()
		os.Exit(1)
	}

	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	cfg, err := config.Load()
	if err != nil {
		logger.Error("configuration error", "error", err)
		os.Exit(1)
	}

	ctx := context.Background()

	switch os.Args[1] {
	case "create-admin":
		fs := flag.NewFlagSet("create-admin", flag.ExitOnError)
		email := fs.String("email", "", "Admin email address")
		password := fs.String("password", "", "Admin password")
		name := fs.String("name", "Admin", "Admin name")
		_ = fs.Parse(os.Args[2:])

		if *email == "" || *password == "" {
			fmt.Println("Error: --email and --password flags are required")
			os.Exit(1)
		}

		repos, closer, err := database.Initialize(ctx, cfg, logger)
		if err != nil {
			logger.Error("failed to connect to database", "error", err)
			os.Exit(1)
		}
		defer closer()

		hashed, _ := bcrypt.GenerateFromPassword([]byte(*password), bcrypt.DefaultCost)
		user := &domain.User{
			ID:           uuid.NewString(),
			Email:        *email,
			PasswordHash: string(hashed),
			Name:         *name,
			Role:         domain.RoleAdmin,
			Active:       true,
		}

		if err := repos.Users.Create(ctx, user); err != nil {
			logger.Error("failed to create admin", "error", err)
			os.Exit(1)
		}
		fmt.Printf("Admin user '%s' created successfully.\n", *email)

	case "export-data":
		fs := flag.NewFlagSet("export-data", flag.ExitOnError)
		outFile := fs.String("out", "authentic_export.json", "Output JSON export file")
		_ = fs.Parse(os.Args[2:])

		repos, closer, err := database.Initialize(ctx, cfg, logger)
		if err != nil {
			logger.Error("failed to connect to database", "error", err)
			os.Exit(1)
		}
		defer closer()

		fmt.Println("Exporting database data...")
		dump := &DatabaseDump{
			ExportedAt: time.Now().UTC(),
			Driver:     cfg.DatabaseDriver,
		}

		keys, _ := repos.APIKeys.ListAll(ctx)
		dump.APIKeys = keys

		payments, _, _ := repos.Payments.List(ctx, domain.PaymentListFilter{Limit: 10000})
		dump.Payments = payments

		orders, _, _ := repos.Orders.ListAll(ctx, 10000, 0)
		dump.Orders = orders

		jobs, _, _ := repos.Provisioning.List(ctx, "", 10000, 0)
		dump.Jobs = jobs

		fileBytes, err := json.MarshalIndent(dump, "", "  ")
		if err != nil {
			logger.Error("failed to marshal export JSON", "error", err)
			os.Exit(1)
		}

		if err := os.WriteFile(*outFile, fileBytes, 0644); err != nil {
			logger.Error("failed to write export file", "error", err)
			os.Exit(1)
		}

		fmt.Printf("Export completed successfully: %s (Records: %d payments, %d keys, %d orders, %d jobs)\n",
			*outFile, len(dump.Payments), len(dump.APIKeys), len(dump.Orders), len(dump.Jobs))

	case "import-data":
		fs := flag.NewFlagSet("import-data", flag.ExitOnError)
		inFile := fs.String("in", "authentic_export.json", "Input JSON file to import")
		_ = fs.Parse(os.Args[2:])

		fileBytes, err := os.ReadFile(*inFile)
		if err != nil {
			logger.Error("failed to read import file", "error", err)
			os.Exit(1)
		}

		var dump DatabaseDump
		if err := json.Unmarshal(fileBytes, &dump); err != nil {
			logger.Error("failed to parse import JSON", "error", err)
			os.Exit(1)
		}

		repos, closer, err := database.Initialize(ctx, cfg, logger)
		if err != nil {
			logger.Error("failed to connect to database", "error", err)
			os.Exit(1)
		}
		defer closer()

		fmt.Printf("Importing data into %s...\n", cfg.DatabaseDriver)
		importedPayments := 0
		for _, p := range dump.Payments {
			if err := repos.Payments.Create(ctx, p); err == nil {
				importedPayments++
			}
		}

		importedKeys := 0
		for _, k := range dump.APIKeys {
			if err := repos.APIKeys.Create(ctx, k); err == nil {
				importedKeys++
			}
		}

		importedOrders := 0
		for _, o := range dump.Orders {
			if err := repos.Orders.Create(ctx, o); err == nil {
				importedOrders++
			}
		}

		fmt.Printf("Import completed: %d payments, %d keys, %d orders imported.\n",
			importedPayments, importedKeys, importedOrders)

	default:
		printUsage()
		os.Exit(1)
	}
}

func printUsage() {
	fmt.Println(`authentic-et CLI Utility

Usage:
  cli create-admin --email <email> --password <pass> [--name <name>]
  cli export-data [--out <filename.json>]
  cli import-data --in <filename.json>

Examples:
  # Create an admin user
  cli create-admin --email admin@authentic-et.local --password MySecretPassword123

  # Export database to JSON for database switching (SQLite -> PostgreSQL)
  DATABASE_DRIVER=sqlite SQLITE_PATH=./data/authentic.db cli export-data --out data_backup.json

  # Import database into PostgreSQL
  DATABASE_DRIVER=postgres DATABASE_URL=postgres://user:pass@localhost/authentic cli import-data --in data_backup.json`)
}
