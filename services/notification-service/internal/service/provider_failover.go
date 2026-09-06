package service

import (
	"fmt"
	"strings"

	"github.com/ecommerce/notification-service/internal/models"
	"github.com/sirupsen/logrus"
)

// FailoverProvider sends through an ordered list of providers, moving to the
// next one when a send fails. It satisfies NotificationProvider itself, so it
// drops into the channel map exactly where a single provider used to sit and
// nothing downstream needs to know there is more than one.
//
// This matters most for the transactional mail this service exists to send: a
// password-reset or order-confirmation email that silently fails because one
// vendor is rate-limiting or having an outage is a user-visible failure with
// no recovery path. Free vendor tiers make this concrete — a 300/day cap is
// reached quietly, and the next provider in the chain absorbs the overflow.
//
// Ordering is significant: put the provider you want to carry normal traffic
// first. A simulated provider placed last turns "everything is down" into a
// logged message rather than a hard failure, which is right for development
// and wrong for production — so it is opt-in via configuration.
type FailoverProvider struct {
	channel   models.Channel
	providers []NotificationProvider
	logger    *logrus.Logger
}

// NewFailoverProvider builds a chain for one channel. Providers are tried in
// the order given. A single provider is returned as-is, since wrapping it adds
// nothing.
func NewFailoverProvider(channel models.Channel, logger *logrus.Logger, providers ...NotificationProvider) NotificationProvider {
	switch len(providers) {
	case 0:
		return nil
	case 1:
		return providers[0]
	}
	return &FailoverProvider{channel: channel, providers: providers, logger: logger}
}

func (p *FailoverProvider) Channel() models.Channel {
	return p.channel
}

// Len reports how many providers are in the chain.
func (p *FailoverProvider) Len() int {
	return len(p.providers)
}

// Send walks the chain and returns the first success. A provider can fail two
// ways — a returned error, or a result with Success false — and both are
// treated as "try the next one".
func (p *FailoverProvider) Send(notification *models.Notification) (*ProviderResult, error) {
	failures := make([]string, 0, len(p.providers))

	for i, provider := range p.providers {
		result, err := provider.Send(notification)

		switch {
		case err != nil:
			failures = append(failures, fmt.Sprintf("%s: %v", providerLabel(provider, i), err))
		case result == nil:
			failures = append(failures, fmt.Sprintf("%s: provider returned no result", providerLabel(provider, i)))
		case !result.Success:
			label := result.ProviderName
			if label == "" {
				label = providerLabel(provider, i)
			}
			failures = append(failures, fmt.Sprintf("%s: %s", label, result.Error))
		default:
			// Delivered. Note which hop it took when it was not the primary,
			// because a chain quietly running on its backup is worth knowing
			// about before the backup also runs out.
			if i > 0 {
				p.logger.WithFields(logrus.Fields{
					"channel":           string(p.channel),
					"provider":          result.ProviderName,
					"chain_position":    i + 1,
					"failed_before_it":  strings.Join(failures, "; "),
					"recipient_present": notification.Recipient != "",
				}).Warn("Notification delivered by a fallback provider")
			}
			return result, nil
		}

		p.logger.WithFields(logrus.Fields{
			"channel":        string(p.channel),
			"provider":       providerLabel(provider, i),
			"chain_position": i + 1,
			"chain_length":   len(p.providers),
		}).Warn("Notification provider failed; trying the next one")
	}

	// Everything failed. Report every reason: with a chain, a single error
	// string hides which vendors were tried and why each refused.
	combined := strings.Join(failures, "; ")
	p.logger.WithFields(logrus.Fields{
		"channel":  string(p.channel),
		"attempts": len(p.providers),
		"failures": combined,
	}).Error("All notification providers failed")

	return &ProviderResult{
		ProviderName: "failover",
		Success:      false,
		Error:        fmt.Sprintf("all %d providers failed: %s", len(p.providers), combined),
	}, nil
}

// providerLabel prefers a provider's self-reported name, falling back to its
// position so log lines stay attributable either way.
func providerLabel(provider NotificationProvider, index int) string {
	if named, ok := provider.(interface{ Name() string }); ok {
		if n := named.Name(); n != "" {
			return n
		}
	}
	return fmt.Sprintf("provider-%d", index+1)
}
