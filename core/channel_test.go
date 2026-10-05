package core

import (
	"testing"
	"testing/synctest"
)

func TestChannelOfferPollUnbuffered(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		RT.GIL.Lock()
		defer RT.GIL.Unlock()
		ch := MakeChannel(make(chan FutureResult))
		value := MakeInt(42)
		if got := procOffer([]Object{ch, value}); got != MakeBoolean(false) {
			t.Fatalf("offer without receiver = %v", got)
		}
		if got := procPoll([]Object{ch}); got != NIL {
			t.Fatalf("poll without sender = %v", got)
		}

		received := make(chan FutureResult, 1)
		go func() { received <- <-ch.ch }()
		synctest.Wait() // The receiver is now blocked on the unbuffered channel.
		if got := procOffer([]Object{ch, value}); got != MakeBoolean(true) {
			t.Fatalf("offer with waiting receiver = %v", got)
		}
		if got := <-received; got.value != value || got.err != nil {
			t.Fatalf("receiver got %+v", got)
		}

		done := make(chan struct{})
		go func() {
			ch.ch <- MakeFutureResult(value, nil)
			close(done)
		}()
		synctest.Wait() // The sender is now blocked on the unbuffered channel.
		if got := procPoll([]Object{ch}); got != value {
			t.Fatalf("poll with waiting sender = %v", got)
		}
		<-done
		ch.Close()
		ch.Close()
		if got := procOffer([]Object{ch, value}); got != MakeBoolean(false) {
			t.Fatalf("offer on closed channel = %v", got)
		}
		if got := procPoll([]Object{ch}); got != NIL {
			t.Fatalf("poll on closed channel = %v", got)
		}
	})
}

func TestChannelOfferClosedUnderlyingChannel(t *testing.T) {
	// Offer must not leak a Go panic, even if the transport was closed directly.
	ch := MakeChannel(make(chan FutureResult, 1))
	close(ch.ch)
	if ch.Offer(MakeInt(1)) {
		t.Fatal("offer on closed transport succeeded")
	}
}

func TestChannelPollGoResults(t *testing.T) {
	for _, name := range []string{"value", "false", "nil", "error"} {
		t.Run(name, func(t *testing.T) {
			synctest.Test(t, func(t *testing.T) {
				RT.GIL.Lock()
				value := Object(MakeInt(42))
				var wantErr Error
				switch name {
				case "false":
					value = MakeBoolean(false)
				case "nil":
					value = NIL
				case "error":
					wantErr = RT.NewError("go failure")
				}
				callback := Proc{Fn: func([]Object) Object {
					if wantErr != nil {
						panic(wantErr)
					}
					return value
				}}
				ch := procGo([]Object{callback}).(*Channel)
				suspended := RT.Suspend()
				synctest.Wait() // The go body has finished and its result channel is closed.
				suspended.Resume()
				defer RT.GIL.Unlock()
				if !ch.isClosed {
					t.Fatal("go result channel is not closed")
				}
				if wantErr != nil {
					func() {
						defer func() {
							if got := recover(); got != wantErr {
								t.Fatalf("poll panic = %v; want original error %v", got, wantErr)
							}
						}()
						procPoll([]Object{ch})
					}()
				} else if got := procPoll([]Object{ch}); got != value {
					t.Fatalf("poll = %v; want %v", got, value)
				}
				if got := procPoll([]Object{ch}); got != NIL {
					t.Fatalf("poll on drained go result channel = %v", got)
				}
			})
		})
	}
}
