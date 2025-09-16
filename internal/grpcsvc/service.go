package grpcsvc

import (
	"context"
	"errors"

	"google.golang.org/protobuf/types/known/structpb"

	"github.com/user/importer/internal/jobs"
)

// ImporterService provides gRPC methods for enqueuing import jobs.
type ImporterService struct {
	Jobs *jobs.Repository
}

func New(jr *jobs.Repository) *ImporterService { return &ImporterService{Jobs: jr} }

// Enqueue expects a Struct with fields: customer_id, product_type, blob_uri. Returns { job_id: number }.
func (s *ImporterService) Enqueue(ctx context.Context, in *structpb.Struct) (*structpb.Struct, error) {
	if in == nil {
		return nil, errors.New("nil request")
	}
	get := func(k string) string {
		if v, ok := in.Fields[k]; ok && v.GetStringValue() != "" {
			return v.GetStringValue()
		}
		return ""
	}
	cust := get("customer_id")
	prod := get("product_type")
	blob := get("blob_uri")
	if cust == "" || prod == "" || blob == "" {
		return nil, errors.New("customer_id, product_type, and blob_uri are required")
	}
	id, err := s.Jobs.Enqueue(ctx, cust, prod, blob)
	if err != nil {
		return nil, err
	}
	resp, _ := structpb.NewStruct(map[string]any{"job_id": id})
	return resp, nil
}


