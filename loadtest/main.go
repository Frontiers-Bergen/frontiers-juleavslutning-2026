// Command loadtest finds out how many button smashers one server can take.
//
// It builds the app from this checkout and runs its own copy on this machine, with a fresh database in a temp
// dir (like browser-tools/browser_check.mjs). Simulated phones then do what real phones do on the evening: join
// all at once (everyone scans the QR code), hold the page, emoji and snow streams, and smash the emoji buttons at
// a human pace during a tug of war round. The presenter's screen watches too.
//
// Never point it at the deployed app: the Hetzner server is shared with other services. Run it locally only.
//
// The number that matters is smash-to-screen time: from sending a smash until each open screen has received it.
// A probe phone smashes ❤️ five times a second (the simulated phones never smash ❤️), and every screen notes when
// its ❤️ count reaches each probe smash. Under 100 ms feels instant (see "Page life cycle" in AGENTS.md).
//
// The load generator runs on the same machine and needs CPU too. Use -server-cores to give the app only as many
// cores as a small server has; the rest are left for the simulated phones.
package main

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"math/rand/v2"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
	"slices"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/andybalholm/brotli"
)

const (
	inviteKey    = "dev-key"
	presenterKey = "loadtest-key"
	appPort      = "8098"

	probeEmoji  = "❤️"
	probeSignal = "HeartEmojiCount"
	probeEvery  = 200 * time.Millisecond

	// Responses under 100 ms feel instant (Jakob Nielsen); the goal for smash-to-screen time at the 99th percentile.
	instant = 100 * time.Millisecond
	// DefaultTugOfWarRules in pages/root/tug_of_war.templ.
	tugOfWarCountdown = 5 * time.Second
)

// The simulated phones smash these; ❤️ is the probe's alone.
var smashEmojis = []string{"👍", "🎉", "😂", "🔥"}

var appURL = "http://localhost:" + appPort + "/" + inviteKey

type settings struct {
	phones      int
	taps        float64
	smashFor    time.Duration
	serverCores int
}

func main() {
	scenario := flag.String("scenario", "evening", "evening: one tug of war round; ramp: double the phones until smash-to-screen time is no longer instant")
	phones := flag.Int("phones", 40, "simulated phones (the first step of a ramp)")
	maxPhones := flag.Int("max-phones", 1280, "the ramp stops here at the latest")
	taps := flag.Float64("taps", 6, "emoji taps per second per phone, on average (a determined thumb manages 6 to 8)")
	smashFor := flag.Duration("smash", 20*time.Second, "how long the phones smash (a tug of war round is 20 s)")
	serverCores := flag.Int("server-cores", 0, "give the app only this many CPU cores (GOMAXPROCS), like a small server; 0 is all")
	flag.Parse()

	if err := run(*scenario, settings{*phones, *taps, *smashFor, *serverCores}, *maxPhones); err != nil {
		fmt.Fprintln(os.Stderr, "loadtest:", err)
		os.Exit(1)
	}
}

func run(scenario string, base settings, maxPhones int) (err error) {
	work, err := os.MkdirTemp("", "loadtest-")
	if err != nil {
		return err
	}
	defer func() { err = errors.Join(err, os.RemoveAll(work)) }()
	binary, err := buildApp(work)
	if err != nil {
		return err
	}

	switch scenario {
	case "evening":
		result, err := runRound(binary, work, base)
		if err != nil {
			return err
		}
		result.print(base)
		return nil
	case "ramp":
		var results []roundResult
		for phones := base.phones; phones <= maxPhones; phones *= 2 {
			step := base
			step.phones = phones
			result, err := runRound(binary, work, step)
			if err != nil {
				return err
			}
			result.print(step)
			results = append(results, result)
			if !result.instant() {
				break
			}
		}
		printRamp(base, results)
		return nil
	default:
		return fmt.Errorf("unknown scenario %q (evening or ramp)", scenario)
	}
}

func buildApp(work string) (string, error) {
	binary := filepath.Join(work, "app")
	if runtime.GOOS == "windows" {
		binary += ".exe"
	}
	for _, command := range [][]string{
		{"go", "tool", "templ", "generate", "-log-level", "error"},
		{"go", "build", "-o", binary, "."},
	} {
		cmd := exec.Command(command[0], command[1:]...)
		cmd.Stdout, cmd.Stderr = os.Stdout, os.Stderr
		if err := cmd.Run(); err != nil {
			return "", fmt.Errorf("%s: %w", strings.Join(command, " "), err)
		}
	}
	return binary, nil
}

// --- The app under test ---

type app struct {
	cmd     *exec.Cmd
	log     *os.File
	workDir string
}

// startApp runs a fresh copy of the app: its own directory, so a new database and no .env.
func startApp(binary, work string, serverCores int) (*app, error) {
	dir, err := os.MkdirTemp(work, "app-")
	if err != nil {
		return nil, err
	}
	log, err := os.Create(filepath.Join(dir, "app.log"))
	if err != nil {
		return nil, err
	}
	cmd := exec.Command(binary)
	cmd.Dir = dir
	cmd.Env = append(os.Environ(), "PORT="+appPort, "HOST=127.0.0.1", "INVITE_KEY="+inviteKey, "PRESENTER_KEY="+presenterKey)
	if serverCores > 0 {
		cmd.Env = append(cmd.Env, "GOMAXPROCS="+strconv.Itoa(serverCores))
	}
	cmd.Stdout, cmd.Stderr = log, log
	if err := cmd.Start(); err != nil {
		return nil, errors.Join(err, log.Close())
	}
	started := &app{cmd: cmd, log: log, workDir: dir}
	for range 80 {
		response, err := http.Get(appURL + "/")
		if err == nil {
			if err := response.Body.Close(); err != nil {
				return nil, errors.Join(err, started.stop())
			}
			return started, nil
		}
		time.Sleep(250 * time.Millisecond)
	}
	return nil, errors.Join(errors.New("the app did not come up"), started.stop())
}

func (a *app) stop() error {
	killErr := a.cmd.Process.Kill()
	// Wait reports the kill itself as an *exec.ExitError; that is the expected end, not a failure.
	var exitErr *exec.ExitError
	if waitErr := a.cmd.Wait(); waitErr != nil && !errors.As(waitErr, &exitErr) {
		killErr = errors.Join(killErr, waitErr)
	}
	return errors.Join(killErr, a.log.Close())
}

// loggedErrors returns the app's ERROR log lines.
func (a *app) loggedErrors() ([]string, error) {
	log, err := os.ReadFile(filepath.Join(a.workDir, "app.log"))
	if err != nil {
		return nil, err
	}
	var errorLines []string
	for line := range strings.Lines(string(log)) {
		if strings.Contains(line, `"level":"ERROR"`) {
			errorLines = append(errorLines, strings.TrimSpace(line))
		}
	}
	return errorLines, nil
}

// cpuTime is the app's CPU time so far. Linux only (from /proc); elsewhere it is zero and not reported.
func (a *app) cpuTime() time.Duration {
	stat, err := os.ReadFile(fmt.Sprintf("/proc/%d/stat", a.cmd.Process.Pid))
	if err != nil {
		return 0
	}
	// The fields after the process name; utime and stime are fields 14 and 15, in clock ticks (100 a second).
	fields := strings.Fields(string(stat[strings.LastIndexByte(string(stat), ')')+1:]))
	if len(fields) < 13 {
		return 0
	}
	user, userErr := strconv.ParseInt(fields[11], 10, 64)
	system, systemErr := strconv.ParseInt(fields[12], 10, 64)
	if userErr != nil || systemErr != nil {
		return 0
	}
	return time.Duration(user+system) * 10 * time.Millisecond
}

// peakMemory is the app's peak resident memory in MB. Linux only, like cpuTime.
func (a *app) peakMemory() int {
	status, err := os.ReadFile(fmt.Sprintf("/proc/%d/status", a.cmd.Process.Pid))
	if err != nil {
		return 0
	}
	for line := range strings.Lines(string(status)) {
		if kilobytes, found := strings.CutPrefix(line, "VmHWM:"); found {
			value, err := strconv.Atoi(strings.TrimSuffix(strings.TrimSpace(kilobytes), " kB"))
			if err != nil {
				return 0
			}
			return value / 1024
		}
	}
	return 0
}

// --- One tug of war round ---

type roundResult struct {
	joinTimes    []time.Duration // from opening the deck to the first live page update
	smashTimes   []time.Duration // a smash POST, there and back
	screenTimes  []time.Duration // from sending a probe smash to a screen receiving it
	teams        map[string]int
	smashes      int64
	failedSmash  int64
	counted      int64
	probes       int64
	balloons     int64
	pagePatches  int64
	streamBytes  int64
	brokenStream int64
	settled      time.Duration // after the last smash, until every screen shows the final counts; 0 if never
	cpuCores     float64
	peakMemoryMB int
	errorsLogged []string
	finalPulls   int64
}

func runRound(binary, work string, s settings) (result roundResult, err error) {
	app, err := startApp(binary, work, s.serverCores)
	if err != nil {
		return result, err
	}
	defer func() { err = errors.Join(err, app.stop()) }()

	ctx, closeStreams := context.WithCancel(context.Background())
	defer closeStreams()
	transport := &http.Transport{MaxIdleConnsPerHost: 100_000, DisableCompression: true}
	stats := &streamStats{}
	probe := &probe{}

	presenter, err := newPresenter(transport)
	if err != nil {
		return result, err
	}
	if err := moveToTugOfWar(presenter); err != nil {
		return result, err
	}
	screens := []*screen{newScreen(presenter, stats, probe)}
	for range s.phones {
		phone, err := newPhone(transport)
		if err != nil {
			return result, err
		}
		screens = append(screens, newScreen(phone, stats, probe))
	}

	// Everyone joins at once, like when the QR code goes up.
	for _, screen := range screens {
		go screen.join(ctx)
	}
	for _, screen := range screens {
		select {
		case <-screen.firstPage:
		case <-time.After(30 * time.Second):
			return result, errors.New("not every phone got its first page within 30 s")
		}
	}

	if status, err := post(presenter, "/root/api/tug-of-war/start", ""); err != nil || status != http.StatusNoContent {
		return result, fmt.Errorf("starting the tug of war: status %d, %w", status, err)
	}
	time.Sleep(tugOfWarCountdown + 100*time.Millisecond)

	// Smash.
	cpuBefore, smashStart := app.cpuTime(), time.Now()
	deadline := smashStart.Add(s.smashFor)
	var smashers sync.WaitGroup
	var smashes smashStats
	for _, screen := range screens[1:] {
		smashers.Go(func() { smashUntil(screen.client, deadline, s.taps, &smashes) })
	}
	probeClient, err := newPhone(transport)
	if err != nil {
		return result, err
	}
	smashers.Go(func() { probe.smashUntil(probeClient, deadline) })
	smashers.Wait()
	smashEnd := time.Now()
	cpuAfter := app.cpuTime()

	// Wait until every screen shows the final counts.
	finalPulls, err := currentPulls()
	if err != nil {
		return result, err
	}
	expectedSmashes := smashes.ok.Load()
	for time.Since(smashEnd) < 15*time.Second {
		if !slices.ContainsFunc(screens, func(screen *screen) bool {
			return !screen.shows(probe.count(), expectedSmashes, finalPulls)
		}) {
			result.settled = max(time.Since(smashEnd), time.Millisecond)
			break
		}
		time.Sleep(5 * time.Millisecond)
	}
	// Read the errors before the phones leave: a stream closed in the middle of sending logs an error too.
	if result.errorsLogged, err = app.loggedErrors(); err != nil {
		return result, err
	}
	closeStreams()

	result.peakMemoryMB = app.peakMemory()
	if cpu := cpuAfter - cpuBefore; cpu > 0 {
		result.cpuCores = cpu.Seconds() / smashEnd.Sub(smashStart).Seconds()
	}
	result.teams = map[string]int{}
	for _, screen := range screens {
		if team, ok := screen.team.Load().(string); ok {
			result.teams[team]++
		}
	}
	for _, screen := range screens[1:] {
		result.joinTimes = append(result.joinTimes, screen.joinTime)
	}
	result.smashTimes = smashes.times
	result.screenTimes = stats.screenTimes
	result.smashes, result.failedSmash = smashes.ok.Load()+smashes.failed.Load(), smashes.failed.Load()
	result.counted = screens[0].smashCount.Load()
	result.probes = probe.count()
	result.balloons = stats.balloons.Load()
	result.pagePatches = stats.pagePatches.Load()
	result.streamBytes = stats.bytes.Load()
	result.brokenStream = stats.broken.Load()
	result.finalPulls = finalPulls
	return result, nil
}

// --- Phones and the presenter ---

type client struct{ *http.Client }

func newPhone(transport *http.Transport) (client, error) {
	jar, err := cookiejar.New(nil)
	if err != nil {
		return client{}, err
	}
	return client{&http.Client{Transport: transport, Jar: jar}}, nil
}

func newPresenter(transport *http.Transport) (client, error) {
	presenter, err := newPhone(transport)
	if err != nil {
		return client{}, err
	}
	login, err := presenter.PostForm("http://localhost:"+appPort+"/presenter/login", url.Values{"key": {presenterKey}})
	if err != nil {
		return client{}, err
	}
	return presenter, drain(login)
}

// moveToTugOfWar steps through the deck until the tug of war slide is shown, like browser_check.mjs.
func moveToTugOfWar(presenter client) error {
	for index := range 100 {
		if status, err := post(presenter, fmt.Sprintf("/root/api/slides/set?index=%d", index), ""); err != nil || status != http.StatusNoContent {
			return fmt.Errorf("moving to slide %d: status %d, %w", index, status, err)
		}
		page, err := attendeePage()
		if err != nil {
			return err
		}
		if strings.Contains(page, "data-tug-of-war-phase") {
			return nil
		}
	}
	return errors.New("no tug of war slide in the deck")
}

// attendeePage is the deck as a new phone sees it: only the active slide.
func attendeePage() (string, error) {
	response, err := http.Get(appURL + "/")
	if err != nil {
		return "", err
	}
	body, err := io.ReadAll(response.Body)
	return string(body), errors.Join(err, response.Body.Close())
}

var (
	tealPulls   = regexp.MustCompile(`data-teal-pulls="(\d+)"`)
	orangePulls = regexp.MustCompile(`data-orange-pulls="(\d+)"`)
	teamPattern = regexp.MustCompile(`data-team="([a-z]+)"`)
)

func pulls(page string) int64 {
	var total int64
	for _, pattern := range []*regexp.Regexp{tealPulls, orangePulls} {
		if match := pattern.FindStringSubmatch(page); match != nil {
			count, err := strconv.ParseInt(match[1], 10, 64)
			if err == nil {
				total += count
			}
		}
	}
	return total
}

func currentPulls() (int64, error) {
	page, err := attendeePage()
	return pulls(page), err
}

func post(phone client, path, body string) (int, error) {
	response, err := phone.Post(appURL+path, "application/json", strings.NewReader(body))
	if err != nil {
		return 0, err
	}
	return response.StatusCode, drain(response)
}

func drain(response *http.Response) error {
	_, err := io.Copy(io.Discard, response.Body)
	return errors.Join(err, response.Body.Close())
}

type smashStats struct {
	ok, failed atomic.Int64
	mu         sync.Mutex
	times      []time.Duration
}

// smashUntil taps the emoji buttons at about taps a second, unevenly, like a thumb.
func smashUntil(phone client, deadline time.Time, taps float64, stats *smashStats) {
	interval := time.Duration(float64(time.Second) / taps)
	for {
		time.Sleep(time.Duration(float64(interval) * (0.5 + rand.Float64())))
		if time.Now().After(deadline) {
			return
		}
		emoji := smashEmojis[rand.IntN(len(smashEmojis))]
		sent := time.Now()
		status, err := post(phone, "/root/api/emoji/increment", `{"emoji":"`+emoji+`"}`)
		took := time.Since(sent)
		if err != nil || status != http.StatusNoContent {
			stats.failed.Add(1)
			continue
		}
		stats.ok.Add(1)
		stats.mu.Lock()
		stats.times = append(stats.times, took)
		stats.mu.Unlock()
	}
}

// probe smashes ❤️ and remembers when each smash was sent, so screens can tell how long it took to reach them.
type probe struct {
	mu     sync.Mutex
	sentAt []time.Time // sentAt[i] is when the probe's smash number i+1 was sent
}

func (p *probe) smashUntil(phone client, deadline time.Time) {
	for time.Now().Before(deadline) {
		p.mu.Lock()
		p.sentAt = append(p.sentAt, time.Now())
		p.mu.Unlock()
		if status, err := post(phone, "/root/api/emoji/increment", `{"emoji":"`+probeEmoji+`"}`); err != nil || status != http.StatusNoContent {
			fmt.Fprintf(os.Stderr, "probe smash failed: status %d, %v\n", status, err)
			p.mu.Lock()
			p.sentAt = p.sentAt[:len(p.sentAt)-1]
			p.mu.Unlock()
		}
		time.Sleep(probeEvery)
	}
}

func (p *probe) count() int64 {
	p.mu.Lock()
	defer p.mu.Unlock()
	return int64(len(p.sentAt))
}

func (p *probe) sentTime(smash int64) (time.Time, bool) {
	p.mu.Lock()
	defer p.mu.Unlock()
	if smash < 1 || smash > int64(len(p.sentAt)) {
		return time.Time{}, false
	}
	return p.sentAt[smash-1], true
}

// --- Screens: what every open page receives ---

type streamStats struct {
	balloons, pagePatches, bytes, broken atomic.Int64
	mu                                   sync.Mutex
	screenTimes                          []time.Duration
}

type screen struct {
	client
	stats *streamStats
	probe *probe

	joinStart time.Time
	firstPage chan struct{}
	firstOnce sync.Once
	joinTime  time.Duration
	team      atomic.Value // the team shown on the last page update

	probeSeen  atomic.Int64 // the ❤️ count on the screen
	smashCount atomic.Int64 // the other emojis' total on the screen
	pullsShown atomic.Int64 // teal plus orange pulls on the screen
}

func newScreen(phone client, stats *streamStats, probe *probe) *screen {
	return &screen{client: phone, stats: stats, probe: probe, firstPage: make(chan struct{})}
}

// join opens the deck and its three live streams, as the page's data-init attributes do.
func (s *screen) join(ctx context.Context) {
	s.joinStart = time.Now()
	response, err := s.Get(appURL + "/")
	if err != nil || drain(response) != nil {
		s.stats.broken.Add(1)
		return
	}
	go s.follow(ctx, "/root/api/", s.onPage)
	go s.follow(ctx, "/root/api/emoji/sse", s.onEmoji)
	go s.follow(ctx, "/root/api/snow/sse", func(string, string) {})
}

func (s *screen) shows(probes, smashes, pulls int64) bool {
	return s.probeSeen.Load() == probes && s.smashCount.Load() == smashes && s.pullsShown.Load() == pulls
}

func (s *screen) onPage(event, data string) {
	if event != "datastar-patch-elements" {
		return
	}
	s.stats.pagePatches.Add(1)
	s.pullsShown.Store(pulls(data))
	// Teams are drawn again when the round starts, so keep the latest.
	if match := teamPattern.FindStringSubmatch(data); match != nil {
		s.team.Store(match[1])
	}
	s.firstOnce.Do(func() {
		s.joinTime = time.Since(s.joinStart)
		close(s.firstPage)
	})
}

func (s *screen) onEmoji(event, data string) {
	switch event {
	case "datastar-patch-elements":
		s.stats.balloons.Add(1)
	case "datastar-patch-signals":
		var counts map[string]int64
		if err := json.Unmarshal([]byte(strings.TrimPrefix(data, "signals ")), &counts); err != nil {
			s.stats.broken.Add(1)
			return
		}
		now := time.Now()
		var others int64
		for name, count := range counts {
			if name != probeSignal {
				others += count
			}
		}
		s.smashCount.Store(others)
		seen := s.probeSeen.Load()
		for smash := seen + 1; smash <= counts[probeSignal]; smash++ {
			if sent, ok := s.probe.sentTime(smash); ok {
				s.stats.mu.Lock()
				s.stats.screenTimes = append(s.stats.screenTimes, now.Sub(sent))
				s.stats.mu.Unlock()
			}
		}
		s.probeSeen.Store(max(seen, counts[probeSignal]))
	}
}

// follow reads a Brotli-compressed Datastar SSE stream, as a phone's browser does, until ctx ends.
func (s *screen) follow(ctx context.Context, path string, onEvent func(event, data string)) {
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, appURL+path, nil)
	if err != nil {
		s.stats.broken.Add(1)
		return
	}
	request.Header.Set("Accept", "text/event-stream")
	request.Header.Set("Accept-Encoding", "br")
	response, err := s.Do(request)
	if err != nil {
		if ctx.Err() == nil {
			s.stats.broken.Add(1)
		}
		return
	}
	defer func() {
		if err := response.Body.Close(); err != nil && ctx.Err() == nil {
			s.stats.broken.Add(1)
		}
	}()
	if response.StatusCode != http.StatusOK || response.Header.Get("Content-Encoding") != "br" {
		s.stats.broken.Add(1)
		return
	}
	lines := bufio.NewScanner(brotli.NewReader(&countingReader{response.Body, &s.stats.bytes}))
	lines.Buffer(make([]byte, 0, 64*1024), 4*1024*1024)
	var event string
	var data []string
	for lines.Scan() {
		line := lines.Text()
		switch {
		case line == "":
			onEvent(event, strings.Join(data, "\n"))
			event, data = "", data[:0]
		case strings.HasPrefix(line, "event: "):
			event = strings.TrimPrefix(line, "event: ")
		case strings.HasPrefix(line, "data: "):
			data = append(data, strings.TrimPrefix(line, "data: "))
		}
	}
	if ctx.Err() == nil {
		s.stats.broken.Add(1)
	}
}

type countingReader struct {
	io.Reader
	total *atomic.Int64
}

func (r *countingReader) Read(p []byte) (int, error) {
	n, err := r.Reader.Read(p)
	r.total.Add(int64(n))
	return n, err
}

// --- Report ---

func (r roundResult) instant() bool {
	return percentile(r.screenTimes, 99) < instant && r.counted == r.smashes-r.failedSmash && r.failedSmash == 0 && r.settled > 0
}

func (r roundResult) print(s settings) {
	cores := "all cores"
	if s.serverCores > 0 {
		cores = fmt.Sprintf("%d cores", s.serverCores)
	}
	screens := int64(s.phones + 1)
	seconds := s.smashFor.Seconds()
	fmt.Printf("\n%d phones, %.0f taps/s each, %s of smashing, the app on %s\n", s.phones, s.taps, s.smashFor, cores)
	fmt.Printf("  joined          first live page after p50 %s, p99 %s; teams this round (with the presenter): teal %d, orange %d\n",
		ms(percentile(r.joinTimes, 50)), ms(percentile(r.joinTimes, 99)), r.teams["teal"], r.teams["orange"])
	fmt.Printf("  smashes         %d sent (%.0f/s), %d failed; POST p50 %s, p99 %s\n",
		r.smashes, float64(r.smashes)/seconds, r.failedSmash, ms(percentile(r.smashTimes, 50)), ms(percentile(r.smashTimes, 99)))
	fmt.Printf("  smash→screen    p50 %s, p95 %s, p99 %s, max %s  (%d probe smashes × %d screens)  %s\n",
		ms(percentile(r.screenTimes, 50)), ms(percentile(r.screenTimes, 95)), ms(percentile(r.screenTimes, 99)),
		ms(percentile(r.screenTimes, 100)), r.probes, screens, verdict(percentile(r.screenTimes, 99) < instant))
	roundEnd := ""
	if r.finalPulls < r.counted+r.probes {
		// Fewer page re-renders after that, so compare rounds that ran their full time.
		roundEnd = " (the round ended early: a team reached the winning lead)"
	}
	fmt.Printf("  counted         %d of %d smashes  %s; tug of war pulls: %d%s\n",
		r.counted, r.smashes-r.failedSmash, verdict(r.counted == r.smashes-r.failedSmash), r.finalPulls, roundEnd)
	expectedBalloons := (r.smashes - r.failedSmash + r.probes) * screens
	fmt.Printf("  balloons        %.1f%% arrived  (NATS drops balloons for a screen that falls behind; counts stay right)\n",
		100*float64(r.balloons)/float64(max(expectedBalloons, 1)))
	fmt.Printf("  per screen      %.1f page updates/s, %.1f KB/s compressed over 3 streams; %d streams broke\n",
		float64(r.pagePatches)/float64(screens)/seconds, float64(r.streamBytes)/1024/float64(screens)/seconds, r.brokenStream)
	server := fmt.Sprintf("%d errors logged", len(r.errorsLogged))
	if r.cpuCores > 0 {
		server = fmt.Sprintf("%.1f cores busy, %d MB peak memory, ", r.cpuCores, r.peakMemoryMB) + server
	}
	fmt.Printf("  server          %s\n", server)
	for _, line := range r.errorsLogged[:min(3, len(r.errorsLogged))] {
		fmt.Printf("                  %s\n", line)
	}
	if r.settled > 0 {
		fmt.Printf("  settled         every screen showed the final counts %s after the last smash\n", ms(r.settled))
	} else {
		fmt.Printf("  settled         ❌ not every screen showed the final counts within 15 s\n")
	}
}

func printRamp(s settings, results []roundResult) {
	fmt.Printf("\nRamp, %.0f taps/s per phone:\n", s.taps)
	fmt.Printf("  %7s  %9s  %12s  %12s  %10s  %s\n", "phones", "smashes/s", "screen p50", "screen p99", "app cores", "")
	phones := s.phones
	for _, r := range results {
		fmt.Printf("  %7d  %9.0f  %12s  %12s  %10.1f  %s\n", phones, float64(r.smashes)/s.smashFor.Seconds(),
			ms(percentile(r.screenTimes, 50)), ms(percentile(r.screenTimes, 99)), r.cpuCores, verdict(r.instant()))
		phones *= 2
	}
}

func percentile(times []time.Duration, p float64) time.Duration {
	if len(times) == 0 {
		return 0
	}
	sorted := slices.Sorted(slices.Values(times))
	return sorted[min(len(sorted)-1, int(float64(len(sorted))*p/100))]
}

func ms(d time.Duration) string {
	return fmt.Sprintf("%.1f ms", float64(d.Microseconds())/1000)
}

func verdict(good bool) string {
	if good {
		return "✅"
	}
	return "❌"
}
