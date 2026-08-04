package model

import (
	"reflect"
	"testing"
)

func TestOpenCodeEntry(t *testing.T) {
	tests := []struct {
		name             string
		caps             ModelCapabilities
		ctx, out         int64
		wantMod          *OpenCodeModalities
		wantAttachment   bool
		wantTools        bool
		wantReasoning    bool
		wantLimitCtx     int64
		wantLimitOut     int64
		wantLimitNonNil  bool
	}{
		{
			name: "no caps, no limits",
			caps: ModelCapabilities{},
			wantMod: &OpenCodeModalities{
				Input:  []string{"text"},
				Output: []string{"text"},
			},
			wantAttachment: false,
			wantTools:      false,
			wantReasoning:  false,
			wantLimitNonNil: false,
		},
		{
			name: "vision only",
			caps: ModelCapabilities{Vision: true},
			wantMod: &OpenCodeModalities{
				Input:  []string{"text", "image"},
				Output: []string{"text"},
			},
			wantAttachment: true,
			wantTools:      false,
			wantReasoning:  false,
			wantLimitNonNil: false,
		},
		{
			name: "audio only",
			caps: ModelCapabilities{Audio: true},
			wantMod: &OpenCodeModalities{
				Input:  []string{"text", "audio"},
				Output: []string{"text", "audio"},
			},
			wantAttachment: false,
			wantTools:      false,
			wantReasoning:  false,
			wantLimitNonNil: false,
		},
		{
			name: "all caps, full limits",
			caps: ModelCapabilities{
				Vision:    true,
				Audio:     true,
				Tools:     true,
				Reasoning: true,
			},
			ctx: 128000,
			out: 8192,
			wantMod: &OpenCodeModalities{
				Input:  []string{"text", "image", "audio"},
				Output: []string{"text", "audio"},
			},
			wantAttachment: true,
			wantTools:      true,
			wantReasoning:  true,
			wantLimitCtx:   128000,
			wantLimitOut:   8192,
			wantLimitNonNil: true,
		},
		{
			name: "ctx only, no out",
			caps: ModelCapabilities{Tools: true},
			ctx:  100000,
			wantMod: &OpenCodeModalities{
				Input:  []string{"text"},
				Output: []string{"text"},
			},
			wantAttachment: false,
			wantTools:      true,
			wantReasoning:  false,
			wantLimitCtx:   100000,
			wantLimitNonNil: true,
		},
		{
			name: "limits zero stay nil",
			caps: ModelCapabilities{},
			wantMod: &OpenCodeModalities{
				Input:  []string{"text"},
				Output: []string{"text"},
			},
			wantLimitNonNil: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			entry := OpenCodeEntry(tt.caps, tt.ctx, tt.out)

			if !reflect.DeepEqual(entry.Modalities, tt.wantMod) {
				t.Errorf("Modalities mismatch: got %+v, want %+v",
					entry.Modalities, tt.wantMod)
			}
			if entry.Attachment != tt.wantAttachment {
				t.Errorf("Attachment: got %v, want %v",
					entry.Attachment, tt.wantAttachment)
			}
			if entry.Tools != tt.wantTools {
				t.Errorf("Tools: got %v, want %v",
					entry.Tools, tt.wantTools)
			}
			if entry.Reasoning != tt.wantReasoning {
				t.Errorf("Reasoning: got %v, want %v",
					entry.Reasoning, tt.wantReasoning)
			}

			if tt.wantLimitNonNil {
				if entry.Limit == nil {
					t.Fatal("Limit expected non-nil, got nil")
				}
				if entry.Limit.Context != tt.wantLimitCtx {
					t.Errorf("Limit.Context: got %d, want %d",
						entry.Limit.Context, tt.wantLimitCtx)
				}
				if entry.Limit.Output != tt.wantLimitOut {
					t.Errorf("Limit.Output: got %d, want %d",
						entry.Limit.Output, tt.wantLimitOut)
				}
			} else if entry.Limit != nil {
				t.Errorf("Limit expected nil, got %+v", entry.Limit)
			}
		})
	}
}
