# elevenlabs-go v0.2.0

![Go version](https://img.shields.io/badge/go-1.18-blue)
![License](https://img.shields.io/github/license/Mliviu79/elevenlabs-go)
![Tests](https://github.com/haguro/elevenlabs-go/actions/workflows/tests.yml/badge.svg?branch=main&event=push)
[![codecov](https://codecov.io/gh/haguro/elevenlabs-go/branch/main/graph/badge.svg?token=UM33DSSTAG)](https://codecov.io/gh/haguro/elevenlabs-go)
[![Go Report Card](https://goreportcard.com/badge/github.com/haguro/elevenlabs-go)](https://goreportcard.com/report/github.com/haguro/elevenlabs-go)
[![Go Reference](https://pkg.go.dev/badge/github.com/haguro/elevenlabs-go.svg)](https://pkg.go.dev/github.com/haguro/elevenlabs-go#section-documentation)

This is a Go client library for the [ElevenLabs](https://elevenlabs.io/) voice cloning and speech synthesis platform. It provides a basic interface for Go programs to interact with the ElevenLabs [API](https://docs.elevenlabs.io/api-reference).

## About this Fork

This repository is a fork of [haguro/elevenlabs-go](https://github.com/haguro/elevenlabs-go) with the following changes:

1. Updated package name to `github.com/Mliviu79/elevenlabs-go`
2. Updated the `TextToSpeechRequest` struct to match the latest ElevenLabs API
3. Added support for the new `/text-to-speech/:voice_id/stream/with-timestamps` endpoint
4. Released as version 0.2.0

## Installation

```bash
go get github.com/Mliviu79/elevenlabs-go@v0.2.0
```

## Example Usage

Make sure to replace `"your-api-key"` in all examples with your actual API key. Refer to the official Elevenlabs [API documentation](https://docs.elevenlabs.io/api-reference/quick-start/introduction) for further details.

Full documentation of this library is available [here](https://pkg.go.dev/github.com/haguro/elevenlabs-go).

### Using a New Client Instance

Using the `NewClient` function returns a new `Client` instance configured with your API key and a timeout duration. Every request method takes the `context.Context` the request is bound to as its first argument; cancelling it aborts the request.

```go
package main

import (
 "context"
 "log"
 "os"
 "time"

 "github.com/Mliviu79/elevenlabs-go"
)

func main() {
 // Create a new client
 client := elevenlabs.NewClient("your-api-key", 30*time.Second)

 // Create a TextToSpeechRequest
 ttsReq := elevenlabs.TextToSpeechRequest{
  Text:    "Hello, world! My name is Adam, nice to meet you!",
  ModelID: "eleven_monolingual_v1",
 }

 // Call the TextToSpeech method on the client, using the "Adam"'s voice ID.
 audio, err := client.TextToSpeech(context.Background(), "pNInz6obpgDQGcFmaJgB", ttsReq)
 if err != nil {
  log.Fatal(err)
 }

 // Write the audio file bytes to disk
 if err := os.WriteFile("adam.mp3", audio, 0644); err != nil {
  log.Fatal(err)
 }

 log.Println("Successfully generated audio file")
}
```

### Using the Default Client and proxy functions

The library has a default client you can configure and use with proxy functions that wrap method calls to the default client. The default client has a default timeout set to 30 seconds, and each proxy function takes the request's context as its first argument like the methods it wraps. You will only need to set your API key at minimum when taking advantage of the default client. Here's the a version of the above example above using shorthand functions only.

```go
package main

import (
 "context"
 "log"
 "os"
 "time"

 el "github.com/Mliviu79/elevenlabs-go"
)

func main() {
 // Set your API key
 el.SetAPIKey("your-api-key")

 // Set a different timeout (optional)
 el.SetTimeout(15 * time.Second)

 // Call the TextToSpeech method on the client, using the "Adam"'s voice ID.
 audio, err := el.TextToSpeech(context.Background(), "pNInz6obpgDQGcFmaJgB", el.TextToSpeechRequest{
   Text:    "Hello, world! My name is Adam, nice to meet you!",
   ModelID: "eleven_monolingual_v1",
  })
 if err != nil {
  log.Fatal(err)
 }

 // Write the audio file bytes to disk
 if err := os.WriteFile("adam.mp3", audio, 0644); err != nil {
  log.Fatal(err)
 }

 log.Println("Successfully generated audio file")
}
```

### Streaming

The Elevenlabs API allows streaming of audio "as it is being generated". In elevenlabs-go, you'll want to pass an `io.Writer` to the `TextToSpeechStream` method where the stream will be continuously copied to. _Note that you will need to set the client timeout to a high enough value to ensure that request does not time out mid-stream_.

```go
package main

import (
 "context"
 "log"
 "os/exec"
 "time"

 "github.com/Mliviu79/elevenlabs-go"
)

func main() {
 message := `The concept of "flushing" typically applies to I/O buffers in many programming 
languages, which store data temporarily in memory before writing it to a more permanent location
like a file or a network connection. Flushing the buffer means writing all the buffered data
immediately, even if the buffer isn't full.`

 // Set your API key
 elevenlabs.SetAPIKey("your-api-key")

 // Set a large enough timeout to ensure the stream is not interrupted.
 elevenlabs.SetTimeout(1 * time.Minute)

 // We'll use mpv to play the audio from the stream piped to standard input
 cmd := exec.CommandContext(context.Background(), "mpv", "--no-cache", "--no-terminal", "--", "fd://0")

 // Get a pipe connected to the mpv's standard input
 pipe, err := cmd.StdinPipe()
 if err != nil {
  log.Fatal(err)
 }

 // Attempt to run the command in a separate process
 if err := cmd.Start(); err != nil {
  log.Fatal(err)
 }

 // Stream the audio to the pipe connected to mpv's standard input
 if err := elevenlabs.TextToSpeechStream(
  context.Background(),
  pipe,
  "pNInz6obpgDQGcFmaJgB",
  elevenlabs.TextToSpeechRequest{
   Text:    message,
   ModelID: "eleven_multilingual_v1",
  }); err != nil {
  log.Fatalf("Got %T error: %q\n", err, err)
 }

 // Close the pipe when all stream has been copied to the pipe
 if err := pipe.Close(); err != nil {
  log.Fatalf("Could not close pipe: %s", err)
 }
 log.Print("Streaming finished.")

 // Wait for mpv to exit. With the pipe closed, it will do that as
 // soon as it finishes playing
 if err := cmd.Wait(); err != nil {
  log.Fatal(err)
 }

 log.Print("All done.")
}
```

## TextToSpeechRequest Struct

The `TextToSpeechRequest` struct has been updated to match the latest ElevenLabs API:

```go
type TextToSpeechRequest struct {
	Text                            string                           `json:"text"`
	ModelID                         string                           `json:"model_id,omitempty"`
	LanguageCode                    string                           `json:"language_code,omitempty"`
	PronunciationDictionaryLocators []PronunciationDictionaryLocator `json:"pronunciation_dictionary_locators,omitempty"`
	// Seed requests deterministic sampling. Nil omits it and the API applies
	// its default; non-nil sends the value, zero included.
	Seed               *uint32  `json:"seed,omitempty"`
	PreviousText       string   `json:"previous_text,omitempty"`
	NextText           string   `json:"next_text,omitempty"`
	PreviousRequestIds []string `json:"previous_request_ids,omitempty"`
	NextRequestIds     []string `json:"next_request_ids,omitempty"`
	// ApplyTextNormalization selects the text normalization mode. Empty omits
	// it and the API applies its default.
	ApplyTextNormalization TextNormalization `json:"apply_text_normalization,omitempty"`
	// ApplyLanguageTextNormalization enables language-specific text
	// normalization. Nil omits it and the API applies its default; non-nil
	// sends the value, false included.
	ApplyLanguageTextNormalization *bool          `json:"apply_language_text_normalization,omitempty"`
	VoiceSettings                  *VoiceSettings `json:"voice_settings,omitempty"`
}
```

## Text-to-Speech with Timestamps

This fork includes support for the new ElevenLabs API endpoint that provides character-level timestamps along with the generated audio. The response contains:

- `audio_base64`: Base64 encoded audio data
- `alignment`: Timestamp information for each character in the original text
- `normalized_alignment`: Timestamp information for each character in the normalized text

### Example Usage

```go
package main

import (
 "context"
 "log"
 "os"
 "time"

 "github.com/Mliviu79/elevenlabs-go"
)

func main() {
 // Create a new client
 client := elevenlabs.NewClient("your-api-key", 30*time.Second)

 // Create a TextToSpeechRequest
 ttsReq := elevenlabs.TextToSpeechRequest{
  Text:    "Hello, world! This is a test with timestamps.",
  ModelID: "eleven_monolingual_v1",
 }

 // Create a file to write the JSON response containing audio and timestamps
 file, err := os.Create("response_with_timestamps.json")
 if err != nil {
  log.Fatal(err)
 }
 defer file.Close()

 // Call the TextToSpeechWithTimestamps method, streaming the JSON response to the file
 err = client.TextToSpeechStreamWithTimestamps(context.Background(), file, "pNInz6obpgDQGcFmaJgB", ttsReq)
 if err != nil {
  log.Fatal(err)
 }

 log.Println("Successfully generated audio with timestamps")
}
```

The response structure includes:

```go
type TextToSpeechWithTimestampsResponse struct {
	AudioBase64         string        `json:"audio_base64"`
	Alignment           AlignmentInfo `json:"alignment"`
	NormalizedAlignment AlignmentInfo `json:"normalized_alignment"`
}

type AlignmentInfo struct {
	Characters                 []string  `json:"characters"`
	CharacterStartTimesSeconds []float64 `json:"character_start_times_seconds"`
	CharacterEndTimesSeconds   []float64 `json:"character_end_times_seconds"`
}
```

## Realtime Speech-to-Text

The package provides the typed protocol of the realtime speech-to-text WebSocket API: the connection options, the audio chunk message and the server events. It opens no connection itself. Dial `SpeechToTextRealtimeURL` with your own WebSocket library, appending the query string built from the options, and send your API key in the `xi-api-key` header of the handshake.

```go
package main

import (
	"encoding/json"
	"fmt"
	"log"

	"github.com/Mliviu79/elevenlabs-go"
)

func main() {
	// A nil pointer leaves a parameter out; a set one is sent, false included.
	includeTimestamps := true
	options := elevenlabs.SpeechToTextRealtimeOptions{
		ModelID:           "scribe_v2_realtime",
		AudioFormat:       "pcm_16000",
		CommitStrategy:    "vad",
		IncludeTimestamps: &includeTimestamps,
		Keyterms:          []string{"ElevenLabs", "Scribe"},
	}

	// Dial this URL with a WebSocket library, sending the API key in the
	// xi-api-key header of the handshake.
	fmt.Println(elevenlabs.SpeechToTextRealtimeURL + "?" + options.Query().Encode())
	// wss://api.elevenlabs.io/v1/speech-to-text/realtime?audio_format=pcm_16000&commit_strategy=vad&include_timestamps=true&keyterms=ElevenLabs&keyterms=Scribe&model_id=scribe_v2_realtime

	// Send audio as it becomes available
	audioMessage, err := json.Marshal(elevenlabs.SpeechToTextRealtimeAudioChunk{
		Audio:      []byte("pcm"),
		SampleRate: 16000,
	})
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println(string(audioMessage))
	// {"message_type":"input_audio_chunk","audio_base_64":"cGNt","commit":false,"sample_rate":16000}

	// Commit the segment without sending more audio
	commitMessage, err := json.Marshal(elevenlabs.SpeechToTextRealtimeAudioChunk{
		Commit:     true,
		SampleRate: 16000,
	})
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println(string(commitMessage))
	// {"message_type":"input_audio_chunk","audio_base_64":"","commit":true,"sample_rate":16000}

	// Two text messages as the server sends them
	messages := []string{
		`{"message_type":"committed_transcript","text":"Hello, world."}`,
		`{"message_type":"auth_error","error":"Invalid API key."}`,
	}

	for _, message := range messages {
		var event elevenlabs.SpeechToTextRealtimeEvent
		if err := json.Unmarshal([]byte(message), &event); err != nil {
			log.Fatal(err)
		}

		// Check for a server error before using the event
		if err := event.Err(); err != nil {
			fmt.Println("server error:", err)
			continue
		}
		fmt.Printf("%s: %s\n", event.MessageType, event.Text)
	}
	// committed_transcript: Hello, world.
	// server error: speech-to-text realtime auth_error: Invalid API key.
}
```

Write each marshalled chunk as a text message. Decode each text message you receive into a `SpeechToTextRealtimeEvent` and check `Err()` first: the server sends an error event and then closes the connection.

## Status and Future Plans

This fork maintains compatibility with the original library while providing updated structs to match the latest ElevenLabs API.

## Contributing

Contributions are welcome! If you have any ideas, improvements, or bug fixes, please open an issue or submit a pull request.

## Looking for a Python library?

The Elevenlabs's official [Python library](https://github.com/elevenlabs/elevenlabs-python) is excellent and fellow Pythonistas are encouraged to use it (and also to give Go, a [go](https://gobyexample.com/) 😉🩵)!

## Disclaimer

This is an independent project and is not affiliated with or endorsed by Elevenlabs. Elevenlabs and its trademarks are the property of their respective owners. The purpose of this project is to provide a client library to facilitate access to the public API provided Elevenlabs within Go programs. Any use of Elevenlabs's trademarks within this project is for identification purposes only and does not imply endorsement, sponsorship, or affiliation.

## License

This project is licensed under the [MIT License](LICENSE).

## Warranty

This code library is provided "as is" and without any warranties whatsoever. Use at your own risk. More details in the [LICENSE](LICENSE) file.
