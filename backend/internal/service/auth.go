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
	"crypto/sha256"
	"encoding/hex"
)

type AuthService struct {
	userRepo     *repository.UserRepository
	groupRepo    *repository.GroupRepository
	jwtSecret    []byte
	pepperSecret string
}

func NewAuthService(userRepo *repository.UserRepository, groupRepo *repository.GroupRepository) *AuthService {
	jwtSecret := os.Getenv("JWT_SECRET")
	if jwtSecret == "" {
		jwtSecret = "default_jwt_secret" // Fallback for dev
	}
	pepper := os.Getenv("PEPPER_SECRET")
	if pepper == "" {
		pepper = "default_pepper_key" // Fallback for dev
	}
	return &AuthService{
		userRepo:     userRepo,
		groupRepo:    groupRepo,
		jwtSecret:    []byte(jwtSecret),
		pepperSecret: pepper,
	}
}

// hashWithPepper combines the password with user-specific data and a system secret.
func (s *AuthService) hashWithPepper(password, username string) string {
	data := fmt.Sprintf("%s:%s:%s", password, username, s.pepperSecret)
	hash := sha256.Sum256([]byte(data))
	return hex.EncodeToString(hash[:])
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

	// Hash password with pepper and username
	pepperedPassword := s.hashWithPepper(req.Password, req.Username)
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(pepperedPassword), bcrypt.DefaultCost)
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

	// Compare password (using pepper and username)
	pepperedPassword := s.hashWithPepper(req.Password, req.Username)
	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(pepperedPassword)); err != nil {
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
