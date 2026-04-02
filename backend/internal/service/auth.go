package service

import (
	"context"
	"expense-tracker/internal/models"
	"expense-tracker/internal/repository"
	"fmt"
	"os"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
)

type AuthService struct {
	userRepo  *repository.UserRepository
	groupRepo *repository.GroupRepository
	jwtSecret []byte
}

func NewAuthService(userRepo *repository.UserRepository, groupRepo *repository.GroupRepository) *AuthService {
	secret := os.Getenv("JWT_SECRET")
	if secret == "" {
		secret = "default_secret_key" // Fallback for safety during dev
	}
	return &AuthService{
		userRepo:  userRepo,
		groupRepo: groupRepo,
		jwtSecret: []byte(secret),
	}
}

// Register creates a new user and an associated group.
func (s *AuthService) Register(ctx context.Context, req models.RegisterRequest) (*models.User, error) {
	// Check if user already exists
	existing, err := s.userRepo.GetByUsername(ctx, req.Username)
	if err != nil {
		return nil, err
	}
	if existing != nil {
		return nil, fmt.Errorf("username already taken")
	}

	// Hash password
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		return nil, fmt.Errorf("failed to hash password: %w", err)
	}

	// Determine role (admin or adminkb)
	role := models.RoleUser
	if req.Username == "admin" || req.Username == "adminkb" {
		role = models.RoleAdmin
	}

	// Create user
	userID := uuid.New().String()
	user := &models.User{
		ID:           userID,
		Username:     req.Username,
		PasswordHash: string(hashedPassword),
		Role:         role,
		Status:       models.StatusActive,
	}

	if err := s.userRepo.CreateUser(ctx, user); err != nil {
		return nil, err
	}

	// Create a private group for the user
	groupID := uuid.New().String()
	group := &models.UserGroup{
		ID:   groupID,
		Name: fmt.Sprintf("%s's Group", req.Username),
	}

	if err := s.groupRepo.CreateGroup(ctx, group); err != nil {
		return nil, err
	}

	// Add user to the group
	if err := s.groupRepo.AddMember(ctx, groupID, userID); err != nil {
		return nil, err
	}

	return user, nil
}

// Login validates user credentials and returns a JWT token.
func (s *AuthService) Login(ctx context.Context, req models.LoginRequest) (string, *models.User, error) {
	user, err := s.userRepo.GetByUsername(ctx, req.Username)
	if err != nil {
		return "", nil, err
	}
	if user == nil {
		return "", nil, fmt.Errorf("invalid username or password")
	}

	if user.Status == models.StatusInactive {
		return "", nil, fmt.Errorf("account is disabled")
	}

	// Compare password
	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(req.Password)); err != nil {
		return "", nil, fmt.Errorf("invalid username or password")
	}

	// Generate JWT
	tokenString, err := s.GenerateToken(user)
	if err != nil {
		return "", nil, fmt.Errorf("failed to generate token: %w", err)
	}

	return tokenString, user, nil
}

func (s *AuthService) GenerateToken(user *models.User) (string, error) {
	// Get User GroupID to embed in token (for easier access)
	// Alternatively, look it up in middleware. For performance, token is better.
	groupID, _ := s.groupRepo.GetUserGroupID(context.Background(), user.ID)

	claims := jwt.MapClaims{
		"sub":      user.ID,
		"username": user.Username,
		"role":     user.Role,
		"groupId":  groupID,
		"exp":      time.Now().Add(time.Hour * 24 * 7).Unix(), // 7 days
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(s.jwtSecret)
}

func (s *AuthService) ValidateToken(tokenString string) (jwt.MapClaims, error) {
	token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return s.jwtSecret, nil
	})

	if err != nil {
		return nil, err
	}

	if claims, ok := token.Claims.(jwt.MapClaims); ok && token.Valid {
		return claims, nil
	}

	return nil, fmt.Errorf("invalid token")
}
