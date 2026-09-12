// Package livefeed continuously drives synthetic sessions through the real
// ingest pipeline so the dashboard shows genuine, varied live activity without
// an operator manually clicking scenarios (report.md §21 "replayable event
// stream", §22 "dataset should create multiple populations").
//
// It reuses the same synthetic generator as the offline evaluation dataset
// (backend/simulator), so live traffic and evaluation traffic are drawn from
// one realistic population — just replayed at human pace instead of in bulk.
package livefeed

import (
	"context"
	"fmt"
	"log/slog"
	"math/rand/v2"
	"time"

	"github.com/holiday-heartbreaks/parallax/backend/internal/ingest"
	"github.com/holiday-heartbreaks/parallax/backend/internal/session"
	"github.com/holiday-heartbreaks/parallax/backend/internal/stream"
	"github.com/holiday-heartbreaks/parallax/backend/simulator"
)

// synthetic customer names — cosmetic variety only, never real people.
var syntheticUsers = []string{
	"user_amaka", "user_chidi", "user_ngozi", "user_tunde", "user_fatima",
	"user_bola", "user_ifeoma", "user_kelechi", "user_yusuf", "user_zainab",
	"user_emeka", "user_grace", "user_hassan", "user_blessing", "user_seun",
}

// Options configures a Feed.
type Options struct {
	Normalizer *ingest.Normalizer
	Sessions   *session.Store
	Bus        *stream.Bus
	Logger     *slog.Logger
	Seed       int64

	// SessionGap bounds the pause between synthetic sessions. Zero uses a
	// sensible default (3-8s) so traffic reads as occasional, not a flood.
	MinSessionGap, MaxSessionGap time.Duration
	// EventGap bounds the pause between events within one session.
	MinEventGap, MaxEventGap time.Duration
}

// Feed generates and replays synthetic sessions until its context is done.
type Feed struct {
	normalizer *ingest.Normalizer
	sessions   *session.Store
	bus        *stream.Bus
	logger     *slog.Logger
	gen        *simulator.Generator

	minSessionGap, maxSessionGap time.Duration
	minEventGap, maxEventGap     time.Duration
}

// New builds a Feed. Sessions and Normalizer are required; Bus may be nil
// (events are still stored, just not published).
func New(opts Options) *Feed {
	logger := opts.Logger
	if logger == nil {
		logger = slog.Default()
	}
	seed := opts.Seed
	if seed == 0 {
		seed = time.Now().UnixNano()
	}
	f := &Feed{
		normalizer:    opts.Normalizer,
		sessions:      opts.Sessions,
		bus:           opts.Bus,
		logger:        logger,
		gen:           simulator.NewGenerator(seed),
		minSessionGap: opts.MinSessionGap,
		maxSessionGap: opts.MaxSessionGap,
		minEventGap:   opts.MinEventGap,
		maxEventGap:   opts.MaxEventGap,
	}
	if f.minSessionGap <= 0 {
		f.minSessionGap = 3 * time.Second
	}
	if f.maxSessionGap <= f.minSessionGap {
		f.maxSessionGap = f.minSessionGap + 5*time.Second
	}
	if f.minEventGap <= 0 {
		f.minEventGap = 200 * time.Millisecond
	}
	if f.maxEventGap <= f.minEventGap {
		f.maxEventGap = f.minEventGap + 450*time.Millisecond
	}
	return f
}

// Run drives the feed until ctx is cancelled. Intended to run in its own
// goroutine for the lifetime of the process.
func (f *Feed) Run(ctx context.Context) {
	f.logger.Info("live feed starting")
	defer f.logger.Info("live feed stopped")

	for {
		select {
		case <-ctx.Done():
			return
		case <-time.After(jitter(f.minSessionGap, f.maxSessionGap)):
		}
		f.runOneSession(ctx)
	}
}

func (f *Feed) runOneSession(ctx context.Context) {
	class, sess := f.generate()
	f.logger.Info("live feed session", "class", class, "session", sess.SessionID, "events", len(sess.Events))

	for _, raw := range sess.Events {
		select {
		case <-ctx.Done():
			return
		case <-time.After(jitter(f.minEventGap, f.maxEventGap)):
		}

		norm, err := f.normalizer.Normalize(raw)
		if err != nil {
			f.logger.Warn("live feed produced an invalid event", "err", err)
			continue
		}
		stored, _ := f.sessions.Append(norm)
		if f.bus != nil {
			f.bus.Publish(stored)
		}
	}
}

// classWeights biases the live demo toward an interesting mix rather than
// report.md's real-world (mostly-legitimate) population — nobody watching a
// dashboard wants to wait ten minutes to see an attack.
const (
	pLegitimate = 0.55
	pTakeover   = 0.15
	pSocialEng  = 0.20
	// remainder (0.10) is accidental
)

// generate produces one freshly-identified labeled session.
func (f *Feed) generate() (string, simulator.LabeledSession) {
	r := rand.Float64()
	var d simulator.Dataset
	var class string
	switch {
	case r < pLegitimate:
		d = f.gen.GenerateDataset(1, 0, 0, 0)
		class = "legitimate"
	case r < pLegitimate+pTakeover:
		d = f.gen.GenerateDataset(0, 1, 0, 0)
		class = "account_takeover"
	case r < pLegitimate+pTakeover+pSocialEng:
		d = f.gen.GenerateDataset(0, 0, 1, 0)
		class = "social_engineering"
	default:
		d = f.gen.GenerateDataset(0, 0, 0, 1)
		class = "accidental"
	}

	var sess simulator.LabeledSession
	switch class {
	case "legitimate":
		sess = d.LegitimateSessions[0]
	case "account_takeover":
		sess = d.AccountTakeoverSessions[0]
	case "social_engineering":
		sess = d.SocialEngineeringSessions[0]
	default:
		sess = d.AccidentalSessions[0]
	}

	rewriteIdentity(&sess)
	return class, sess
}

// rewriteIdentity gives a templated session (the generator reuses fixed ids
// like "sess_legit_00000" per class) a fresh unique id, a varied synthetic
// user, and timestamps anchored to now — preserving relative spacing between
// events — so repeated ticks don't collide on the same session or user.
func rewriteIdentity(sess *simulator.LabeledSession) {
	freshID := fmt.Sprintf("%s_%d%03d", sess.SessionID, time.Now().UnixNano(), rand.IntN(1000))
	freshUser := syntheticUsers[rand.IntN(len(syntheticUsers))]

	var base int64
	if len(sess.Events) > 0 {
		base = sess.Events[0].Timestamp
	}
	now := time.Now().Unix()

	sess.SessionID = freshID
	sess.UserID = freshUser
	for i := range sess.Events {
		delta := sess.Events[i].Timestamp - base
		sess.Events[i].SessionID = freshID
		sess.Events[i].UserID = freshUser
		sess.Events[i].EventID = fmt.Sprintf("%s_ev%d", freshID, i+1)
		sess.Events[i].Timestamp = now + delta
	}
}

func jitter(min, max time.Duration) time.Duration {
	if max <= min {
		return min
	}
	return min + time.Duration(rand.Int64N(int64(max-min)))
}
