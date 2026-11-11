package proxy

import (
	"context"
	"net"
	"testing"
	"time"

	"github.com/dop251/goja"
)

// testVM converts Go values for the predicate calls below. The predicates read
// goja.Value arguments, so the tests go through a real runtime rather than
// hand-rolling Value implementations, which would not exercise the same
// conversion the script path uses.
var testVM = goja.New()

// toValue converts one argument for a predicate call.
func toValue(s string) goja.Value { return testVM.ToValue(s) }

// TestWeekdayRange covers the single-day, range, wrapping and GMT forms.
//
// These predicates read the wall clock, so the assertions are computed from
// time.Now() rather than hard-coded, and the expectation is expressed the way a
// PAC author would write the call.
func TestWeekdayRange(t *testing.T) {
	names := []string{"SUN", "MON", "TUE", "WED", "THU", "FRI", "SAT"}
	today := time.Now().Weekday()

	// A single day matches only today.
	for i, name := range names {
		want := time.Weekday(i) == today
		if got := weekdayRange(name); got != want {
			t.Errorf("weekdayRange(%q) = %v, want %v", name, got, want)
		}
	}

	// Today..today is today.
	if !weekdayRange(names[today], toValue(names[today])) {
		t.Errorf("weekdayRange(%s, %s) = false, want true", names[today], names[today])
	}

	// A non-wrapping range includes today when today falls inside it, using the
	// same day ordering the function does.
	from := names[(int(today)+6)%7] // yesterday
	if !weekdayRange(from, toValue(names[today])) {
		t.Error("a range ending today excluded today")
	}

	// A trailing GMT is accepted.
	if got := weekdayRange(names[today], toValue("GMT")); !got {
		t.Error("GMT form rejected today")
	}

	// An unknown day name matches nothing.
	if weekdayRange("XYZ") {
		t.Error("an unknown weekday matched")
	}
}

// TestWeekdayRangeWrapping covers SAT-SUN and the reverse across the week
// boundary, which is the case a naive `>= && <=` gets wrong.
func TestWeekdayRangeWrapping(t *testing.T) {
	// FRI-MON wraps, so it includes today for every day except TUE..THU.
	today := time.Now().Weekday()
	want := today == time.Friday || today == time.Saturday || today == time.Sunday || today == time.Monday
	if got := weekdayRange("FRI", toValue("MON")); got != want {
		t.Errorf("weekdayRange(FRI, MON) = %v, want %v (today %s)", got, want, today)
	}
}

// TestDateRange covers no arguments, a single day, a range and a malformed
// range.
func TestDateRange(t *testing.T) {
	today := time.Now().Day()

	if !dateRange() {
		t.Error("no arguments should always match")
	}
	if !dateRange(toValue(itoa(today))) {
		t.Error("today did not match a single-day range")
	}
	if today != 1 && dateRange(toValue(itoa(today-1))) {
		t.Error("yesterday matched a single-day range")
	}
	// A range spanning today.
	lo, hi := itoa(today), itoa(today)
	if !dateRange(toValue(lo), toValue(hi)) {
		t.Error("today did not fall inside today..today")
	}
	// A wrapping range: 25..5 includes today only for the last days and first
	// days of a month.
	wantWrap := today >= 25 || today <= 5
	if got := dateRange(toValue("25"), toValue("5")); got != wantWrap {
		t.Errorf("dateRange(25, 5) = %v, want %v (today %d)", got, wantWrap, today)
	}
	// GMT is accepted and ignored.
	if !dateRange(toValue(itoa(today)), toValue("GMT")) {
		t.Error("GMT date range rejected today")
	}
	// A non-numeric argument does not match.
	if dateRange(toValue("nope")) {
		t.Error("a malformed date matched")
	}
}

// TestTimeRange covers hour, hour-range, wrapping range and malformed input.
func TestTimeRange(t *testing.T) {
	hour := time.Now().Hour()

	if !timeRange() {
		t.Error("no arguments should always match")
	}
	if !timeRange(toValue(itoa(hour))) {
		t.Error("the current hour did not match a single-hour range")
	}
	if other := (hour + 6) % 24; timeRange(toValue(itoa(other))) {
		// The +6 hour is a different hour, so it must not match.
		t.Errorf("hour %d matched a range for hour %d", hour, other)
	}
	if !timeRange(toValue(itoa(hour)), toValue(itoa(hour))) {
		t.Error("the current hour did not fall inside its own range")
	}
	// A wrapping range that spans midnight includes the current hour when it is
	// late or early.
	wantWrap := hour >= 22 || hour <= 2
	if got := timeRange(toValue("22"), toValue("2")); got != wantWrap {
		t.Errorf("timeRange(22, 2) = %v, want %v (hour %d)", got, wantWrap, hour)
	}
	if timeRange(toValue("abc")) {
		t.Error("a malformed time matched")
	}
}

// TestPacHelperFunctionsInstalled covers the whole installed set through a
// script, which is what actually runs.
func TestPacHelperFunctionsInstalled(t *testing.T) {
	const source = `
function FindProxyForURL(url, host) {
    if (typeof dnsDomainIs !== "function") return "PROXY missing:1";
    if (typeof shExpMatch !== "function") return "PROXY missing:2";
    if (typeof isPlainHostName !== "function") return "PROXY missing:3";
    if (typeof isInNet !== "function") return "PROXY missing:4";
    if (typeof dnsResolve !== "function") return "PROXY missing:5";
    if (typeof myIpAddress !== "function") return "PROXY missing:6";
    if (typeof localHostOrDomainIs !== "function") return "PROXY missing:7";
    if (typeof dnsDomainLevels !== "function") return "PROXY missing:8";
    if (typeof weekdayRange !== "function") return "PROXY missing:9";
    if (typeof dateRange !== "function") return "PROXY missing:10";
    if (typeof timeRange !== "function") return "PROXY missing:11";
    if (typeof alert !== "function") return "PROXY missing:12";
    return "PROXY all:1";
}
`
	res := evalPAC(t, source, "http://a/")
	if res.Attempts[0].Host != "all" {
		t.Fatalf("raw = %q: a spec global is missing or not a function", res.Raw)
	}
}

// TestPACAlertIsSilent makes sure the alert global does not panic or block.
func TestPACAlertIsSilent(t *testing.T) {
	res := evalPAC(t, `
function FindProxyForURL(url, host) { alert("hello " + host); return "PROXY ok:1"; }
`, "http://a/")
	if res.Attempts[0].Host != "ok" {
		t.Errorf("raw = %q", res.Raw)
	}
}

// TestIsInNetIPv6 covers the 16-byte branch the IPv4 test does not reach.
func TestIsInNetIPv6(t *testing.T) {
	g := &pacGlobals{
		ctx:        context.Background(),
		lookupHost: func(context.Context, string) ([]string, error) { return nil, nil },
	}
	if !g.isInNet("2001:db8::1", "2001:db8::", "ffff:ffff::") {
		t.Error("an address inside its IPv6 net did not match")
	}
	if g.isInNet("2001:db9::1", "2001:db8::", "ffff:ffff::") {
		t.Error("an address outside its IPv6 net matched")
	}
	// A malformed mask or pattern is a non-match, not a panic.
	if g.isInNet("10.0.0.1", "not-an-ip", "255.0.0.0") {
		t.Error("a malformed pattern matched")
	}
	if g.isInNet("10.0.0.1", "10.0.0.0", "not-a-mask") {
		t.Error("a malformed mask matched")
	}
}

// TestDetectMyIP covers the self-address helper, which must return something
// parseable even on a host with no external interface.
func TestDetectMyIP(t *testing.T) {
	ip := detectMyIP()
	if net.ParseIP(ip) == nil {
		t.Fatalf("detectMyIP() = %q, not an IP", ip)
	}
}

// TestPACDirectiveString covers the log rendering.
func TestPACDirectiveString(t *testing.T) {
	cases := map[PACDirective]string{
		{Kind: PACDirect}:                       "DIRECT",
		{Kind: PACHTTP, Host: "h", Port: "1"}:   "PROXY h:1",
		{Kind: PACHTTPS, Host: "h", Port: "2"}:  "HTTPS h:2",
		{Kind: PACSOCKS5, Host: "h", Port: "3"}: "SOCKS5 h:3",
	}
	for d, want := range cases {
		if got := d.String(); got != want {
			t.Errorf("%+v.String() = %q, want %q", d, got, want)
		}
	}
}

// TestPACSourceNil covers the adapter's nil guard.
func TestPACSourceNil(t *testing.T) {
	src := NewPACSource(nil)
	if _, err := src.Evaluate(context.Background(), mustURL(t, "http://a/")); err == nil {
		t.Fatal("a nil PAC resolver evaluated")
	}
}

// TestPACSplitHostPort covers the address forms a PAC file may contain.
func TestPACSplitHostPort(t *testing.T) {
	cases := map[string][2]string{
		"host:8080":     {"host", "8080"},
		"host":          {"host", ""},
		"[::1]:8080":    {"::1", "8080"},
		"[::1]":         {"::1", ""},
		"10.0.0.1:3128": {"10.0.0.1", "3128"},
	}
	for in, want := range cases {
		h, p := splitHostPort(in)
		if h != want[0] || p != want[1] {
			t.Errorf("splitHostPort(%q) = (%q, %q), want %v", in, h, p, want)
		}
	}
}

// itoa is a tiny int-to-string helper for the date/time tests.
func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	if n < 0 {
		return "-" + itoa(-n)
	}
	var b []byte
	for n > 0 {
		b = append([]byte{byte('0' + n%10)}, b...)
		n /= 10
	}
	return string(b)
}
