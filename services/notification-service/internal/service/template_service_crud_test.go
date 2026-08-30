package service

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/ecommerce/notification-service/internal/models"
	repoMocks "github.com/ecommerce/notification-service/internal/repository/mocks"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// stubProvider stands in for an email/SMS provider so TestSend can be driven
// without reaching a real gateway.
type stubProvider struct {
	channel models.Channel
	result  *ProviderResult
	err     error
	sent    []*models.Notification
}

func (p *stubProvider) Send(n *models.Notification) (*ProviderResult, error) {
	p.sent = append(p.sent, n)
	if p.err != nil {
		return nil, p.err
	}
	if p.result != nil {
		return p.result, nil
	}
	return &ProviderResult{ProviderName: "stub", MessageID: "msg-1", Success: true}, nil
}

func (p *stubProvider) Channel() models.Channel { return p.channel }

// newTemplateServiceWithProvider wires a service that has an email provider.
func newTemplateServiceWithProvider(p *stubProvider) (*templateService, *repoMocks.MockNotificationRepository) {
	mockRepo := new(repoMocks.MockNotificationRepository)
	logger := logrus.New()
	logger.SetLevel(logrus.PanicLevel)
	return &templateService{
		repo:      mockRepo,
		providers: map[models.Channel]NotificationProvider{p.channel: p},
		logger:    logger,
	}, mockRepo
}

func strPtr(s string) *string { return &s }
func boolPtr(b bool) *bool    { return &b }

func sampleTemplate() *models.NotificationTemplate {
	return &models.NotificationTemplate{
		ID:              "tmpl-1",
		TenantID:        "tenant-1",
		Type:            models.TypeWelcome,
		Channel:         models.ChannelEmail,
		Name:            "Welcome email",
		SubjectTemplate: "Welcome, {{.UserName}}!",
		BodyTemplate:    "<p>Hello {{.UserName}}</p>",
		IsActive:        true,
	}
}

// ------------------------------------------------------------------- List/Get

func TestTemplateList(t *testing.T) {
	svc, mockRepo := newTestTemplateService()
	want := []models.NotificationTemplate{*sampleTemplate()}
	mockRepo.On("ListTemplates", mock.Anything, "tenant-1").Return(want, nil)

	got, err := svc.List(context.Background(), "tenant-1")

	assert.NoError(t, err)
	assert.Equal(t, want, got)
	mockRepo.AssertExpectations(t)
}

// Listing without a tenant would return every tenant's templates, so it is
// refused before the repository is consulted.
func TestTemplateListRequiresTenant(t *testing.T) {
	svc, mockRepo := newTestTemplateService()

	_, err := svc.List(context.Background(), "")

	assert.EqualError(t, err, "tenant_id is required")
	mockRepo.AssertNotCalled(t, "ListTemplates", mock.Anything, mock.Anything)
}

func TestTemplateListPropagatesRepoError(t *testing.T) {
	svc, mockRepo := newTestTemplateService()
	mockRepo.On("ListTemplates", mock.Anything, "tenant-1").Return(nil, errors.New("mongo down"))

	_, err := svc.List(context.Background(), "tenant-1")

	assert.Error(t, err)
}

func TestTemplateGet(t *testing.T) {
	svc, mockRepo := newTestTemplateService()
	want := sampleTemplate()
	mockRepo.On("GetTemplate", mock.Anything, "tenant-1", "tmpl-1").Return(want, nil)

	got, err := svc.Get(context.Background(), "tenant-1", "tmpl-1")

	assert.NoError(t, err)
	assert.Equal(t, want, got)
	mockRepo.AssertExpectations(t)
}

func TestTemplateGetPropagatesRepoError(t *testing.T) {
	svc, mockRepo := newTestTemplateService()
	mockRepo.On("GetTemplate", mock.Anything, "tenant-1", "ghost").Return(nil, errors.New("not found"))

	_, err := svc.Get(context.Background(), "tenant-1", "ghost")

	assert.Error(t, err)
}

// -------------------------------------------------------------------- Create

func TestTemplateCreate(t *testing.T) {
	svc, mockRepo := newTestTemplateService()
	var captured *models.NotificationTemplate
	mockRepo.On("CreateTemplate", mock.Anything, mock.AnythingOfType("*models.NotificationTemplate")).
		Run(func(args mock.Arguments) {
			captured = args.Get(1).(*models.NotificationTemplate)
		}).
		Return(nil)

	got, err := svc.Create(context.Background(), "tenant-1", &models.CreateTemplateRequest{
		Type:            "  welcome  ",
		Channel:         " email ",
		Name:            "  Welcome email  ",
		SubjectTemplate: "Hi {{.UserName}}",
		BodyTemplate:    "<p>Hi</p>",
		IsActive:        true,
	})

	assert.NoError(t, err)
	assert.NotEmpty(t, got.ID, "a new template should be assigned an id")
	assert.Equal(t, "tenant-1", got.TenantID)
	// Surrounding whitespace is trimmed so lookups by (channel, type) match.
	assert.Equal(t, models.TypeWelcome, got.Type)
	assert.Equal(t, models.ChannelEmail, got.Channel)
	assert.Equal(t, "Welcome email", got.Name)
	assert.Equal(t, got, captured)
	mockRepo.AssertExpectations(t)
}

func TestTemplateCreateValidation(t *testing.T) {
	tests := []struct {
		name     string
		tenantID string
		req      *models.CreateTemplateRequest
		wantErr  string
	}{
		{
			name:     "missing tenant",
			tenantID: "",
			req:      &models.CreateTemplateRequest{Type: "welcome", Channel: "email", Name: "n", BodyTemplate: "b"},
			wantErr:  "tenant_id is required",
		},
		{
			name:     "missing type",
			tenantID: "tenant-1",
			req:      &models.CreateTemplateRequest{Channel: "email", Name: "n", BodyTemplate: "b"},
			wantErr:  "type is required",
		},
		{
			name:     "blank type",
			tenantID: "tenant-1",
			req:      &models.CreateTemplateRequest{Type: "   ", Channel: "email", Name: "n", BodyTemplate: "b"},
			wantErr:  "type is required",
		},
		{
			name:     "missing name",
			tenantID: "tenant-1",
			req:      &models.CreateTemplateRequest{Type: "welcome", Channel: "email", BodyTemplate: "b"},
			wantErr:  "name is required",
		},
		{
			name:     "unknown channel",
			tenantID: "tenant-1",
			req:      &models.CreateTemplateRequest{Type: "welcome", Channel: "carrier-pigeon", Name: "n", BodyTemplate: "b"},
			wantErr:  "invalid channel",
		},
		{
			name:     "missing body",
			tenantID: "tenant-1",
			req:      &models.CreateTemplateRequest{Type: "welcome", Channel: "email", Name: "n"},
			wantErr:  "body_template is required",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc, mockRepo := newTestTemplateService()

			_, err := svc.Create(context.Background(), tt.tenantID, tt.req)

			if assert.Error(t, err) {
				assert.Contains(t, err.Error(), tt.wantErr)
			}
			mockRepo.AssertNotCalled(t, "CreateTemplate", mock.Anything, mock.Anything)
		})
	}
}

func TestTemplateCreateAcceptsEverySupportedChannel(t *testing.T) {
	for _, channel := range []string{"email", "sms", "push"} {
		t.Run(channel, func(t *testing.T) {
			svc, mockRepo := newTestTemplateService()
			mockRepo.On("CreateTemplate", mock.Anything, mock.Anything).Return(nil)

			got, err := svc.Create(context.Background(), "tenant-1", &models.CreateTemplateRequest{
				Type: "welcome", Channel: channel, Name: "n", BodyTemplate: "b",
			})

			assert.NoError(t, err)
			assert.Equal(t, models.Channel(channel), got.Channel)
		})
	}
}

// Templates are validated at save time so an admin gets the error while
// editing rather than a 500 at send time.
func TestTemplateCreateRejectsUnparsableTemplates(t *testing.T) {
	tests := []struct {
		name    string
		req     *models.CreateTemplateRequest
		wantErr string
	}{
		{
			name:    "broken subject",
			req:     &models.CreateTemplateRequest{Type: "welcome", Channel: "email", Name: "n", SubjectTemplate: "{{.Unclosed", BodyTemplate: "ok"},
			wantErr: "subject template invalid",
		},
		{
			name:    "broken body",
			req:     &models.CreateTemplateRequest{Type: "welcome", Channel: "email", Name: "n", BodyTemplate: "{{range}}"},
			wantErr: "body template invalid",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc, mockRepo := newTestTemplateService()

			_, err := svc.Create(context.Background(), "tenant-1", tt.req)

			if assert.Error(t, err) {
				assert.Contains(t, err.Error(), tt.wantErr)
			}
			mockRepo.AssertNotCalled(t, "CreateTemplate", mock.Anything, mock.Anything)
		})
	}
}

func TestTemplateCreatePropagatesRepoError(t *testing.T) {
	svc, mockRepo := newTestTemplateService()
	mockRepo.On("CreateTemplate", mock.Anything, mock.Anything).Return(errors.New("duplicate key"))

	_, err := svc.Create(context.Background(), "tenant-1", &models.CreateTemplateRequest{
		Type: "welcome", Channel: "email", Name: "n", BodyTemplate: "b",
	})

	assert.Error(t, err)
}

// -------------------------------------------------------------------- Update

func TestTemplateUpdateAppliesOnlySuppliedFields(t *testing.T) {
	svc, mockRepo := newTestTemplateService()
	existing := sampleTemplate()
	mockRepo.On("GetTemplate", mock.Anything, "tenant-1", "tmpl-1").Return(existing, nil)
	mockRepo.On("UpdateTemplate", mock.Anything, "tenant-1", "tmpl-1", mock.Anything).Return(nil)

	got, err := svc.Update(context.Background(), "tenant-1", "tmpl-1", &models.UpdateTemplateRequest{
		Name:     strPtr("  Renamed  "),
		IsActive: boolPtr(false),
	})

	assert.NoError(t, err)
	assert.Equal(t, "Renamed", got.Name)
	assert.False(t, got.IsActive)
	// Untouched fields keep their stored values.
	assert.Equal(t, "Welcome, {{.UserName}}!", got.SubjectTemplate)
	assert.Equal(t, models.ChannelEmail, got.Channel)
	mockRepo.AssertExpectations(t)
}

func TestTemplateUpdateAllFields(t *testing.T) {
	svc, mockRepo := newTestTemplateService()
	mockRepo.On("GetTemplate", mock.Anything, "tenant-1", "tmpl-1").Return(sampleTemplate(), nil)
	mockRepo.On("UpdateTemplate", mock.Anything, "tenant-1", "tmpl-1", mock.Anything).Return(nil)

	got, err := svc.Update(context.Background(), "tenant-1", "tmpl-1", &models.UpdateTemplateRequest{
		Type:            strPtr("order_shipped"),
		Channel:         strPtr("sms"),
		Name:            strPtr("Shipped"),
		SubjectTemplate: strPtr("Shipped {{.OrderID}}"),
		BodyTemplate:    strPtr("Your order {{.OrderID}} shipped"),
		IsActive:        boolPtr(true),
	})

	assert.NoError(t, err)
	assert.Equal(t, models.TypeOrderShipped, got.Type)
	assert.Equal(t, models.ChannelSMS, got.Channel)
	assert.Equal(t, "Shipped", got.Name)
	assert.Equal(t, "Shipped {{.OrderID}}", got.SubjectTemplate)
	assert.True(t, got.IsActive)
}

func TestTemplateUpdateRejectsInvalidInput(t *testing.T) {
	tests := []struct {
		name    string
		req     *models.UpdateTemplateRequest
		wantErr string
	}{
		{name: "unknown channel", req: &models.UpdateTemplateRequest{Channel: strPtr("carrier-pigeon")}, wantErr: "invalid channel"},
		{name: "broken subject", req: &models.UpdateTemplateRequest{SubjectTemplate: strPtr("{{.Unclosed")}, wantErr: "subject template invalid"},
		{name: "broken body", req: &models.UpdateTemplateRequest{BodyTemplate: strPtr("{{range}}")}, wantErr: "body template invalid"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc, mockRepo := newTestTemplateService()
			mockRepo.On("GetTemplate", mock.Anything, "tenant-1", "tmpl-1").Return(sampleTemplate(), nil)

			_, err := svc.Update(context.Background(), "tenant-1", "tmpl-1", tt.req)

			if assert.Error(t, err) {
				assert.Contains(t, err.Error(), tt.wantErr)
			}
			mockRepo.AssertNotCalled(t, "UpdateTemplate", mock.Anything, mock.Anything, mock.Anything, mock.Anything)
		})
	}
}

func TestTemplateUpdateOnMissingTemplate(t *testing.T) {
	svc, mockRepo := newTestTemplateService()
	mockRepo.On("GetTemplate", mock.Anything, "tenant-1", "ghost").Return(nil, errors.New("not found"))

	_, err := svc.Update(context.Background(), "tenant-1", "ghost", &models.UpdateTemplateRequest{Name: strPtr("x")})

	assert.Error(t, err)
}

func TestTemplateUpdatePropagatesRepoError(t *testing.T) {
	svc, mockRepo := newTestTemplateService()
	mockRepo.On("GetTemplate", mock.Anything, "tenant-1", "tmpl-1").Return(sampleTemplate(), nil)
	mockRepo.On("UpdateTemplate", mock.Anything, "tenant-1", "tmpl-1", mock.Anything).Return(errors.New("mongo down"))

	_, err := svc.Update(context.Background(), "tenant-1", "tmpl-1", &models.UpdateTemplateRequest{Name: strPtr("x")})

	assert.Error(t, err)
}

// -------------------------------------------------------------------- Delete

func TestTemplateDelete(t *testing.T) {
	svc, mockRepo := newTestTemplateService()
	mockRepo.On("DeleteTemplate", mock.Anything, "tenant-1", "tmpl-1").Return(nil)

	assert.NoError(t, svc.Delete(context.Background(), "tenant-1", "tmpl-1"))
	mockRepo.AssertExpectations(t)
}

func TestTemplateDeletePropagatesRepoError(t *testing.T) {
	svc, mockRepo := newTestTemplateService()
	mockRepo.On("DeleteTemplate", mock.Anything, "tenant-1", "tmpl-1").Return(errors.New("mongo down"))

	assert.Error(t, svc.Delete(context.Background(), "tenant-1", "tmpl-1"))
}

// ------------------------------------------------------------------- Preview

func TestTemplatePreviewUsesSuppliedVars(t *testing.T) {
	svc, mockRepo := newTestTemplateService()
	mockRepo.On("GetTemplate", mock.Anything, "tenant-1", "tmpl-1").Return(sampleTemplate(), nil)

	got, err := svc.Preview(context.Background(), "tenant-1", "tmpl-1", map[string]interface{}{"UserName": "Alice"})

	assert.NoError(t, err)
	assert.Equal(t, "Welcome, Alice!", got.Subject)
	assert.Contains(t, got.Body, "Hello Alice")
}

// Without overrides the preview still renders, using the sample defaults, so
// an admin never sees raw {{.Placeholders}}.
func TestTemplatePreviewFallsBackToSampleVars(t *testing.T) {
	svc, mockRepo := newTestTemplateService()
	mockRepo.On("GetTemplate", mock.Anything, "tenant-1", "tmpl-1").Return(sampleTemplate(), nil)

	got, err := svc.Preview(context.Background(), "tenant-1", "tmpl-1", nil)

	assert.NoError(t, err)
	assert.NotContains(t, got.Subject, "{{")
	assert.NotEmpty(t, strings.TrimSpace(got.Subject))
}

func TestTemplatePreviewOnMissingTemplate(t *testing.T) {
	svc, mockRepo := newTestTemplateService()
	mockRepo.On("GetTemplate", mock.Anything, "tenant-1", "ghost").Return(nil, errors.New("not found"))

	_, err := svc.Preview(context.Background(), "tenant-1", "ghost", nil)

	assert.Error(t, err)
}

// A template stored before validation existed can still be unrenderable; the
// preview must report that rather than panic.
func TestTemplatePreviewSurfacesRenderFailure(t *testing.T) {
	svc, mockRepo := newTestTemplateService()
	broken := sampleTemplate()
	broken.SubjectTemplate = "{{.UserName.Missing}}"
	mockRepo.On("GetTemplate", mock.Anything, "tenant-1", "tmpl-1").Return(broken, nil)

	_, err := svc.Preview(context.Background(), "tenant-1", "tmpl-1", nil)

	assert.Error(t, err)
}

// ------------------------------------------------------------------ TestSend

func TestTemplateTestSend(t *testing.T) {
	provider := &stubProvider{channel: models.ChannelEmail}
	svc, mockRepo := newTemplateServiceWithProvider(provider)
	mockRepo.On("GetTemplate", mock.Anything, "tenant-1", "tmpl-1").Return(sampleTemplate(), nil)

	err := svc.TestSend(context.Background(), "tenant-1", "tmpl-1", "admin@example.com", map[string]interface{}{"UserName": "Alice"})

	assert.NoError(t, err)
	if assert.Len(t, provider.sent, 1) {
		sent := provider.sent[0]
		assert.Equal(t, "admin@example.com", sent.Recipient)
		assert.Equal(t, "Welcome, Alice!", sent.Subject)
		// The record is flagged so a test send is never mistaken for a real one.
		assert.Equal(t, true, sent.Metadata["test_send"])
		assert.Equal(t, "tmpl-1", sent.Metadata["template_id"])
		// It is filed against the template's tenant, not a caller-supplied one.
		assert.Equal(t, "tenant-1", sent.TenantID)
	}
}

func TestTemplateTestSendRequiresEmail(t *testing.T) {
	provider := &stubProvider{channel: models.ChannelEmail}
	svc, mockRepo := newTemplateServiceWithProvider(provider)

	err := svc.TestSend(context.Background(), "tenant-1", "tmpl-1", "", nil)

	assert.EqualError(t, err, "email is required")
	mockRepo.AssertNotCalled(t, "GetTemplate", mock.Anything, mock.Anything, mock.Anything)
}

// GetTemplate is tenant-scoped, so another tenant's id resolves to not-found —
// this is what stops a caller rendering someone else's template to their own
// address.
func TestTemplateTestSendOnMissingTemplate(t *testing.T) {
	provider := &stubProvider{channel: models.ChannelEmail}
	svc, mockRepo := newTemplateServiceWithProvider(provider)
	mockRepo.On("GetTemplate", mock.Anything, "tenant-1", "other-tenants-template").Return(nil, errors.New("not found"))

	err := svc.TestSend(context.Background(), "tenant-1", "other-tenants-template", "attacker@example.com", nil)

	assert.Error(t, err)
	assert.Empty(t, provider.sent, "nothing may be dispatched for a template the tenant cannot see")
}

func TestTemplateTestSendWithoutAProviderForTheChannel(t *testing.T) {
	// Only an email provider is configured, but the template is SMS.
	provider := &stubProvider{channel: models.ChannelEmail}
	svc, mockRepo := newTemplateServiceWithProvider(provider)
	smsTemplate := sampleTemplate()
	smsTemplate.Channel = models.ChannelSMS
	mockRepo.On("GetTemplate", mock.Anything, "tenant-1", "tmpl-1").Return(smsTemplate, nil)

	err := svc.TestSend(context.Background(), "tenant-1", "tmpl-1", "admin@example.com", nil)

	if assert.Error(t, err) {
		assert.Contains(t, err.Error(), "no provider configured for channel")
	}
}

func TestTemplateTestSendSurfacesRenderFailure(t *testing.T) {
	provider := &stubProvider{channel: models.ChannelEmail}
	svc, mockRepo := newTemplateServiceWithProvider(provider)
	broken := sampleTemplate()
	broken.BodyTemplate = "{{.UserName.Missing}}"
	mockRepo.On("GetTemplate", mock.Anything, "tenant-1", "tmpl-1").Return(broken, nil)

	err := svc.TestSend(context.Background(), "tenant-1", "tmpl-1", "admin@example.com", nil)

	assert.Error(t, err)
	assert.Empty(t, provider.sent)
}

func TestTemplateTestSendSurfacesProviderError(t *testing.T) {
	provider := &stubProvider{channel: models.ChannelEmail, err: errors.New("smtp refused")}
	svc, mockRepo := newTemplateServiceWithProvider(provider)
	mockRepo.On("GetTemplate", mock.Anything, "tenant-1", "tmpl-1").Return(sampleTemplate(), nil)

	err := svc.TestSend(context.Background(), "tenant-1", "tmpl-1", "admin@example.com", nil)

	if assert.Error(t, err) {
		assert.Contains(t, err.Error(), "test send failed")
	}
}

// A provider can return no error but still report failure; that must not be
// reported to the admin as a success.
func TestTemplateTestSendSurfacesUnsuccessfulResult(t *testing.T) {
	provider := &stubProvider{
		channel: models.ChannelEmail,
		result:  &ProviderResult{ProviderName: "stub", Success: false, Error: "mailbox full"},
	}
	svc, mockRepo := newTemplateServiceWithProvider(provider)
	mockRepo.On("GetTemplate", mock.Anything, "tenant-1", "tmpl-1").Return(sampleTemplate(), nil)

	err := svc.TestSend(context.Background(), "tenant-1", "tmpl-1", "admin@example.com", nil)

	if assert.Error(t, err) {
		assert.Contains(t, err.Error(), "mailbox full")
	}
}

// ------------------------------------------------------------ RenderTemplate

func TestRenderTemplate(t *testing.T) {
	subject, body, err := RenderTemplate(&models.NotificationTemplate{
		SubjectTemplate: "Order {{.OrderID}}",
		BodyTemplate:    "<p>Total {{.Total}}</p>",
	}, map[string]interface{}{"OrderID": "order-1", "Total": "৳1,000.00"})

	assert.NoError(t, err)
	assert.Equal(t, "Order order-1", subject)
	assert.Equal(t, "<p>Total ৳1,000.00</p>", body)
}

// An empty subject is legitimate for SMS/push templates.
func TestRenderTemplateAllowsEmptyParts(t *testing.T) {
	subject, body, err := RenderTemplate(&models.NotificationTemplate{BodyTemplate: "just a body"}, nil)

	assert.NoError(t, err)
	assert.Empty(t, subject)
	assert.Equal(t, "just a body", body)
}

// missingkey=zero keeps an unknown placeholder from being a hard error, so
// rendering still succeeds. Because vars is a map[string]interface{}, the
// "zero" it substitutes is a nil interface, which prints as "<no value>" —
// this pins that behaviour so a change to it is a deliberate one. In practice
// MergeSampleVars backfills the standard placeholders before this runs, so
// only a placeholder the admin invented reaches this path.
func TestRenderTemplateHandlesMissingVars(t *testing.T) {
	subject, _, err := RenderTemplate(&models.NotificationTemplate{
		SubjectTemplate: "Hi {{.NotSupplied}}",
		BodyTemplate:    "b",
	}, map[string]interface{}{})

	assert.NoError(t, err, "an unknown placeholder must not fail the render")
	assert.Equal(t, "Hi <no value>", subject)
}

// The standard placeholders are backfilled, so a preview of a real template
// never shows the marker.
func TestRenderTemplateWithMergedVarsHasNoPlaceholderMarkers(t *testing.T) {
	vars := MergeSampleVars(models.TypeOrderConfirmation, nil)

	subject, body, err := RenderTemplate(&models.NotificationTemplate{
		SubjectTemplate: "Order {{.OrderID}} for {{.CustomerName}}",
		BodyTemplate:    "<p>{{.TenantName}} — {{.Total}}</p>",
	}, vars)

	assert.NoError(t, err)
	assert.NotContains(t, subject, "<no value>")
	assert.NotContains(t, body, "<no value>")
}

// Body copy is authored as raw HTML by admins, so text/template must leave it
// alone rather than double-encoding it.
func TestRenderTemplateDoesNotEscapeAuthoredHTML(t *testing.T) {
	_, body, err := RenderTemplate(&models.NotificationTemplate{
		BodyTemplate: `<a href="{{.URL}}">Click</a>`,
	}, map[string]interface{}{"URL": "https://shop.example.com/a?b=c&d=e"})

	assert.NoError(t, err)
	assert.Contains(t, body, "https://shop.example.com/a?b=c&d=e")
	assert.NotContains(t, body, "&amp;amp;")
}

func TestRenderTemplateReportsBadTemplates(t *testing.T) {
	tests := []struct {
		name     string
		template *models.NotificationTemplate
		wantErr  string
	}{
		{
			name:     "subject fails to parse",
			template: &models.NotificationTemplate{SubjectTemplate: "{{.Unclosed", BodyTemplate: "b"},
			wantErr:  "subject",
		},
		{
			name:     "body fails to parse",
			template: &models.NotificationTemplate{SubjectTemplate: "s", BodyTemplate: "{{range}}"},
			wantErr:  "body",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, _, err := RenderTemplate(tt.template, nil)

			if assert.Error(t, err) {
				assert.Contains(t, err.Error(), tt.wantErr)
			}
		})
	}
}

func TestMergeSampleVarsLayersOverridesOnDefaults(t *testing.T) {
	got := MergeSampleVars(models.TypeOrderConfirmation, map[string]interface{}{"OrderID": "order-99"})

	assert.Equal(t, "order-99", got["OrderID"], "the override should win")
	// The always-available placeholders survive so a preview is never blank.
	assert.NotEmpty(t, got["TenantName"])
	assert.NotEmpty(t, got["CustomerName"])
}

func TestMergeSampleVarsWithoutOverrides(t *testing.T) {
	got := MergeSampleVars(models.TypeWelcome, nil)

	assert.NotEmpty(t, got["UserName"])
	assert.NotEmpty(t, got["TenantName"])
}
