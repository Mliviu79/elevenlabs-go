package elevenlabs_test

import (
	"encoding/json"
	"errors"
	"fmt"
	"reflect"
	"strings"
	"testing"

	"github.com/Mliviu79/elevenlabs-go"
)

// Every fixture and expectation in this file was written by hand from the ElevenLabs
// documentation as fetched on 2026-11-18, and names the page it came from:
//
//   - "Realtime", the API reference at /docs/api-reference/speech-to-text/v-1-speech-to-text-realtime
//     (an AsyncAPI 2.6.0 document), called the API reference below.
//   - The guides under /docs/eleven-api/guides/how-to/speech-to-text/realtime: "Realtime event
//     reference", "Transcripts and commit strategies", "Speaker diarization", "Transcript editing"
//     and "Server-side streaming".
//   - "Keyterm prompting", at /docs/eleven-api/guides/how-to/speech-to-text/batch/keyterm-prompting.
//
// Fixture and type come from the same documents, so a passing test proves the package agrees with
// that reading of them. Values the documents do not give (timings, identifiers, error texts) are
// this repository's own and are marked as such.

func TestSpeechToTextRealtimeOptionsQuery(t *testing.T) {
	t.Run("zero value sets nothing", func(t *testing.T) {
		query := elevenlabs.SpeechToTextRealtimeOptions{}.Query()
		if query == nil {
			t.Errorf("Query() returned a nil map; a caller that sets a parameter on it would panic")
		}
		if len(query) != 0 {
			t.Errorf("Query() of the zero value carries %d parameters: %v", len(query), query)
		}
		if got := query.Encode(); got != "" {
			t.Errorf("Query().Encode() of the zero value = %q, want the empty string", got)
		}
	})

	testCases := []struct {
		name    string
		options elevenlabs.SpeechToTextRealtimeOptions
		want    string
	}{
		{
			// The 23 parameter names are the query binding of the API reference. Values: the
			// model, format, strategy and redaction mode are documented enum members; 1.5 and 100
			// are the VAD example of "Transcripts and commit strategies", which also gives the
			// token and the 1000 ms keepalive; the entity categories are the ones the API
			// reference lists. The server would refuse this combination (transcript editing
			// excludes entity detection, for one): the case pins the encoding, not a session.
			//
			// Keys are sorted, as url.Values.Encode emits them. Each list is two repeated keys in
			// the order given, deliberately not alphabetical. diarize, enable_logging and
			// no_verbatim are false, min_silence_duration_ms and vad_threshold are zero, and all
			// five are still sent because their pointers are set.
			name: "every parameter set",
			options: elevenlabs.SpeechToTextRealtimeOptions{
				ModelID:                  "scribe_v3_live",
				AudioFormat:              "pcm_16000",
				LanguageCode:             "en",
				SecondaryLanguages:       []string{"fr", "es"},
				CommitStrategy:           "vad",
				VADThreshold:             ptr(0.0),
				VADSilenceThresholdSecs:  ptr(1.5),
				MinSpeechDurationMs:      ptr(100),
				MinSilenceDurationMs:     ptr(0),
				IncludeTimestamps:        ptr(true),
				IncludeLanguageDetection: ptr(true),
				Keyterms:                 []string{"Scribe", "ElevenLabs"},
				NoVerbatim:               ptr(false),
				MultilangMode:            ptr(true),
				Diarize:                  ptr(false),
				EntityDetection:          []string{"pii", "pci"},
				EntityRedaction:          []string{"phi", "offensive_language"},
				EntityRedactionMode:      "redacted",
				TranscriptEdit:           "write-dates-as-ISO-8601",
				FilterBackgroundAudio:    ptr(true),
				KeepaliveIntervalMs:      ptr(1000),
				EnableLogging:            ptr(false),
				Token:                    "sutkn_1234567890",
			},
			want: strings.Join([]string{
				"audio_format=pcm_16000",
				"commit_strategy=vad",
				"diarize=false",
				"enable_logging=false",
				"entity_detection=pii",
				"entity_detection=pci",
				"entity_redaction=phi",
				"entity_redaction=offensive_language",
				"entity_redaction_mode=redacted",
				"filter_background_audio=true",
				"include_language_detection=true",
				"include_timestamps=true",
				"keepalive_interval_ms=1000",
				"keyterms=Scribe",
				"keyterms=ElevenLabs",
				"language_code=en",
				"min_silence_duration_ms=0",
				"min_speech_duration_ms=100",
				"model_id=scribe_v3_live",
				"multilang_mode=true",
				"no_verbatim=false",
				"secondary_languages=fr",
				"secondary_languages=es",
				"token=sutkn_1234567890",
				"transcript_edit=write-dates-as-ISO-8601",
				"vad_silence_threshold_secs=1.5",
				"vad_threshold=0",
			}, "&"),
		},
		{
			// The query string "Transcripts and commit strategies" gives for turn prediction,
			// with its keys in sorted order.
			name: "only what is set is sent",
			options: elevenlabs.SpeechToTextRealtimeOptions{
				ModelID:        "scribe_v3_live",
				AudioFormat:    "pcm_16000",
				CommitStrategy: "turn_prediction",
				Keyterms:       []string{},
			},
			want: "audio_format=pcm_16000&commit_strategy=turn_prediction&model_id=scribe_v3_live",
		},
		{
			// The query string "Keyterm prompting" gives for the WebSocket API, sorted.
			name: "keyterms repeat the key",
			options: elevenlabs.SpeechToTextRealtimeOptions{
				ModelID:  "scribe_v3_live",
				Keyterms: []string{"ElevenLabs", "AnotherTerm"},
			},
			want: "keyterms=ElevenLabs&keyterms=AnotherTerm&model_id=scribe_v3_live",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			if got := tc.options.Query().Encode(); got != tc.want {
				t.Errorf("Query().Encode() differs\n got: %s\nwant: %s", got, tc.want)
			}
		})
	}
}

func TestSpeechToTextRealtimeAudioChunkJSON(t *testing.T) {
	testCases := []struct {
		name  string
		chunk elevenlabs.SpeechToTextRealtimeAudioChunk
		want  string
	}{
		{
			// The final commit message of the WebSocket examples in "Server-side streaming",
			// byte for byte.
			name:  "commit only, nil audio",
			chunk: elevenlabs.SpeechToTextRealtimeAudioChunk{Commit: true, SampleRate: 16000},
			want:  `{"message_type":"input_audio_chunk","audio_base_64":"","commit":true,"sample_rate":16000}`,
		},
		{
			name:  "commit only, empty audio",
			chunk: elevenlabs.SpeechToTextRealtimeAudioChunk{Audio: []byte{}, Commit: true, SampleRate: 16000},
			want:  `{"message_type":"input_audio_chunk","audio_base_64":"","commit":true,"sample_rate":16000}`,
		},
		{
			// The audio message of the same examples. The bytes 0xfb 0xff are "+/8=" in the
			// standard alphabet with padding and "-_8" in the URL-safe one without.
			name:  "audio only",
			chunk: elevenlabs.SpeechToTextRealtimeAudioChunk{Audio: []byte{0xfb, 0xff}, SampleRate: 16000},
			want:  `{"message_type":"input_audio_chunk","audio_base_64":"+/8=","commit":false,"sample_rate":16000}`,
		},
		{
			// The four properties the API reference marks required on InputAudioChunk.
			name:  "zero value carries every required key",
			chunk: elevenlabs.SpeechToTextRealtimeAudioChunk{},
			want:  `{"message_type":"input_audio_chunk","audio_base_64":"","commit":false,"sample_rate":0}`,
		},
		{
			// previous_text is a property of InputAudioChunk in the API reference; the text is
			// the example of "Transcripts and commit strategies".
			name: "first chunk with previous text",
			chunk: elevenlabs.SpeechToTextRealtimeAudioChunk{
				Audio:        []byte{0x00, 0x01, 0x02, 0x03},
				SampleRate:   16000,
				PreviousText: "The previous text context",
			},
			want: `{"message_type":"input_audio_chunk","audio_base_64":"AAECAw==","commit":false,"sample_rate":16000,"previous_text":"The previous text context"}`,
		},
		{
			// The keyterm library message of "Keyterm prompting", with real audio in place of
			// its placeholder.
			name: "first chunk with a keyterm library",
			chunk: elevenlabs.SpeechToTextRealtimeAudioChunk{
				Audio:          []byte{0x00, 0x01, 0x02, 0x03},
				SampleRate:     16000,
				KeytermLibrary: []string{"Acetaminophen", "Metoprolol", "Atorvastatin"},
			},
			want: `{"message_type":"input_audio_chunk","audio_base_64":"AAECAw==","commit":false,"sample_rate":16000,"keyterm_library":["Acetaminophen","Metoprolol","Atorvastatin"]}`,
		},
		{
			name: "first chunk with previous text and a keyterm library",
			chunk: elevenlabs.SpeechToTextRealtimeAudioChunk{
				Audio:          []byte{0x00, 0x01, 0x02, 0x03},
				SampleRate:     16000,
				PreviousText:   "The previous text context",
				KeytermLibrary: []string{"Acetaminophen", "Metoprolol", "Atorvastatin"},
			},
			want: `{"message_type":"input_audio_chunk","audio_base_64":"AAECAw==","commit":false,"sample_rate":16000,"previous_text":"The previous text context","keyterm_library":["Acetaminophen","Metoprolol","Atorvastatin"]}`,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			got, err := json.Marshal(tc.chunk)
			if err != nil {
				t.Fatalf("marshalling the chunk: %v", err)
			}
			if string(got) != tc.want {
				t.Errorf("chunk marshalled differently\n got: %s\nwant: %s", got, tc.want)
			}
			if strings.Contains(string(got), "null") {
				t.Errorf("chunk carries a null, which the API does not document: %s", got)
			}

			gotFromPointer, err := json.Marshal(&tc.chunk)
			if err != nil {
				t.Fatalf("marshalling a pointer to the chunk: %v", err)
			}
			if string(gotFromPointer) != tc.want {
				t.Errorf("a pointer to the chunk marshalled differently\n got: %s\nwant: %s", gotFromPointer, tc.want)
			}
		})
	}
}

func TestSpeechToTextRealtimeEventDecode(t *testing.T) {
	render := func(event elevenlabs.SpeechToTextRealtimeEvent) string {
		rendered, err := json.Marshal(event)
		if err != nil {
			return fmt.Sprintf("%+v", event)
		}
		return string(rendered)
	}

	testCases := []struct {
		name    string
		payload string
		want    elevenlabs.SpeechToTextRealtimeEvent
		wantErr bool
	}{
		{
			// SessionStarted and all 14 properties of MessagesSessionStartedConfig in the API
			// reference. The VAD values 1.5, 0.4 and 100 are the example of "Transcripts and
			// commit strategies"; the session id and the remaining values are this repository's.
			name: "session_started",
			payload: `{
				"message_type": "session_started",
				"session_id": "0b0a72b57fd743ecbd9ad8e0a3bbd00a",
				"config": {
					"sample_rate": 16000,
					"audio_format": "pcm_16000",
					"language_code": "en",
					"commit_strategy": "vad",
					"vad_silence_threshold_secs": 1.5,
					"vad_threshold": 0.4,
					"min_speech_duration_ms": 100,
					"min_silence_duration_ms": 250,
					"model_id": "scribe_v2_realtime",
					"enable_logging": true,
					"include_timestamps": true,
					"include_language_detection": false,
					"keyterms": ["ElevenLabs", "Scribe"],
					"no_verbatim": false
				}
			}`,
			want: elevenlabs.SpeechToTextRealtimeEvent{
				MessageType: elevenlabs.SpeechToTextRealtimeSessionStarted,
				SessionID:   "0b0a72b57fd743ecbd9ad8e0a3bbd00a",
				Config: &elevenlabs.SpeechToTextRealtimeSessionConfig{
					SampleRate:               ptr(16000),
					AudioFormat:              "pcm_16000",
					LanguageCode:             "en",
					CommitStrategy:           "vad",
					VADSilenceThresholdSecs:  ptr(1.5),
					VADThreshold:             ptr(0.4),
					MinSpeechDurationMs:      ptr(100),
					MinSilenceDurationMs:     ptr(250),
					ModelID:                  "scribe_v2_realtime",
					EnableLogging:            ptr(true),
					IncludeTimestamps:        ptr(true),
					IncludeLanguageDetection: ptr(false),
					Keyterms:                 []string{"ElevenLabs", "Scribe"},
					NoVerbatim:               ptr(false),
				},
			},
		},
		{
			// "Transcripts and commit strategies" says the config echoes keepalive_interval_ms,
			// null when disabled; the API reference's schema does not list it, so it decodes as
			// an unknown field. The four booleans are the reverse pattern of the case above, so
			// between them the two cases tell each boolean from the other three.
			name: "session_started with a config property the schema does not list",
			payload: `{
				"message_type": "session_started",
				"session_id": "5c1f0d3a9b6e4a27a1c8d2e4f6a8b0c2",
				"config": {
					"keepalive_interval_ms": null,
					"enable_logging": true,
					"include_timestamps": false,
					"include_language_detection": true,
					"no_verbatim": false
				}
			}`,
			want: elevenlabs.SpeechToTextRealtimeEvent{
				MessageType: elevenlabs.SpeechToTextRealtimeSessionStarted,
				SessionID:   "5c1f0d3a9b6e4a27a1c8d2e4f6a8b0c2",
				Config: &elevenlabs.SpeechToTextRealtimeSessionConfig{
					EnableLogging:            ptr(true),
					IncludeTimestamps:        ptr(false),
					IncludeLanguageDetection: ptr(true),
					NoVerbatim:               ptr(false),
				},
			},
		},
		{
			// The turn prediction example of "Transcripts and commit strategies".
			name: "partial_transcript",
			payload: `{
				"message_type": "partial_transcript",
				"text": "My phone number is 415",
				"turn_probability": 0.08
			}`,
			want: elevenlabs.SpeechToTextRealtimeEvent{
				MessageType:     elevenlabs.SpeechToTextRealtimePartialTranscript,
				Text:            "My phone number is 415",
				TurnProbability: ptr(0.08),
			},
		},
		{
			// "Realtime event reference", Language code: always present with language
			// detection, and null until the segment has enough speech.
			name:    "partial_transcript with a null language code",
			payload: `{"message_type": "partial_transcript", "text": "Hi", "language_code": null}`,
			want: elevenlabs.SpeechToTextRealtimeEvent{
				MessageType: elevenlabs.SpeechToTextRealtimePartialTranscript,
				Text:        "Hi",
			},
		},
		{
			// "Transcripts and commit strategies": a keepalive is a partial transcript with
			// empty text when the segment has no uncommitted text.
			name:    "partial_transcript sent as a keepalive",
			payload: `{"message_type": "partial_transcript", "text": ""}`,
			want: elevenlabs.SpeechToTextRealtimeEvent{
				MessageType: elevenlabs.SpeechToTextRealtimePartialTranscript,
			},
		},
		{
			// "Speaker diarization", Partial transcripts: draft words whose speaker is null
			// while unresolved and whose timings are absent until aligned.
			name: "partial_transcript with draft words",
			payload: `{
				"message_type": "partial_transcript",
				"text": "Hello there",
				"words": [
					{"text": "Hello", "speaker_id": "speaker_0"},
					{"text": "there", "speaker_id": null, "start": null, "end": null}
				]
			}`,
			want: elevenlabs.SpeechToTextRealtimeEvent{
				MessageType: elevenlabs.SpeechToTextRealtimePartialTranscript,
				Text:        "Hello there",
				Words: []elevenlabs.SpeechToTextRealtimeWord{
					{Text: "Hello", SpeakerID: ptr("speaker_0")},
					{Text: "there"},
				},
			},
		},
		{
			// The committed transcript example of "Speaker diarization".
			name: "committed_transcript",
			payload: `{
				"message_type": "committed_transcript",
				"segment_index": 0,
				"text": "Hello."
			}`,
			want: elevenlabs.SpeechToTextRealtimeEvent{
				MessageType:  elevenlabs.SpeechToTextRealtimeCommittedTranscript,
				Text:         "Hello.",
				SegmentIndex: ptr(0),
			},
		},
		{
			// CommittedTranscriptWithTimestamps and TranscriptionWord in the API reference,
			// where a word's characters are strings. Texts and timings are this repository's.
			name: "committed_transcript_with_timestamps",
			payload: `{
				"message_type": "committed_transcript_with_timestamps",
				"text": "Hello ",
				"language_code": "en",
				"words": [
					{
						"text": "Hello",
						"start": 0,
						"end": 0.42,
						"type": "word",
						"speaker_id": "speaker_0",
						"logprob": -0.05,
						"characters": ["H", "e", "l", "l", "o"]
					},
					{
						"text": " ",
						"start": 0.42,
						"end": 0.5,
						"type": "spacing",
						"logprob": 0,
						"characters": [" "]
					}
				]
			}`,
			want: elevenlabs.SpeechToTextRealtimeEvent{
				MessageType:  elevenlabs.SpeechToTextRealtimeCommittedTranscriptWithTimestamps,
				Text:         "Hello ",
				LanguageCode: ptr("en"),
				Words: []elevenlabs.SpeechToTextRealtimeWord{
					{
						Text:      "Hello",
						Start:     ptr(0.0),
						End:       ptr(0.42),
						Type:      "word",
						SpeakerID: ptr("speaker_0"),
						Logprob:   ptr(-0.05),
						Characters: []elevenlabs.SpeechToTextRealtimeCharacter{
							{Text: "H"}, {Text: "e"}, {Text: "l"}, {Text: "l"}, {Text: "o"},
						},
					},
					{
						Text:       " ",
						Start:      ptr(0.42),
						End:        ptr(0.5),
						Type:       "spacing",
						Logprob:    ptr(0.0),
						Characters: []elevenlabs.SpeechToTextRealtimeCharacter{{Text: " "}},
					},
				},
			},
		},
		{
			// The same schema, whose language_code and words may both be null.
			name: "committed_transcript_with_timestamps with null words",
			payload: `{
				"message_type": "committed_transcript_with_timestamps",
				"text": "Hello",
				"language_code": null,
				"words": null
			}`,
			want: elevenlabs.SpeechToTextRealtimeEvent{
				MessageType: elevenlabs.SpeechToTextRealtimeCommittedTranscriptWithTimestamps,
				Text:        "Hello",
			},
		},
		{
			// A character sent as a string is a JSON string: an escape in it is decoded.
			name: "committed_transcript_with_timestamps with escaped characters",
			payload: `{
				"message_type": "committed_transcript_with_timestamps",
				"text": "\"é",
				"words": [{"text": "\"é", "type": "word", "characters": ["\"", "é"]}]
			}`,
			want: elevenlabs.SpeechToTextRealtimeEvent{
				MessageType: elevenlabs.SpeechToTextRealtimeCommittedTranscriptWithTimestamps,
				Text:        `"é`,
				Words: []elevenlabs.SpeechToTextRealtimeWord{
					{
						Text:       `"é`,
						Type:       "word",
						Characters: []elevenlabs.SpeechToTextRealtimeCharacter{{Text: `"`}, {Text: "é"}},
					},
				},
			},
		},
		{
			// The diarized transcript example of "Speaker diarization".
			name: "diarized_transcript",
			payload: `{
				"message_type": "diarized_transcript",
				"segment_index": 0,
				"text": "Hello.",
				"words": [
					{
						"text": "Hello.",
						"speaker_id": "speaker_0"
					}
				]
			}`,
			want: elevenlabs.SpeechToTextRealtimeEvent{
				MessageType:  elevenlabs.SpeechToTextRealtimeDiarizedTranscript,
				Text:         "Hello.",
				SegmentIndex: ptr(0),
				Words: []elevenlabs.SpeechToTextRealtimeWord{
					{Text: "Hello.", SpeakerID: ptr("speaker_0")},
				},
			},
		},
		{
			// DiarizedTranscriptWordsItems in the API reference, the full word: its characters
			// are objects with nullable timings, its speaker may be null and it carries a
			// channel index. Texts and timings are this repository's.
			name: "diarized_transcript with character objects",
			payload: `{
				"message_type": "diarized_transcript",
				"segment_index": 1,
				"text": "Hi",
				"words": [
					{
						"text": "Hi",
						"start": 0.1,
						"end": 0.3,
						"type": "word",
						"speaker_id": null,
						"logprob": -0.2,
						"characters": [
							{"text": "H", "start": 0.1, "end": 0.2},
							{"text": "i", "start": null, "end": null}
						],
						"channel_index": 0
					}
				]
			}`,
			want: elevenlabs.SpeechToTextRealtimeEvent{
				MessageType:  elevenlabs.SpeechToTextRealtimeDiarizedTranscript,
				Text:         "Hi",
				SegmentIndex: ptr(1),
				Words: []elevenlabs.SpeechToTextRealtimeWord{
					{
						Text:    "Hi",
						Start:   ptr(0.1),
						End:     ptr(0.3),
						Type:    "word",
						Logprob: ptr(-0.2),
						Characters: []elevenlabs.SpeechToTextRealtimeCharacter{
							{Text: "H", Start: ptr(0.1), End: ptr(0.2)},
							{Text: "i"},
						},
						ChannelIndex: ptr(0),
					},
				},
			},
		},
		{
			// The schema lets a word's characters and channel index be null. A null inside the
			// list is not documented; it decodes as an empty character.
			name: "diarized_transcript with null characters",
			payload: `{
				"message_type": "diarized_transcript",
				"segment_index": 2,
				"text": "Hi you",
				"words": [
					{"text": "Hi", "type": "word", "logprob": -0.2, "characters": [null], "channel_index": null},
					{"text": "you", "type": "word", "logprob": -0.3, "characters": null}
				]
			}`,
			want: elevenlabs.SpeechToTextRealtimeEvent{
				MessageType:  elevenlabs.SpeechToTextRealtimeDiarizedTranscript,
				Text:         "Hi you",
				SegmentIndex: ptr(2),
				Words: []elevenlabs.SpeechToTextRealtimeWord{
					{
						Text:       "Hi",
						Type:       "word",
						Logprob:    ptr(-0.2),
						Characters: []elevenlabs.SpeechToTextRealtimeCharacter{{}},
					},
					{Text: "you", Type: "word", Logprob: ptr(-0.3)},
				},
			},
		},
		{
			// The speaker merge example of "Speaker diarization".
			name: "speakers_merged",
			payload: `{
				"message_type": "speakers_merged",
				"from_speaker_id": "speaker_1",
				"to_speaker_id": "speaker_0"
			}`,
			want: elevenlabs.SpeechToTextRealtimeEvent{
				MessageType:   elevenlabs.SpeechToTextRealtimeSpeakersMerged,
				FromSpeakerID: "speaker_1",
				ToSpeakerID:   "speaker_0",
			},
		},
		{
			// CommittedTranscriptEntities and DetectedEntity in the API reference, which names
			// person_name as an entity type. The text and positions are this repository's.
			name: "committed_transcript_entities",
			payload: `{
				"message_type": "committed_transcript_entities",
				"text": "My name is John Smith.",
				"entities": [
					{"text": "John Smith", "entity_type": "person_name", "start_char": 11, "end_char": 21}
				]
			}`,
			want: elevenlabs.SpeechToTextRealtimeEvent{
				MessageType: elevenlabs.SpeechToTextRealtimeCommittedTranscriptEntities,
				Text:        "My name is John Smith.",
				Entities: []elevenlabs.SpeechToTextRealtimeEntity{
					{Text: "John Smith", EntityType: "person_name", StartChar: 11, EndChar: 21},
				},
			},
		},
		{
			// The same schema: "Detected entities. Empty if none were found."
			name:    "committed_transcript_entities with none found",
			payload: `{"message_type": "committed_transcript_entities", "text": "Hello.", "entities": []}`,
			want: elevenlabs.SpeechToTextRealtimeEvent{
				MessageType: elevenlabs.SpeechToTextRealtimeCommittedTranscriptEntities,
				Text:        "Hello.",
				Entities:    []elevenlabs.SpeechToTextRealtimeEntity{},
			},
		},
		{
			// RedactedTranscript in the API reference. The marker follows the
			// enumerated_entity_type form the reference describes, {ENTITY_TYPE_N}; the text is
			// this repository's.
			name: "redacted_transcript",
			payload: `{
				"message_type": "redacted_transcript",
				"text": "My name is John Smith.",
				"redacted_text": "My name is {PERSON_NAME_1}."
			}`,
			want: elevenlabs.SpeechToTextRealtimeEvent{
				MessageType:  elevenlabs.SpeechToTextRealtimeRedactedTranscript,
				Text:         "My name is John Smith.",
				RedactedText: "My name is {PERSON_NAME_1}.",
			},
		},
		{
			// The edited transcript example of "Transcript editing".
			name: "edited_transcript",
			payload: `{
				"message_type": "edited_transcript",
				"text": "our next meeting is on the twelfth of July twenty twenty-six",
				"edited_text": "our next meeting is on 2026-07-12"
			}`,
			want: elevenlabs.SpeechToTextRealtimeEvent{
				MessageType: elevenlabs.SpeechToTextRealtimeEditedTranscript,
				Text:        "our next meeting is on the twelfth of July twenty twenty-six",
				EditedText:  "our next meeting is on 2026-07-12",
			},
		},
		{
			// ScribeWarning in the API reference. The warning text is this repository's.
			name:    "warning",
			payload: `{"message_type": "warning", "warning": "audio is arriving faster than realtime"}`,
			want: elevenlabs.SpeechToTextRealtimeEvent{
				MessageType: elevenlabs.SpeechToTextRealtimeWarning,
				Warning:     "audio is arriving faster than realtime",
			},
		},
		{
			// Neither the type nor the extra field is documented: the API may add both.
			name: "an unknown message type with an unknown field",
			payload: `{
				"message_type": "some_future_event",
				"text": "Hello.",
				"some_future_field": {"nested": [1, 2, 3]}
			}`,
			want: elevenlabs.SpeechToTextRealtimeEvent{
				MessageType: elevenlabs.SpeechToTextRealtimeMessageType("some_future_event"),
				Text:        "Hello.",
			},
		},
		{
			name: "a character that is neither a string nor an object",
			payload: `{
				"message_type": "diarized_transcript",
				"segment_index": 0,
				"text": "Hi",
				"words": [{"text": "Hi", "characters": [42]}]
			}`,
			wantErr: true,
		},
		{
			name:    "invalid JSON",
			payload: `{"message_type": "partial_transcript", "text": `,
			wantErr: true,
		},
	}

	// The 14 error events of the API reference, each with the "error" property its schema
	// requires. The invalid_request text is the one "Speaker diarization" documents; the others
	// are this repository's. The wire name is a literal beside the constant so that a wrong
	// constant cannot agree with itself.
	errorEvents := []struct {
		messageType elevenlabs.SpeechToTextRealtimeMessageType
		wire        string
		message     string
	}{
		{elevenlabs.SpeechToTextRealtimeErrorGeneric, "error", "entity redaction failed"},
		{elevenlabs.SpeechToTextRealtimeErrorAuth, "auth_error", "the API key is not valid"},
		{elevenlabs.SpeechToTextRealtimeErrorQuotaExceeded, "quota_exceeded", "the usage quota is spent"},
		{elevenlabs.SpeechToTextRealtimeErrorCommitThrottled, "commit_throttled", "too many commits"},
		{elevenlabs.SpeechToTextRealtimeErrorUnacceptedTerms, "unaccepted_terms", "the terms are not accepted"},
		{elevenlabs.SpeechToTextRealtimeErrorRateLimited, "rate_limited", "too many requests"},
		{elevenlabs.SpeechToTextRealtimeErrorQueueOverflow, "queue_overflow", "the queue is full"},
		{elevenlabs.SpeechToTextRealtimeErrorResourceExhausted, "resource_exhausted", "the server is at capacity"},
		{elevenlabs.SpeechToTextRealtimeErrorSessionTimeLimitExceeded, "session_time_limit_exceeded", "the session is too long"},
		{elevenlabs.SpeechToTextRealtimeErrorInput, "input_error", "the chunk could not be read"},
		{elevenlabs.SpeechToTextRealtimeErrorInvalidRequest, "invalid_request", "The 'diarize' parameter is only supported by scribe_v3_live."},
		{elevenlabs.SpeechToTextRealtimeErrorChunkSizeExceeded, "chunk_size_exceeded", "the chunk is too large"},
		{elevenlabs.SpeechToTextRealtimeErrorInsufficientAudioActivity, "insufficient_audio_activity", "not enough audio was sent"},
		{elevenlabs.SpeechToTextRealtimeErrorTranscriber, "transcriber_error", "the transcriber failed"},
	}
	for _, errorEvent := range errorEvents {
		testCases = append(testCases, struct {
			name    string
			payload string
			want    elevenlabs.SpeechToTextRealtimeEvent
			wantErr bool
		}{
			name:    errorEvent.wire,
			payload: `{"message_type": "` + errorEvent.wire + `", "error": "` + errorEvent.message + `"}`,
			want: elevenlabs.SpeechToTextRealtimeEvent{
				MessageType:  errorEvent.messageType,
				ErrorMessage: errorEvent.message,
			},
		})
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			var got elevenlabs.SpeechToTextRealtimeEvent
			err := json.Unmarshal([]byte(tc.payload), &got)
			if tc.wantErr {
				if err == nil {
					t.Errorf("decoding succeeded, want an error; decoded %s", render(got))
				}
				return
			}
			if err != nil {
				t.Fatalf("decoding the event: %v", err)
			}
			if !reflect.DeepEqual(got, tc.want) {
				t.Errorf("decoded event differs\n got: %s\nwant: %s", render(got), render(tc.want))
			}
		})
	}
}

func TestSpeechToTextRealtimeEventErr(t *testing.T) {
	const prefix = "speech-to-text realtime"

	// decode builds the event from a payload, the way a consumer receives it, so that the
	// message type under test comes from the wire name and never from the package's constant.
	decode := func(t *testing.T, payload string) elevenlabs.SpeechToTextRealtimeEvent {
		t.Helper()
		var event elevenlabs.SpeechToTextRealtimeEvent
		if err := json.Unmarshal([]byte(payload), &event); err != nil {
			t.Fatalf("decoding %s: %v", payload, err)
		}
		return event
	}

	// The 24 message types of the API reference's V1SpeechToTextRealtimeSubscribe, in its
	// order. The wire name and the error text are literals beside the constant so that a wrong
	// constant cannot agree with itself. The 14 error types are the "Error handling" table of
	// "Realtime event reference"; the ten others, warning included, are the controls that must
	// go the other way.
	messageTypes := []struct {
		messageType elevenlabs.SpeechToTextRealtimeMessageType
		wire        string
		isError     bool
		wantText    string
	}{
		{elevenlabs.SpeechToTextRealtimeSessionStarted, "session_started", false, ""},
		{elevenlabs.SpeechToTextRealtimePartialTranscript, "partial_transcript", false, ""},
		{elevenlabs.SpeechToTextRealtimeCommittedTranscript, "committed_transcript", false, ""},
		{elevenlabs.SpeechToTextRealtimeDiarizedTranscript, "diarized_transcript", false, ""},
		{elevenlabs.SpeechToTextRealtimeSpeakersMerged, "speakers_merged", false, ""},
		{elevenlabs.SpeechToTextRealtimeCommittedTranscriptWithTimestamps, "committed_transcript_with_timestamps", false, ""},
		{elevenlabs.SpeechToTextRealtimeCommittedTranscriptEntities, "committed_transcript_entities", false, ""},
		{elevenlabs.SpeechToTextRealtimeRedactedTranscript, "redacted_transcript", false, ""},
		{elevenlabs.SpeechToTextRealtimeEditedTranscript, "edited_transcript", false, ""},
		{elevenlabs.SpeechToTextRealtimeWarning, "warning", false, ""},
		{elevenlabs.SpeechToTextRealtimeErrorGeneric, "error", true, "speech-to-text realtime error: the server said no"},
		{elevenlabs.SpeechToTextRealtimeErrorAuth, "auth_error", true, "speech-to-text realtime auth_error: the server said no"},
		{elevenlabs.SpeechToTextRealtimeErrorQuotaExceeded, "quota_exceeded", true, "speech-to-text realtime quota_exceeded: the server said no"},
		{elevenlabs.SpeechToTextRealtimeErrorCommitThrottled, "commit_throttled", true, "speech-to-text realtime commit_throttled: the server said no"},
		{elevenlabs.SpeechToTextRealtimeErrorUnacceptedTerms, "unaccepted_terms", true, "speech-to-text realtime unaccepted_terms: the server said no"},
		{elevenlabs.SpeechToTextRealtimeErrorRateLimited, "rate_limited", true, "speech-to-text realtime rate_limited: the server said no"},
		{elevenlabs.SpeechToTextRealtimeErrorQueueOverflow, "queue_overflow", true, "speech-to-text realtime queue_overflow: the server said no"},
		{elevenlabs.SpeechToTextRealtimeErrorResourceExhausted, "resource_exhausted", true, "speech-to-text realtime resource_exhausted: the server said no"},
		{elevenlabs.SpeechToTextRealtimeErrorSessionTimeLimitExceeded, "session_time_limit_exceeded", true, "speech-to-text realtime session_time_limit_exceeded: the server said no"},
		{elevenlabs.SpeechToTextRealtimeErrorInput, "input_error", true, "speech-to-text realtime input_error: the server said no"},
		{elevenlabs.SpeechToTextRealtimeErrorInvalidRequest, "invalid_request", true, "speech-to-text realtime invalid_request: the server said no"},
		{elevenlabs.SpeechToTextRealtimeErrorChunkSizeExceeded, "chunk_size_exceeded", true, "speech-to-text realtime chunk_size_exceeded: the server said no"},
		{elevenlabs.SpeechToTextRealtimeErrorInsufficientAudioActivity, "insufficient_audio_activity", true, "speech-to-text realtime insufficient_audio_activity: the server said no"},
		{elevenlabs.SpeechToTextRealtimeErrorTranscriber, "transcriber_error", true, "speech-to-text realtime transcriber_error: the server said no"},
	}

	// assertError checks that err is the realtime error a consumer recovers with errors.As,
	// through a wrap of its own as well, and that it carries the type, the message and the text.
	assertError := func(t *testing.T, err error, wantType, wantMessage, wantText string) {
		t.Helper()
		if err == nil {
			t.Errorf("Err() = nil, want an error of type %s", wantType)
			return
		}
		var realtimeErr *elevenlabs.SpeechToTextRealtimeError
		if !errors.As(fmt.Errorf("reading an event: %w", err), &realtimeErr) {
			t.Errorf("Err() returned %T, which errors.As does not recover as *SpeechToTextRealtimeError", err)
			return
		}
		if string(realtimeErr.Type) != wantType {
			t.Errorf("error Type = %q, want %q", realtimeErr.Type, wantType)
		}
		if realtimeErr.Message != wantMessage {
			t.Errorf("error Message = %q, want %q", realtimeErr.Message, wantMessage)
		}
		if got := err.Error(); got != wantText {
			t.Errorf("Error() = %q, want %q", got, wantText)
		}
	}

	for _, tc := range messageTypes {
		t.Run(tc.wire, func(t *testing.T) {
			if string(tc.messageType) != tc.wire {
				t.Errorf("the constant is %q, want the documented %q", tc.messageType, tc.wire)
			}

			if !tc.isError {
				event := decode(t, `{"message_type": "`+tc.wire+`", "text": "Hello."}`)
				if err := event.Err(); err != nil {
					t.Errorf("Err() = %v, want nil: %s is not an error event", err, tc.wire)
				}
				return
			}

			event := decode(t, `{"message_type": "`+tc.wire+`", "error": "the server said no"}`)
			assertError(t, event.Err(), tc.wire, "the server said no", tc.wantText)

			// With no message, only the type says the event is an error.
			withoutMessage := decode(t, `{"message_type": "`+tc.wire+`", "error": ""}`)
			assertError(t, withoutMessage.Err(), tc.wire, "", prefix+" "+tc.wire)
		})
	}

	t.Run("a warning with its text is not an error", func(t *testing.T) {
		event := decode(t, `{"message_type": "warning", "warning": "audio is arriving faster than realtime"}`)
		if err := event.Err(); err != nil {
			t.Errorf("Err() = %v, want nil: a warning is not an error event", err)
		}
	})

	t.Run("an undocumented type without an error field is not an error", func(t *testing.T) {
		event := decode(t, `{"message_type": "some_future_event", "text": "Hello."}`)
		if err := event.Err(); err != nil {
			t.Errorf("Err() = %v, want nil", err)
		}
	})

	t.Run("an undocumented type carrying an error field is an error", func(t *testing.T) {
		event := decode(t, `{"message_type": "some_future_error", "error": "try again later"}`)
		assertError(t, event.Err(), "some_future_error", "try again later",
			"speech-to-text realtime some_future_error: try again later")
	})

	t.Run("a documented type carrying an error field is an error", func(t *testing.T) {
		event := decode(t, `{"message_type": "committed_transcript", "text": "Hello.", "error": "try again later"}`)
		assertError(t, event.Err(), "committed_transcript", "try again later",
			"speech-to-text realtime committed_transcript: try again later")
	})

	errorTexts := []struct {
		name string
		err  *elevenlabs.SpeechToTextRealtimeError
		want string
	}{
		{
			name: "type and message",
			err: &elevenlabs.SpeechToTextRealtimeError{
				Type:    elevenlabs.SpeechToTextRealtimeMessageType("auth_error"),
				Message: "Invalid API key.",
			},
			want: "speech-to-text realtime auth_error: Invalid API key.",
		},
		{
			name: "type only",
			err:  &elevenlabs.SpeechToTextRealtimeError{Type: elevenlabs.SpeechToTextRealtimeMessageType("auth_error")},
			want: "speech-to-text realtime auth_error",
		},
		{
			name: "message only",
			err:  &elevenlabs.SpeechToTextRealtimeError{Message: "Invalid API key."},
			want: "speech-to-text realtime: Invalid API key.",
		},
		{
			name: "zero value",
			err:  &elevenlabs.SpeechToTextRealtimeError{},
			want: "speech-to-text realtime",
		},
	}
	for _, tc := range errorTexts {
		t.Run("Error text with "+tc.name, func(t *testing.T) {
			if got := tc.err.Error(); got != tc.want {
				t.Errorf("Error() = %q, want %q", got, tc.want)
			}
		})
	}
}
