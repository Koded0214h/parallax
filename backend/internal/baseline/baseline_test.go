package baseline

import (
	"testing"
	"time"
)

func TestBaselineAmountDeviation(t *testing.T) {
	b := DefaultBaseline("user_001")
	b.TypicalAmountMean = 50000.0
	b.TypicalAmountStdDev = 10000.0

	ratio, zScore := b.AmountDeviation(150000.0)
	if ratio != 3.0 {
		t.Fatalf("expected ratio 3.0, got %f", ratio)
	}
	if zScore != 10.0 {
		t.Fatalf("expected zScore 10.0, got %f", zScore)
	}
}

func TestBaselineHourTypical(t *testing.T) {
	b := DefaultBaseline("user_001")
	b.TypicalStartHour = 8
	b.TypicalEndHour = 20

	if !b.IsHourTypical(14) {
		t.Errorf("expected 14:00 to be typical")
	}
	if b.IsHourTypical(3) {
		t.Errorf("expected 03:00 to be atypical")
	}

	// Wrap around (e.g. night shift 22 to 6)
	b.TypicalStartHour = 22
	b.TypicalEndHour = 6
	if !b.IsHourTypical(23) || !b.IsHourTypical(4) {
		t.Errorf("expected 23:00 and 04:00 to be typical for night schedule")
	}
	if b.IsHourTypical(12) {
		t.Errorf("expected 12:00 to be atypical for night schedule")
	}
}

func TestStoreRecordTransfer(t *testing.T) {
	store := NewStore()
	userID := "user_test_record"

	store.RecordTransfer(userID, 20000.0, "ben_1", "dev_1", time.Now().UTC())
	store.RecordTransfer(userID, 40000.0, "ben_1", "dev_1", time.Now().UTC())

	b, ok := store.Get(userID)
	if !ok {
		t.Fatalf("expected user baseline to exist")
	}
	if b.TotalTransfers < 2 {
		t.Errorf("expected at least 2 transfers, got %d", b.TotalTransfers)
	}
	if ben, ok := b.KnownBeneficiaries["ben_1"]; !ok || ben.TransferCount != 2 {
		t.Errorf("expected beneficiary ben_1 with 2 transfers, got %+v", ben)
	}
	if dev, ok := b.KnownDevices["dev_1"]; !ok || dev.UseCount != 2 {
		t.Errorf("expected device dev_1 with 2 uses, got %+v", dev)
	}
}
