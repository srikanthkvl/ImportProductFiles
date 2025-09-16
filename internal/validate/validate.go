package validate

import (
	"fmt"

	"github.com/user/importer/internal/parser"
	"github.com/user/importer/internal/products"
)

// Records validates the records against product schema required fields.
func Records(productType string, records []parser.Record) error {
	req, err := products.SchemaRequiredFields(productType)
	if err != nil {
		return err
	}
	for idx, rec := range records {
		for _, field := range req {
			if rec[field] == "" {
				return fmt.Errorf("record %d missing required field %s", idx, field)
			}
		}
	}
	return nil
}


