package application_test

import (
	"context"
	"errors"
	"log/slog"
	"regexp"
	"slices"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/TarasDemchenkooo/backend/services/auth/internal/application"
	"github.com/TarasDemchenkooo/backend/services/auth/internal/domain"
)

const codeTTL = 15 * time.Minute

var (
	errBoom                 = errors.New("boom")
	verificationCodePattern = regexp.MustCompile(`^\d{6}$`)
)

type ctxKey struct{}

type fakeUsers struct {
	calls    *[]string
	err      error
	assignID uuid.UUID
	saved    []domain.User
	ctxValue any
}

func (f *fakeUsers) CreateOrReplaceUnverified(ctx context.Context, u *domain.User) error {
	*f.calls = append(*f.calls, "users.save")
	f.ctxValue = ctx.Value(ctxKey{})

	if f.err != nil {
		return f.err
	}

	if f.assignID != uuid.Nil {
		u.ID = f.assignID
	}

	f.saved = append(f.saved, *u)

	return nil
}

type savedCode struct {
	email domain.Email
	code  string
	ttl   time.Duration
}

type fakeCodes struct {
	calls    *[]string
	err      error
	saved    []savedCode
	ctxValue any
}

func (f *fakeCodes) Save(ctx context.Context, email domain.Email, code string, ttl time.Duration) error {
	*f.calls = append(*f.calls, "codes.save")
	f.ctxValue = ctx.Value(ctxKey{})

	if f.err != nil {
		return f.err
	}

	f.saved = append(f.saved, savedCode{email: email, code: code, ttl: ttl})

	return nil
}

type fakeEvents struct {
	calls     *[]string
	err       error
	published []application.UserRegisteredEvent
	ctxValue  any
}

func (f *fakeEvents) PublishUserRegistered(ctx context.Context, e application.UserRegisteredEvent) error {
	*f.calls = append(*f.calls, "events.publish")
	f.ctxValue = ctx.Value(ctxKey{})

	if f.err != nil {
		return f.err
	}

	f.published = append(f.published, e)

	return nil
}

type fakeHasher struct {
	calls  *[]string
	err    error
	hashed []domain.RawPassword
}

func (f *fakeHasher) Hash(password domain.RawPassword) (string, error) {
	*f.calls = append(*f.calls, "hasher.hash")

	if f.err != nil {
		return "", f.err
	}

	f.hashed = append(f.hashed, password)

	return "hashed:" + password.String(), nil
}

type fixture struct {
	calls  []string
	users  *fakeUsers
	codes  *fakeCodes
	events *fakeEvents
	hasher *fakeHasher
	uc     *application.RegisterUseCase
}

func newFixture() *fixture {
	f := &fixture{}
	f.users = &fakeUsers{calls: &f.calls}
	f.codes = &fakeCodes{calls: &f.calls}
	f.events = &fakeEvents{calls: &f.calls}
	f.hasher = &fakeHasher{calls: &f.calls}
	f.uc = application.NewRegisterUseCase(f.users, f.codes, f.events, f.hasher, codeTTL, slog.New(slog.DiscardHandler))

	return f
}

func validInput() application.RegisterInput {
	return application.RegisterInput{
		Email:    "user@example.com",
		Password: "Str0ngPassw0rd!",
		Role:     "athlete",
	}
}

func TestRegister_Success(t *testing.T) {
	f := newFixture()
	ctx := context.WithValue(t.Context(), ctxKey{}, "request")
	before := time.Now().UTC()

	out, err := f.uc.Execute(ctx, application.RegisterInput{
		Email:    "  User@Example.COM ",
		Password: "Str0ngPassw0rd!",
		Role:     "trainer",
	})
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}

	wantCalls := []string{"hasher.hash", "users.save", "codes.save", "events.publish"}
	if !slices.Equal(f.calls, wantCalls) {
		t.Errorf("calls = %v, want %v", f.calls, wantCalls)
	}

	if len(f.users.saved) != 1 {
		t.Fatalf("users saved = %d, want 1", len(f.users.saved))
	}

	user := f.users.saved[0]
	if user.ID == uuid.Nil {
		t.Error("saved user has no id")
	}

	if out.UserID != user.ID {
		t.Errorf("output user id = %s, want saved user id %s", out.UserID, user.ID)
	}

	if out.ExpiresIn != codeTTL {
		t.Errorf("output expires in = %s, want %s", out.ExpiresIn, codeTTL)
	}

	if user.Email != "user@example.com" {
		t.Errorf("saved email = %q, want normalized user@example.com", user.Email)
	}

	if user.Role != domain.RoleTrainer {
		t.Errorf("saved role = %q, want %q", user.Role, domain.RoleTrainer)
	}

	if !slices.Equal(f.hasher.hashed, []domain.RawPassword{"Str0ngPassw0rd!"}) {
		t.Errorf("hashed passwords = %v, want the raw password once", f.hasher.hashed)
	}

	if user.PasswordHash != "hashed:Str0ngPassw0rd!" {
		t.Errorf("saved password hash = %q, want the hasher output", user.PasswordHash)
	}

	if user.IsVerified {
		t.Error("saved user is verified, want unverified")
	}

	if user.CreatedAt.Before(before) || !user.CreatedAt.Equal(user.UpdatedAt) || user.CreatedAt.Location() != time.UTC {
		t.Errorf("created at = %s, updated at = %s, want equal UTC times not before %s", user.CreatedAt, user.UpdatedAt, before)
	}

	if len(f.codes.saved) != 1 {
		t.Fatalf("codes saved = %d, want 1", len(f.codes.saved))
	}

	code := f.codes.saved[0]
	if code.email != "user@example.com" || code.ttl != codeTTL {
		t.Errorf("code saved for %q with ttl %s, want user@example.com with %s", code.email, code.ttl, codeTTL)
	}

	if !verificationCodePattern.MatchString(code.code) {
		t.Errorf("code = %q, want 6 digits", code.code)
	}

	wantEvent := application.UserRegisteredEvent{UserID: user.ID, Email: "user@example.com", Code: code.code}
	if !slices.Equal(f.events.published, []application.UserRegisteredEvent{wantEvent}) {
		t.Errorf("published events = %+v, want [%+v]", f.events.published, wantEvent)
	}

	for name, got := range map[string]any{"users": f.users.ctxValue, "codes": f.codes.ctxValue, "events": f.events.ctxValue} {
		if got != "request" {
			t.Errorf("%s did not receive the request context", name)
		}
	}
}

func TestRegister_UsesUserIDAssignedByRepository(t *testing.T) {
	f := newFixture()
	existingID := uuid.New()
	f.users.assignID = existingID

	out, err := f.uc.Execute(t.Context(), validInput())
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}

	if out.UserID != existingID {
		t.Errorf("output user id = %s, want repository id %s", out.UserID, existingID)
	}

	if len(f.events.published) != 1 || f.events.published[0].UserID != existingID {
		t.Errorf("published events = %+v, want user id %s", f.events.published, existingID)
	}
}

func TestRegister_AcceptsSupportedRoles(t *testing.T) {
	for _, role := range []domain.Role{domain.RoleAthlete, domain.RoleTrainer} {
		t.Run(role.String(), func(t *testing.T) {
			f := newFixture()
			in := validInput()
			in.Role = role.String()

			if _, err := f.uc.Execute(t.Context(), in); err != nil {
				t.Fatalf("Execute() error = %v", err)
			}

			if len(f.users.saved) != 1 || f.users.saved[0].Role != role {
				t.Errorf("saved users = %+v, want one with role %q", f.users.saved, role)
			}
		})
	}
}

func TestRegister_NormalizesEmail(t *testing.T) {
	tests := []struct {
		name  string
		email string
	}{
		{name: "upper case", email: "USER@EXAMPLE.COM"},
		{name: "mixed case", email: "User@Example.com"},
		{name: "surrounding spaces", email: " \t user@example.com \n"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f := newFixture()
			in := validInput()
			in.Email = tt.email

			if _, err := f.uc.Execute(t.Context(), in); err != nil {
				t.Fatalf("Execute() error = %v", err)
			}

			if len(f.users.saved) != 1 || f.users.saved[0].Email != "user@example.com" {
				t.Errorf("saved users = %+v, want email user@example.com", f.users.saved)
			}

			if len(f.codes.saved) != 1 || f.codes.saved[0].email != "user@example.com" {
				t.Errorf("saved codes = %+v, want email user@example.com", f.codes.saved)
			}
		})
	}
}

func TestRegister_PasswordLengthIsCountedInCharacters(t *testing.T) {
	tests := []struct {
		name     string
		password string
		wantErr  error
	}{
		{name: "8 ascii characters", password: "12345678"},
		{name: "8 cyrillic characters", password: "пароль12"},
		{name: "7 ascii characters", password: "1234567", wantErr: domain.ErrWeakPassword},
		{name: "7 cyrillic characters longer than 8 bytes", password: "пароль1", wantErr: domain.ErrWeakPassword},
		{name: "empty", password: "", wantErr: domain.ErrWeakPassword},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f := newFixture()
			in := validInput()
			in.Password = tt.password

			_, err := f.uc.Execute(t.Context(), in)
			if !errors.Is(err, tt.wantErr) {
				t.Fatalf("Execute() error = %v, want %v", err, tt.wantErr)
			}
		})
	}
}

func TestRegister_RejectsInvalidInputWithoutSideEffects(t *testing.T) {
	tests := []struct {
		name    string
		input   application.RegisterInput
		wantErr error
	}{
		{name: "empty email", input: application.RegisterInput{Email: "", Password: "Str0ngPassw0rd!", Role: "athlete"}, wantErr: domain.ErrInvalidEmail},
		{name: "email without at sign", input: application.RegisterInput{Email: "user.example.com", Password: "Str0ngPassw0rd!", Role: "athlete"}, wantErr: domain.ErrInvalidEmail},
		{name: "email without domain", input: application.RegisterInput{Email: "user@", Password: "Str0ngPassw0rd!", Role: "athlete"}, wantErr: domain.ErrInvalidEmail},
		{name: "email with display name", input: application.RegisterInput{Email: "User <user@example.com>", Password: "Str0ngPassw0rd!", Role: "athlete"}, wantErr: domain.ErrInvalidEmail},
		{name: "weak password", input: application.RegisterInput{Email: "user@example.com", Password: "short", Role: "athlete"}, wantErr: domain.ErrWeakPassword},
		{name: "empty role", input: application.RegisterInput{Email: "user@example.com", Password: "Str0ngPassw0rd!", Role: ""}, wantErr: domain.ErrInvalidRole},
		{name: "unknown role", input: application.RegisterInput{Email: "user@example.com", Password: "Str0ngPassw0rd!", Role: "admin"}, wantErr: domain.ErrInvalidRole},
		{name: "role in another case", input: application.RegisterInput{Email: "user@example.com", Password: "Str0ngPassw0rd!", Role: "Athlete"}, wantErr: domain.ErrInvalidRole},
		{name: "email is checked before password", input: application.RegisterInput{Email: "bad", Password: "short", Role: "admin"}, wantErr: domain.ErrInvalidEmail},
		{name: "password is checked before role", input: application.RegisterInput{Email: "user@example.com", Password: "short", Role: "admin"}, wantErr: domain.ErrWeakPassword},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f := newFixture()

			out, err := f.uc.Execute(t.Context(), tt.input)
			if !errors.Is(err, tt.wantErr) {
				t.Fatalf("Execute() error = %v, want %v", err, tt.wantErr)
			}

			if out != (application.RegisterOutput{}) {
				t.Errorf("output = %+v, want zero value on error", out)
			}

			if len(f.calls) != 0 {
				t.Errorf("calls = %v, want no dependency to be called", f.calls)
			}
		})
	}
}

func TestRegister_StopsOnDependencyFailure(t *testing.T) {
	tests := []struct {
		name      string
		setup     func(f *fixture)
		wantErr   error
		wantCalls []string
	}{
		{
			name:      "hasher fails",
			setup:     func(f *fixture) { f.hasher.err = errBoom },
			wantErr:   errBoom,
			wantCalls: []string{"hasher.hash"},
		},
		{
			name:      "email already taken",
			setup:     func(f *fixture) { f.users.err = domain.ErrEmailAlreadyTaken },
			wantErr:   domain.ErrEmailAlreadyTaken,
			wantCalls: []string{"hasher.hash", "users.save"},
		},
		{
			name:      "user repository fails",
			setup:     func(f *fixture) { f.users.err = errBoom },
			wantErr:   errBoom,
			wantCalls: []string{"hasher.hash", "users.save"},
		},
		{
			name:      "code store fails",
			setup:     func(f *fixture) { f.codes.err = errBoom },
			wantErr:   errBoom,
			wantCalls: []string{"hasher.hash", "users.save", "codes.save"},
		},
		{
			name:      "event publisher fails",
			setup:     func(f *fixture) { f.events.err = errBoom },
			wantErr:   errBoom,
			wantCalls: []string{"hasher.hash", "users.save", "codes.save", "events.publish"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f := newFixture()
			tt.setup(f)

			out, err := f.uc.Execute(t.Context(), validInput())
			if !errors.Is(err, tt.wantErr) {
				t.Fatalf("Execute() error = %v, want it to wrap %v", err, tt.wantErr)
			}

			if out != (application.RegisterOutput{}) {
				t.Errorf("output = %+v, want zero value on error", out)
			}

			if !slices.Equal(f.calls, tt.wantCalls) {
				t.Errorf("calls = %v, want %v", f.calls, tt.wantCalls)
			}

			if len(f.events.published) != 0 {
				t.Errorf("events published = %d, want 0 when registration fails", len(f.events.published))
			}
		})
	}
}

func TestRegister_GeneratesNewCodeForEveryRegistration(t *testing.T) {
	f := newFixture()

	for range 20 {
		if _, err := f.uc.Execute(t.Context(), validInput()); err != nil {
			t.Fatalf("Execute() error = %v", err)
		}
	}

	unique := make(map[string]struct{}, len(f.codes.saved))
	for _, c := range f.codes.saved {
		unique[c.code] = struct{}{}
	}

	if len(unique) < 2 {
		t.Errorf("20 registrations produced %d distinct codes, want them to differ", len(unique))
	}
}
