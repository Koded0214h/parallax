package baseline

import (
	"math"
	"sync"
	"time"
)

// Beneficiary represents a known payee for a user.
type Beneficiary struct {
	BeneficiaryID string    `json:"beneficiary_id"`
	Name          string    `json:"name"`
	AccountNumber string    `json:"account_number"`
	BankName      string    `json:"bank_name"`
	CreatedAt     time.Time `json:"created_at"`
	TransferCount int       `json:"transfer_count"`
	TotalSent     float64   `json:"total_sent"`
}

// Device represents a known hardware/browser fingerprint.
type Device struct {
	DeviceID    string    `json:"device_id"`
	DeviceModel string    `json:"device_model"`
	UserAgent   string    `json:"user_agent"`
	FirstSeen   time.Time `json:"first_seen"`
	LastSeen    time.Time `json:"last_seen"`
	UseCount    int       `json:"use_count"`
}

// UserBaseline defines historical behavioral metrics for a synthetic user (prd.md §11).
type UserBaseline struct {
	UserID string `json:"user_id"`

	// Financial baseline
	TypicalAmountMin    float64 `json:"typical_amount_min"`
	TypicalAmountMax    float64 `json:"typical_amount_max"`
	TypicalAmountMean   float64 `json:"typical_amount_mean"`
	TypicalAmountStdDev float64 `json:"typical_amount_std_dev"`
	TotalTransfers      int     `json:"total_transfers"`

	// Temporal baseline (hours 0-23 in local/WAT timezone)
	TypicalStartHour int `json:"typical_start_hour"`
	TypicalEndHour   int `json:"typical_end_hour"`

	// Known entities
	KnownBeneficiaries map[string]Beneficiary `json:"known_beneficiaries"`
	KnownDevices       map[string]Device      `json:"known_devices"`

	// Interaction dynamics
	TypicalSessionDurationSec float64 `json:"typical_session_duration_sec"`
	TypicalEventsPerMinute    float64 `json:"typical_events_per_minute"`

	// Velocity limits
	MaxHourlyTransfers int     `json:"max_hourly_transfers"`
	MaxDailyAmount     float64 `json:"max_daily_amount"`

	// Security state timestamps
	LastPasswordChange time.Time `json:"last_password_change"`
	LastPINChange      time.Time `json:"last_pin_change"`
}

// Clone performs a deep copy of the baseline.
func (b UserBaseline) Clone() UserBaseline {
	cp := b
	cp.KnownBeneficiaries = make(map[string]Beneficiary, len(b.KnownBeneficiaries))
	for k, v := range b.KnownBeneficiaries {
		cp.KnownBeneficiaries[k] = v
	}
	cp.KnownDevices = make(map[string]Device, len(b.KnownDevices))
	for k, v := range b.KnownDevices {
		cp.KnownDevices[k] = v
	}
	return cp
}

// AmountDeviation computes how many standard deviations or multiples the amount is above normal.
func (b UserBaseline) AmountDeviation(amount float64) (ratio float64, zScore float64) {
	if b.TypicalAmountMean <= 0 {
		return 1.0, 0.0
	}
	ratio = amount / b.TypicalAmountMean
	if b.TypicalAmountStdDev > 0 {
		zScore = (amount - b.TypicalAmountMean) / b.TypicalAmountStdDev
	}
	return ratio, zScore
}

// IsHourTypical reports whether the given hour (0-23) falls within typical active hours.
func (b UserBaseline) IsHourTypical(hour int) bool {
	if b.TypicalStartHour <= b.TypicalEndHour {
		return hour >= b.TypicalStartHour && hour <= b.TypicalEndHour
	}
	// Handles overnight wrap-around (e.g., 22:00 to 06:00)
	return hour >= b.TypicalStartHour || hour <= b.TypicalEndHour
}

// Store holds thread-safe in-memory customer baselines.
type Store struct {
	mu        sync.RWMutex
	baselines map[string]UserBaseline
}

// NewStore initializes an empty baseline store.
func NewStore() *Store {
	s := &Store{
		baselines: make(map[string]UserBaseline),
	}
	s.seedDefaults()
	return s
}

// Get retrieves a copy of a user baseline.
func (s *Store) Get(userID string) (UserBaseline, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	b, ok := s.baselines[userID]
	if !ok {
		return UserBaseline{}, false
	}
	return b.Clone(), true
}

// Set saves or updates a user baseline.
func (s *Store) Set(b UserBaseline) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.baselines[b.UserID] = b.Clone()
}

// GetOrCreate returns the existing baseline or creates a reasonable default.
func (s *Store) GetOrCreate(userID string) UserBaseline {
	s.mu.Lock()
	defer s.mu.Unlock()
	if b, ok := s.baselines[userID]; ok {
		return b.Clone()
	}
	created := DefaultBaseline(userID)
	s.baselines[userID] = created.Clone()
	return created
}

// RecordTransfer updates baseline statistics following a successful transfer.
func (s *Store) RecordTransfer(userID string, amount float64, beneficiaryID, deviceID string, t time.Time) {
	s.mu.Lock()
	defer s.mu.Unlock()

	b, ok := s.baselines[userID]
	if !ok {
		b = DefaultBaseline(userID)
	}

	// Update running mean and variance using Welford's algorithm
	b.TotalTransfers++
	n := float64(b.TotalTransfers)
	delta := amount - b.TypicalAmountMean
	b.TypicalAmountMean += delta / n
	delta2 := amount - b.TypicalAmountMean
	if n > 1 {
		variance := (b.TypicalAmountStdDev*b.TypicalAmountStdDev*(n-1) + delta*delta2) / n
		b.TypicalAmountStdDev = math.Sqrt(variance)
	}
	if amount < b.TypicalAmountMin || b.TypicalAmountMin == 0 {
		b.TypicalAmountMin = amount
	}
	if amount > b.TypicalAmountMax {
		b.TypicalAmountMax = amount
	}

	// Update beneficiary record
	if beneficiaryID != "" {
		if ben, exists := b.KnownBeneficiaries[beneficiaryID]; exists {
			ben.TransferCount++
			ben.TotalSent += amount
			b.KnownBeneficiaries[beneficiaryID] = ben
		} else {
			b.KnownBeneficiaries[beneficiaryID] = Beneficiary{
				BeneficiaryID: beneficiaryID,
				Name:          "Beneficiary " + beneficiaryID,
				CreatedAt:     t,
				TransferCount: 1,
				TotalSent:     amount,
			}
		}
	}

	// Update device record
	if deviceID != "" {
		if dev, exists := b.KnownDevices[deviceID]; exists {
			dev.LastSeen = t
			dev.UseCount++
			b.KnownDevices[deviceID] = dev
		} else {
			b.KnownDevices[deviceID] = Device{
				DeviceID:  deviceID,
				FirstSeen: t,
				LastSeen:  t,
				UseCount:  1,
			}
		}
	}

	s.baselines[userID] = b
}

// Reset clears all baselines and re-seeds defaults.
func (s *Store) Reset() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.baselines = make(map[string]UserBaseline)
	s.seedDefaultsLocked()
}

// DefaultBaseline creates a synthetic baseline for a standard user (prd.md §11).
func DefaultBaseline(userID string) UserBaseline {
	now := time.Now().UTC()
	return UserBaseline{
		UserID:                    userID,
		TypicalAmountMin:          10000.0,
		TypicalAmountMax:          75000.0,
		TypicalAmountMean:         35000.0,
		TypicalAmountStdDev:       12500.0,
		TotalTransfers:            48,
		TypicalStartHour:          8,
		TypicalEndHour:            21,
		TypicalSessionDurationSec: 45.0,
		TypicalEventsPerMinute:    12.0,
		MaxHourlyTransfers:        4,
		MaxDailyAmount:            250000.0,
		LastPasswordChange:        now.Add(-60 * 24 * time.Hour),
		LastPINChange:             now.Add(-90 * 24 * time.Hour),
		KnownBeneficiaries: map[string]Beneficiary{
			"ben_mother": {
				BeneficiaryID: "ben_mother",
				Name:          "Amina Adeleke",
				AccountNumber: "0123456789",
				BankName:      "First Bank of Nigeria",
				CreatedAt:     now.Add(-180 * 24 * time.Hour),
				TransferCount: 24,
				TotalSent:     480000.0,
			},
			"ben_landlord": {
				BeneficiaryID: "ben_landlord",
				Name:          "Babajide Properties",
				AccountNumber: "2233445566",
				BankName:      "GTBank",
				CreatedAt:     now.Add(-120 * 24 * time.Hour),
				TransferCount: 6,
				TotalSent:     900000.0,
			},
			"ben_groceries": {
				BeneficiaryID: "ben_groceries",
				Name:          "Kano Fresh Mart",
				AccountNumber: "3344556677",
				BankName:      "Access Bank",
				CreatedAt:     now.Add(-60 * 24 * time.Hour),
				TransferCount: 15,
				TotalSent:     150000.0,
			},
		},
		KnownDevices: map[string]Device{
			"device_primary": {
				DeviceID:    "device_primary",
				DeviceModel: "Samsung Galaxy S22",
				UserAgent:   "ParallaxApp/1.0 (Android 13; SM-S901B)",
				FirstSeen:   now.Add(-180 * 24 * time.Hour),
				LastSeen:    now.Add(-2 * time.Hour),
				UseCount:    320,
			},
		},
	}
}

func (s *Store) seedDefaults() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.seedDefaultsLocked()
}

func (s *Store) seedDefaultsLocked() {
	userIDs := []string{"user_001", "user_002", "user_demo", "user_toheeb", "user_fiope"}
	for _, id := range userIDs {
		s.baselines[id] = DefaultBaseline(id)
	}
}
