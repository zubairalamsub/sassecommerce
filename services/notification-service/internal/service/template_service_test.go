package service

import (
	"context"
	"testing"

	"github.com/ecommerce/notification-service/internal/models"
	repoMocks "github.com/ecommerce/notification-service/internal/repository/mocks"
	deftpl "github.com/ecommerce/notification-service/internal/templates"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// newTestTemplateService spins up a templateService with a mocked repo and an
// in-memory logger. No providers are wired — the install-defaults path does
// not touch any provider, so a nil map is sufficient.
func newTestTemplateService() (*templateService, *repoMocks.MockNotificationRepository) {
	mockRepo := new(repoMocks.MockNotificationRepository)
	logger := logrus.New()
	logger.SetLevel(logrus.PanicLevel)
	return &templateService{
		repo:      mockRepo,
		providers: map[models.Channel]NotificationProvider{},
		logger:    logger,
	}, mockRepo
}

// TestInstallDefaults_EmptyTenant verifies that a tenant with no existing
// templates gets every starter-pack entry created exactly once, with
// IsActive=true and the correct tenant scoping.
func TestInstallDefaults_EmptyTenant(t *testing.T) {
	svc, mockRepo := newTestTemplateService()
	ctx := context.Background()
	tenantID := "tenant-1"

	expected := deftpl.Defaults()

	// No existing templates → every default should be inserted.
	mockRepo.On("ListTemplates", ctx, tenantID).Return([]models.NotificationTemplate{}, nil)
	mockRepo.On("CreateTemplate", ctx, mock.AnythingOfType("*models.NotificationTemplate")).
		Return(nil).Times(len(expected))

	res, err := svc.InstallDefaults(ctx, tenantID, false)

	assert.NoError(t, err)
	assert.NotNil(t, res)
	assert.Equal(t, len(expected), res.Created, "should create one row per default template")
	assert.Equal(t, 0, res.Updated)
	assert.Equal(t, 0, res.Skipped)
	assert.Len(t, res.Templates, len(expected))

	// Every returned template should be tenant-scoped and active.
	for _, tpl := range res.Templates {
		assert.Equal(t, tenantID, tpl.TenantID)
		assert.True(t, tpl.IsActive)
		assert.NotEmpty(t, tpl.ID)
		assert.NotEmpty(t, tpl.BodyTemplate)
	}

	mockRepo.AssertExpectations(t)
}

// TestInstallDefaults_SkipExisting confirms that when force=false we leave
// duplicates alone and report them in the Skipped counter.
func TestInstallDefaults_SkipExisting(t *testing.T) {
	svc, mockRepo := newTestTemplateService()
	ctx := context.Background()
	tenantID := "tenant-1"

	expected := deftpl.Defaults()
	// Simulate that two templates already exist for this tenant: welcome and
	// order_confirmation. The rest should still be created.
	preexisting := []models.NotificationTemplate{
		{ID: "t-1", TenantID: tenantID, Type: models.TypeWelcome, Channel: models.ChannelEmail, Name: "Custom welcome"},
		{ID: "t-2", TenantID: tenantID, Type: models.TypeOrderConfirmation, Channel: models.ChannelEmail, Name: "Custom order"},
	}
	mockRepo.On("ListTemplates", ctx, tenantID).Return(preexisting, nil)
	mockRepo.On("CreateTemplate", ctx, mock.AnythingOfType("*models.NotificationTemplate")).
		Return(nil).Times(len(expected) - len(preexisting))

	res, err := svc.InstallDefaults(ctx, tenantID, false)

	assert.NoError(t, err)
	assert.Equal(t, len(expected)-len(preexisting), res.Created)
	assert.Equal(t, len(preexisting), res.Skipped)
	assert.Equal(t, 0, res.Updated)

	// Skipped templates must not appear in the touched list.
	for _, tpl := range res.Templates {
		assert.NotEqual(t, models.TypeWelcome, tpl.Type)
		assert.NotEqual(t, models.TypeOrderConfirmation, tpl.Type)
	}

	mockRepo.AssertExpectations(t)
}

// TestInstallDefaults_ForceOverwrites confirms force=true updates duplicates
// in place rather than skipping them.
func TestInstallDefaults_ForceOverwrites(t *testing.T) {
	svc, mockRepo := newTestTemplateService()
	ctx := context.Background()
	tenantID := "tenant-1"

	expected := deftpl.Defaults()
	preexisting := []models.NotificationTemplate{
		{ID: "t-1", TenantID: tenantID, Type: models.TypeWelcome, Channel: models.ChannelEmail, Name: "Custom welcome"},
	}
	mockRepo.On("ListTemplates", ctx, tenantID).Return(preexisting, nil)
	mockRepo.On("UpdateTemplate", ctx, tenantID, "t-1", mock.AnythingOfType("*models.NotificationTemplate")).Return(nil).Once()
	mockRepo.On("CreateTemplate", ctx, mock.AnythingOfType("*models.NotificationTemplate")).
		Return(nil).Times(len(expected) - len(preexisting))

	res, err := svc.InstallDefaults(ctx, tenantID, true)

	assert.NoError(t, err)
	assert.Equal(t, len(expected)-len(preexisting), res.Created)
	assert.Equal(t, 0, res.Skipped)
	assert.Equal(t, len(preexisting), res.Updated)

	mockRepo.AssertExpectations(t)
}

// TestInstallDefaults_MissingTenant covers the input-validation branch.
func TestInstallDefaults_MissingTenant(t *testing.T) {
	svc, _ := newTestTemplateService()
	_, err := svc.InstallDefaults(context.Background(), "", false)
	assert.Error(t, err)
}
