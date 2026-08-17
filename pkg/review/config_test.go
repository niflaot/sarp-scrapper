package review

import "testing"

func TestConfigValidateThresholds(t *testing.T) {
	valid := Config{StateFile: "state.json", HouseMinValue: 500000, BusinessMinValue: 100000}
	if err := valid.Validate(); err != nil {
		t.Fatalf("Validate() error = %v", err)
	}
	valid.HouseMinValue = -1
	if err := valid.Validate(); err == nil {
		t.Fatal("Validate() expected negative house threshold error")
	}
}
