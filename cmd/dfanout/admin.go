package main

import (
	"context"
	"fmt"

	pb "github.com/dfanout/dfanout/proto"
	"github.com/gogo/protobuf/jsonpb"
	"github.com/jackc/pgx/v5"
)

const maxEndpoints = 10

var protoMarshaler = &jsonpb.Marshaler{EnumsAsInts: true, EmitDefaults: true, OrigName: true}

type adminService struct {
	pgConn *pgx.Conn
}

func (s *adminService) GetFanout(ctx context.Context, req *pb.GetFanoutRequest) (*pb.GetFanoutResponse, error) {
	rows, err := s.pgConn.Query(ctx,
		`SELECT endpoint_name, is_primary, http_endpoint
		 FROM endpoints
		 WHERE fanout_name = $1
		 ORDER BY is_primary DESC`, req.FanName)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var endpoints []*pb.Endpoint
	var (
		endpointName string
		primary      bool
		httpEndpoint string
	)
	for rows.Next() {
		if err := rows.Scan(&endpointName, &primary, &httpEndpoint); err != nil {
			return nil, err
		}
		var endpoint pb.HTTPEndpoint
		if err := jsonpb.UnmarshalString(httpEndpoint, &endpoint); err != nil {
			return nil, err
		}
		endpoints = append(endpoints, &pb.Endpoint{
			Name:    endpointName,
			Primary: primary,
			Destination: &pb.Endpoint_HttpEndpoint{
				HttpEndpoint: &endpoint,
			},
		})
	}
	return &pb.GetFanoutResponse{Endpoints: endpoints}, nil
}

func (s *adminService) CreateFanout(ctx context.Context, req *pb.CreateFanoutRequest) (resp *pb.CreateFanoutResponse, err error) {
	if n := len(req.Endpoints); n > maxEndpoints {
		return nil, fmt.Errorf("a maximum of 10 endpoints are allowed, %d provided", n)
	}

	tx, err := s.pgConn.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return nil, err
	}
	defer func() {
		if err != nil {
			tx.Rollback(ctx)
		}
	}()

	for _, e := range req.Endpoints {
		if err := s.insertEndpoint(ctx, tx, req.FanoutName, e); err != nil {
			return nil, err
		}
	}
	if err := s.validatePrimaryCount(ctx, tx, req.FanoutName); err != nil {
		return nil, err
	}
	if err := tx.Commit(ctx); err != nil {
		return nil, err
	}
	return &pb.CreateFanoutResponse{}, nil
}

func (s *adminService) UpdateFanout(ctx context.Context, req *pb.UpdateFanoutRequest) (*pb.UpdateFanoutRequest, error) {
	tx, err := s.pgConn.BeginTx(ctx, pgx.TxOptions{
		IsoLevel: pgx.Serializable,
	})
	if err != nil {
		return nil, err
	}
	defer func() {
		if err != nil {
			tx.Rollback(ctx)
		}
	}()

	for _, e := range req.EndpointsToDelete {
		_, err := tx.Exec(ctx,
			`DELETE FROM endpoints WHERE fanout_name = $1 AND endpoint_name = $2`, req.FanoutName, e)
		if err != nil {
			return nil, err
		}
	}
	for _, e := range req.EndpointsToInsert {
		if err := s.insertEndpoint(ctx, tx, req.FanoutName, e); err != nil {
			return nil, err
		}
	}
	for _, e := range req.EndpointsToUpdate {
		if err := s.updateEndpoint(ctx, tx, req.FanoutName, e); err != nil {
			return nil, err
		}
	}

	if err := s.validatePrimaryCount(ctx, tx, req.FanoutName); err != nil {
		return nil, err
	}
	if err := tx.Commit(ctx); err != nil {
		return nil, err
	}
	// TODO: Implement update and insert. Check primary count.
	return &pb.UpdateFanoutRequest{}, nil
}

func (s *adminService) DeleteFanout(ctx context.Context, req *pb.DeleteFanoutRequest) (*pb.DeleteFanoutResponse, error) {
	_, err := s.pgConn.Exec(ctx,
		`DELETE FROM endpoints WHERE fanout_name = $1`, req.FanoutName)
	if err != nil {
		return nil, err
	}
	return &pb.DeleteFanoutResponse{}, nil
}

func (s *adminService) insertEndpoint(ctx context.Context, tx pgx.Tx, fanout string, e *pb.Endpoint) error {
	switch endpoint := e.Destination.(type) {
	case *pb.Endpoint_HttpEndpoint:
		// TODO: Validate the endpoint.
		httpEndpoint, err := protoMarshaler.MarshalToString(endpoint.HttpEndpoint)
		if err != nil {
			return err
		}
		_, err = tx.Exec(ctx,
			`INSERT INTO endpoints (fanout_name, endpoint_name, is_primary, http_endpoint, created_at, updated_at)
			 VALUES ($1, $2, $3, $4, NOW(), NOW())`, fanout, e.Name, e.Primary, httpEndpoint)
		return err
	default:
		panic("not supported endpoint")
	}
}

func (s *adminService) updateEndpoint(ctx context.Context, tx pgx.Tx, fanout string, e *pb.Endpoint) error {
	switch endpoint := e.Destination.(type) {
	case *pb.Endpoint_HttpEndpoint:
		// TODO: Validate the endpoint.
		httpEndpoint, err := protoMarshaler.MarshalToString(endpoint.HttpEndpoint)
		if err != nil {
			return err
		}
		_, err = tx.Exec(ctx,
			`UPDATE endpoints
			 SET is_primary = $1, http_endpoint = $2, updated_at = NOW()
			 WHERE fanout_name = $3 AND endpoint_name = $4`, e.Primary, httpEndpoint, fanout, e.Name)
		return err
	default:
		panic("not supported endpoint")
	}
}

func (s *adminService) validatePrimaryCount(ctx context.Context, tx pgx.Tx, fanout string) error {
	row := tx.QueryRow(ctx,
		`SELECT COUNT(*)
		 FROM endpoints
		 WHERE fanout_name = $1 AND is_primary = TRUE`, fanout)

	var count int
	if err := row.Scan(&count); err != nil {
		return err
	}
	if count != 1 {
		return fmt.Errorf("need one primary endpoint; found %v", count)
	}
	return nil
}
