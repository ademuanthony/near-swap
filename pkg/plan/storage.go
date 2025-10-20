package plan

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
)

const (
	DefaultStorageFileName = ".near-swap-plans.json"
)

// Storage handles persistence of trading plans
type Storage struct {
	filePath string
	mu       sync.RWMutex
	plans    map[string]*TradingPlan
}

// PlanStorage represents the JSON structure for storage
type PlanStorage struct {
	Plans map[string]*TradingPlan `json:"plans"`
}

// NewStorage creates a new storage instance
func NewStorage(filePath string) (*Storage, error) {
	if filePath == "" {
		// Default to home directory
		home, err := os.UserHomeDir()
		if err != nil {
			return nil, fmt.Errorf("failed to get home directory: %w", err)
		}
		filePath = filepath.Join(home, DefaultStorageFileName)
	}

	storage := &Storage{
		filePath: filePath,
		plans:    make(map[string]*TradingPlan),
	}

	// Load existing plans if file exists
	if err := storage.load(); err != nil {
		// If file doesn't exist, that's okay - we'll create it on first save
		if !os.IsNotExist(err) {
			return nil, fmt.Errorf("failed to load plans: %w", err)
		}
	}

	return storage, nil
}

// load reads plans from the storage file
func (s *Storage) load() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	data, err := os.ReadFile(s.filePath)
	if err != nil {
		return err
	}

	var planStorage PlanStorage
	if err := json.Unmarshal(data, &planStorage); err != nil {
		return fmt.Errorf("failed to unmarshal plans: %w", err)
	}

	s.plans = planStorage.Plans
	if s.plans == nil {
		s.plans = make(map[string]*TradingPlan)
	}

	return nil
}

// save writes plans to the storage file
func (s *Storage) save() error {
	s.mu.RLock()
	defer s.mu.RUnlock()

	planStorage := PlanStorage{
		Plans: s.plans,
	}

	data, err := json.MarshalIndent(planStorage, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal plans: %w", err)
	}

	// Ensure directory exists
	dir := filepath.Dir(s.filePath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create directory: %w", err)
	}

	// Write to temporary file first, then rename for atomic write
	tempFile := s.filePath + ".tmp"
	if err := os.WriteFile(tempFile, data, 0600); err != nil {
		return fmt.Errorf("failed to write plans: %w", err)
	}

	if err := os.Rename(tempFile, s.filePath); err != nil {
		return fmt.Errorf("failed to rename temp file: %w", err)
	}

	return nil
}

// Create adds a new plan to storage
func (s *Storage) Create(plan *TradingPlan) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, exists := s.plans[plan.Name]; exists {
		return fmt.Errorf("plan '%s' already exists", plan.Name)
	}

	s.plans[plan.Name] = plan

	// Release lock before saving
	s.mu.Unlock()
	err := s.save()
	s.mu.Lock()

	return err
}

// Get retrieves a plan by name
func (s *Storage) Get(name string) (*TradingPlan, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	plan, exists := s.plans[name]
	if !exists {
		return nil, fmt.Errorf("plan '%s' not found", name)
	}

	return plan, nil
}

// Update modifies an existing plan
func (s *Storage) Update(plan *TradingPlan) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, exists := s.plans[plan.Name]; !exists {
		return fmt.Errorf("plan '%s' not found", plan.Name)
	}

	s.plans[plan.Name] = plan

	// Release lock before saving
	s.mu.Unlock()
	err := s.save()
	s.mu.Lock()

	return err
}

// Delete removes a plan from storage
func (s *Storage) Delete(name string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, exists := s.plans[name]; !exists {
		return fmt.Errorf("plan '%s' not found", name)
	}

	delete(s.plans, name)

	// Release lock before saving
	s.mu.Unlock()
	err := s.save()
	s.mu.Lock()

	return err
}

// List returns all plans
func (s *Storage) List() []*TradingPlan {
	s.mu.RLock()
	defer s.mu.RUnlock()

	plans := make([]*TradingPlan, 0, len(s.plans))
	for _, plan := range s.plans {
		plans = append(plans, plan)
	}

	return plans
}

// ListByStatus returns plans filtered by status
func (s *Storage) ListByStatus(status PlanStatus) []*TradingPlan {
	s.mu.RLock()
	defer s.mu.RUnlock()

	plans := make([]*TradingPlan, 0)
	for _, plan := range s.plans {
		if plan.Status == status {
			plans = append(plans, plan)
		}
	}

	return plans
}

// Exists checks if a plan with the given name exists
func (s *Storage) Exists(name string) bool {
	s.mu.RLock()
	defer s.mu.RUnlock()

	_, exists := s.plans[name]
	return exists
}

// Count returns the total number of plans
func (s *Storage) Count() int {
	s.mu.RLock()
	defer s.mu.RUnlock()

	return len(s.plans)
}

// GetFilePath returns the storage file path
func (s *Storage) GetFilePath() string {
	return s.filePath
}
