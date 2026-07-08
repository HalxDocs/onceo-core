package benchmarks

import (
	"net/http"
	"os"
	"testing"

	onceo "github.com/HalxDocs/onceo-core"
	"github.com/HalxDocs/onceo-core/internal/testutil"
	"github.com/HalxDocs/onceo-core/providers/paystack"
)

func BenchmarkPaystackVerify(b *testing.B) {
	secret := "sk_test_bench"
	body, err := os.ReadFile("../providers/paystack/testdata/charge_success.json")
	if err != nil {
		b.Fatal(err)
	}

	headers := http.Header{}
	headers.Set("X-Paystack-Signature", testutil.SignHMACSHA512([]byte(secret), body))

	p := &paystack.Provider{SecretKey: secret}
	store := onceo.NewMemoryStore()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = onceo.Process(nil, p, store, headers, body)
	}
}

func BenchmarkPaystackParse(b *testing.B) {
	body, err := os.ReadFile("../providers/paystack/testdata/charge_success.json")
	if err != nil {
		b.Fatal(err)
	}

	p := &paystack.Provider{}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = p.Parse(body)
	}
}

func BenchmarkPaystackNormalize(b *testing.B) {
	body, err := os.ReadFile("../providers/paystack/testdata/charge_success.json")
	if err != nil {
		b.Fatal(err)
	}

	p := &paystack.Provider{}
	parsed, _ := p.Parse(body)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = p.Normalize(parsed)
	}
}
