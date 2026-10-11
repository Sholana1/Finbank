package grpcapi

import (
	"context"
	"errors"
	"log/slog"

	identityv1 "github.com/sholana1/fintechdemo/gen/identity/v1"
	"github.com/sholana1/fintechdemo/identity/internal/app"
	"github.com/sholana1/fintechdemo/identity/internal/domain"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type userRegister interface {
	Execute(ctx context.Context, in app.RegisterUserInput) (domain.User, error)
}

type Handler struct {
	identityv1.UnimplementedIdentityServiceServer
	registerUser userRegister
	logger       *slog.Logger
}

func NewHandler(registerUser userRegister, logger *slog.Logger) *Handler {
	return &Handler{registerUser: registerUser, logger: logger}
}

func (h *Handler) RegisterUser(ctx context.Context, req *identityv1.RegisterUserInput) (*identityv1.RegisterUserResponse, error) {
	user, err := h.registerUser.Execute(ctx, app.RegisterUserInput{
		Email:       req.GetEmail(),
		PhoneNumber: req.GetPhoneNumber(),
		Password:    req.GetPassword(),
	})
	if err != nil {
		return nil, h.toStatus("RegisterUser", err)
	}
	return &identityv1.RegisterUserResponse{
		UserId: user.ID,
		Status: user.Status,
	}, nil
}

func (h *Handler) toStatus(method string, err error) error {
	switch {
	case errors.Is(err, domain.ErrInvalidEmail),
		errors.Is(err, domain.ErrInvalidPhoneNumber),
		errors.Is(err, domain.ErrWeakPassword):
		return status.Error(codes.InvalidArgument, rootMessage(err))
	case errors.Is(err, domain.ErrEmailTaken),
		errors.Is(err, domain.ErrPhoneNumberTaken):
		return status.Error(codes.AlreadyExists, rootMessage(err))
	default:
		h.logger.Error("request failed", "method", method, "error", err)
		return status.Error(codes.Internal, "internal error")
	}
}

func rootMessage(err error) string {
	for {
		inner := errors.Unwrap(err)
		if inner == nil {
			return err.Error()
		}
		err = inner
	}
}

func toProtoStatus(s domain.UserStatus) identityv1.UserStatus {
	switch s {
	case domain.UserStatusPendingVerification:
		return identityv1.UserStatus_USER_STATUS_PENDING_VERIFICATION
	case domain.UserStatusActive:
		return identityv1.UserStatus_USER_STATUS_ACTIVE
	case domain.UserStatusSuspended:
		return identityv1.UserStatus_USER_STATUS_SUSPENDED
	case domain.UserStatusClosed:
		return identityv1.UserStatus_USER_STATUS_CLOSED
	default:
		return identityv1.UserStatus_USER_STATUS_UNSPECIFIED
	}
}
