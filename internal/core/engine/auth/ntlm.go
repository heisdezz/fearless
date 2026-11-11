package auth

import (
	"context"
	"crypto/hmac"
	"crypto/md5"
	"crypto/rand"
	"encoding/base64"
	"encoding/binary"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"
	"unicode/utf16"

	"golang.org/x/crypto/md4"

	"fearless/internal/core/model"
)

// ntlmProvider implements NTLM over HTTP ([MS-NTHT]).
//
// Bruno does not implement NTLM itself: it wraps axios with the `axios-ntlm`
// package (bruno-electron/src/ipc/network/index.js:178). The algorithm therefore
// comes from [MS-NLMP], which is what both axios-ntlm and curl implement.
//
// The handshake needs two round trips:
//
//  1. Send the request with `Authorization: NTLM <type 1>`. The server answers
//     401 with `WWW-Authenticate: NTLM <type 2>`, carrying its challenge.
//  2. Retry with `Authorization: NTLM <type 3>`, which carries the proof.
//
// If the first request already succeeds, the server did not require NTLM and the
// response is returned unchanged.
type ntlmProvider struct{}

func (ntlmProvider) Mode() model.AuthMode { return model.AuthNTLM }

// Prepare supplies the handshake transport.
//
// Credentials are interpolated before the transport runs, so `{{password}}`
// works; the transport never sees a template.
func (ntlmProvider) Prepare(_ context.Context, cfg *model.Auth, deps Deps) (Prepared, error) {
	creds := ntlmCredentials{
		domain:   interpolate(deps, model.DerefString(cfg.Domain)),
		username: interpolate(deps, model.DerefString(cfg.Username)),
		password: interpolate(deps, model.DerefString(cfg.Password)),
	}
	// An empty user name cannot produce a valid proof. Contributing nothing
	// leaves the server's own 401 as the explanation, which is more useful than
	// a handshake that fails locally with a less specific message.
	if creds.username == "" {
		return Prepared{}, nil
	}
	return Prepared{
		Transport: &ntlmTransport{
			creds:  creds,
			rewind: deps.Rewind,
			rand:   ntlmRandom,
			now:    time.Now,
		},
	}, nil
}

// ntlmRandom returns n cryptographically random bytes.
func ntlmRandom(n int) []byte {
	b := make([]byte, n)
	if _, err := rand.Read(b); err != nil {
		panic("auth: crypto/rand failed: " + err.Error())
	}
	return b
}

// ntlmCredentials are the values every message derives from.
type ntlmCredentials struct {
	domain   string
	username string
	password string
}

// ntlmTransport performs the handshake.
type ntlmTransport struct {
	next   http.RoundTripper
	creds  ntlmCredentials
	rewind func() (*http.Request, error)
	rand   func(int) []byte
	now    func() time.Time
}

// Wrap attaches the next round tripper.
func (t *ntlmTransport) Wrap(next http.RoundTripper) http.RoundTripper {
	if next == nil {
		next = http.DefaultTransport
	}
	wrapped := *t
	wrapped.next = next
	return &wrapped
}

// RoundTrip sends the type 1 message, then the type 3 message.
func (t *ntlmTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	next := t.next
	if next == nil {
		next = http.DefaultTransport
	}

	// A replayed request that already carries a type 3 message is the tail end of
	// a handshake in progress; sending it directly is the only correct action.
	if ntlmMessageType(req.Header.Get("Authorization")) >= 3 {
		return next.RoundTrip(req)
	}

	// Without a replayable request the handshake cannot complete. Sending the
	// request as-is and returning the server's 401 is honest.
	if t.rewind == nil {
		return next.RoundTrip(req)
	}

	msg1, err := ntlmNegotiateMessage()
	if err != nil {
		return nil, err
	}
	first, err := t.rewind()
	if err != nil {
		return nil, err
	}
	first.Header.Set("Authorization", "NTLM "+base64.StdEncoding.EncodeToString(msg1))

	resp1, err := next.RoundTrip(first)
	if err != nil {
		return nil, err
	}
	if resp1.StatusCode != http.StatusUnauthorized {
		return resp1, nil
	}

	msg2, ok := findNTLMChallenge(resp1.Header)
	if !ok {
		return resp1, nil
	}
	// The 401's body must be drained before the retry, so the connection can be
	// reused. NTLM authenticates a *connection*, so a new socket restarts the
	// negotiation and the type 3 message would be rejected.
	_ = closeAndDrain(resp1.Body)

	msg3, err := t.creds.authenticateMessage(msg2, t.rand, t.now)
	if err != nil {
		return nil, err
	}
	third, err := t.rewind()
	if err != nil {
		return nil, err
	}
	third.Header.Set("Authorization", "NTLM "+base64.StdEncoding.EncodeToString(msg3))
	return next.RoundTrip(third)
}

// ntlmMessageType reports the message type in an Authorization value, or 0.
func ntlmMessageType(authz string) uint32 {
	rest, ok := strings.CutPrefix(authz, "NTLM ")
	if !ok {
		if rest, ok = strings.CutPrefix(authz, "ntlm "); !ok {
			return 0
		}
	}
	raw, err := base64.StdEncoding.DecodeString(strings.TrimSpace(rest))
	if err != nil || len(raw) < 12 || !hasNTLMSignature(raw) {
		return 0
	}
	return binary.LittleEndian.Uint32(raw[8:])
}

// NTLM message signature and type numbers, [MS-NLMP] §2.2.1.1.
var (
	ntlmSignature = []byte("NTLMSSP\x00")
	ntlmType1     = uint32(1)
	ntlmType2     = uint32(2)
	ntlmType3     = uint32(3)
)

// Negotiation flags, [MS-NLMP] §2.2.2.5.
//
// The set matches curl's NTLMv2 default, which is what servers are tested
// against: Unicode strings, extended session security, 128-bit keys, and a
// request for the target info block (without which NTLMv2 cannot be computed).
const (
	ntlmFlagNegotiateUnicode    = 0x00000001
	ntlmFlagNegotiateOEM        = 0x00000002
	ntlmFlagRequestTarget       = 0x00000004
	ntlmFlagNegotiateNTLM       = 0x00000200
	ntlmFlagNegotiateAlwaysSign = 0x00008000
	ntlmFlagNegotiateExtended   = 0x00080000
	ntlmFlagNegotiateTargetInfo = 0x00800000
	ntlmFlagNegotiateVersion    = 0x02000000
	ntlmFlagNegotiate128        = 0x20000000
	ntlmFlagNegotiateKeyExch    = 0x40000000
	ntlmFlagNegotiate56         = 0x80000000
)

// ntlmNegotiateFlags returns the flag word sent in both messages.
//
// Key exchange is deliberately excluded: without it no session key is sent, so
// the server cannot decrypt traffic it did not originate, and there is no key to
// derive wrong.
func ntlmNegotiateFlags() uint32 {
	return ntlmFlagNegotiateUnicode |
		ntlmFlagNegotiateOEM |
		ntlmFlagRequestTarget |
		ntlmFlagNegotiateNTLM |
		ntlmFlagNegotiateAlwaysSign |
		ntlmFlagNegotiateExtended |
		ntlmFlagNegotiateTargetInfo |
		ntlmFlagNegotiateVersion |
		ntlmFlagNegotiate128 |
		ntlmFlagNegotiate56
}

// ntlmVersionBlock is the 8-byte version field: 10.0 build 19041, NTLM rev 15.
var ntlmVersionBlock = []byte{10, 0, 0x9D, 0x99, 0, 0, 0, 15}

// ntlmNegotiateMessage builds the type 1 message, [MS-NLMP] §2.2.1.1.
//
// Layout (32 bytes before the version block):
//
//	0  signature(8) 8 type(4) 12 flags(4)
//	16 domain secbuf(8) 24 workstation secbuf(8) 32 version(8)
//
// Both security buffers are empty and point at offset 40, the first byte past
// the message.
func ntlmNegotiateMessage() ([]byte, error) {
	const msgLen = 40
	buf := make([]byte, msgLen)
	copy(buf[0:], ntlmSignature)
	binary.LittleEndian.PutUint32(buf[8:], ntlmType1)
	binary.LittleEndian.PutUint32(buf[12:], ntlmNegotiateFlags())

	for _, off := range []int{16, 24} {
		binary.LittleEndian.PutUint16(buf[off:], 0)        // length
		binary.LittleEndian.PutUint16(buf[off+2:], 0)      // max length
		binary.LittleEndian.PutUint32(buf[off+4:], msgLen) // offset
	}
	copy(buf[32:], ntlmVersionBlock)
	return buf, nil
}

// ntlmChallenge is a parsed type 2 message.
type ntlmChallenge struct {
	flags      uint32
	serverChal [8]byte
	targetInfo []byte
}

// ntlmParseChallenge decodes the binary payload of a type 2 message,
// [MS-NLMP] §2.2.1.2.
func ntlmParseChallenge(payload []byte) (ntlmChallenge, error) {
	var ch ntlmChallenge
	if len(payload) < 32 || !hasNTLMSignature(payload) {
		return ch, errors.New("auth: NTLM type 2 message too short or not signed")
	}
	if typ := binary.LittleEndian.Uint32(payload[8:]); typ != ntlmType2 {
		return ch, fmt.Errorf("auth: expected NTLM type 2 message, got type %d", typ)
	}
	ch.flags = binary.LittleEndian.Uint32(payload[20:])
	copy(ch.serverChal[:], payload[24:32])

	// Target info is the last security buffer (offset 40). It is optional in the
	// spec but mandatory for NTLMv2, so a missing one is reported by the caller.
	if len(payload) >= 48 {
		targetLen := int(binary.LittleEndian.Uint16(payload[40:]))
		targetOff := int(binary.LittleEndian.Uint32(payload[44:]))
		if targetLen > 0 && targetOff >= 0 && targetOff+targetLen <= len(payload) {
			ch.targetInfo = payload[targetOff : targetOff+targetLen]
		}
	}
	return ch, nil
}

// findNTLMChallenge extracts a type 2 message from response headers.
//
// A bare `NTLM` (no payload) is refused: it is legal for pre-Windows 2000
// servers but carries no target info, so no NTLMv2 proof can be built and the
// 401 should stand rather than receive an unverifiable message.
func findNTLMChallenge(h http.Header) ([]byte, bool) {
	for _, value := range h.Values("WWW-Authenticate") {
		rest, ok := strings.CutPrefix(strings.TrimSpace(value), "NTLM ")
		if !ok {
			if rest, ok = strings.CutPrefix(strings.TrimSpace(value), "ntlm "); !ok {
				continue
			}
		}
		raw, err := base64.StdEncoding.DecodeString(strings.TrimSpace(rest))
		if err != nil {
			continue
		}
		if _, err := ntlmParseChallenge(raw); err != nil {
			continue
		}
		return raw, true
	}
	return nil, false
}

// hasNTLMSignature reports whether b starts with the NTLMSSP signature.
func hasNTLMSignature(b []byte) bool {
	if len(b) < len(ntlmSignature) {
		return false
	}
	for i, c := range ntlmSignature {
		if b[i] != c {
			return false
		}
	}
	return true
}

// authenticateMessage builds the type 3 message for a challenge.
//
// This is the NTLMv2 path ([MS-NLMP] §3.3.1):
//
//	NTOWFv2  = HMAC-MD5(MD4(UTF16(Upper(user) + domain)), UTF16(password))
//	proof    = HMAC-MD5(NTOWFv2, serverChallenge || blob)
func (c ntlmCredentials) authenticateMessage(msg2 []byte, random func(int) []byte, now func() time.Time) ([]byte, error) {
	ch, err := ntlmParseChallenge(msg2)
	if err != nil {
		return nil, err
	}
	if len(ch.targetInfo) == 0 {
		return nil, errors.New("auth: NTLM type 2 message has no target info, which NTLMv2 requires")
	}

	proof, blob := c.ntlmV2Response(ch.serverChal, ch.targetInfo, random, now)
	ntResponse := append(append([]byte{}, proof...), blob...)

	user := utf16le(strings.ToUpper(c.username))
	domain := utf16le(c.domain)

	// Both authentication fields are sized and offset from the payload that
	// follows the header. An empty LM response is correct for NTLMv2.
	const headerLen = 64
	out := make([]byte, headerLen, headerLen+len(ntResponse)+len(domain)+len(user))
	copy(out[0:], ntlmSignature)
	binary.LittleEndian.PutUint32(out[8:], ntlmType3)

	// Offsets are absolute from the start of the message. They are accumulated
	// separately from the payload so each field points at its own bytes.
	cursor := headerLen
	writeSecBuf := func(off int, data []byte) {
		binary.LittleEndian.PutUint16(out[off:], uint16(len(data)))
		binary.LittleEndian.PutUint16(out[off+2:], uint16(len(data)))
		binary.LittleEndian.PutUint32(out[off+4:], uint32(cursor))
		cursor += len(data)
	}

	writeSecBuf(12, nil)        // LM response: absent in NTLMv2
	writeSecBuf(20, ntResponse) // NT response
	writeSecBuf(28, domain)
	writeSecBuf(36, user)
	writeSecBuf(44, nil) // workstation
	writeSecBuf(52, nil) // session key: none without key exchange

	// The server's flags govern string encoding, but the message type is
	// mandatory: a server may echo back a flag word that omits NegotiateNTLM,
	// and a type 3 without it is rejected.
	binary.LittleEndian.PutUint32(out[60:], (ch.flags|ntlmFlagNegotiateNTLM)&^ntlmFlagNegotiateKeyExch)

	out = append(out, ntResponse...)
	out = append(out, domain...)
	out = append(out, user...)
	return out, nil
}

// ntlmV2Response computes the NTLMv2 proof and blob.
//
// The blob ([MS-NLMP] §2.2.2.7) is:
//
//	0x01010000 | reserved(4) | timestamp(8) | clientNonce(8) | reserved(4) |
//	targetInfo | reserved(4)
//
// The trailing four bytes must be zero; servers reject a blob that is not
// terminated, and every field is little-endian.
func (c ntlmCredentials) ntlmV2Response(serverChal [8]byte, targetInfo []byte, random func(int) []byte, now func() time.Time) ([]byte, []byte) {
	clientNonce := random(8)

	blob := make([]byte, 0, 32+len(targetInfo)+4)
	blob = append(blob, 0x01, 0x01, 0x00, 0x00) // blob signature
	blob = append(blob, 0, 0, 0, 0)             // reserved
	ts := make([]byte, 8)

	// 100-nanosecond ticks since 1601-01-01 UTC.
	const windowsEpochOffsetSeconds = 11644473600
	ticks := (uint64(now().UTC().Unix()) + windowsEpochOffsetSeconds) * 10_000_000
	binary.LittleEndian.PutUint64(ts, ticks)
	blob = append(blob, ts...)

	blob = append(blob, clientNonce...)
	blob = append(blob, 0, 0, 0, 0) // reserved
	blob = append(blob, targetInfo...)
	blob = append(blob, 0, 0, 0, 0) // terminator

	key := ntlmNTowfv2(c.username, c.domain, c.password)
	msg := make([]byte, 0, 8+len(blob))
	msg = append(msg, serverChal[:]...)
	msg = append(msg, blob...)
	return hmacMD5(key, msg), blob
}

// ntlmNTowfv1 returns MD4(UTF16LE(password)), [MS-NLMP] §3.3.1.
//
// MD4 is mandated by the protocol; it is not used here as a general-purpose
// hash.
func ntlmNTowfv1(password string) []byte {
	h := md4.New()
	h.Write(utf16le(password))
	return h.Sum(nil)
}

// ntlmNTowfv2 derives the NTLMv2 key from the credentials.
func ntlmNTowfv2(user, domain, password string) []byte {
	return hmacMD5(ntlmNTowfv1(password), utf16le(strings.ToUpper(user)+domain))
}

// hmacMD5 computes HMAC-MD5, which NTLMv2 specifies for its proofs.
func hmacMD5(key, data []byte) []byte {
	mac := hmac.New(md5.New, key)
	mac.Write(data)
	return mac.Sum(nil)
}

// utf16le encodes a string as little-endian UTF-16, as every NTLM string field
// requires.
func utf16le(s string) []byte {
	units := utf16.Encode([]rune(s))
	out := make([]byte, len(units)*2)
	for i, u := range units {
		binary.LittleEndian.PutUint16(out[i*2:], u)
	}
	return out
}
