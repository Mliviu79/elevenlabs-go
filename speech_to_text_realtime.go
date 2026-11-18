package elevenlabs

import (
	"encoding/base64"
	"encoding/json"
	"net/url"
	"strconv"
)

// SpeechToTextRealtimeURL is the production endpoint of the realtime speech-to-text WebSocket
// API. The package opens no connection itself: a caller dials this address with its own WebSocket
// library, appends the query string of [SpeechToTextRealtimeOptions.Query], and authenticates by
// sending its API key in the xi-api-key header of the handshake.
//
// The API reference also lists the servers api.us.elevenlabs.io, api.eu.residency.elevenlabs.io,
// api.in.residency.elevenlabs.io and api.sg.residency.elevenlabs.io: a caller whose account is
// served from one of them replaces the host.
const SpeechToTextRealtimeURL = "wss://api.elevenlabs.io/v1/speech-to-text/realtime"

// SpeechToTextRealtimeOptions holds the connection parameters of the realtime speech-to-text
// API, which travel in the query string of the WebSocket URL. Every member is optional: an empty
// string, a nil pointer and an empty list are left out and the API applies its default; a non-nil
// pointer sends its value, false and zero included. Values are sent as given: the server checks
// them and answers a rejected parameter with an invalid_request error event.
type SpeechToTextRealtimeOptions struct {
	// ModelID is the model_id parameter: "scribe_v2_realtime", the API's default, or
	// "scribe_v3_live".
	ModelID string
	// AudioFormat is the audio_format parameter, the encoding of the audio the caller sends:
	// "pcm_8000", "pcm_16000" (the API's default), "pcm_22050", "pcm_24000", "pcm_44100" or
	// "pcm_48000" for 16-bit little-endian PCM at that sample rate, or "ulaw_8000" for 8-bit
	// μ-law. The audio is mono.
	AudioFormat string
	// LanguageCode is the language_code parameter: the language of the speech, as an ISO 639-1
	// or ISO 639-3 code.
	LanguageCode string
	// SecondaryLanguages is the secondary_languages parameter, sent once per language: further
	// languages the model recognises the speech within, beside LanguageCode.
	SecondaryLanguages []string
	// CommitStrategy is the commit_strategy parameter, which decides when a segment of speech is
	// committed. "manual", the API's default for every model, needs an explicit commit (see
	// SpeechToTextRealtimeAudioChunk.Commit); "vad" commits when it detects silence;
	// "turn_prediction" also commits as soon as the model predicts that the speaker has finished
	// their turn, and needs the scribe_v3_live model.
	CommitStrategy string
	// VADThreshold is the vad_threshold parameter, the threshold of voice activity detection
	// under the "vad" and "turn_prediction" strategies. With FilterBackgroundAudio the API's
	// default is 0.15.
	VADThreshold *float64
	// VADSilenceThresholdSecs is the vad_silence_threshold_secs parameter: the silence, in
	// seconds, after which the "vad" and "turn_prediction" strategies commit. The API accepts
	// 0.192 to 3.0.
	VADSilenceThresholdSecs *float64
	// MinSpeechDurationMs is the min_speech_duration_ms parameter, the minimum speech duration
	// in milliseconds for voice activity detection.
	MinSpeechDurationMs *int
	// MinSilenceDurationMs is the min_silence_duration_ms parameter, the minimum silence
	// duration in milliseconds for voice activity detection.
	MinSilenceDurationMs *int
	// IncludeTimestamps is the include_timestamps parameter, false by default. When true, each
	// committed transcript is followed by a committed_transcript_with_timestamps event carrying
	// its words and their timings.
	IncludeTimestamps *bool
	// IncludeLanguageDetection is the include_language_detection parameter, false by default.
	// When true, partial_transcript and committed_transcript_with_timestamps events carry the
	// language of the speech.
	IncludeLanguageDetection *bool
	// Keyterms is the keyterms parameter, sent once per term: words or phrases the model is
	// biased towards. The API accepts up to 50 terms of up to 20 characters each.
	Keyterms []string
	// NoVerbatim is the no_verbatim parameter, false by default. When true, filler words and
	// disfluencies are removed from the transcript.
	NoVerbatim *bool
	// MultilangMode is the multilang_mode parameter, false by default. When true, the model
	// follows speakers who switch languages during the session. It needs the scribe_v3_live
	// model.
	MultilangMode *bool
	// Diarize is the diarize parameter, false by default. When true, words carry a speaker,
	// committed transcripts carry a segment index, each is followed by a diarized_transcript
	// event, and speakers_merged events revise earlier speakers. It needs the scribe_v3_live
	// model and cannot be combined with FilterBackgroundAudio.
	Diarize *bool
	// EntityDetection is the entity_detection parameter, sent once per value: "all", or entity
	// types or categories ("pii", "phi", "pci", "other", "offensive_language"). The entities
	// detected in each committed transcript arrive in a committed_transcript_entities event. It
	// cannot be combined with TranscriptEdit.
	EntityDetection []string
	// EntityRedaction is the entity_redaction parameter, sent once per value, in the format of
	// EntityDetection. Each committed transcript is followed by a redacted_transcript event
	// carrying its text with those entities replaced. It cannot be combined with TranscriptEdit.
	EntityRedaction []string
	// EntityRedactionMode is the entity_redaction_mode parameter, which sets what replaces a
	// redacted entity: "redacted" writes {REDACTED}, "entity_type" writes {ENTITY_TYPE} and
	// "enumerated_entity_type", the API's default, writes {ENTITY_TYPE_N}, numbering each
	// occurrence. It is used only with EntityRedaction.
	EntityRedactionMode string
	// TranscriptEdit is the transcript_edit parameter: an instruction in natural language, up to
	// 2000 characters, applied to every committed transcript. The edited text arrives in an
	// edited_transcript event. It cannot be combined with EntityDetection or EntityRedaction.
	TranscriptEdit string
	// FilterBackgroundAudio is the filter_background_audio parameter, false by default. When
	// true, voice activity detection ignores background speech and ambient noise. It cannot be
	// combined with Diarize.
	FilterBackgroundAudio *bool
	// KeepaliveIntervalMs is the keepalive_interval_ms parameter, from 500 to 10000
	// milliseconds, off by default. About once per interval, while the caller keeps sending
	// audio, the server sends a partial_transcript event as a keepalive: with empty text, or
	// repeating the latest partial text of the segment in progress.
	KeepaliveIntervalMs *int
	// EnableLogging is the enable_logging parameter, true by default. False selects zero
	// retention mode, which the API offers to enterprise customers only.
	EnableLogging *bool
	// Token is the token parameter: a single-use token that authenticates the connection
	// instead of the xi-api-key header, which is then not sent. The token travels in the URL, so
	// whatever records the URL records the token.
	Token string
}

// Query returns the options as the query parameters of the WebSocket URL, under the API's
// parameter names. It holds only what is set: an empty string, a nil pointer and an empty list
// are left out, so the zero value yields an empty map, never a nil one. A list is sent as one key
// repeated per element, in order. A caller encodes the result and appends it to
// [SpeechToTextRealtimeURL] after a "?".
func (o SpeechToTextRealtimeOptions) Query() url.Values {
	q := url.Values{}
	setString := func(name, value string) {
		if value != "" {
			q.Set(name, value)
		}
	}
	setBool := func(name string, value *bool) {
		if value != nil {
			q.Set(name, strconv.FormatBool(*value))
		}
	}
	setInt := func(name string, value *int) {
		if value != nil {
			q.Set(name, strconv.Itoa(*value))
		}
	}
	setFloat := func(name string, value *float64) {
		if value != nil {
			q.Set(name, strconv.FormatFloat(*value, 'f', -1, 64))
		}
	}
	addEach := func(name string, values []string) {
		for _, value := range values {
			q.Add(name, value)
		}
	}

	setString("model_id", o.ModelID)
	setString("audio_format", o.AudioFormat)
	setString("language_code", o.LanguageCode)
	addEach("secondary_languages", o.SecondaryLanguages)
	setString("commit_strategy", o.CommitStrategy)
	setFloat("vad_threshold", o.VADThreshold)
	setFloat("vad_silence_threshold_secs", o.VADSilenceThresholdSecs)
	setInt("min_speech_duration_ms", o.MinSpeechDurationMs)
	setInt("min_silence_duration_ms", o.MinSilenceDurationMs)
	setBool("include_timestamps", o.IncludeTimestamps)
	setBool("include_language_detection", o.IncludeLanguageDetection)
	addEach("keyterms", o.Keyterms)
	setBool("no_verbatim", o.NoVerbatim)
	setBool("multilang_mode", o.MultilangMode)
	setBool("diarize", o.Diarize)
	addEach("entity_detection", o.EntityDetection)
	addEach("entity_redaction", o.EntityRedaction)
	setString("entity_redaction_mode", o.EntityRedactionMode)
	setString("transcript_edit", o.TranscriptEdit)
	setBool("filter_background_audio", o.FilterBackgroundAudio)
	setInt("keepalive_interval_ms", o.KeepaliveIntervalMs)
	setBool("enable_logging", o.EnableLogging)
	setString("token", o.Token)
	return q
}

// SpeechToTextRealtimeAudioChunk is the input_audio_chunk message, the one message a client
// sends. A caller marshals it with [json.Marshal] and writes the result as one text message.
type SpeechToTextRealtimeAudioChunk struct {
	// Audio is the raw audio, in the audio format of the session. It is sent base64-encoded as
	// audio_base_64. A chunk with no audio and Commit set is the documented message that commits
	// without sending audio.
	Audio []byte
	// Commit is the commit property: true commits the segment after this chunk. It is always
	// sent.
	Commit bool
	// SampleRate is the sample_rate property, the sample rate of the audio in Hz. It is always
	// sent.
	SampleRate int
	// PreviousText is the previous_text property, text that gives the model context. It is left
	// out when empty. The API accepts it on the first chunk of a session only and answers it with
	// an error on a later one.
	PreviousText string
	// KeytermLibrary is the keyterm_library property, a large vocabulary the model consults when
	// it hears a likely match: up to 10,000 terms of up to 50 characters each and 2 MB of text in
	// total. It is left out when empty. It needs the scribe_v3_live model, and the API accepts it
	// on the first chunk of a session only: on a later one it answers with an input_error event
	// and closes the connection.
	KeytermLibrary []string
}

// MarshalJSON encodes the chunk as the input_audio_chunk message: message_type first, then
// audio_base_64 in standard base64 with padding (the empty string, never null, when there is no
// audio), commit and sample_rate, then previous_text and keyterm_library when they are set.
func (c SpeechToTextRealtimeAudioChunk) MarshalJSON() ([]byte, error) {
	type inputAudioChunk struct {
		MessageType    string   `json:"message_type"`
		AudioBase64    string   `json:"audio_base_64"`
		Commit         bool     `json:"commit"`
		SampleRate     int      `json:"sample_rate"`
		PreviousText   string   `json:"previous_text,omitempty"`
		KeytermLibrary []string `json:"keyterm_library,omitempty"`
	}
	return json.Marshal(inputAudioChunk{
		MessageType:    "input_audio_chunk",
		AudioBase64:    base64.StdEncoding.EncodeToString(c.Audio),
		Commit:         c.Commit,
		SampleRate:     c.SampleRate,
		PreviousText:   c.PreviousText,
		KeytermLibrary: c.KeytermLibrary,
	})
}

// SpeechToTextRealtimeMessageType is the message_type of a server event. The API adds types over
// time, so a value outside the constants below is expected and is not, by itself, an error.
type SpeechToTextRealtimeMessageType string

const (
	// SpeechToTextRealtimeSessionStarted is sent once the connection is established, with the
	// identifier and the configuration of the session.
	SpeechToTextRealtimeSessionStarted SpeechToTextRealtimeMessageType = "session_started"
	// SpeechToTextRealtimePartialTranscript carries the interim text of the segment in progress;
	// a later one replaces it. The server also sends it as a keepalive.
	SpeechToTextRealtimePartialTranscript SpeechToTextRealtimeMessageType = "partial_transcript"
	// SpeechToTextRealtimeCommittedTranscript carries the final text of a segment, which does not
	// change afterwards.
	SpeechToTextRealtimeCommittedTranscript SpeechToTextRealtimeMessageType = "committed_transcript"
	// SpeechToTextRealtimeCommittedTranscriptWithTimestamps carries a committed transcript again
	// with its words, their timings and the detected language. It is sent when timestamps or
	// language detection are requested, and it may arrive before the committed_transcript event
	// of the same segment as well as after it.
	SpeechToTextRealtimeCommittedTranscriptWithTimestamps SpeechToTextRealtimeMessageType = "committed_transcript_with_timestamps"
	// SpeechToTextRealtimeDiarizedTranscript follows a committed transcript with its words and
	// their speakers. It is sent with diarization.
	SpeechToTextRealtimeDiarizedTranscript SpeechToTextRealtimeMessageType = "diarized_transcript"
	// SpeechToTextRealtimeSpeakersMerged says that two speaker identifiers are one person. It is
	// sent with diarization.
	SpeechToTextRealtimeSpeakersMerged SpeechToTextRealtimeMessageType = "speakers_merged"
	// SpeechToTextRealtimeCommittedTranscriptEntities follows a committed transcript with the
	// entities detected in it. It is sent with entity detection.
	SpeechToTextRealtimeCommittedTranscriptEntities SpeechToTextRealtimeMessageType = "committed_transcript_entities"
	// SpeechToTextRealtimeRedactedTranscript follows a committed transcript with its text after
	// redaction. It is sent with entity redaction.
	SpeechToTextRealtimeRedactedTranscript SpeechToTextRealtimeMessageType = "redacted_transcript"
	// SpeechToTextRealtimeEditedTranscript follows a committed transcript with its text after the
	// edit instruction. It is sent with transcript editing.
	SpeechToTextRealtimeEditedTranscript SpeechToTextRealtimeMessageType = "edited_transcript"
	// SpeechToTextRealtimeWarning carries a warning text. It is not one of the error events.
	SpeechToTextRealtimeWarning SpeechToTextRealtimeMessageType = "warning"

	// SpeechToTextRealtimeErrorGeneric is a generic server error. It is also sent when entity
	// redaction fails.
	SpeechToTextRealtimeErrorGeneric SpeechToTextRealtimeMessageType = "error"
	// SpeechToTextRealtimeErrorAuth says the request could not be authenticated.
	SpeechToTextRealtimeErrorAuth SpeechToTextRealtimeMessageType = "auth_error"
	// SpeechToTextRealtimeErrorQuotaExceeded says the usage quota is exceeded.
	SpeechToTextRealtimeErrorQuotaExceeded SpeechToTextRealtimeMessageType = "quota_exceeded"
	// SpeechToTextRealtimeErrorCommitThrottled says a commit was refused because too many were
	// made in a short time.
	SpeechToTextRealtimeErrorCommitThrottled SpeechToTextRealtimeMessageType = "commit_throttled"
	// SpeechToTextRealtimeErrorUnacceptedTerms says the account has not accepted the terms of
	// service of the speech-to-text models.
	SpeechToTextRealtimeErrorUnacceptedTerms SpeechToTextRealtimeMessageType = "unaccepted_terms"
	// SpeechToTextRealtimeErrorRateLimited says too many requests were made in a short time.
	SpeechToTextRealtimeErrorRateLimited SpeechToTextRealtimeMessageType = "rate_limited"
	// SpeechToTextRealtimeErrorQueueOverflow says the processing queue is full.
	SpeechToTextRealtimeErrorQueueOverflow SpeechToTextRealtimeMessageType = "queue_overflow"
	// SpeechToTextRealtimeErrorResourceExhausted says the server is at capacity.
	SpeechToTextRealtimeErrorResourceExhausted SpeechToTextRealtimeMessageType = "resource_exhausted"
	// SpeechToTextRealtimeErrorSessionTimeLimitExceeded says the session reached its maximum
	// duration.
	SpeechToTextRealtimeErrorSessionTimeLimitExceeded SpeechToTextRealtimeMessageType = "session_time_limit_exceeded"
	// SpeechToTextRealtimeErrorInput says an audio chunk could not be processed, likely because
	// of its format or its parameters.
	SpeechToTextRealtimeErrorInput SpeechToTextRealtimeMessageType = "input_error"
	// SpeechToTextRealtimeErrorInvalidRequest says the connection parameters were rejected.
	SpeechToTextRealtimeErrorInvalidRequest SpeechToTextRealtimeMessageType = "invalid_request"
	// SpeechToTextRealtimeErrorChunkSizeExceeded says an audio chunk is too large.
	SpeechToTextRealtimeErrorChunkSizeExceeded SpeechToTextRealtimeMessageType = "chunk_size_exceeded"
	// SpeechToTextRealtimeErrorInsufficientAudioActivity says too little audio activity was sent
	// to keep the connection open.
	SpeechToTextRealtimeErrorInsufficientAudioActivity SpeechToTextRealtimeMessageType = "insufficient_audio_activity"
	// SpeechToTextRealtimeErrorTranscriber says the audio could not be transcribed.
	SpeechToTextRealtimeErrorTranscriber SpeechToTextRealtimeMessageType = "transcriber_error"
)

// SpeechToTextRealtimeEvent is an event the server sends. One struct decodes every event with
// [json.Unmarshal]: MessageType says which event it is, and a field the event does not carry
// stays zero. A message type or a field this package does not know decodes without error. A
// caller checks Err before it uses an event.
type SpeechToTextRealtimeEvent struct {
	// MessageType is the message_type property, which names the event.
	MessageType SpeechToTextRealtimeMessageType `json:"message_type"`
	// SessionID is the session_id property of a session_started event, the identifier of the
	// session.
	SessionID string `json:"session_id"`
	// Config is the config property of a session_started event, the configuration the session
	// runs with. It is nil on every other event.
	Config *SpeechToTextRealtimeSessionConfig `json:"config"`
	// Text is the text property: the interim text of a partial_transcript, the committed text of
	// a committed_transcript, and that same committed text on the events that follow it
	// (committed_transcript_with_timestamps, diarized_transcript, committed_transcript_entities,
	// redacted_transcript and edited_transcript).
	Text string `json:"text"`
	// LanguageCode is the language_code property of partial_transcript and
	// committed_transcript_with_timestamps events, sent with language detection. It is nil when
	// the event carries none or carries null, as a partial transcript does until the language is
	// known.
	LanguageCode *string `json:"language_code"`
	// Words is the words property: the timed words of a committed_transcript_with_timestamps,
	// the words and speakers of a diarized_transcript and, with diarization, the draft words of a
	// partial_transcript.
	Words []SpeechToTextRealtimeWord `json:"words"`
	// TurnProbability is the turn_probability property of a partial_transcript: the estimated
	// probability that the speaker has finished their turn. The scribe_v3_live model sends it.
	TurnProbability *float64 `json:"turn_probability"`
	// SegmentIndex is the segment_index property, sent with diarization: the zero-based index of
	// the committed segment, on a committed_transcript and on the diarized_transcript that
	// belongs to it.
	SegmentIndex *int `json:"segment_index"`
	// FromSpeakerID is the from_speaker_id property of a speakers_merged event: the speaker
	// identifier that every earlier partial and diarized transcript must now read as ToSpeakerID.
	// Merges chain, so a caller applies them in the order received.
	FromSpeakerID string `json:"from_speaker_id"`
	// ToSpeakerID is the to_speaker_id property of a speakers_merged event: the speaker
	// identifier that replaces FromSpeakerID.
	ToSpeakerID string `json:"to_speaker_id"`
	// Entities is the entities property of a committed_transcript_entities event. It is empty
	// when none were found.
	Entities []SpeechToTextRealtimeEntity `json:"entities"`
	// RedactedText is the redacted_text property of a redacted_transcript event: Text with the
	// redacted entities replaced by markers.
	RedactedText string `json:"redacted_text"`
	// EditedText is the edited_text property of an edited_transcript event: Text after the edit
	// instruction, identical to Text when nothing was edited.
	EditedText string `json:"edited_text"`
	// Warning is the warning property of a warning event.
	Warning string `json:"warning"`
	// ErrorMessage is the error property of an error event, the server's description of what
	// went wrong. Err reports it together with the event's type.
	ErrorMessage string `json:"error"`
}

// Err returns a [*SpeechToTextRealtimeError] when the event is a server error, and nil
// otherwise. An event is an error when its type is one of the 14 documented error types, or when
// it carries a non-empty error property, so that an error type the API adds later is still
// reported. The server sends the error event and then closes the connection.
func (e SpeechToTextRealtimeEvent) Err() error {
	switch e.MessageType {
	case SpeechToTextRealtimeErrorGeneric,
		SpeechToTextRealtimeErrorAuth,
		SpeechToTextRealtimeErrorQuotaExceeded,
		SpeechToTextRealtimeErrorCommitThrottled,
		SpeechToTextRealtimeErrorUnacceptedTerms,
		SpeechToTextRealtimeErrorRateLimited,
		SpeechToTextRealtimeErrorQueueOverflow,
		SpeechToTextRealtimeErrorResourceExhausted,
		SpeechToTextRealtimeErrorSessionTimeLimitExceeded,
		SpeechToTextRealtimeErrorInput,
		SpeechToTextRealtimeErrorInvalidRequest,
		SpeechToTextRealtimeErrorChunkSizeExceeded,
		SpeechToTextRealtimeErrorInsufficientAudioActivity,
		SpeechToTextRealtimeErrorTranscriber:
		return &SpeechToTextRealtimeError{Type: e.MessageType, Message: e.ErrorMessage}
	}
	if e.ErrorMessage != "" {
		return &SpeechToTextRealtimeError{Type: e.MessageType, Message: e.ErrorMessage}
	}
	return nil
}

// SpeechToTextRealtimeError is the error [SpeechToTextRealtimeEvent.Err] returns for an error
// event of the realtime speech-to-text API. A caller recovers it with [errors.As] and classifies
// it by Type, never by the error's text.
type SpeechToTextRealtimeError struct {
	// Type is the message type of the error event, one of the SpeechToTextRealtimeError
	// constants unless the API sent a type this package does not know.
	Type SpeechToTextRealtimeMessageType
	// Message is the server's description of the error, as sent. It may be empty.
	Message string
}

// Error returns "speech-to-text realtime" followed by the type and, after a colon, the message,
// each only when it is present.
func (e *SpeechToTextRealtimeError) Error() string {
	text := "speech-to-text realtime"
	if e.Type != "" {
		text += " " + string(e.Type)
	}
	if e.Message != "" {
		text += ": " + e.Message
	}
	return text
}

// SpeechToTextRealtimeSessionConfig is the configuration a session_started event reports: the
// settings the session runs with, as the API sends them. That is not the config schema of the API
// reference: the API reports the commit strategy as vad_commit_strategy and logging as
// disable_logging, and sends properties the schema does not list. A pointer member is nil when
// the event does not carry the property or carries null.
type SpeechToTextRealtimeSessionConfig struct {
	// SampleRate is the sample_rate property, the sample rate of the audio in Hz.
	SampleRate *int `json:"sample_rate"`
	// AudioFormat is the audio_format property, the encoding of the audio, in the values of
	// SpeechToTextRealtimeOptions.AudioFormat.
	AudioFormat string `json:"audio_format"`
	// LanguageCode is the language_code property, the language the connection asked for. It is
	// empty when the connection named none.
	LanguageCode string `json:"language_code"`
	// SecondaryLanguages is the secondary_languages property, the further languages the
	// connection asked for.
	SecondaryLanguages []string `json:"secondary_languages"`
	// TimestampsGranularity is the timestamps_granularity property, such as "word".
	TimestampsGranularity string `json:"timestamps_granularity"`
	// VADCommitStrategy is the vad_commit_strategy property: true when the session commits on
	// detected silence, as it does under the "vad" and "turn_prediction" strategies, and false
	// under "manual".
	VADCommitStrategy *bool `json:"vad_commit_strategy"`
	// VADSilenceThresholdSecs is the vad_silence_threshold_secs property, the silence threshold
	// in seconds.
	VADSilenceThresholdSecs *float64 `json:"vad_silence_threshold_secs"`
	// VADThreshold is the vad_threshold property, the threshold of voice activity detection.
	VADThreshold *float64 `json:"vad_threshold"`
	// MinSpeechDurationMs is the min_speech_duration_ms property, the minimum speech duration in
	// milliseconds.
	MinSpeechDurationMs *int `json:"min_speech_duration_ms"`
	// MinSilenceDurationMs is the min_silence_duration_ms property, the minimum silence duration
	// in milliseconds.
	MinSilenceDurationMs *int `json:"min_silence_duration_ms"`
	// MaxTokensToRecompute is the max_tokens_to_recompute property. The API's documentation does
	// not describe it.
	MaxTokensToRecompute *int `json:"max_tokens_to_recompute"`
	// ModelID is the model_id property, the model that transcribes the session.
	ModelID string `json:"model_id"`
	// DisableLogging is the disable_logging property. True means the session runs in zero
	// retention mode, which a connection asks for with EnableLogging set to false.
	DisableLogging *bool `json:"disable_logging"`
	// IncludeTimestamps is the include_timestamps property: whether the session sends word
	// timings with its committed transcripts.
	IncludeTimestamps *bool `json:"include_timestamps"`
	// IncludeLanguageDetection is the include_language_detection property: whether the session
	// sends the detected language with its transcripts.
	IncludeLanguageDetection *bool `json:"include_language_detection"`
	// FilterBackgroundAudio is the filter_background_audio property: whether voice activity
	// detection ignores background speech and ambient noise.
	FilterBackgroundAudio *bool `json:"filter_background_audio"`
	// Keyterms is the keyterms property, the terms the model is biased towards.
	Keyterms []string `json:"keyterms"`
	// NoVerbatim is the no_verbatim property: whether filler words and disfluencies are removed
	// from the transcript.
	NoVerbatim *bool `json:"no_verbatim"`
	// EntityDetection is the entity_detection property: the entity types the session detects.
	// The API expands a category the connection asked for, such as "pii", into its entity types.
	// It is nil when entity detection is off.
	EntityDetection []string `json:"entity_detection"`
	// EntityRedaction is the entity_redaction property: the entity types the session redacts,
	// expanded as EntityDetection is. It is nil when entity redaction is off.
	EntityRedaction []string `json:"entity_redaction"`
	// EntityRedactionMode is the entity_redaction_mode property, in the values of
	// SpeechToTextRealtimeOptions.EntityRedactionMode.
	EntityRedactionMode string `json:"entity_redaction_mode"`
	// KeepaliveIntervalMs is the keepalive_interval_ms property, the interval in milliseconds at
	// which the server sends keepalive partial transcripts. It is nil when keepalives are off.
	KeepaliveIntervalMs *int `json:"keepalive_interval_ms"`
}

// SpeechToTextRealtimeWord is a word of a transcript, on a committed_transcript_with_timestamps
// event, a diarized_transcript event, or among the draft words of a partial_transcript. It
// decodes both word shapes of the API reference. A pointer member is nil when the word does not
// carry the property or carries null.
type SpeechToTextRealtimeWord struct {
	// Text is the text property, the word or sound that was transcribed.
	Text string `json:"text"`
	// Start is the start property, the start time of the word in seconds. It is nil for a word
	// that is not aligned yet.
	Start *float64 `json:"start"`
	// End is the end property, the end time of the word in seconds. It is nil for a word that is
	// not aligned yet.
	End *float64 `json:"end"`
	// Type is the type property: "word", "spacing", or "audio_event" for a sound that is not a
	// word, such as laughter.
	Type string `json:"type"`
	// SpeakerID is the speaker_id property, such as "speaker_0". It is nil while the speaker is
	// not identified.
	SpeakerID *string `json:"speaker_id"`
	// Logprob is the logprob property, the log of the probability with which the word was
	// predicted: at most 0, and higher for a more confident prediction.
	Logprob *float64 `json:"logprob"`
	// Characters is the characters property, the characters that make up the word, each with its
	// timings when the API sends it as an object.
	Characters []SpeechToTextRealtimeCharacter `json:"characters"`
	// ChannelIndex is the channel_index property, the channel the word was spoken on in
	// multichannel audio. It is nil for single-channel audio.
	ChannelIndex *int `json:"channel_index"`
}

// SpeechToTextRealtimeCharacter is one character of a word. It has two shapes: the API reference
// describes the characters of a committed_transcript_with_timestamps word as JSON strings and
// those of a diarized word as objects with text, start and end, and the API sends objects on
// committed_transcript_with_timestamps too. Both decode into this type.
type SpeechToTextRealtimeCharacter struct {
	// Text is the character that was transcribed.
	Text string `json:"text"`
	// Start is the start time of the character in seconds. It is nil when the character was sent
	// as a string or carries no timing.
	Start *float64 `json:"start"`
	// End is the end time of the character in seconds. It is nil when the character was sent as
	// a string or carries no timing.
	End *float64 `json:"end"`
}

// UnmarshalJSON decodes either shape of a character: a JSON string sets Text and leaves Start
// and End nil, and an object sets all three. A JSON null leaves the character as it is, and any
// other JSON value is an error.
func (c *SpeechToTextRealtimeCharacter) UnmarshalJSON(b []byte) error {
	if string(b) == "null" {
		return nil
	}
	if len(b) > 0 && b[0] == '"' {
		var text string
		if err := json.Unmarshal(b, &text); err != nil {
			return err
		}
		*c = SpeechToTextRealtimeCharacter{Text: text}
		return nil
	}
	// A type with the same fields and no methods, so that decoding it does not call this method
	// again.
	type character SpeechToTextRealtimeCharacter
	var decoded character
	if err := json.Unmarshal(b, &decoded); err != nil {
		return err
	}
	*c = SpeechToTextRealtimeCharacter(decoded)
	return nil
}

// SpeechToTextRealtimeEntity is an entity detected in a committed transcript, as a
// committed_transcript_entities event reports it.
type SpeechToTextRealtimeEntity struct {
	// Text is the text property, the text that was identified as an entity.
	Text string `json:"text"`
	// EntityType is the entity_type property, such as "credit_card", "email_address" or
	// "person_name".
	EntityType string `json:"entity_type"`
	// StartChar is the start_char property, the position in the transcript text at which the
	// entity starts.
	StartChar int `json:"start_char"`
	// EndChar is the end_char property, the position in the transcript text at which the entity
	// ends.
	EndChar int `json:"end_char"`
}
