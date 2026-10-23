package proxy

import (
	"context"
	"fmt"
	"time"

	"github.com/h-varmazyar/gatec/internal/models"
	"github.com/h-varmazyar/gatec/internal/repositories"
	"github.com/h-varmazyar/gatec/pkg/paginator"
)

// Service provides proxy management functionality
type Service struct {
	proxyRepo           *repositories.ProxyRepository
	proxyAssignmentRepo *repositories.ProxyAssignmentRepository
}

// NewService creates a new proxy service
func NewService(
	proxyRepo *repositories.ProxyRepository,
	proxyAssignmentRepo *repositories.ProxyAssignmentRepository,
) *Service {
	return &Service{
		proxyRepo:           proxyRepo,
		proxyAssignmentRepo: proxyAssignmentRepo,
	}
}

// --- CRUD Operations ---

// Create creates a new proxy
func (s *Service) Create(ctx context.Context, proxy *models.Proxy) error {
	if proxy.Server == "" {
		return fmt.Errorf("server is required")
	}
	if proxy.Port <= 0 {
		return fmt.Errorf("port must be positive")
	}
	if proxy.Type == "" {
		proxy.Type = models.ProxyTypeMTProto
	}
	if proxy.Status == "" {
		proxy.Status = models.ProxyStatusUnknown
	}

	// Check for duplicate proxy (same server and port)
	existingProxies, err := s.proxyRepo.ListAll(ctx)
	if err != nil {
		return fmt.Errorf("failed to check existing proxies: %w", err)
	}

	for _, existing := range existingProxies {
		if existing.Server == proxy.Server && existing.Port == proxy.Port {
			return fmt.Errorf("proxy with server %s and port %d already exists", proxy.Server, proxy.Port)
		}
	}

	return s.proxyRepo.Create(ctx, proxy)
}

// GetByID returns a proxy by ID
func (s *Service) GetByID(ctx context.Context, id uint) (*models.Proxy, error) {
	return s.proxyRepo.GetByID(ctx, id)
}

// Update updates an existing proxy
func (s *Service) Update(ctx context.Context, proxy *models.Proxy) error {
	existing, err := s.proxyRepo.GetByID(ctx, proxy.ID)
	if err != nil {
		return fmt.Errorf("proxy not found: %w", err)
	}

	// Only allow updating certain fields
	existing.Server = proxy.Server
	existing.Port = proxy.Port
	existing.Type = proxy.Type
	existing.Secret = proxy.Secret
	existing.Username = proxy.Username
	existing.Password = proxy.Password
	existing.HTTPOnly = proxy.HTTPOnly
	existing.IsEnable = proxy.IsEnable

	return s.proxyRepo.Create(ctx, existing)
}

// Delete deletes a proxy
func (s *Service) Delete(ctx context.Context, id uint) error {
	// Check if proxy has active assignments
	assignments, err := s.proxyAssignmentRepo.ListActiveAssignments(ctx)
	if err != nil {
		return fmt.Errorf("failed to check assignments: %w", err)
	}

	for _, assignment := range assignments {
		if assignment.ProxyID == id {
			return fmt.Errorf("cannot delete proxy %d: has active assignments", id)
		}
	}

	// Soft delete (through GORM Model)
	// We just disable it
	proxy, err := s.proxyRepo.GetByID(ctx, id)
	if err != nil {
		return fmt.Errorf("proxy not found: %w", err)
	}

	proxy.IsEnable = false
	return s.proxyRepo.Create(ctx, proxy)
}

// List returns paginated list of proxies
func (s *Service) List(ctx context.Context, pg paginator.Params) ([]models.Proxy, int64, error) {
	proxies, err := s.proxyRepo.ListAll(ctx)
	if err != nil {
		return nil, 0, err
	}

	// Manual pagination since repo doesn't support it yet
	total := int64(len(proxies))
	start := int64((pg.Page - 1) * pg.Limit())
	end := int64(pg.Page * pg.Limit())

	if start >= total {
		return []models.Proxy{}, total, nil
	}
	if end > total {
		end = total
	}

	return proxies[start:end], total, nil
}

// ListActive returns all active (enabled + active status) proxies
func (s *Service) ListActive(ctx context.Context) ([]models.Proxy, error) {
	return s.proxyRepo.ListActive(ctx)
}

// ListEnabled returns all enabled proxies regardless of status
func (s *Service) ListEnabled(ctx context.Context) ([]models.Proxy, error) {
	return s.proxyRepo.ListEnabled(ctx)
}

// GetStats returns proxy statistics
func (s *Service) GetStats(ctx context.Context) (*ProxyStats, error) {
	allProxies, err := s.proxyRepo.ListAll(ctx)
	if err != nil {
		return nil, err
	}

	enabledProxies, err := s.proxyRepo.ListEnabled(ctx)
	if err != nil {
		return nil, err
	}

	activeProxies, err := s.proxyRepo.ListActive(ctx)
	if err != nil {
		return nil, err
	}

	assignments, err := s.proxyAssignmentRepo.ListActiveAssignments(ctx)
	if err != nil {
		return nil, err
	}

	stats := &ProxyStats{
		Total:       len(allProxies),
		Enabled:     len(enabledProxies),
		Active:      len(activeProxies),
		Assignments: len(assignments),
	}

	return stats, nil
}

// ToggleStatus toggles the enable status of a proxy
func (s *Service) ToggleStatus(ctx context.Context, id uint, enable bool) error {
	proxy, err := s.proxyRepo.GetByID(ctx, id)
	if err != nil {
		return fmt.Errorf("proxy not found: %w", err)
	}

	proxy.IsEnable = enable
	if !enable {
		proxy.Status = models.ProxyStatusInactive
	}

	return s.proxyRepo.Create(ctx, proxy)
}

// ForceCheck triggers an immediate health check for a specific proxy
func (s *Service) ForceCheck(ctx context.Context, id uint) error {
	proxy, err := s.proxyRepo.GetByID(ctx, id)
	if err != nil {
		return fmt.Errorf("proxy not found: %w", err)
	}

	// Just update the last_checked_at timestamp for now
	// Actual checking is done by the ProxyHealthChecker worker
	now := time.Now()
	proxy.LastCheckedAt = &now
	return s.proxyRepo.Create(ctx, proxy)
}

// --- Types ---

// ProxyStats contains statistics about proxies
type ProxyStats struct {
	Total       int `json:"total"`
	Enabled     int `json:"enabled"`
	Active      int `json:"active"`
	Assignments int `json:"assignments"`
}
