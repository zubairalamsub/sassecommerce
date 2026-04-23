package e2e

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/ecommerce/user-service/internal/api"
	"github.com/ecommerce/user-service/internal/middleware"
	"github.com/ecommerce/user-service/internal/models"
	"github.com/ecommerce/user-service/internal/repository"
	"github.com/ecommerce/user-service/internal/service"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

type E2ETestSuite struct {
	suite.Suite
	db          *gorm.DB
	router      *gin.Engine
	authService service.AuthService
}

func (suite *E2ETestSuite) SetupSuite() {
	// Create in-memory SQLite database
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	assert.NoError(suite.T(), err)

	// Run migrations
	err = db.AutoMigrate(&models.User{}, &models.RefreshToken{})
	assert.NoError(suite.T(), err)

	suite.db = db

	// Initialize services
	logger := logrus.New()
	logger.SetOutput(io.Discard)

	userRepo := repository.NewUserRepository(db)
	tokenConfig := models.TokenConfig{
		SecretKey:      "test-secret-key",
		ExpirationTime: 24 * time.Hour,
		Issuer:         "test-service",
	}

	suite.authService = service.NewAuthService(userRepo, tokenConfig, logger)
	userService := service.NewUserService(userRepo, logger)

	// Initialize handlers
	authHandler := api.NewAuthHandler(suite.authService, logger)
	userHandler := api.NewUserHandler(userService, logger)

	// Setup router
	gin.SetMode(gin.TestMode)
	router := gin.New()

	v1 := router.Group("/api/v1")
	{
		auth := v1.Group("/auth")
		{
			auth.POST("/register", authHandler.Register)
			auth.POST("/login", authHandler.Login)
		}

		authProtected := v1.Group("/auth")
		authProtected.Use(middleware.AuthMiddleware(suite.authService))
		{
			authProtected.GET("/profile", authHandler.GetProfile)
			authProtected.POST("/change-password", authHandler.ChangePassword)
		}

		users := v1.Group("/users")
		users.Use(middleware.AuthMiddleware(suite.authService))
		{
			users.GET("", userHandler.ListUsers)
			users.GET("/:id", userHandler.GetUser)
			users.PUT("/:id", userHandler.UpdateUser)
			users.DELETE("/:id", userHandler.DeleteUser)
			users.PATCH("/:id/role", middleware.RequireRole(models.UserRoleAdmin), userHandler.UpdateUserRole)
			users.PATCH("/:id/status", middleware.RequireRole(models.UserRoleAdmin), userHandler.UpdateUserStatus)
		}
	}

	suite.router = router
}

func (suite *E2ETestSuite) TearDownSuite() {
	sqlDB, _ := suite.db.DB()
	sqlDB.Close()
}

func (suite *E2ETestSuite) TestRegister_Success() {
	reqBody := models.RegisterRequest{
		TenantID:  uuid.New().String(),
		Email:     "e2e@example.com",
		Username:  "e2euser",
		Password:  "password123",
		FirstName: "E2E",
		LastName:  "User",
	}

	body, _ := json.Marshal(reqBody)
	req, _ := http.NewRequest("POST", "/api/v1/auth/register", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	suite.router.ServeHTTP(w, req)

	assert.Equal(suite.T(), http.StatusCreated, w.Code)

	var response map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &response)
	assert.True(suite.T(), response["success"].(bool))
	assert.NotNil(suite.T(), response["data"])
}

func (suite *E2ETestSuite) TestRegister_DuplicateEmail() {
	tenantID := uuid.New().String()

	// First registration
	reqBody := models.RegisterRequest{
		TenantID:  tenantID,
		Email:     "duplicate@example.com",
		Username:  "user1",
		Password:  "password123",
		FirstName: "User",
		LastName:  "One",
	}

	body, _ := json.Marshal(reqBody)
	req, _ := http.NewRequest("POST", "/api/v1/auth/register", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	suite.router.ServeHTTP(w, req)
	assert.Equal(suite.T(), http.StatusCreated, w.Code)

	// Second registration with same email
	reqBody.Username = "user2"
	body, _ = json.Marshal(reqBody)
	req, _ = http.NewRequest("POST", "/api/v1/auth/register", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	w = httptest.NewRecorder()
	suite.router.ServeHTTP(w, req)

	assert.Equal(suite.T(), http.StatusConflict, w.Code)

	var response map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &response)
	assert.False(suite.T(), response["success"].(bool))
}

func (suite *E2ETestSuite) TestLogin_Success() {
	tenantID := uuid.New().String()

	// Register user
	registerReq := models.RegisterRequest{
		TenantID:  tenantID,
		Email:     "login@example.com",
		Username:  "loginuser",
		Password:  "password123",
		FirstName: "Login",
		LastName:  "User",
	}

	body, _ := json.Marshal(registerReq)
	req, _ := http.NewRequest("POST", "/api/v1/auth/register", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	suite.router.ServeHTTP(w, req)
	assert.Equal(suite.T(), http.StatusCreated, w.Code)

	// Login
	loginReq := models.LoginRequest{
		TenantID: tenantID,
		Email:    "login@example.com",
		Password: "password123",
	}

	body, _ = json.Marshal(loginReq)
	req, _ = http.NewRequest("POST", "/api/v1/auth/login", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	w = httptest.NewRecorder()
	suite.router.ServeHTTP(w, req)

	assert.Equal(suite.T(), http.StatusOK, w.Code)

	var response map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &response)
	assert.True(suite.T(), response["success"].(bool))

	data := response["data"].(map[string]interface{})
	assert.NotEmpty(suite.T(), data["token"])
	assert.NotNil(suite.T(), data["user"])
}

func (suite *E2ETestSuite) TestLogin_InvalidCredentials() {
	loginReq := models.LoginRequest{
		TenantID: uuid.New().String(),
		Email:    "nonexistent@example.com",
		Password: "wrongpassword",
	}

	body, _ := json.Marshal(loginReq)
	req, _ := http.NewRequest("POST", "/api/v1/auth/login", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	suite.router.ServeHTTP(w, req)

	assert.Equal(suite.T(), http.StatusUnauthorized, w.Code)
}

func (suite *E2ETestSuite) TestGetProfile_Success() {
	tenantID := uuid.New().String()

	// Register and login
	token := suite.registerAndLogin(tenantID, "profile@example.com", "profileuser", "password123")

	// Get profile
	req, _ := http.NewRequest("GET", "/api/v1/auth/profile", nil)
	req.Header.Set("Authorization", "Bearer "+token)

	w := httptest.NewRecorder()
	suite.router.ServeHTTP(w, req)

	assert.Equal(suite.T(), http.StatusOK, w.Code)

	var response map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &response)
	assert.True(suite.T(), response["success"].(bool))
}

func (suite *E2ETestSuite) TestGetProfile_Unauthorized() {
	req, _ := http.NewRequest("GET", "/api/v1/auth/profile", nil)

	w := httptest.NewRecorder()
	suite.router.ServeHTTP(w, req)

	assert.Equal(suite.T(), http.StatusUnauthorized, w.Code)
}

func (suite *E2ETestSuite) TestChangePassword_Success() {
	tenantID := uuid.New().String()
	token := suite.registerAndLogin(tenantID, "changepw@example.com", "changepwuser", "oldpassword123")

	changeReq := models.ChangePasswordRequest{
		OldPassword: "oldpassword123",
		NewPassword: "newpassword123",
	}

	body, _ := json.Marshal(changeReq)
	req, _ := http.NewRequest("POST", "/api/v1/auth/change-password", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)

	w := httptest.NewRecorder()
	suite.router.ServeHTTP(w, req)

	assert.Equal(suite.T(), http.StatusOK, w.Code)
}

func (suite *E2ETestSuite) TestListUsers_Success() {
	tenantID := uuid.New().String()

	// Create multiple users
	for i := 0; i < 3; i++ {
		email := "listuser" + uuid.New().String() + "@example.com"
		username := "listuser" + uuid.New().String()
		suite.registerAndLogin(tenantID, email, username, "password123")
	}

	// Get token for one user
	token := suite.registerAndLogin(tenantID, "listmain@example.com", "listmain", "password123")

	// List users
	req, _ := http.NewRequest("GET", "/api/v1/users?page=1&page_size=10", nil)
	req.Header.Set("Authorization", "Bearer "+token)

	w := httptest.NewRecorder()
	suite.router.ServeHTTP(w, req)

	assert.Equal(suite.T(), http.StatusOK, w.Code)

	var response map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &response)
	assert.True(suite.T(), response["success"].(bool))
	assert.NotNil(suite.T(), response["data"])
	assert.NotNil(suite.T(), response["pagination"])
}

// Helper function to register and login a user
func (suite *E2ETestSuite) registerAndLogin(tenantID, email, username, password string) string {
	// Register
	registerReq := models.RegisterRequest{
		TenantID:  tenantID,
		Email:     email,
		Username:  username,
		Password:  password,
		FirstName: "Test",
		LastName:  "User",
	}

	body, _ := json.Marshal(registerReq)
	req, _ := http.NewRequest("POST", "/api/v1/auth/register", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	suite.router.ServeHTTP(w, req)

	// Login
	loginReq := models.LoginRequest{
		TenantID: tenantID,
		Email:    email,
		Password: password,
	}

	body, _ = json.Marshal(loginReq)
	req, _ = http.NewRequest("POST", "/api/v1/auth/login", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	w = httptest.NewRecorder()
	suite.router.ServeHTTP(w, req)

	var response map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &response)
	data := response["data"].(map[string]interface{})
	return data["token"].(string)
}

func TestE2ETestSuite(t *testing.T) {
	suite.Run(t, new(E2ETestSuite))
}
