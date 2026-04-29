package handler

import (
	"context"
	"errors"
	"log"

	createv1 "buf.build/gen/go/budget-planner-platform/bp-user/protocolbuffers/go/create_user/v1"
	deletev1 "buf.build/gen/go/budget-planner-platform/bp-user/protocolbuffers/go/delete_user/v1"
	getv1 "buf.build/gen/go/budget-planner-platform/bp-user/protocolbuffers/go/get_user/v1"
	updatev1 "buf.build/gen/go/budget-planner-platform/bp-user/protocolbuffers/go/update_user/v1"
	userv1 "buf.build/gen/go/budget-planner-platform/bp-user/protocolbuffers/go/user/v1"
	"buf.build/gen/go/budget-planner-platform/bp-user/grpc/go/user_service/v1/user_servicev1grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/yaroslav/bp-user-service/internal/model"
	"github.com/yaroslav/bp-user-service/internal/repository"
)

// UserHandler implements the UserServiceServer gRPC interface.
type UserHandler struct {
	user_servicev1grpc.UnimplementedUserServiceServer
	repo *repository.UserRepository
}

// NewUserHandler creates a new UserHandler.
func NewUserHandler(repo *repository.UserRepository) *UserHandler {
	return &UserHandler{repo: repo}
}

func (h *UserHandler) CreateUser(ctx context.Context, req *createv1.CreateUserRequest) (*createv1.CreateUserResponse, error) {
	userID, err := extractUserID(ctx)
	if err != nil {
		return nil, err
	}

	u := &model.User{
		ID:          userID,
		Email:       req.GetEmail(),
		DisplayName: req.GetDisplayName(),
		AvatarURL:   req.GetAvatarUrl(),
		Currency:    req.GetCurrency(),
		Timezone:    req.GetTimezone(),
	}

	if u.Currency == "" {
		u.Currency = "USD"
	}
	if u.Timezone == "" {
		u.Timezone = "UTC"
	}

	created, err := h.repo.Create(ctx, u)
	if err != nil {
		if errors.Is(err, repository.ErrAlreadyExists) {
			return nil, status.Error(codes.AlreadyExists, "user already exists")
		}
		log.Printf("CreateUser error: %v", err)
		return nil, status.Error(codes.Internal, "failed to create user")
	}

	return &createv1.CreateUserResponse{User: toProto(created)}, nil
}

func (h *UserHandler) GetUser(ctx context.Context, req *getv1.GetUserRequest) (*getv1.GetUserResponse, error) {
	u, err := h.repo.GetByID(ctx, req.GetId())
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return nil, status.Error(codes.NotFound, "user not found")
		}
		log.Printf("GetUser error: %v", err)
		return nil, status.Error(codes.Internal, "failed to get user")
	}

	return &getv1.GetUserResponse{User: toProto(u)}, nil
}

func (h *UserHandler) UpdateUser(ctx context.Context, req *updatev1.UpdateUserRequest) (*updatev1.UpdateUserResponse, error) {
	fields := repository.UpdateFields{
		DisplayName: req.DisplayName,
		AvatarURL:   req.AvatarUrl,
		Currency:    req.Currency,
		Timezone:    req.Timezone,
	}

	u, err := h.repo.Update(ctx, req.GetId(), fields)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return nil, status.Error(codes.NotFound, "user not found")
		}
		log.Printf("UpdateUser error: %v", err)
		return nil, status.Error(codes.Internal, "failed to update user")
	}

	return &updatev1.UpdateUserResponse{User: toProto(u)}, nil
}

func (h *UserHandler) DeleteUser(ctx context.Context, req *deletev1.DeleteUserRequest) (*deletev1.DeleteUserResponse, error) {
	err := h.repo.SoftDelete(ctx, req.GetId())
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return nil, status.Error(codes.NotFound, "user not found")
		}
		log.Printf("DeleteUser error: %v", err)
		return nil, status.Error(codes.Internal, "failed to delete user")
	}

	return &deletev1.DeleteUserResponse{}, nil
}

// extractUserID reads the x-user-id from gRPC metadata.
func extractUserID(ctx context.Context) (string, error) {
	md, ok := metadata.FromIncomingContext(ctx)
	if !ok {
		return "", status.Error(codes.Unauthenticated, "missing metadata")
	}

	values := md.Get("x-user-id")
	if len(values) == 0 || values[0] == "" {
		return "", status.Error(codes.Unauthenticated, "missing x-user-id metadata")
	}

	return values[0], nil
}

// toProto converts a domain User to the protobuf User message.
func toProto(u *model.User) *userv1.User {
	return &userv1.User{
		Id:          u.ID,
		Email:       u.Email,
		DisplayName: u.DisplayName,
		AvatarUrl:   u.AvatarURL,
		Currency:    u.Currency,
		Timezone:    u.Timezone,
		CreateTime:  timestamppb.New(u.CreatedAt),
		UpdateTime:  timestamppb.New(u.UpdatedAt),
	}
}
