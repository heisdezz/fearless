package auth

import (
	"encoding/base64"
	"encoding/binary"
	"encoding/hex"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"fearless/internal/core/model"
)

// MS-NLMP §4.2 test vectors. The protocol document publishes these so an
// implementation can be checked without a domain controller.
const (
	msUser     = "User"
	msDomain   = "Domain"
	msPassword = "Password"

	// §4.2.2: the NTOWFv1 for "Password".
	msNTOWFv1 = "A4F49C406510BDCAB6824EE7C30FD852"
	// §4.2.4: the NTOWFv2 for User/Domain/Password.
	msNTOWFv2 = "0C868A403BFD7A93A3001EF22EF02E3F"
	// §4.2.4.1.1: the server challenge.
	msServerChallenge = "0123456789abcdef"
	// §4.2.4.1.1: the client nonce.
	msClientNonce = "ffffff0011223344"
	// §4.2.4.1.1: target info for the domain.
	msTargetInfo = "02000c0044006f006d00610069006e0001000c0053006500720076006500720000000000"
	// §4.2.4.2.2: the expected NTProofStr.
	msNTProofStr = "C2606A75B2A0947CFD475D653F30C598"
)

func msCredentials() ntlmCredentials {
	return ntlmCredentials{domain: msDomain, username: msUser, password: msPassword}
}

// msRandom returns the published client nonce regardless of n, which only ever
// requests 8 bytes here.
func msRandom(int) []byte {
	b, _ := hex.DecodeString(msClientNonce)
	return b
}

// TestNTLMNTowfv1MatchesVector pins the MD4-based key derivation.
func TestNTLMNTowfv1MatchesVector(t *testing.T) {
	if got := fmt.Sprintf("%X", ntlmNTowfv1(msPassword)); got != msNTOWFv1 {
		t.Errorf("NTOWFv1 = %s, want %s", got, msNTOWFv1)
	}
}

// TestNTLMNTowfv2MatchesVector pins the HMAC-MD5 over the upper-cased
// user name concatenated with the domain.
func TestNTLMNTowfv2MatchesVector(t *testing.T) {
	if got := fmt.Sprintf("%X", ntlmNTowfv2(msUser, msDomain, msPassword)); got != msNTOWFv2 {
		t.Errorf("NTOWFv2 = %s, want %s", got, msNTOWFv2)
	}
	// The user name is case-insensitive: a lowercase login must derive the same
	// key, because Windows compares it that way.
	if got := fmt.Sprintf("%X", ntlmNTowfv2(strings.ToLower(msUser), msDomain, msPassword)); got != msNTOWFv2 {
		t.Errorf("lowercase NTOWFv2 = %s, want %s", got, msNTOWFv2)
	}
}

// TestNTLMNTowfv2IsDomainSensitive proves the domain participates.
func TestNTLMNTowfv2IsDomainSensitive(t *testing.T) {
	other := fmt.Sprintf("%X", ntlmNTowfv2(msUser, "Other", msPassword))
	if other == msNTOWFv2 {
		t.Error("a different domain must derive a different key")
	}
}

// TestNTLMV2ResponseMatchesVector pins the type 3 proof against §4.2.4.2.
//
// This is the assertion that matters: it covers the blob layout, the timestamp
// encoding, the challenge ordering and the HMAC in one comparison.
func TestNTLMV2ResponseMatchesVector(t *testing.T) {
	targetInfo, err := hex.DecodeString(msTargetInfo)
	if err != nil {
		t.Fatal(err)
	}
	chalBytes, _ := hex.DecodeString(msServerChallenge)
	var chal [8]byte
	copy(chal[:], chalBytes)

	// The published blob carries timestamp 0x0090d336b734c301 ticks since
	// 1601-01-01. This constant is the epoch conversion, not a magic date.
	const wantTicks = uint64(0x0090d336b734c301)
	const epochOffsetSeconds = 11644473600
	when := time.Unix(int64(wantTicks/10_000_000)-epochOffsetSeconds, 0).UTC()

	proof, blob := msCredentials().ntlmV2Response(chal, targetInfo, msRandom, func() time.Time { return when })

	if got := fmt.Sprintf("%X", proof); got != msNTProofStr {
		t.Errorf("NTProofStr = %s, want %s", got, msNTProofStr)
	}

	// Blob shape, verbatim from §4.2.4.1.1: signature, reserved, timestamp,
	// client nonce, reserved, target info, zero terminator.
	wantBlob := "0101000000000000008E1AB736D39000" +
		strings.ToUpper(msClientNonce) + "00000000" +
		strings.ToUpper(msTargetInfo) + "00000000"
	if got := strings.ToUpper(fmt.Sprintf("%X", blob)); got != wantBlob {
		t.Errorf("blob mismatch\n got %s\nwant %s", got, wantBlob)
	}
	if wantLen := 28 + len(targetInfo) + 4; len(blob) != wantLen {
		t.Errorf("blob length = %d, want %d", len(blob), wantLen)
	}
}

// TestNTLMV2ResponseIsNonceDependent ensures the proof is not reusable.
func TestNTLMV2ResponseIsNonceDependent(t *testing.T) {
	targetInfo, _ := hex.DecodeString(msTargetInfo)
	var chal [8]byte
	when := time.Unix(1700000000, 0)

	proofA, _ := msCredentials().ntlmV2Response(chal, targetInfo,
		func(int) []byte { return make([]byte, 8) }, func() time.Time { return when })
	proofB, _ := msCredentials().ntlmV2Response(chal, targetInfo,
		func(int) []byte { return []byte{1, 2, 3, 4, 5, 6, 7, 8} }, func() time.Time { return when })

	if string(proofA) == string(proofB) {
		t.Error("a different client nonce must change the proof")
	}
}

// TestNTLMNegotiateMessage covers the type 1 layout.
func TestNTLMNegotiateMessage(t *testing.T) {
	msg, err := ntlmNegotiateMessage()
	if err != nil {
		t.Fatal(err)
	}
	if len(msg) != 40 {
		t.Fatalf("length = %d, want 40", len(msg))
	}
	if !hasNTLMSignature(msg) {
		t.Error("missing NTLMSSP signature")
	}
	if typ := binary.LittleEndian.Uint32(msg[8:]); typ != ntlmType1 {
		t.Errorf("type = %d, want 1", typ)
	}
	flags := binary.LittleEndian.Uint32(msg[12:])
	for _, want := range []uint32{
		ntlmFlagNegotiateUnicode, ntlmFlagNegotiateNTLM,
		ntlmFlagNegotiateExtended, ntlmFlagNegotiateTargetInfo,
	} {
		if flags&want == 0 {
			t.Errorf("flag %#x not set in %#x", want, flags)
		}
	}
	if flags&ntlmFlagNegotiateKeyExch != 0 {
		t.Error("key exchange must not be requested when no session key is sent")
	}
	// Empty domain/workstation buffers point past the message.
	for _, off := range []int{16, 24} {
		if l := binary.LittleEndian.Uint16(msg[off:]); l != 0 {
			t.Errorf("buffer at %d has length %d", off, l)
		}
		if o := binary.LittleEndian.Uint32(msg[off+4:]); o != 40 {
			t.Errorf("buffer at %d has offset %d, want 40", off, o)
		}
	}
}

// TestNTLMParseChallenge covers a well-formed type 2 message.
func TestNTLMParseChallenge(t *testing.T) {
	msg2 := buildChallengeMessage(t, msServerChallenge, msTargetInfo, ntlmNegotiateFlags())

	ch, err := ntlmParseChallenge(msg2)
	if err != nil {
		t.Fatal(err)
	}
	if got := fmt.Sprintf("%X", ch.serverChal[:]); !strings.EqualFold(got, msServerChallenge) {
		t.Errorf("challenge = %s, want %s", got, msServerChallenge)
	}
	if got := fmt.Sprintf("%X", ch.targetInfo); !strings.EqualFold(got, msTargetInfo) {
		t.Errorf("target info = %s, want %s", got, msTargetInfo)
	}
}

// TestNTLMParseChallengeRejectsMalformed guards against panics on bad input.
func TestNTLMParseChallengeRejectsMalformed(t *testing.T) {
	cases := map[string][]byte{
		"empty":         nil,
		"short":         []byte("NTLMSSP\x00"),
		"bad signature": append([]byte("XXXXSSP\x00"), make([]byte, 32)...),
		"wrong type":    withType(make([]byte, 48), ntlmType1),
	}
	for name, payload := range cases {
		if _, err := ntlmParseChallenge(payload); err == nil {
			t.Errorf("%s: expected an error", name)
		}
	}
}

// TestNTLMParseChallengeToleratesOutOfRangeTargetInfo makes sure a truncated
// message does not slice out of bounds.
func TestNTLMParseChallengeToleratesOutOfRangeTargetInfo(t *testing.T) {
	msg := buildChallengeMessage(t, msServerChallenge, msTargetInfo, ntlmNegotiateFlags())
	// Claim a target info range that runs past the end of the message.
	binary.LittleEndian.PutUint32(msg[44:], uint32(len(msg)+100))

	ch, err := ntlmParseChallenge(msg)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if len(ch.targetInfo) != 0 {
		t.Error("an out-of-range target info must be ignored, not sliced")
	}
}

// TestNTLMFindChallenge covers header extraction and the bare-NTLM refusal.
func TestNTLMFindChallenge(t *testing.T) {
	msg2 := buildChallengeMessage(t, msServerChallenge, msTargetInfo, ntlmNegotiateFlags())
	b64 := base64.StdEncoding.EncodeToString(msg2)

	h := http.Header{}
	h.Add("WWW-Authenticate", "Negotiate")
	h.Add("WWW-Authenticate", "NTLM "+b64)
	if _, ok := findNTLMChallenge(h); !ok {
		t.Error("type 2 message not found among multiple challenges")
	}

	// A bare `NTLM` cannot carry a target info block, so it is refused.
	bare := http.Header{}
	bare.Set("WWW-Authenticate", "NTLM")
	if _, ok := findNTLMChallenge(bare); ok {
		t.Error("a bare NTLM challenge must not be accepted")
	}

	// Not NTLM at all.
	digestOnly := http.Header{}
	digestOnly.Set("WWW-Authenticate", `Digest realm="x"`)
	if _, ok := findNTLMChallenge(digestOnly); ok {
		t.Error("a Digest challenge is not an NTLM challenge")
	}

	// Corrupt base64.
	bad := http.Header{}
	bad.Set("WWW-Authenticate", "NTLM !!!notbase64!!!")
	if _, ok := findNTLMChallenge(bad); ok {
		t.Error("invalid base64 must not parse")
	}
}

// TestNTLMAuthenticateMessage covers the type 3 layout and field offsets.
func TestNTLMAuthenticateMessage(t *testing.T) {
	msg2 := buildChallengeMessage(t, msServerChallenge, msTargetInfo, ntlmNegotiateFlags())

	msg3, err := msCredentials().authenticateMessage(msg2, msRandom, func() time.Time {
		return time.Unix(1700000000, 0)
	})
	if err != nil {
		t.Fatal(err)
	}
	if typ := binary.LittleEndian.Uint32(msg3[8:]); typ != ntlmType3 {
		t.Fatalf("type = %d, want 3", typ)
	}
	if flags := binary.LittleEndian.Uint32(msg3[60:]); flags&ntlmFlagNegotiateNTLM == 0 {
		t.Error("NegotiateNTLM must be set on the type 3 message")
	}

	// Every security buffer must describe bytes actually present, and the fields
	// must not overlap.
	seen := make(map[[2]int]bool)
	check := func(off int, name string, wantData []byte) {
		l := int(binary.LittleEndian.Uint16(msg3[off:]))
		pos := int(binary.LittleEndian.Uint32(msg3[off+4:]))
		if l == 0 {
			return
		}
		if pos+l > len(msg3) {
			t.Fatalf("%s buffer [%d,%d) exceeds message length %d", name, pos, pos+l, len(msg3))
		}
		if seen[[2]int{pos, l}] {
			t.Errorf("%s buffer overlaps another field", name)
		}
		seen[[2]int{pos, l}] = true
		if wantData != nil && string(msg3[pos:pos+l]) != string(wantData) {
			t.Errorf("%s = %X, want %X", name, msg3[pos:pos+l], wantData)
		}
	}
	check(12, "lm", nil) // empty
	check(20, "nt", nil)
	check(28, "domain", utf16le(msDomain))
	check(36, "user", utf16le(strings.ToUpper(msUser)))
	check(44, "workstation", nil)
	check(52, "session key", nil)

	// The LM field must be empty for NTLMv2.
	if l := binary.LittleEndian.Uint16(msg3[12:]); l != 0 {
		t.Errorf("LM response length = %d, want 0 in NTLMv2", l)
	}
}

// TestNTLMAuthenticateMessageRequiresTargetInfo refuses to guess.
func TestNTLMAuthenticateMessageRequiresTargetInfo(t *testing.T) {
	msg2 := buildChallengeMessage(t, msServerChallenge, "", ntlmNegotiateFlags())
	_, err := msCredentials().authenticateMessage(msg2, msRandom, time.Now)
	if err == nil || !strings.Contains(err.Error(), "target info") {
		t.Fatalf("err = %v", err)
	}
}

// TestNTLMTransportCompletesHandshake runs the whole exchange against a server
// that verifies the proof, so a wrong message fails rather than passing by luck.
func TestNTLMTransportCompletesHandshake(t *testing.T) {
	var sawType1, sawType3 bool

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		authz := r.Header.Get("Authorization")
		switch ntlmMessageType(authz) {
		case 0:
			w.WriteHeader(http.StatusUnauthorized)
		case 1:
			sawType1 = true
			msg2 := buildChallengeMessage(t, msServerChallenge, msTargetInfo, ntlmNegotiateFlags())
			w.Header().Set("WWW-Authenticate", "NTLM "+base64.StdEncoding.EncodeToString(msg2))
			w.WriteHeader(http.StatusUnauthorized)
		case 3:
			sawType3 = true
			// Recompute the proof the way a server would.
			raw, _ := decodeStd(strings.TrimPrefix(authz, "NTLM "))
			ntLen := int(binary.LittleEndian.Uint16(raw[20:]))
			ntOff := int(binary.LittleEndian.Uint32(raw[24:]))
			ntResponse := raw[ntOff : ntOff+ntLen]
			proof, blob := ntResponse[:16], ntResponse[16:]

			targetInfo, _ := hex.DecodeString(msTargetInfo)
			chalBytes, _ := hex.DecodeString(msServerChallenge)
			key := ntlmNTowfv2(msUser, msDomain, msPassword)
			want := hmacMD5(key, append(append([]byte{}, chalBytes...), blob...))
			if string(proof) != string(want) {
				w.WriteHeader(http.StatusUnauthorized)
				return
			}
			if len(targetInfo) == 0 {
				t.Error("server has no target info")
			}
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte("authenticated"))
		default:
			t.Errorf("unexpected message type in %q", authz)
			w.WriteHeader(http.StatusUnauthorized)
		}
	}))
	defer server.Close()

	req, err := http.NewRequest(http.MethodGet, server.URL, nil)
	if err != nil {
		t.Fatal(err)
	}

	tr := &ntlmTransport{
		creds:  msCredentials(),
		rewind: func() (*http.Request, error) { return http.NewRequest(http.MethodGet, server.URL, nil) },
		rand:   msRandom,
		now:    time.Now,
	}
	client := &http.Client{Transport: tr.Wrap(http.DefaultTransport)}
	resp, err := client.Do(req)
	if err != nil {
		t.Fatalf("request: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want 200", resp.StatusCode)
	}
	if !sawType1 || !sawType3 {
		t.Errorf("handshake incomplete: type1=%v type3=%v", sawType1, sawType3)
	}
}

// TestNTLMTransportPassesThroughWhenNotChallenged covers a server that does not
// require NTLM.
func TestNTLMTransportPassesThroughWhenNotChallenged(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	tr := &ntlmTransport{
		creds:  msCredentials(),
		rewind: func() (*http.Request, error) { return http.NewRequest(http.MethodGet, server.URL, nil) },
		rand:   msRandom,
		now:    time.Now,
	}
	client := &http.Client{Transport: tr.Wrap(http.DefaultTransport)}
	resp, err := client.Get(server.URL)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Errorf("status = %d", resp.StatusCode)
	}
}

// TestNTLMTransportWithoutRewindIsSafe never panics and never hangs.
func TestNTLMTransportWithoutRewindIsSafe(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("WWW-Authenticate", "NTLM")
		w.WriteHeader(http.StatusUnauthorized)
	}))
	defer server.Close()

	tr := &ntlmTransport{creds: msCredentials()}
	client := &http.Client{Transport: tr.Wrap(http.DefaultTransport)}
	resp, err := client.Get(server.URL)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusUnauthorized {
		t.Errorf("status = %d, want the server's 401", resp.StatusCode)
	}
}

// TestNTLMTransportLeavesNonNTLMChallengeAlone keeps a Digest server working.
func TestNTLMTransportLeavesNonNTLMChallengeAlone(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("WWW-Authenticate", `Digest realm="x", nonce="n"`)
		w.WriteHeader(http.StatusUnauthorized)
	}))
	defer server.Close()

	tr := &ntlmTransport{
		creds:  msCredentials(),
		rewind: func() (*http.Request, error) { return http.NewRequest(http.MethodGet, server.URL, nil) },
		rand:   msRandom,
		now:    time.Now,
	}
	client := &http.Client{Transport: tr.Wrap(http.DefaultTransport)}
	resp, err := client.Get(server.URL)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusUnauthorized {
		t.Errorf("status = %d", resp.StatusCode)
	}
}

// TestNTLMProvider registration and no-op behaviour.
func TestNTLMProvider(t *testing.T) {
	if _, ok := Get(model.AuthNTLM); !ok {
		t.Fatal("ntlm provider not registered")
	}

	// A missing user name contributes nothing rather than starting a handshake
	// that cannot succeed.
	p, err := Apply(t.Context(), []*model.Auth{{Mode: model.AuthNTLM}}, Deps{})
	if err != nil {
		t.Fatalf("apply: %v", err)
	}
	if p.Transport != nil {
		t.Error("empty credentials must not supply a transport")
	}

	// With credentials, a transport is supplied.
	p, err = Apply(t.Context(), []*model.Auth{{
		Mode:     model.AuthNTLM,
		Username: strp("user"),
		Password: strp("pass"),
		Domain:   strp("DOM"),
	}}, Deps{})
	if err != nil {
		t.Fatalf("apply: %v", err)
	}
	if p.Transport == nil {
		t.Fatal("expected a transport")
	}
}

// TestNTLMProviderInterpolates makes {{vars}} in credentials resolve.
func TestNTLMProviderInterpolates(t *testing.T) {
	p, err := Apply(t.Context(), []*model.Auth{{
		Mode:     model.AuthNTLM,
		Username: strp("{{user}}"),
		Password: strp("{{pass}}"),
	}}, Deps{Interpolate: func(s string) string {
		return strings.NewReplacer("{{user}}", "alice", "{{pass}}", "s3cret").Replace(s)
	}})
	if err != nil {
		t.Fatalf("apply: %v", err)
	}
	tr, ok := p.Transport.(*ntlmTransport)
	if !ok {
		t.Fatalf("transport type %T", p.Transport)
	}
	if tr.creds.username != "alice" || tr.creds.password != "s3cret" {
		t.Errorf("credentials not interpolated: %+v", tr.creds)
	}
}

// --- helpers -------------------------------------------------------------

// buildChallengeMessage assembles a type 2 message with a target info block.
func buildChallengeMessage(t *testing.T, challengeHex, targetInfoHex string, flags uint32) []byte {
	t.Helper()
	chal, err := hex.DecodeString(challengeHex)
	if err != nil {
		t.Fatal(err)
	}
	targetInfo, err := hex.DecodeString(targetInfoHex)
	if err != nil {
		t.Fatal(err)
	}

	const headerLen = 48
	msg := make([]byte, headerLen+len(targetInfo))
	copy(msg[0:], ntlmSignature)
	binary.LittleEndian.PutUint32(msg[8:], ntlmType2)

	// Target name security buffer: empty, pointing past the message.
	binary.LittleEndian.PutUint32(msg[12:], 0)
	binary.LittleEndian.PutUint32(msg[16:], uint32(headerLen+len(targetInfo)))

	binary.LittleEndian.PutUint32(msg[20:], flags)
	copy(msg[24:32], chal)

	// Target info security buffer and version block.
	binary.LittleEndian.PutUint16(msg[40:], uint16(len(targetInfo)))
	binary.LittleEndian.PutUint16(msg[42:], uint16(len(targetInfo)))
	binary.LittleEndian.PutUint32(msg[44:], headerLen)
	// Version block sits at 48 in the header, but this message has no version
	// field (a 48-byte header is valid); target info starts right after.
	copy(msg[headerLen:], targetInfo)
	return msg
}

func withType(msg []byte, typ uint32) []byte {
	copy(msg[0:], ntlmSignature)
	binary.LittleEndian.PutUint32(msg[8:], typ)
	return msg
}

func decodeStd(s string) ([]byte, error) {
	return base64.StdEncoding.DecodeString(s)
}
