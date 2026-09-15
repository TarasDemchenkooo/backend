package application

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/google/uuid"

	"github.com/TarasDemchenkooo/backend/services/auth/internal/domain"
)

type RegisterInput struct {
	Email    string
	Password string
	Role     string
}

type RegisterOutput struct {
	UserID    uuid.UUID
	ExpiresIn time.Duration
}

type RegisterUseCase struct {
	users  UserRepository
	codes  CodeStore
	events EventPublisher
	hasher PasswordHasher

	codeTTL time.Duration
	log     *slog.Logger
}

func NewRegisterUseCase(
	users UserRepository,
	codes CodeStore,
	events EventPublisher,
	hasher PasswordHasher,
	codeTTL time.Duration,
	log *slog.Logger,
) *RegisterUseCase {
	return &RegisterUseCase{
		users:   users,
		codes:   codes,
		events:  events,
		hasher:  hasher,
		codeTTL: codeTTL,
		log:     log.With(slog.String("usecase", "register")),
	}
}

func (uc *RegisterUseCase) Execute(ctx context.Context, in RegisterInput) (RegisterOutput, error) {
	email, err := domain.NewEmail(in.Email)
	if err != nil {
		uc.log.DebugContext(ctx, "invalid email rejected", "error", err)
		return RegisterOutput{}, err
	}

	rawPassword, err := domain.NewRawPassword(in.Password)
	if err != nil {
		uc.log.DebugContext(ctx, "weak password rejected", "error", err)
		return RegisterOutput{}, err
	}

	role, err := domain.NewRole(in.Role)
	if err != nil {
		uc.log.DebugContext(ctx, "invalid role rejected", "role", in.Role, "error", err)
		return RegisterOutput{}, err
	}

	hash, err := uc.hasher.Hash(rawPassword)
	if err != nil {
		return RegisterOutput{}, fmt.Errorf("hash password: %w", err)
	}

	user := domain.NewUser(email, hash, role)
	if err := uc.users.CreateOrReplaceUnverified(ctx, user); err != nil {
		return RegisterOutput{}, fmt.Errorf("save user: %w", err)
	}

	log := uc.log.With(slog.String("user_id", user.ID.String()))
	log.InfoContext(ctx, "unverified user stored", "role", role.String())

	code, err := newVerificationCode()
	if err != nil {
		return RegisterOutput{}, err
	}

	if err := uc.codes.Save(ctx, email, code, uc.codeTTL); err != nil {
		return RegisterOutput{}, fmt.Errorf("save verification code: %w", err)
	}

	log.DebugContext(ctx, "verification code stored", "ttl", uc.codeTTL)

	event := UserRegisteredEvent{
		UserID: user.ID,
		Email:  email.String(),
		Code:   code,
	}
	if err := uc.events.PublishUserRegistered(ctx, event); err != nil {
		return RegisterOutput{}, fmt.Errorf("publish user registered: %w", err)
	}

	log.InfoContext(ctx, "user registered")

	return RegisterOutput{UserID: user.ID, ExpiresIn: uc.codeTTL}, nil
}
