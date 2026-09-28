package routing

import (
	"context"
	"fmt"
	"time"

	"github.com/premsanyal/aegisroad/services/api-gateway/internal/models"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	routingpb "github.com/premsanyal/aegisroad/gen/go/routing"
)

type Client struct {
	conn   *grpc.ClientConn
	client routingpb.RoutingServiceClient
}

func NewClient(addr string) (*Client, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	conn, err := grpc.DialContext(ctx, addr,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithBlock(),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to routing engine: %w", err)
	}

	return &Client{
		conn:   conn,
		client: routingpb.NewRoutingServiceClient(conn),
	}, nil
}

func (c *Client) CalculateRoute(ctx context.Context, req *models.RouteRequest) (*models.RouteResponse, error) {
	pbReq := toPBRouteRequest(req)

	resp, err := c.client.CalculateRoute(ctx, pbReq)
	if err != nil {
		return nil, fmt.Errorf("routing engine error: %w", err)
	}

	return fromPBRouteResponse(resp), nil
}

func (c *Client) StreamRouteUpdates(ctx context.Context, req *models.RouteRequest) (<-chan *models.RouteUpdate, error) {
	pbReq := toPBRouteRequest(req)

	stream, err := c.client.StreamRouteUpdates(ctx, pbReq)
	if err != nil {
		return nil, fmt.Errorf("failed to start route stream: %w", err)
	}

	updates := make(chan *models.RouteUpdate, 100)

	go func() {
		defer close(updates)
		for {
			pbUpdate, err := stream.Recv()
			if err != nil {
				// Log error but don't block
				return
			}
			updates <- fromPBRouteUpdate(pbUpdate)
		}
	}()

	return updates, nil
}

func (c *Client) ReserveCorridor(ctx context.Context, req *models.CorridorRequest) (*models.CorridorResponse, error) {
	pbReq := toPBCorridorRequest(req)

	resp, err := c.client.ReserveCorridor(ctx, pbReq)
	if err != nil {
		return nil, fmt.Errorf("corridor reservation failed: %w", err)
	}

	return fromPBCorridorResponse(resp), nil
}

func (c *Client) ReleaseCorridor(ctx context.Context, reservationID string) error {
	_, err := c.client.ReleaseCorridor(ctx, &routingpb.CorridorRelease{
		ReservationId: reservationID,
	})
	return err
}

func (c *Client) Close() error {
	return c.conn.Close()
}

// Conversion functions would go here
func toPBRouteRequest(req *models.RouteRequest) *routingpb.RouteRequest {
	return &routingpb.RouteRequest{
		RequestId:       req.RequestID,
		PriorityLevel:   req.PriorityLevel,
		Origin:          toPBPoint(req.Origin),
		Destination:     toPBPoint(req.Destination),
		VehicleId:       req.VehicleID,
		VehicleType:     req.VehicleType,
		AvoidSegments:   req.AvoidSegments,
		ReturnAlternatives: req.ReturnAlternatives,
		MaxAlternatives: int32(req.MaxAlternatives),
		Profile:         toPBProfile(req.Profile),
	}
}

func toPBPoint(p models.Point) *routingpb.Point {
	return &routingpb.Point{
		Longitude: p.Longitude,
		Latitude:  p.Latitude,
	}
}

func toPBProfile(p models.RoutingProfile) routingpb.RoutingProfile {
	switch p {
	case models.ProfileCar:
		return routingpb.RoutingProfile_PROFILE_CAR
	case models.ProfileBike:
		return routingpb.RoutingProfile_PROFILE_BIKE
	case models.ProfileFoot:
		return routingpb.RoutingProfile_PROFILE_FOOT
	case models.ProfileEmergency:
		return routingpb.RoutingProfile_PROFILE_EMERGENCY
	default:
		return routingpb.RoutingProfile_PROFILE_CAR
	}
}

func fromPBRouteResponse(resp *routingpb.RouteResponse) *models.RouteResponse {
	// Implementation
	return &models.RouteResponse{}
}

func fromPBRouteUpdate(update *routingpb.RouteUpdate) *models.RouteUpdate {
	return &models.RouteUpdate{}
}

func toPBCorridorRequest(req *models.CorridorRequest) *routingpb.CorridorRequest {
	return &routingpb.CorridorRequest{}
}

func fromPBCorridorResponse(resp *routingpb.CorridorResponse) *models.CorridorResponse {
	return &models.CorridorResponse{}
}