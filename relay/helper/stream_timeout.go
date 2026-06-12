package helper

import (
	"fmt"
	"io"
	"net/http"
	"sync"
	"time"

	"github.com/QuantumNous/new-api/logger"
	relaycommon "github.com/QuantumNous/new-api/relay/common"

	"github.com/gin-gonic/gin"
)

func ChannelStreamIdleTimeout(info *relaycommon.RelayInfo) time.Duration {
	if info == nil || info.ChannelMeta == nil || info.ChannelSetting.StreamTimeout <= 0 {
		return 0
	}
	return time.Duration(info.ChannelSetting.StreamTimeout) * time.Second
}

func WrapResponseBodyWithStreamIdleTimeout(c *gin.Context, resp *http.Response, info *relaycommon.RelayInfo) {
	timeout := ChannelStreamIdleTimeout(info)
	if timeout <= 0 || resp == nil || resp.Body == nil {
		return
	}
	if _, ok := resp.Body.(*streamIdleTimeoutReadCloser); ok {
		return
	}

	resp.Body = newStreamIdleTimeoutReadCloser(resp.Body, timeout, func() {
		info.MarkStreamIdleTimeoutTriggered()
		if info.StreamStatus != nil {
			info.StreamStatus.SetEndReason(relaycommon.StreamEndReasonDone, nil)
		}
		logger.LogInfo(c, fmt.Sprintf("channel stream idle timeout reached: %d seconds", int64(timeout.Seconds())))
	})
}

type streamIdleTimeoutReadCloser struct {
	source    io.ReadCloser
	reader    *io.PipeReader
	writer    *io.PipeWriter
	timeout   time.Duration
	onTimeout func()

	activity chan struct{}
	done     chan struct{}
	doneOnce sync.Once
}

func newStreamIdleTimeoutReadCloser(source io.ReadCloser, timeout time.Duration, onTimeout func()) *streamIdleTimeoutReadCloser {
	reader, writer := io.Pipe()
	wrapper := &streamIdleTimeoutReadCloser{
		source:    source,
		reader:    reader,
		writer:    writer,
		timeout:   timeout,
		onTimeout: onTimeout,
		activity:  make(chan struct{}, 1),
		done:      make(chan struct{}),
	}
	go wrapper.copyLoop()
	go wrapper.timeoutLoop()
	return wrapper
}

func (r *streamIdleTimeoutReadCloser) Read(p []byte) (int, error) {
	return r.reader.Read(p)
}

func (r *streamIdleTimeoutReadCloser) Close() error {
	r.finish()
	err := r.source.Close()
	_ = r.reader.Close()
	_ = r.writer.Close()
	return err
}

func (r *streamIdleTimeoutReadCloser) finish() {
	r.doneOnce.Do(func() {
		close(r.done)
	})
}

func (r *streamIdleTimeoutReadCloser) markActivity() {
	select {
	case r.activity <- struct{}{}:
	default:
	}
}

func (r *streamIdleTimeoutReadCloser) copyLoop() {
	defer r.finish()

	buf := make([]byte, 32*1024)
	for {
		n, err := r.source.Read(buf)
		if n > 0 {
			r.markActivity()
			if _, writeErr := r.writer.Write(buf[:n]); writeErr != nil {
				_ = r.writer.CloseWithError(writeErr)
				return
			}
		}
		if err != nil {
			if err == io.EOF {
				_ = r.writer.Close()
			} else {
				_ = r.writer.CloseWithError(err)
			}
			return
		}
	}
}

func (r *streamIdleTimeoutReadCloser) timeoutLoop() {
	timer := time.NewTimer(r.timeout)
	defer timer.Stop()

	for {
		select {
		case <-r.activity:
			resetTimer(timer, r.timeout)
		case <-timer.C:
			if r.onTimeout != nil {
				r.onTimeout()
			}
			_ = r.source.Close()
			_ = r.writer.Close()
			r.finish()
			return
		case <-r.done:
			return
		}
	}
}

func resetTimer(timer *time.Timer, timeout time.Duration) {
	if !timer.Stop() {
		select {
		case <-timer.C:
		default:
		}
	}
	timer.Reset(timeout)
}
