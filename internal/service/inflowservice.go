package service

import (
	"context"

	pb "ReservoirFloodPrediction/api/inflow/v1"
)

type InflowServiceService struct {
	pb.UnimplementedInflowServiceServer
}

func NewInflowServiceService() *InflowServiceService {
	return &InflowServiceService{}
}

func (s *InflowServiceService) SimulateRainfall(ctx context.Context, req *pb.SimulateRainfallRequest) (*pb.SimulateRainfallReply, error) {
	return &pb.SimulateRainfallReply{}, nil
}
func (s *InflowServiceService) PredictRainfall(ctx context.Context, req *pb.PredictRainfallRequest) (*pb.PredictRainfallReply, error) {
	return &pb.PredictRainfallReply{}, nil
}
func (s *InflowServiceService) CalculateInflowVolume(ctx context.Context, req *pb.CalculateInflowVolumeRequest) (*pb.CalculateInflowVolumeReply, error) {
	return &pb.CalculateInflowVolumeReply{}, nil
}
