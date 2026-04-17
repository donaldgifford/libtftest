package localstack

import "testing"

func TestConfig_ResolveImage(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		cfg     Config
		envImg  string
		envAuth string
		want    string
	}{
		{
			name: "explicit image wins",
			cfg:  Config{Image: "custom/localstack:v1"},
			want: "custom/localstack:v1",
		},
		{
			name: "default community image",
			cfg:  Config{},
			want: defaultImage,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got := tt.cfg.ResolveImage()
			if got != tt.want {
				t.Errorf("ResolveImage() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestConfig_ResolveImage_EnvOverride(t *testing.T) {
	t.Setenv("LIBTFTEST_LOCALSTACK_IMAGE", "mirror/localstack:custom")
	t.Setenv("LOCALSTACK_AUTH_TOKEN", "")

	cfg := Config{}
	got := cfg.ResolveImage()
	want := "mirror/localstack:custom"

	if got != want {
		t.Errorf("ResolveImage() with env = %q, want %q", got, want)
	}
}

func TestConfig_ResolveImage_ProDefault(t *testing.T) {
	t.Setenv("LIBTFTEST_LOCALSTACK_IMAGE", "")
	t.Setenv("LOCALSTACK_AUTH_TOKEN", "some-token")

	cfg := Config{Edition: EditionAuto}
	got := cfg.ResolveImage()

	if got != defaultProImage {
		t.Errorf("ResolveImage() with pro token = %q, want %q", got, defaultProImage)
	}
}

func TestConfig_Env(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		cfg      Config
		wantKeys []string
	}{
		{
			name:     "basic env",
			cfg:      Config{},
			wantKeys: []string{"DEBUG"},
		},
		{
			name:     "with auth token",
			cfg:      Config{AuthToken: "test-token"},
			wantKeys: []string{"DEBUG", "LOCALSTACK_AUTH_TOKEN"},
		},
		{
			name:     "with services",
			cfg:      Config{Services: []string{"s3", "sqs"}},
			wantKeys: []string{"DEBUG", "SERVICES"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			env := tt.cfg.Env()
			for _, key := range tt.wantKeys {
				if _, ok := env[key]; !ok {
					t.Errorf("Env() missing key %q", key)
				}
			}
		})
	}
}

func TestConfig_Env_ServicesJoined(t *testing.T) {
	t.Parallel()

	cfg := Config{Services: []string{"s3", "sqs", "iam"}}
	env := cfg.Env()

	want := "s3,sqs,iam"
	if env["SERVICES"] != want {
		t.Errorf("Env()[SERVICES] = %q, want %q", env["SERVICES"], want)
	}
}
