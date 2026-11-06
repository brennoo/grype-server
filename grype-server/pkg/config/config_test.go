package config

import (
	"testing"

	"github.com/spf13/viper"
)

func TestLoadConfig(t *testing.T) {
	viper.Reset()
	t.Cleanup(viper.Reset)

	viper.Set(RestServerPort, 9991)
	viper.Set(GRPCServerPort, 9992)
	viper.Set(HealthCheckAddress, ":8080")
	viper.Set(DbRootDir, "./")
	viper.Set(DbUpdateURL, "https://grype.anchore.io/databases")

	conf := LoadConfig()

	if conf.RestServerPort != 9991 {
		t.Errorf("RestServerPort = %d, want 9991", conf.RestServerPort)
	}
	if conf.GRPCServerPort != 9992 {
		t.Errorf("GRPCServerPort = %d, want 9992", conf.GRPCServerPort)
	}
	if conf.HealthCheckAddress != ":8080" {
		t.Errorf("HealthCheckAddress = %q, want %q", conf.HealthCheckAddress, ":8080")
	}
	if conf.DbRootDir != "./" {
		t.Errorf("DbRootDir = %q, want %q", conf.DbRootDir, "./")
	}
	if conf.DbUpdateURL != "https://grype.anchore.io/databases" {
		t.Errorf("DbUpdateURL = %q, want %q", conf.DbUpdateURL, "https://grype.anchore.io/databases")
	}
}
