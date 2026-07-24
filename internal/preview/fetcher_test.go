package preview

import (
	"context"
	"errors"
	"net"
	"testing"
)

type fakeResolver map[string][]net.IP

func (f fakeResolver) LookupIPAddr(_ context.Context, host string) ([]net.IPAddr, error) {
	ips, ok := f[host]
	if !ok {
		return nil, errors.New("not found")
	}
	result := make([]net.IPAddr, 0, len(ips))
	for _, ip := range ips {
		result = append(result, net.IPAddr{IP: ip})
	}
	return result, nil
}

func TestValidateURLBlocksInternalNetworks(t *testing.T) {
	resolver := fakeResolver{
		"loopback.example": {net.ParseIP("127.0.0.1")},
		"private.example":  {net.ParseIP("192.168.1.20")},
		"metadata.example": {net.ParseIP("169.254.169.254")},
		"mixed.example":    {net.ParseIP("93.184.216.34"), net.ParseIP("10.0.0.2")},
		"public.example":   {net.ParseIP("93.184.216.34")},
	}
	for _, host := range []string{"loopback.example", "private.example", "metadata.example", "mixed.example"} {
		t.Run(host, func(t *testing.T) {
			_, err := validateURL(context.Background(), resolver, "https://"+host+"/product")
			if !errors.Is(err, ErrUnsafeURL) {
				t.Fatalf("error = %v, want ErrUnsafeURL", err)
			}
		})
	}
	if _, err := validateURL(context.Background(), resolver, "https://public.example/product"); err != nil {
		t.Fatalf("public URL rejected: %v", err)
	}
}

func TestIsPublicIP(t *testing.T) {
	tests := map[string]bool{
		"8.8.8.8": true, "1.1.1.1": true,
		"127.0.0.1": false, "10.0.0.1": false, "172.16.0.1": false,
		"192.168.1.1": false, "169.254.169.254": false, "::1": false,
		"fc00::1": false, "2001:4860:4860::8888": true,
	}
	for raw, want := range tests {
		if got := IsPublicIP(net.ParseIP(raw)); got != want {
			t.Errorf("IsPublicIP(%s) = %v, want %v", raw, got, want)
		}
	}
}
