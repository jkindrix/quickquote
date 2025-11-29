package service

import (
	"context"
	"sync"

	"github.com/google/uuid"

	"github.com/jkindrix/quickquote/internal/domain"
	"github.com/jkindrix/quickquote/internal/repository"
)

// MockCallRepository is a mock implementation of domain.CallRepository for testing.
type MockCallRepository struct {
	mu           sync.RWMutex
	calls        map[uuid.UUID]*domain.Call
	byProviderID map[string]*domain.Call

	// For tracking method calls
	CreateCalls         int
	UpdateCalls         int
	GetByIDCalls        int
	GetByProviderIDCalls int
	ListCalls           int
	CountCalls          int

	// For injecting errors
	CreateError         error
	UpdateError         error
	GetByIDError        error
	GetByProviderIDError error
	ListError           error
	CountError          error
}

func NewMockCallRepository() *MockCallRepository {
	return &MockCallRepository{
		calls:        make(map[uuid.UUID]*domain.Call),
		byProviderID: make(map[string]*domain.Call),
	}
}

func (m *MockCallRepository) Create(ctx context.Context, call *domain.Call) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.CreateCalls++
	if m.CreateError != nil {
		return m.CreateError
	}
	m.calls[call.ID] = call
	m.byProviderID[call.ProviderCallID] = call
	return nil
}

func (m *MockCallRepository) GetByID(ctx context.Context, id uuid.UUID) (*domain.Call, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	m.GetByIDCalls++
	if m.GetByIDError != nil {
		return nil, m.GetByIDError
	}
	if call, ok := m.calls[id]; ok {
		return call, nil
	}
	return nil, repository.ErrNotFound
}

func (m *MockCallRepository) GetByProviderCallID(ctx context.Context, providerCallID string) (*domain.Call, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	m.GetByProviderIDCalls++
	if m.GetByProviderIDError != nil {
		return nil, m.GetByProviderIDError
	}
	if call, ok := m.byProviderID[providerCallID]; ok {
		return call, nil
	}
	return nil, repository.ErrNotFound
}

func (m *MockCallRepository) Update(ctx context.Context, call *domain.Call) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.UpdateCalls++
	if m.UpdateError != nil {
		return m.UpdateError
	}
	if _, ok := m.calls[call.ID]; !ok {
		return repository.ErrNotFound
	}
	m.calls[call.ID] = call
	m.byProviderID[call.ProviderCallID] = call
	return nil
}

func (m *MockCallRepository) List(ctx context.Context, limit, offset int) ([]*domain.Call, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	m.ListCalls++
	if m.ListError != nil {
		return nil, m.ListError
	}
	var result []*domain.Call
	for _, call := range m.calls {
		result = append(result, call)
	}
	// Apply pagination
	if offset >= len(result) {
		return []*domain.Call{}, nil
	}
	end := offset + limit
	if end > len(result) {
		end = len(result)
	}
	return result[offset:end], nil
}

func (m *MockCallRepository) Count(ctx context.Context) (int, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	m.CountCalls++
	if m.CountError != nil {
		return 0, m.CountError
	}
	return len(m.calls), nil
}

func (m *MockCallRepository) ListByStatus(ctx context.Context, status domain.CallStatus, limit, offset int) ([]*domain.Call, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	var result []*domain.Call
	for _, call := range m.calls {
		if call.Status == status {
			result = append(result, call)
		}
	}
	if offset >= len(result) {
		return []*domain.Call{}, nil
	}
	end := offset + limit
	if end > len(result) {
		end = len(result)
	}
	return result[offset:end], nil
}

// MockQuoteGenerator is a mock implementation of QuoteGenerator for testing.
type MockQuoteGenerator struct {
	GenerateQuoteCalls int
	GenerateQuoteError error
	GeneratedQuote     string
}

func NewMockQuoteGenerator() *MockQuoteGenerator {
	return &MockQuoteGenerator{
		GeneratedQuote: "Test generated quote summary",
	}
}

func (m *MockQuoteGenerator) GenerateQuote(ctx context.Context, transcript string, extractedData *domain.ExtractedData) (string, error) {
	m.GenerateQuoteCalls++
	if m.GenerateQuoteError != nil {
		return "", m.GenerateQuoteError
	}
	return m.GeneratedQuote, nil
}

// MockUserRepository is a mock implementation of domain.UserRepository for testing.
type MockUserRepository struct {
	mu    sync.RWMutex
	users map[uuid.UUID]*domain.User
	byEmail map[string]*domain.User

	CreateCalls     int
	GetByIDCalls    int
	GetByEmailCalls int
	UpdateCalls     int

	CreateError     error
	GetByIDError    error
	GetByEmailError error
	UpdateError     error
}

func NewMockUserRepository() *MockUserRepository {
	return &MockUserRepository{
		users:   make(map[uuid.UUID]*domain.User),
		byEmail: make(map[string]*domain.User),
	}
}

func (m *MockUserRepository) Create(ctx context.Context, user *domain.User) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.CreateCalls++
	if m.CreateError != nil {
		return m.CreateError
	}
	m.users[user.ID] = user
	m.byEmail[user.Email] = user
	return nil
}

func (m *MockUserRepository) GetByID(ctx context.Context, id uuid.UUID) (*domain.User, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	m.GetByIDCalls++
	if m.GetByIDError != nil {
		return nil, m.GetByIDError
	}
	if user, ok := m.users[id]; ok {
		return user, nil
	}
	return nil, repository.ErrNotFound
}

func (m *MockUserRepository) GetByEmail(ctx context.Context, email string) (*domain.User, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	m.GetByEmailCalls++
	if m.GetByEmailError != nil {
		return nil, m.GetByEmailError
	}
	if user, ok := m.byEmail[email]; ok {
		return user, nil
	}
	return nil, repository.ErrNotFound
}

func (m *MockUserRepository) Update(ctx context.Context, user *domain.User) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.UpdateCalls++
	if m.UpdateError != nil {
		return m.UpdateError
	}
	if _, ok := m.users[user.ID]; !ok {
		return repository.ErrNotFound
	}
	m.users[user.ID] = user
	m.byEmail[user.Email] = user
	return nil
}

func (m *MockUserRepository) Count(ctx context.Context) (int64, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return int64(len(m.users)), nil
}

// MockSessionRepository is a mock implementation of domain.SessionRepository for testing.
type MockSessionRepository struct {
	mu       sync.RWMutex
	sessions map[string]*domain.Session
	byUserID map[uuid.UUID][]*domain.Session

	CreateCalls         int
	GetByTokenCalls     int
	UpdateCalls         int
	DeleteCalls         int
	DeleteExpiredCalls  int
	DeleteByUserIDCalls int

	CreateError         error
	GetByTokenError     error
	UpdateError         error
	DeleteError         error
	DeleteExpiredError  error
	DeleteByUserIDError error
}

func NewMockSessionRepository() *MockSessionRepository {
	return &MockSessionRepository{
		sessions: make(map[string]*domain.Session),
		byUserID: make(map[uuid.UUID][]*domain.Session),
	}
}

func (m *MockSessionRepository) Create(ctx context.Context, session *domain.Session) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.CreateCalls++
	if m.CreateError != nil {
		return m.CreateError
	}
	m.sessions[session.Token] = session
	m.byUserID[session.UserID] = append(m.byUserID[session.UserID], session)
	return nil
}

func (m *MockSessionRepository) GetByToken(ctx context.Context, token string) (*domain.Session, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	m.GetByTokenCalls++
	if m.GetByTokenError != nil {
		return nil, m.GetByTokenError
	}
	if session, ok := m.sessions[token]; ok {
		return session, nil
	}
	return nil, repository.ErrNotFound
}

func (m *MockSessionRepository) Update(ctx context.Context, session *domain.Session) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.UpdateCalls++
	if m.UpdateError != nil {
		return m.UpdateError
	}
	// Find old token for this session ID and remove it
	for token, s := range m.sessions {
		if s.ID == session.ID {
			delete(m.sessions, token)
			break
		}
	}
	// Add with new token
	m.sessions[session.Token] = session
	return nil
}

func (m *MockSessionRepository) Delete(ctx context.Context, token string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.DeleteCalls++
	if m.DeleteError != nil {
		return m.DeleteError
	}
	delete(m.sessions, token)
	return nil
}

func (m *MockSessionRepository) DeleteExpired(ctx context.Context) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.DeleteExpiredCalls++
	if m.DeleteExpiredError != nil {
		return m.DeleteExpiredError
	}
	// Simplified: just return success
	return nil
}

func (m *MockSessionRepository) DeleteByUserID(ctx context.Context, userID uuid.UUID) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.DeleteByUserIDCalls++
	if m.DeleteByUserIDError != nil {
		return m.DeleteByUserIDError
	}
	for _, session := range m.byUserID[userID] {
		delete(m.sessions, session.Token)
	}
	delete(m.byUserID, userID)
	return nil
}
