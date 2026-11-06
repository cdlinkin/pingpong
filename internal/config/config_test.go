package config

import "testing"

func TestLoad(t *testing.T) {
	t.Setenv("DATABASE_URL", "postgres://localhost/test")
	t.Setenv("JWT_SECRET", "secret")
	t.Setenv("LOG_LEVEL", "debug")
	t.Setenv("LOG_FORMAT", "json")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.DatabaseURL != "postgres://localhost/test" || cfg.JWTSecret != "secret" || cfg.LogLevel != "debug" || cfg.LogFormat != "json" {
		t.Fatalf("unexpected config: %+v", cfg)
	}
}

func TestLoadWithoutSecret(t *testing.T) {
	t.Setenv("JWT_SECRET", "")

	if _, err := Load(); err == nil {
		t.Fatal("expected error when JWT_SECRET is empty")
	}
}

func TestLoadUploadDir(t *testing.T) {
	t.Setenv("JWT_SECRET", "secret")

	t.Setenv("UPLOAD_DIR", "")
	cfg, err := Load()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.UploadDir != "uploads" {
		t.Fatalf("expected default uploads, got %q", cfg.UploadDir)
	}

	t.Setenv("UPLOAD_DIR", "/data/files")
	cfg, err = Load()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.UploadDir != "/data/files" {
		t.Fatalf("expected /data/files, got %q", cfg.UploadDir)
	}
}

func TestLoadHTTPAddr(t *testing.T) {
	t.Setenv("JWT_SECRET", "secret")

	t.Setenv("HTTP_ADDR", "")
	cfg, err := Load()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.HTTPAddr != ":8080" {
		t.Fatalf("expected default http address, got %q", cfg.HTTPAddr)
	}

	t.Setenv("HTTP_ADDR", ":9090")
	cfg, err = Load()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.HTTPAddr != ":9090" {
		t.Fatalf("expected ':9090', got %q", cfg.HTTPAddr)
	}
}
