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

func TestLoadStorage(t *testing.T) {
	t.Setenv("JWT_SECRET", "secret")
	t.Setenv("STORAGE", "")
	t.Setenv("S3_BUCKET", "")
	t.Setenv("S3_USE_SSL", "")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.Storage != "local" || cfg.S3Bucket != "pingpong" || cfg.S3UseSSL {
		t.Fatalf("unexpected defaults: %+v", cfg)
	}

	t.Setenv("STORAGE", "s3")
	t.Setenv("S3_ENDPOINT", "localhost:9000")
	t.Setenv("S3_ACCESS_KEY", "access")
	t.Setenv("S3_SECRET_KEY", "secret-key")
	t.Setenv("S3_USE_SSL", "true")

	cfg, err = Load()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.Storage != "s3" || cfg.S3Endpoint != "localhost:9000" || !cfg.S3UseSSL {
		t.Fatalf("unexpected s3 config: %+v", cfg)
	}
}

func TestLoadStorageErrors(t *testing.T) {
	tests := []struct {
		name string
		env  map[string]string
	}{
		{"unknown storage", map[string]string{"STORAGE": "ftp"}},
		{"s3 without endpoint", map[string]string{"STORAGE": "s3", "S3_ENDPOINT": "", "S3_ACCESS_KEY": "a", "S3_SECRET_KEY": "b"}},
		{"s3 without keys", map[string]string{"STORAGE": "s3", "S3_ENDPOINT": "localhost:9000", "S3_ACCESS_KEY": "", "S3_SECRET_KEY": ""}},
		{"bad ssl flag", map[string]string{"S3_USE_SSL": "maybe"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Setenv("JWT_SECRET", "secret")
			t.Setenv("STORAGE", "")
			t.Setenv("S3_USE_SSL", "")
			for k, v := range tt.env {
				t.Setenv(k, v)
			}

			if _, err := Load(); err == nil {
				t.Fatal("expected error")
			}
		})
	}
}
