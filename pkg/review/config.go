package review

import "fmt"

type Config struct {
	StateFile         string `env:"LISTINGS_STATE_FILE" envDefault:".cache/listings-state.json"`
	HouseMinValue     int64  `env:"LISTINGS_HOUSE_MIN_VALUE" envDefault:"500000"`
	BusinessMinValue  int64  `env:"LISTINGS_BUSINESS_MIN_VALUE" envDefault:"100000"`
	WarehouseMinValue int64  `env:"LISTINGS_WAREHOUSE_MIN_VALUE" envDefault:"100000"`
}

func (c Config) Validate() error {
	if c.StateFile == "" {
		return fmt.Errorf("LISTINGS_STATE_FILE must not be empty")
	}
	if c.HouseMinValue < 0 {
		return fmt.Errorf("LISTINGS_HOUSE_MIN_VALUE must not be negative")
	}
	if c.BusinessMinValue < 0 {
		return fmt.Errorf("LISTINGS_BUSINESS_MIN_VALUE must not be negative")
	}
	if c.WarehouseMinValue < 0 {
		return fmt.Errorf("LISTINGS_WAREHOUSE_MIN_VALUE must not be negative")
	}
	return nil
}
