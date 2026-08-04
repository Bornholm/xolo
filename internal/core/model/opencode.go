package model

type OpenCodeModalities struct {
	Input  []string `json:"input"`
	Output []string `json:"output"`
}

type OpenCodeLimit struct {
	Context int64 `json:"context,omitempty"`
	Output  int64 `json:"output,omitempty"`
}

type OpenCodeModelEntry struct {
	Name       string              `json:"name"`
	Limit      *OpenCodeLimit      `json:"limit,omitempty"`
	Modalities *OpenCodeModalities `json:"modalities,omitempty"`
	Attachment bool                `json:"attachment,omitempty"`
	Tools      bool                `json:"tools,omitempty"`
	Reasoning  bool                `json:"reasoning,omitempty"`
}

func OpenCodeEntry(caps ModelCapabilities, ctxWindow, outWindow int64) OpenCodeModelEntry {
	input := []string{"text"}
	output := []string{"text"}
	attachment := false

	if caps.Vision {
		input = append(input, "image")
		attachment = true
	}
	if caps.Audio {
		input = append(input, "audio")
		output = append(output, "audio")
	}

	entry := OpenCodeModelEntry{
		Modalities: &OpenCodeModalities{Input: input, Output: output},
		Attachment: attachment,
		Tools:      caps.Tools,
		Reasoning:  caps.Reasoning,
	}

	if ctxWindow > 0 || outWindow > 0 {
		lim := &OpenCodeLimit{}
		if ctxWindow > 0 {
			lim.Context = ctxWindow
		}
		if outWindow > 0 {
			lim.Output = outWindow
		}
		entry.Limit = lim
	}

	return entry
}
