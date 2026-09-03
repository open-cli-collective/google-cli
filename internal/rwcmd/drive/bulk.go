package drive

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/open-cli-collective/google-cli/internal/bulk"
)

func resolveFileIDs(cmd *cobra.Command, args []string, stdin bool, query string, dryRun bool) (WriteClient, []string, error) {
	var client WriteClient
	ids, err := bulk.ResolveIDs(bulk.Config{Args: args, Stdin: stdin, Query: query}, func(q string) ([]string, error) {
		var err error
		client, err = newWriteClient(cmd.Context())
		if err != nil {
			return nil, fmt.Errorf("creating Drive client: %w", err)
		}
		return client.SearchFileIDs(cmd.Context(), q, 0)
	})
	if err != nil || dryRun {
		return client, ids, err
	}
	if client == nil {
		client, err = newWriteClient(cmd.Context())
		if err != nil {
			return nil, nil, fmt.Errorf("creating Drive client: %w", err)
		}
	}
	return client, ids, nil
}

func printBulkResult(action string, ids []string, dryRun bool) error {
	result := &bulk.Result{Action: action, IDs: ids, Count: len(ids), DryRun: dryRun, ItemNoun: "file"}
	if err := result.Print(); err != nil {
		return err
	}
	if dryRun {
		for _, id := range ids {
			fmt.Println(id)
		}
	}
	return nil
}
