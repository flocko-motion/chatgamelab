package adapters

import (
	"bufio"
	"bytes"
	"io"
)

// ReadEvents reads a server-sent event stream and hands each event's data to
// on, until the stream ends or on returns an error, which it returns as is. A
// stream that breaks partway may pass, and is marked so.
func ReadEvents(call string, r io.Reader, on func(data []byte) error) error {
	scanner := bufio.NewScanner(r)
	// A reply's last event carries the whole response, which can be long.
	scanner.Buffer(make([]byte, 64*1024), 4*1024*1024)

	var data []byte
	flush := func() error {
		if len(data) == 0 {
			return nil
		}
		event := data
		data = nil
		if bytes.Equal(event, []byte("[DONE]")) {
			return nil
		}
		return on(event)
	}

	for scanner.Scan() {
		line := scanner.Bytes()
		if len(line) == 0 {
			if err := flush(); err != nil {
				return err
			}
			continue
		}
		if field, ok := bytes.CutPrefix(line, []byte("data:")); ok {
			if len(data) > 0 {
				data = append(data, '\n')
			}
			data = append(data, bytes.TrimPrefix(field, []byte(" "))...)
		}
		// Comments, event names and ids are not needed: every payload names
		// its own type.
	}
	if err := scanner.Err(); err != nil {
		return Transient(call, err)
	}
	return flush()
}
