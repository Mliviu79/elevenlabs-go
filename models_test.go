package elevenlabs_test

import (
	"encoding/json"
	"reflect"
	"testing"

	"github.com/Mliviu79/elevenlabs-go"
)

// decodeObject decodes a JSON object into a generic map so that two bodies can
// be compared regardless of key order.
func decodeObject(t *testing.T, body []byte) map[string]any {
	t.Helper()
	var m map[string]any
	if err := json.Unmarshal(body, &m); err != nil {
		t.Fatalf("decoding %s: %v", body, err)
	}
	return m
}

func TestTextToSpeechRequestCarriesExplicitValues(t *testing.T) {
	testCases := []struct {
		name string
		body string
	}{
		{
			name: "zero and false settings",
			body: `{
				"text": "hello",
				"voice_settings": {
					"stability": 0,
					"similarity_boost": 0,
					"style": 0,
					"speed": 1,
					"use_speaker_boost": false
				},
				"seed": 0,
				"apply_language_text_normalization": false
			}`,
		},
		{
			name: "zero and false settings with a normalization mode",
			body: `{
				"text": "hello",
				"voice_settings": {
					"stability": 0,
					"similarity_boost": 0,
					"style": 0,
					"speed": 1,
					"use_speaker_boost": false
				},
				"seed": 0,
				"apply_text_normalization": "off",
				"apply_language_text_normalization": false
			}`,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			var req elevenlabs.TextToSpeechRequest
			if err := json.Unmarshal([]byte(tc.body), &req); err != nil {
				t.Fatalf("TextToSpeechRequest cannot carry a value the API accepts: %v", err)
			}
			got, err := json.Marshal(req)
			if err != nil {
				t.Fatalf("marshalling TextToSpeechRequest: %v", err)
			}
			want := decodeObject(t, []byte(tc.body))
			if gotMap := decodeObject(t, got); !reflect.DeepEqual(gotMap, want) {
				t.Errorf("request body changed in transit\n got: %s\nwant: %s", got, tc.body)
			}
		})
	}
}

func TestTextToSpeechRequestOmitsUnsetFields(t *testing.T) {
	got, err := json.Marshal(elevenlabs.TextToSpeechRequest{Text: "hello"})
	if err != nil {
		t.Fatalf("marshalling TextToSpeechRequest: %v", err)
	}
	want := map[string]any{"text": "hello"}
	if gotMap := decodeObject(t, got); !reflect.DeepEqual(gotMap, want) {
		t.Errorf("unset optional fields reached the body: %s", got)
	}
}
