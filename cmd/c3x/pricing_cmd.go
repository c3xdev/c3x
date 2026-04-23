package main

import (
	"fmt"
	"strings"

	"github.com/c3xdev/c3x/internal/localpricing"
	"github.com/c3xdev/c3x/internal/settings"
	"github.com/spf13/cobra"
)

func pricingCmd(ctx *settings.Session) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "pricing",
		Short: "Manage local pricing data for offline estimation",
		Long:  "Download and manage cloud provider pricing data for fully offline cost estimation.",
	}

	cmd.AddCommand(pricingSyncCmd(ctx))
	cmd.AddCommand(pricingStatusCmd(ctx))

	return cmd
}

func pricingSyncCmd(ctx *settings.Session) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "sync",
		Short: "Download pricing data from cloud providers",
		Long: `Download pricing data from AWS, Azure, and Google Cloud into a local SQLite database.
Once synced, use --offline flag with estimate to run without any network calls.`,
		Example: `  Sync all providers:

      c3x pricing sync

  Sync specific providers:

      c3x pricing sync --providers aws,azure

  Then estimate offline:

      c3x estimate --path . --offline`,
		RunE: func(cmd *cobra.Command, args []string) error {
			providerStr, _ := cmd.Flags().GetString("providers")
			providers := strings.Split(providerStr, ",")

			dbPath, _ := cmd.Flags().GetString("db-path")
			if dbPath == "" {
				dbPath = localpricing.DefaultPath()
			}

			fmt.Printf("Syncing pricing data to %s\n", dbPath)

			store, err := localpricing.Open(dbPath)
			if err != nil {
				return err
			}
			defer store.Close()

			err = localpricing.Sync(localpricing.SyncOptions{
				Providers: providers,
				Store:     store,
				OnProgress: func(provider string, count int) {
					fmt.Printf("  [%s] %d items synced\n", provider, count)
				},
			})
			if err != nil {
				return err
			}

			count, _ := store.ProductCount()
			fmt.Printf("\nDone! %d products in local database.\n", count)
			fmt.Println("Use 'c3x estimate --path . --offline' to estimate without network calls.")

			return nil
		},
	}

	cmd.Flags().String("providers", "aws,azure", "Comma-separated list of providers to sync")
	cmd.Flags().String("db-path", "", "Path to pricing database (default: ~/.c3x/pricing.db)")

	return cmd
}

func pricingStatusCmd(ctx *settings.Session) *cobra.Command {
	return &cobra.Command{
		Use:   "status",
		Short: "Show local pricing database status",
		RunE: func(cmd *cobra.Command, args []string) error {
			dbPath := localpricing.DefaultPath()

			if !localpricing.Exists(dbPath) {
				fmt.Println("No local pricing database found.")
				fmt.Println("Run 'c3x pricing sync' to download pricing data.")
				return nil
			}

			store, err := localpricing.Open(dbPath)
			if err != nil {
				return err
			}
			defer store.Close()

			count, _ := store.ProductCount()
			lastSync, _ := store.GetMetadata("last_sync")

			fmt.Printf("Database: %s\n", dbPath)
			fmt.Printf("Products: %d\n", count)
			if lastSync != "" {
				fmt.Printf("Last sync: %s\n", lastSync)
			}

			return nil
		},
	}
}
