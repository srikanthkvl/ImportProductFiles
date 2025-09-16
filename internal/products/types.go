package products

import (
	"errors"
	"fmt"
)

// ProductType enumerates supported product types.
const (
	ProductUsers         = "users"
	ProductOrganizations = "organizations"
	ProductCourses       = "courses"
)

var supported = map[string]struct{}{
	ProductUsers:         {},
	ProductOrganizations: {},
	ProductCourses:       {},
}

// ValidateProductType checks if the product type is supported.
func ValidateProductType(pt string) error {
	if _, ok := supported[pt]; !ok {
		return fmt.Errorf("unsupported product type: %s", pt)
	}
	return nil
}

// TargetTableFor maps product type to target table name in customer DB.
func TargetTableFor(pt string) (string, error) {
	if err := ValidateProductType(pt); err != nil {
		return "", err
	}
	return pt, nil
}

// SchemaRequiredFields returns required field names for a product type.
func SchemaRequiredFields(pt string) ([]string, error) {
	switch pt {
	case ProductUsers:
		return []string{"id", "email", "name"}, nil
	case ProductOrganizations:
		return []string{"id", "name"}, nil
	case ProductCourses:
		return []string{"id", "title"}, nil
	default:
		return nil, errors.New("unknown product type")
	}
}


