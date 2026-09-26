package worker_test

import (
	"context"
	"encoding/json"
	"errors"
	"sync"
	"testing"
	"time"

	amqp "github.com/rabbitmq/amqp091-go"
	"go.uber.org/mock/gomock"

	mediasvcmocks "github.com/labib0x9/ffgif/internal/app/media/mocks"
	mailermocks "github.com/labib0x9/ffgif/internal/port/mailer/mocks"
	"github.com/labib0x9/ffgif/internal/port/queue"
	queuemocks "github.com/labib0x9/ffgif/internal/port/queue/mocks"
	"github.com/labib0x9/ffgif/internal/worker"
)

// ---------------------------------------------------------------------------
// a recording amqp.Acknowledger
// ---------------------------------------------------------------------------

type ackCall struct {
	op       string // "ack" | "nack" | "reject"
	multiple bool
	requeue  bool
}

// recordingAcker stands in for the channel a real Delivery is bound to, so the
// worker's disposition of each message is observable without a broker.
type recordingAcker struct {
	mu    sync.Mutex
	calls []ackCall
	done  chan struct{}
	once  sync.Once
}

func newAcker() *recordingAcker {
	return &recordingAcker{done: make(chan struct{})}
}

func (a *recordingAcker) record(c ackCall) error {
	a.mu.Lock()
	a.calls = append(a.calls, c)
	a.mu.Unlock()
	a.once.Do(func() { close(a.done) })
	return nil
}

func (a *recordingAcker) Ack(tag uint64, multiple bool) error {
	return a.record(ackCall{op: "ack", multiple: multiple})
}

func (a *recordingAcker) Nack(tag uint64, multiple, requeue bool) error {
	return a.record(ackCall{op: "nack", multiple: multiple, requeue: requeue})
}

func (a *recordingAcker) Reject(tag uint64, requeue bool) error {
	return a.record(ackCall{op: "reject", requeue: requeue})
}

// wait blocks until the worker has dispositioned a message, or fails the test.
func (a *recordingAcker) wait(t *testing.T) ackCall {
	t.Helper()
	select {
	case <-a.done:
	case <-time.After(3 * time.Second):
		t.Fatal("the worker never acked or nacked the delivery")
	}
	a.mu.Lock()
	defer a.mu.Unlock()
	if len(a.calls) == 0 {
		t.Fatal("no acknowledgement recorded")
	}
	return a.calls[0]
}

// deathHeader builds the x-death header RabbitMQ attaches to a message that has
// already been dead-lettered `count` times.
func deathHeader(count int64) amqp.Table {
	return amqp.Table{
		"x-death": []any{
			amqp.Table{
				"count":  count,
				"reason": "rejected",
				"queue":  "process",
			},
		},
	}
}

func delivery(t *testing.T, acker amqp.Acknowledger, headers amqp.Table, payload any) amqp.Delivery {
	t.Helper()
	body, err := json.Marshal(payload)
	if err != nil {
		t.Fatalf("marshal payload: %v", err)
	}
	return amqp.Delivery{
		Acknowledger: acker,
		DeliveryTag:  1,
		Headers:      headers,
		Body:         body,
	}
}

func rawDelivery(acker amqp.Acknowledger, body string) amqp.Delivery {
	return amqp.Delivery{Acknowledger: acker, DeliveryTag: 1, Body: []byte(body)}
}

// ===========================================================================
// Video conversion worker
// ===========================================================================

type videoHarness struct {
	q    *queuemocks.MockQueue
	srv  *mediasvcmocks.MockService
	ch   chan amqp.Delivery
	stop context.CancelFunc
	wg   *sync.WaitGroup
}

func startVideoWorker(t *testing.T) *videoHarness {
	t.Helper()
	ctrl := gomock.NewController(t)
	h := &videoHarness{
		q:   queuemocks.NewMockQueue(ctrl),
		srv: mediasvcmocks.NewMockService(ctrl),
		ch:  make(chan amqp.Delivery, 1),
		wg:  &sync.WaitGroup{},
	}

	ctx, cancel := context.WithCancel(context.Background())
	h.stop = cancel

	h.q.EXPECT().
		ConsumeVideo(gomock.Any(), gomock.Eq("video-worker"), gomock.Eq(2)).
		Return((<-chan amqp.Delivery)(h.ch), nil).
		Times(1)
	h.q.EXPECT().CloseConsumerChannel(gomock.Eq("video-worker")).Return(nil).AnyTimes()

	w := worker.NewVideoWorker(h.srv, h.q)
	h.wg.Add(1)
	go func() {
		defer h.wg.Done()
		_ = w.Run(ctx, "video-worker", 2)
	}()

	t.Cleanup(func() {
		cancel()
		h.wg.Wait()
	})
	return h
}

// EXPECTED TO FAIL: internal/worker/converter_worker.go answers every
// processing failure with `d.Nack(false, false)` — requeue=false, i.e. straight
// to the dead-letter queue on the very first attempt. The retryCount() helper
// that reads the x-death header is commented out at the bottom of the file and
// the maxRetries field (set to 2 in NewVideoWorker) is never read.
//
// A transient failure — ffmpeg OOM, MinIO blip, a network hiccup — permanently
// loses the user's conversion instead of being retried.
//
// Contract: while x-death count < maxRetries, a failure must requeue
// (Nack(false, true)); only once the budget is spent may it dead-letter.
func TestVideoWorker_TransientFailureIsRetriedNotDeadLettered(t *testing.T) {
	h := startVideoWorker(t)
	acker := newAcker()

	h.srv.EXPECT().
		Process(gomock.Any(), gomock.Any()).
		Return(errors.New("ffmpeg: signal killed")).
		Times(1)

	// first redelivery attempt: the message has been dead-lettered 0 times so far
	h.ch <- delivery(t, acker, deathHeader(0), queue.VideoMessage{
		UserID: "user-1", JobId: "job-1", Key: "upload-1", Start: 0, End: 5,
	})

	got := h.acker(t, acker)
	if got.op != "nack" {
		t.Fatalf("disposition = %s, want nack", got.op)
	}
	if !got.requeue {
		t.Error("the first transient failure was dead-lettered immediately " +
			"(Nack requeue=false); the x-death retry budget of 2 is never consulted")
	}
}

func (h *videoHarness) acker(t *testing.T, a *recordingAcker) ackCall {
	t.Helper()
	return a.wait(t)
}

// Once the retry budget is spent the message SHOULD go to the DLQ. This is the
// half the current code gets right, so it must keep working after a fix.
func TestVideoWorker_ExhaustedRetryBudgetDeadLetters(t *testing.T) {
	h := startVideoWorker(t)
	acker := newAcker()

	h.srv.EXPECT().
		Process(gomock.Any(), gomock.Any()).
		Return(errors.New("ffmpeg: still failing")).
		Times(1)

	// already retried twice — maxRetries for the video worker is 2
	h.ch <- delivery(t, acker, deathHeader(2), queue.VideoMessage{JobId: "job-1", Key: "upload-1"})

	got := acker.wait(t)
	if got.op != "nack" {
		t.Fatalf("disposition = %s, want nack", got.op)
	}
	if got.requeue {
		t.Error("a message that exhausted its retry budget was requeued; it will loop forever")
	}
}

// A message that cannot even be parsed can never succeed, so retrying it is
// pointless — it must go straight to the DLQ. The current code gets this right.
func TestVideoWorker_UnparseableMessageIsDeadLetteredImmediately(t *testing.T) {
	h := startVideoWorker(t)
	acker := newAcker()

	// Process must not be called for a body that does not decode.
	h.ch <- rawDelivery(acker, `{"user_id": `)

	got := acker.wait(t)
	if got.op != "nack" {
		t.Fatalf("disposition = %s, want nack", got.op)
	}
	if got.requeue {
		t.Error("a permanently malformed message was requeued and will loop forever")
	}
}

func TestVideoWorker_SuccessAcksTheMessage(t *testing.T) {
	h := startVideoWorker(t)
	acker := newAcker()

	h.srv.EXPECT().
		Process(gomock.Any(), gomock.Any()).
		DoAndReturn(func(_ context.Context, msg queue.VideoMessage) error {
			// the decoded message must reach the service intact
			if msg.JobId != "job-1" || msg.Key != "upload-1" {
				t.Errorf("service got %+v, want JobId=job-1 Key=upload-1", msg)
			}
			if msg.Width != 640 || msg.FPS != 24 {
				t.Errorf("service got width=%d fps=%d, want 640/24", msg.Width, msg.FPS)
			}
			return nil
		}).
		Times(1)

	h.ch <- delivery(t, acker, nil, queue.VideoMessage{
		JobId: "job-1", Key: "upload-1", Width: 640, FPS: 24,
	})

	got := acker.wait(t)
	if got.op != "ack" {
		t.Errorf("disposition = %s, want ack on success", got.op)
	}
	if got.multiple {
		t.Error("Ack(multiple=true) would acknowledge other in-flight deliveries too")
	}
}

// ===========================================================================
// Email worker
// ===========================================================================

type emailHarness struct {
	q      *queuemocks.MockQueue
	mailer *mailermocks.MockEmailSender
	ch     chan amqp.Delivery
	wg     *sync.WaitGroup
}

func startEmailWorker(t *testing.T) *emailHarness {
	t.Helper()
	ctrl := gomock.NewController(t)
	h := &emailHarness{
		q:      queuemocks.NewMockQueue(ctrl),
		mailer: mailermocks.NewMockEmailSender(ctrl),
		ch:     make(chan amqp.Delivery, 1),
		wg:     &sync.WaitGroup{},
	}

	ctx, cancel := context.WithCancel(context.Background())
	h.q.EXPECT().
		ConsumeEmail(gomock.Any(), gomock.Eq("email-worker"), gomock.Eq(1)).
		Return((<-chan amqp.Delivery)(h.ch), nil).
		Times(1)
	h.q.EXPECT().CloseConsumerChannel(gomock.Any()).Return(nil).AnyTimes()

	w := worker.NewEmailWorker(h.q, h.mailer)
	h.wg.Add(1)
	go func() {
		defer h.wg.Done()
		_ = w.Run(ctx, "email-worker", 1)
	}()

	t.Cleanup(func() {
		cancel()
		h.wg.Wait()
	})
	return h
}

// EXPECTED TO FAIL: internal/app/share/create_by_token.go publishes an email job
// with Name: "share", but the switch in internal/worker/email_worker.go has no
// case for it — signup, forgot-password, resend-verify and reset-password only.
// Every share notification therefore falls through to `default:`, is logged as
// an "unknown email job type" and dead-lettered. mailer.EmailSender declares
// SendShareNotification and nothing in the codebase ever calls it.
//
// Net effect: share-by-token links are created and returned by the API but the
// recipient is never emailed.
func TestEmailWorker_HandlesShareNotifications(t *testing.T) {
	h := startEmailWorker(t)
	acker := newAcker()

	h.mailer.EXPECT().
		SendShareNotification(gomock.Eq("friend@example.com"), gomock.Eq("share-token-abc")).
		Return(nil).
		AnyTimes()

	h.ch <- delivery(t, acker, nil, queue.EmailMessage{
		To: "friend@example.com", Name: "share", Token: "share-token-abc",
	})

	got := acker.wait(t)
	if got.op != "ack" {
		t.Errorf("a 'share' email job was %sed instead of being delivered; "+
			"the email worker has no case for it and mailer.SendShareNotification "+
			"is never called anywhere in the codebase", got.op)
	}
}

func TestEmailWorker_RoutesEachJobTypeToTheRightMailerCall(t *testing.T) {
	tests := []struct {
		name   string
		expect func(*mailermocks.MockEmailSender)
	}{
		{
			name: "signup",
			expect: func(m *mailermocks.MockEmailSender) {
				m.EXPECT().SendVerificationToken(gomock.Eq("u@example.com"), gomock.Eq("tok")).Return(nil).Times(1)
			},
		},
		{
			name: "forgot-password",
			expect: func(m *mailermocks.MockEmailSender) {
				m.EXPECT().SendResetPassword(gomock.Eq("u@example.com"), gomock.Eq("tok")).Return(nil).Times(1)
			},
		},
		{
			name: "resend-verify",
			expect: func(m *mailermocks.MockEmailSender) {
				m.EXPECT().SendVerificationToken(gomock.Eq("u@example.com"), gomock.Eq("tok")).Return(nil).Times(1)
			},
		},
		{
			name: "reset-password",
			expect: func(m *mailermocks.MockEmailSender) {
				m.EXPECT().SendResetNotification(gomock.Eq("u@example.com")).Return(nil).Times(1)
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			h := startEmailWorker(t)
			acker := newAcker()
			tc.expect(h.mailer)

			h.ch <- delivery(t, acker, nil, queue.EmailMessage{
				To: "u@example.com", Name: tc.name, Token: "tok",
			})

			if got := acker.wait(t); got.op != "ack" {
				t.Errorf("disposition = %s, want ack", got.op)
			}
		})
	}
}

// EXPECTED TO FAIL: the same first-failure-is-fatal pattern as the video
// worker. An SMTP timeout — the single most common transient failure in an
// email pipeline — permanently drops the verification mail, so the user can
// never confirm their account.
func TestEmailWorker_SmtpTimeoutIsRetriedNotDeadLettered(t *testing.T) {
	h := startEmailWorker(t)
	acker := newAcker()

	h.mailer.EXPECT().
		SendVerificationToken(gomock.Any(), gomock.Any()).
		Return(errors.New("dial tcp smtp.example.com:587: i/o timeout")).
		Times(1)

	// maxRetries is 3 for the email worker; this delivery has used 1
	h.ch <- delivery(t, acker, deathHeader(1), queue.EmailMessage{
		To: "u@example.com", Name: "signup", Token: "tok",
	})

	got := acker.wait(t)
	if got.op != "nack" {
		t.Fatalf("disposition = %s, want nack", got.op)
	}
	if !got.requeue {
		t.Error("a transient SMTP failure was dead-lettered on the first attempt; " +
			"the user never receives their verification email")
	}
}

func TestEmailWorker_UnknownJobTypeIsDeadLettered(t *testing.T) {
	h := startEmailWorker(t)
	acker := newAcker()

	// no mailer call is expected for a job type nobody handles
	h.ch <- delivery(t, acker, nil, queue.EmailMessage{
		To: "u@example.com", Name: "definitely-not-a-real-job-type",
	})

	got := acker.wait(t)
	if got.op != "nack" {
		t.Fatalf("disposition = %s, want nack", got.op)
	}
	if got.requeue {
		t.Error("an unroutable job type was requeued and will loop forever")
	}
}

// ===========================================================================
// Save-metadata worker
// ===========================================================================

type saveHarness struct {
	q   *queuemocks.MockQueue
	srv *mediasvcmocks.MockService
	ch  chan amqp.Delivery
	wg  *sync.WaitGroup
}

func startSaveWorker(t *testing.T) *saveHarness {
	t.Helper()
	ctrl := gomock.NewController(t)
	h := &saveHarness{
		q:   queuemocks.NewMockQueue(ctrl),
		srv: mediasvcmocks.NewMockService(ctrl),
		ch:  make(chan amqp.Delivery, 1),
		wg:  &sync.WaitGroup{},
	}

	ctx, cancel := context.WithCancel(context.Background())
	h.q.EXPECT().
		ConsumeSave(gomock.Any(), gomock.Eq("save-worker"), gomock.Eq(1)).
		Return((<-chan amqp.Delivery)(h.ch), nil).
		Times(1)
	h.q.EXPECT().CloseConsumerChannel(gomock.Any()).Return(nil).AnyTimes()

	w := worker.NewSaveVideoWorker(h.q, h.srv)
	h.wg.Add(1)
	go func() {
		defer h.wg.Done()
		_ = w.Run(ctx, "save-worker", 1)
	}()

	t.Cleanup(func() {
		cancel()
		h.wg.Wait()
	})
	return h
}

// EXPECTED TO FAIL: internal/worker/save_metadata_worker.go collapses every
// non-ErrInvalidUserID failure into `d.Nack(false, false)`. A MinIO StatObject
// blip loses the upload's metadata for good — the user's video is in the bucket
// but never appears in their library.
func TestSaveWorker_TransientStorageFailureIsRetried(t *testing.T) {
	h := startSaveWorker(t)
	acker := newAcker()

	h.srv.EXPECT().
		SaveMetadata(gomock.Any(), gomock.Any()).
		Return(errors.New("minio: RequestTimeout")).
		Times(1)

	h.ch <- delivery(t, acker, deathHeader(0), queue.SaveVideoMessage{
		Key: "user-1:clip.mp4", UserID: "user-1", Filename: "clip.mp4",
	})

	got := acker.wait(t)
	if got.op != "nack" {
		t.Fatalf("disposition = %s, want nack", got.op)
	}
	if !got.requeue {
		t.Error("a transient storage failure was dead-lettered on the first attempt; " +
			"the uploaded video never shows up in the user's library")
	}
}

// An unparseable user id can never succeed, so dead-lettering it is correct.
func TestSaveWorker_InvalidUserIdIsDeadLetteredImmediately(t *testing.T) {
	h := startSaveWorker(t)
	acker := newAcker()

	h.srv.EXPECT().
		SaveMetadata(gomock.Any(), gomock.Any()).
		Return(errors.New("invalid user id")).
		Times(1)

	h.ch <- delivery(t, acker, deathHeader(9), queue.SaveVideoMessage{
		Key: "k", UserID: "not-a-uuid",
	})

	got := acker.wait(t)
	if got.op != "nack" || got.requeue {
		t.Errorf("disposition = %+v, want a non-requeueing nack", got)
	}
}

func TestSaveWorker_SuccessAcks(t *testing.T) {
	h := startSaveWorker(t)
	acker := newAcker()

	h.srv.EXPECT().
		SaveMetadata(gomock.Any(), gomock.Any()).
		DoAndReturn(func(_ context.Context, msg queue.SaveVideoMessage) error {
			if msg.Key != "user-1:clip.mp4" || msg.Filename != "clip.mp4" {
				t.Errorf("service got %+v", msg)
			}
			return nil
		}).
		Times(1)

	h.ch <- delivery(t, acker, nil, queue.SaveVideoMessage{
		Key: "user-1:clip.mp4", UserID: "user-1", Filename: "clip.mp4",
	})

	if got := acker.wait(t); got.op != "ack" {
		t.Errorf("disposition = %s, want ack", got.op)
	}
}

// ===========================================================================
// Shutdown
// ===========================================================================

// Cancelling the context must stop the loop and release the consumer channel,
// rather than leaking a goroutine per worker.
func TestVideoWorker_StopsOnContextCancel(t *testing.T) {
	ctrl := gomock.NewController(t)
	q := queuemocks.NewMockQueue(ctrl)
	srv := mediasvcmocks.NewMockService(ctrl)
	ch := make(chan amqp.Delivery)

	q.EXPECT().ConsumeVideo(gomock.Any(), gomock.Any(), gomock.Any()).
		Return((<-chan amqp.Delivery)(ch), nil).Times(1)
	q.EXPECT().CloseConsumerChannel(gomock.Eq("video-worker")).Return(nil).Times(1)

	ctx, cancel := context.WithCancel(context.Background())
	w := worker.NewVideoWorker(srv, q)

	done := make(chan error, 1)
	go func() { done <- w.Run(ctx, "video-worker", 1) }()

	cancel()
	select {
	case err := <-done:
		if err != nil {
			t.Errorf("Run returned %v on a clean shutdown, want nil", err)
		}
	case <-time.After(3 * time.Second):
		t.Fatal("the worker did not shut down when its context was cancelled")
	}
}

// A closed delivery channel (broker connection dropped) must be reported so the
// supervisor can reconnect, not silently swallowed.
func TestVideoWorker_ClosedDeliveryChannelIsReported(t *testing.T) {
	ctrl := gomock.NewController(t)
	q := queuemocks.NewMockQueue(ctrl)
	srv := mediasvcmocks.NewMockService(ctrl)
	ch := make(chan amqp.Delivery)

	q.EXPECT().ConsumeVideo(gomock.Any(), gomock.Any(), gomock.Any()).
		Return((<-chan amqp.Delivery)(ch), nil).Times(1)
	q.EXPECT().CloseConsumerChannel(gomock.Any()).Return(nil).Times(1)

	w := worker.NewVideoWorker(srv, q)
	done := make(chan error, 1)
	go func() { done <- w.Run(context.Background(), "video-worker", 1) }()

	close(ch)
	select {
	case err := <-done:
		if !errors.Is(err, queue.ErrConsumerChannelClosed) {
			t.Errorf("Run returned %v, want ErrConsumerChannelClosed", err)
		}
	case <-time.After(3 * time.Second):
		t.Fatal("the worker did not notice its delivery channel closing")
	}
}

// A failure to start consuming must surface immediately.
func TestVideoWorker_ConsumeErrorIsReturned(t *testing.T) {
	ctrl := gomock.NewController(t)
	q := queuemocks.NewMockQueue(ctrl)
	srv := mediasvcmocks.NewMockService(ctrl)
	boom := errors.New("channel setup error")

	q.EXPECT().ConsumeVideo(gomock.Any(), gomock.Any(), gomock.Any()).
		Return(nil, boom).Times(1)
	// CloseConsumerChannel must not run — the defer is registered after the
	// error check, so a leaked close here would be a bug.

	w := worker.NewVideoWorker(srv, q)
	if err := w.Run(context.Background(), "video-worker", 1); !errors.Is(err, boom) {
		t.Errorf("Run returned %v, want the consume error", err)
	}
}
