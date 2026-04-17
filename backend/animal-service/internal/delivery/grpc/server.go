package grpcserver

import (
	"context"
	"fmt"
	"log/slog"
	"net"

	"github.com/go-playground/validator/v10"
	animalv1 "github.com/petmatch/petmatch/gen/go/petmatch/animal/v1"
	"github.com/petmatch/petmatch/internal/adapter/pbconv"
	appanimal "github.com/petmatch/petmatch/internal/app/animal"
	domain "github.com/petmatch/petmatch/internal/domain/animal"
	"github.com/prometheus/client_golang/prometheus"
	"google.golang.org/grpc"
	"google.golang.org/grpc/health"
	healthgrpc "google.golang.org/grpc/health/grpc_health_v1"
	"google.golang.org/grpc/reflection"
)

// Server wraps the gRPC runtime.
type Server struct {
	addr   string
	server *grpc.Server
	health *health.Server
}

// NewServer creates a gRPC server for AnimalService.
func NewServer(addr string, service *appanimal.Service, registry *prometheus.Registry, logger *slog.Logger) *Server {
	metrics := NewMetrics(registry)
	grpcServer := grpc.NewServer(grpc.ChainUnaryInterceptor(UnaryInterceptor(logger, metrics)))
	animalv1.RegisterAnimalServiceServer(grpcServer, NewAnimalHandler(service))
	healthServer := health.NewServer()
	healthServer.SetServingStatus("petmatch.animal.v1.AnimalService", healthgrpc.HealthCheckResponse_SERVING)
	healthgrpc.RegisterHealthServer(grpcServer, healthServer)
	reflection.Register(grpcServer)
	return &Server{addr: addr, server: grpcServer, health: healthServer}
}

// ListenAndServe starts serving gRPC traffic.
func (s *Server) ListenAndServe() error {
	listener, err := net.Listen("tcp", s.addr)
	if err != nil {
		return fmt.Errorf("listen grpc %s: %w", s.addr, err)
	}
	if err := s.server.Serve(listener); err != nil {
		return fmt.Errorf("serve grpc: %w", err)
	}
	return nil
}

// GracefulStop drains active RPCs.
func (s *Server) GracefulStop() {
	s.health.SetServingStatus("petmatch.animal.v1.AnimalService", healthgrpc.HealthCheckResponse_NOT_SERVING)
	s.server.GracefulStop()
}

// Stop immediately stops the server.
func (s *Server) Stop() {
	s.server.Stop()
}

// AnimalHandler implements petmatch.animal.v1.AnimalService.
type AnimalHandler struct {
	animalv1.UnimplementedAnimalServiceServer
	service  *appanimal.Service
	validate *validator.Validate
}

// NewAnimalHandler creates an AnimalService gRPC handler.
func NewAnimalHandler(service *appanimal.Service) *AnimalHandler {
	return &AnimalHandler{service: service, validate: validator.New(validator.WithRequiredStructEnabled())}
}

// CreateAnimal creates a draft animal profile.
func (h *AnimalHandler) CreateAnimal(ctx context.Context, req *animalv1.CreateAnimalRequest) (*animalv1.CreateAnimalResponse, error) {
	profile := pbconv.FromAnimalProfile(req.GetAnimal())
	ownerProfileID := req.GetOwnerProfileId()
	if ownerProfileID == "" {
		ownerProfileID = profile.OwnerProfileID
	}
	ownerType := domain.OwnerType(req.GetOwnerType())
	if ownerType == domain.OwnerTypeUnspecified {
		ownerType = profile.OwnerType
	}
	if err := h.validateRequest(struct {
		OwnerProfileID string `validate:"required"`
		Name           string `validate:"required"`
	}{
		OwnerProfileID: ownerProfileID,
		Name:           profile.Name,
	}); err != nil {
		return nil, toStatusError(err)
	}
	cmd := appanimal.CreateAnimalCommand{
		ActorID:        actorIDFromContext(ctx, ownerProfileID),
		OwnerProfileID: ownerProfileID,
		OwnerType:      ownerType,
		IdempotencyKey: req.GetIdempotencyKey(),
		Name:           profile.Name,
		Species:        profile.Species,
		Breed:          profile.Breed,
		Sex:            profile.Sex,
		Size:           profile.Size,
		AgeMonths:      profile.AgeMonths,
		Description:    profile.Description,
		Traits:         profile.Traits,
		MedicalNotes:   profile.MedicalNotes,
		Vaccinated:     profile.Vaccinated,
		Sterilized:     profile.Sterilized,
		Location:       profile.Location,
		Photos:         profile.Photos,
	}
	animalProfile, err := h.service.Create(ctx, cmd)
	if err != nil {
		return nil, toStatusError(err)
	}
	return &animalv1.CreateAnimalResponse{Animal: pbconv.ToAnimalProfile(animalProfile)}, nil
}

// GetAnimal returns one animal profile.
func (h *AnimalHandler) GetAnimal(ctx context.Context, req *animalv1.GetAnimalRequest) (*animalv1.GetAnimalResponse, error) {
	if err := h.validateRequest(struct {
		AnimalID string `validate:"required"`
	}{AnimalID: req.GetAnimalId()}); err != nil {
		return nil, toStatusError(err)
	}
	profile, err := h.service.Get(ctx, req.GetAnimalId())
	if err != nil {
		return nil, toStatusError(err)
	}
	return &animalv1.GetAnimalResponse{Animal: pbconv.ToAnimalProfile(profile)}, nil
}

// BatchGetAnimals returns animals by IDs.
func (h *AnimalHandler) BatchGetAnimals(ctx context.Context, req *animalv1.BatchGetAnimalsRequest) (*animalv1.BatchGetAnimalsResponse, error) {
	if err := h.validateRequest(struct {
		AnimalIDs []string `validate:"required,min=1,dive,required"`
	}{AnimalIDs: req.GetAnimalIds()}); err != nil {
		return nil, toStatusError(err)
	}
	profiles, err := h.service.BatchGet(ctx, req.GetAnimalIds())
	if err != nil {
		return nil, toStatusError(err)
	}
	result := make([]*animalv1.AnimalProfile, 0, len(profiles))
	for _, profile := range profiles {
		result = append(result, pbconv.ToAnimalProfile(profile))
	}
	return &animalv1.BatchGetAnimalsResponse{Animals: result}, nil
}

// SearchAnimals searches public animal profiles.
func (h *AnimalHandler) SearchAnimals(ctx context.Context, req *animalv1.SearchAnimalsRequest) (*animalv1.SearchAnimalsResponse, error) {
	result, err := h.service.Search(ctx, searchQueryFromProto(req))
	if err != nil {
		return nil, toStatusError(err)
	}
	animals := make([]*animalv1.AnimalProfile, 0, len(result.Items))
	for _, profile := range result.Items {
		animals = append(animals, pbconv.ToAnimalProfile(profile))
	}
	return &animalv1.SearchAnimalsResponse{Animals: animals, Page: pbconv.ToPageResponse(result.NextPageToken, result.TotalSize)}, nil
}

// UpdateAnimal patches mutable profile fields.
func (h *AnimalHandler) UpdateAnimal(ctx context.Context, req *animalv1.UpdateAnimalRequest) (*animalv1.UpdateAnimalResponse, error) {
	if err := h.validateRequest(struct {
		AnimalID   string   `validate:"required"`
		UpdateMask []string `validate:"required,min=1,dive,required"`
	}{AnimalID: req.GetAnimalId(), UpdateMask: req.GetUpdateMask().GetPaths()}); err != nil {
		return nil, toStatusError(err)
	}
	updated, err := h.service.Update(ctx, appanimal.UpdateAnimalCommand{
		ActorID:    actorIDFromContext(ctx, ""),
		AnimalID:   req.GetAnimalId(),
		Patch:      pbconv.FromAnimalProfile(req.GetAnimal()),
		UpdateMask: req.GetUpdateMask().GetPaths(),
	})
	if err != nil {
		return nil, toStatusError(err)
	}
	return &animalv1.UpdateAnimalResponse{Animal: pbconv.ToAnimalProfile(updated)}, nil
}

// PublishAnimal publishes an animal profile.
func (h *AnimalHandler) PublishAnimal(ctx context.Context, req *animalv1.PublishAnimalRequest) (*animalv1.PublishAnimalResponse, error) {
	if err := h.validateRequest(struct {
		AnimalID string `validate:"required"`
	}{AnimalID: req.GetAnimalId()}); err != nil {
		return nil, toStatusError(err)
	}
	updated, err := h.service.Publish(ctx, appanimal.PublishAnimalCommand{
		ActorID:  actorIDFromContext(ctx, ""),
		AnimalID: req.GetAnimalId(),
	})
	if err != nil {
		return nil, toStatusError(err)
	}
	return &animalv1.PublishAnimalResponse{Animal: pbconv.ToAnimalProfile(updated)}, nil
}

// ArchiveAnimal archives an animal profile.
func (h *AnimalHandler) ArchiveAnimal(ctx context.Context, req *animalv1.ArchiveAnimalRequest) (*animalv1.ArchiveAnimalResponse, error) {
	if err := h.validateRequest(struct {
		AnimalID string `validate:"required"`
		Reason   string `validate:"required"`
	}{AnimalID: req.GetAnimalId(), Reason: req.GetReason()}); err != nil {
		return nil, toStatusError(err)
	}
	updated, err := h.service.Archive(ctx, appanimal.ArchiveAnimalCommand{
		ActorID:  actorIDFromContext(ctx, ""),
		AnimalID: req.GetAnimalId(),
		Reason:   req.GetReason(),
	})
	if err != nil {
		return nil, toStatusError(err)
	}
	return &animalv1.ArchiveAnimalResponse{Animal: pbconv.ToAnimalProfile(updated)}, nil
}

// AddAnimalPhoto adds a profile photo.
func (h *AnimalHandler) AddAnimalPhoto(ctx context.Context, req *animalv1.AddAnimalPhotoRequest) (*animalv1.AddAnimalPhotoResponse, error) {
	if err := h.validateRequest(struct {
		AnimalID string `validate:"required"`
		PhotoURL string `validate:"required,url"`
	}{AnimalID: req.GetAnimalId(), PhotoURL: req.GetPhoto().GetUrl()}); err != nil {
		return nil, toStatusError(err)
	}
	updated, err := h.service.AddPhoto(ctx, appanimal.AddPhotoCommand{
		ActorID:        actorIDFromContext(ctx, ""),
		AnimalID:       req.GetAnimalId(),
		IdempotencyKey: req.GetIdempotencyKey(),
		Photo:          pbconv.FromPhoto(req.GetPhoto()),
	})
	if err != nil {
		return nil, toStatusError(err)
	}
	return &animalv1.AddAnimalPhotoResponse{Animal: pbconv.ToAnimalProfile(updated)}, nil
}

// RemoveAnimalPhoto removes a profile photo.
func (h *AnimalHandler) RemoveAnimalPhoto(ctx context.Context, req *animalv1.RemoveAnimalPhotoRequest) (*animalv1.RemoveAnimalPhotoResponse, error) {
	if err := h.validateRequest(struct {
		AnimalID string `validate:"required"`
		PhotoID  string `validate:"required"`
	}{AnimalID: req.GetAnimalId(), PhotoID: req.GetPhotoId()}); err != nil {
		return nil, toStatusError(err)
	}
	updated, err := h.service.RemovePhoto(ctx, appanimal.RemovePhotoCommand{
		ActorID:  actorIDFromContext(ctx, ""),
		AnimalID: req.GetAnimalId(),
		PhotoID:  req.GetPhotoId(),
	})
	if err != nil {
		return nil, toStatusError(err)
	}
	return &animalv1.RemoveAnimalPhotoResponse{Animal: pbconv.ToAnimalProfile(updated)}, nil
}

// ListOwnerAnimals lists profiles by owner.
func (h *AnimalHandler) ListOwnerAnimals(ctx context.Context, req *animalv1.ListOwnerAnimalsRequest) (*animalv1.ListOwnerAnimalsResponse, error) {
	if err := h.validateRequest(struct {
		OwnerProfileID string `validate:"required"`
	}{OwnerProfileID: req.GetOwnerProfileId()}); err != nil {
		return nil, toStatusError(err)
	}
	statuses := make([]domain.Status, 0, len(req.GetStatuses()))
	for _, status := range req.GetStatuses() {
		statuses = append(statuses, domain.Status(status))
	}
	result, err := h.service.ListOwnerAnimals(ctx, req.GetOwnerProfileId(), statuses, req.GetPage().GetPageSize(), req.GetPage().GetPageToken())
	if err != nil {
		return nil, toStatusError(err)
	}
	animals := make([]*animalv1.AnimalProfile, 0, len(result.Items))
	for _, profile := range result.Items {
		animals = append(animals, pbconv.ToAnimalProfile(profile))
	}
	return &animalv1.ListOwnerAnimalsResponse{Animals: animals, Page: pbconv.ToPageResponse(result.NextPageToken, result.TotalSize)}, nil
}

func (h *AnimalHandler) validateRequest(value any) error {
	if err := h.validate.Struct(value); err != nil {
		return fmt.Errorf("%w: %v", domain.ErrInvalidArgument, err)
	}
	return nil
}

func searchQueryFromProto(req *animalv1.SearchAnimalsRequest) domain.SearchQuery {
	filter := req.GetFilter()
	query := domain.SearchQuery{
		PageSize:  req.GetPage().GetPageSize(),
		PageToken: req.GetPage().GetPageToken(),
	}
	if filter == nil {
		return query
	}
	query.Breeds = append([]string(nil), filter.GetBreeds()...)
	query.Traits = append([]string(nil), filter.GetTraits()...)
	query.City = filter.City
	query.RadiusKM = filter.RadiusKm
	query.Vaccinated = filter.Vaccinated
	query.Sterilized = filter.Sterilized
	query.BoostedOnly = filter.BoostedOnly
	query.OwnerProfileID = filter.GetOwnerProfileId()
	for _, value := range filter.GetSpecies() {
		query.Species = append(query.Species, domain.Species(value))
	}
	for _, value := range filter.GetSexes() {
		query.Sexes = append(query.Sexes, domain.Sex(value))
	}
	for _, value := range filter.GetSizes() {
		query.Sizes = append(query.Sizes, domain.Size(value))
	}
	for _, value := range filter.GetStatuses() {
		query.Statuses = append(query.Statuses, domain.Status(value))
	}
	query.MinAgeMonths = filter.MinAgeMonths
	query.MaxAgeMonths = filter.MaxAgeMonths
	if filter.GetNear() != nil {
		lat := filter.GetNear().GetLatitude()
		lon := filter.GetNear().GetLongitude()
		query.NearLatitude = &lat
		query.NearLongitude = &lon
	}
	return query
}
