package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/anthropics/anthropic-sdk-go"
	"github.com/anthropics/anthropic-sdk-go/option"
)

// keepTurnText says whether a turn's text belongs in the answer. Text written
// alongside a read_page call is narration before reading ("let me check");
// text beside contact_team alone is the answer itself, written before the
// model offers the email.
func keepTurnText(called []string) bool {
	for _, n := range called {
		if n == "read_page" {
			return false
		}
	}
	return true
}

// ---------- Claude Haiku 5.5 through the Anthropic API ----------

// Haiku 5.5 list prices per million tokens for prompts up to 100K tokens.
// Cache writes are billed at 1.25x input and cache reads at 0.1x input.
const (
	haikuIn  = 0.10
	haikuOut = 0.50
)

type Haiku struct{ client anthropic.Client }

func NewHaiku(key string) *Haiku {
	return &Haiku{client: anthropic.NewClient(option.WithAPIKey(key), option.WithRequestTimeout(60*time.Second))}
}

func (h *Haiku) Name() string  { return "haiku" }
func (h *Haiku) Model() string { return "claude-haiku-5-5" }

func haikuCost(u Usage) float64 {
	return (float64(u.In)*haikuIn + float64(u.CacheWrite)*haikuIn*1.25 +
		float64(u.CacheRead)*haikuIn*0.1 + float64(u.Out)*haikuOut) / 1e6
}

func (h *Haiku) Ask(sys string, hist []Turn, t *Tools, emit Emit) (*AskResponse, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 90*time.Second)
	defer cancel()

	readPage := anthropic.ToolParam{
		Name:        "read_page",
		Description: anthropic.String(toolReadDesc),
		InputSchema: anthropic.ToolInputSchemaParam{
			Properties: map[string]any{"path": map[string]any{"type": "string", "description": "Path from the index, e.g. services/tokenfuse.md"}},
			Required:   []string{"path"},
		},
	}
	contact := anthropic.ToolParam{
		Name:        "contact_team",
		Description: anthropic.String(toolContactDesc),
		InputSchema: anthropic.ToolInputSchemaParam{
			Properties: map[string]any{"summary": map[string]any{"type": "string", "description": "One line: what the visitor wants from the team"}},
			Required:   []string{"summary"},
		},
	}
	tools := []anthropic.ToolUnionParam{{OfTool: &readPage}, {OfTool: &contact}}

	msgs := make([]anthropic.MessageParam, 0, len(hist)+2*maxToolLoops)
	for _, turn := range hist {
		if turn.Role == "user" {
			msgs = append(msgs, anthropic.NewUserMessage(anthropic.NewTextBlock(turn.Text)))
		} else {
			msgs = append(msgs, anthropic.NewAssistantMessage(anthropic.NewTextBlock(turn.Text)))
		}
	}

	out := &AskResponse{}
	var kept []string
	shown := "" // what the browser shows now: kept text plus this turn's stream
	for i := 0; i < maxToolLoops; i++ {
		stream := h.client.Messages.NewStreaming(ctx, anthropic.MessageNewParams{
			Model:     h.Model(),
			MaxTokens: 8000,
			System: []anthropic.TextBlockParam{{
				Text:         sys,
				CacheControl: anthropic.NewCacheControlEphemeralParam(),
			}},
			OutputConfig: anthropic.OutputConfigParam{Effort: anthropic.OutputConfigEffortLow},
			Tools:        tools,
			Messages:     msgs,
		})
		resp := anthropic.Message{}
		reading, newBlock := false, false
		for stream.Next() {
			ev := stream.Current()
			if err := resp.Accumulate(ev); err != nil {
				return out, err
			}
			switch e := ev.AsAny().(type) {
			case anthropic.ContentBlockStartEvent:
				newBlock = e.ContentBlock.Type == "text"
				if e.ContentBlock.Type == "tool_use" && e.ContentBlock.Name == "read_page" && !reading {
					// What streamed this turn was narration before reading.
					reading = true
					shown = strings.Join(kept, "\n\n")
					emit(Event{T: "reset", Text: shown})
				}
			case anthropic.ContentBlockDeltaEvent:
				d, ok := e.Delta.AsAny().(anthropic.TextDelta)
				if !ok || reading || d.Text == "" {
					continue
				}
				if newBlock && shown != "" {
					emit(Event{T: "delta", Text: "\n\n"})
					shown += "\n\n"
				}
				newBlock = false
				emit(Event{T: "delta", Text: d.Text})
				shown += d.Text
			}
		}
		if err := stream.Err(); err != nil {
			return out, err
		}
		out.Calls++
		out.Usage.In += resp.Usage.InputTokens
		out.Usage.Out += resp.Usage.OutputTokens
		out.Usage.CacheWrite += resp.Usage.CacheCreationInputTokens
		out.Usage.CacheRead += resp.Usage.CacheReadInputTokens
		out.CostUSD = haikuCost(out.Usage)

		if resp.StopReason == anthropic.StopReasonRefusal {
			t.OfferContact = true
			out.Answer = "I can't help with that one here. If it is for the team, point at the address beside this chat to write to them."
			return out, nil
		}

		msgs = append(msgs, resp.ToParam())
		results := []anthropic.ContentBlockParamUnion{}
		var texts, called []string
		for _, block := range resp.Content {
			switch v := block.AsAny().(type) {
			case anthropic.TextBlock:
				if s := strings.TrimSpace(v.Text); s != "" {
					texts = append(texts, s)
				}
			case anthropic.ToolUseBlock:
				called = append(called, v.Name)
				res, isErr := t.Run(v.Name, []byte(v.JSON.Input.Raw()))
				results = append(results, anthropic.NewToolResultBlock(v.ID, res, isErr))
			}
		}
		if keepTurnText(called) {
			kept = append(kept, texts...)
		}
		if resp.StopReason != anthropic.StopReasonToolUse {
			break
		}
		msgs = append(msgs, anthropic.NewUserMessage(results...))
	}
	out.Answer = strings.Join(kept, "\n\n")
	if out.Answer == "" {
		return out, errors.New("the model returned no answer")
	}
	return out, nil
}

// ---------- A free model through OpenRouter's chat completions API ----------

// Free is one OpenRouter free route. With deny set, OpenRouter routes only to
// hosts that do not train on prompts; on 2026-11-01 that left two of the
// thirteen free tool-capable models answering.
type Free struct {
	id     string
	key    string
	models []string
	deny   bool
	http   *http.Client
	last   string
}

func NewFree(id, key string, models []string, deny bool) *Free {
	return &Free{id: id, key: key, models: models, deny: deny, http: &http.Client{Timeout: 90 * time.Second}, last: strings.TrimSpace(models[0])}
}

func (f *Free) Name() string  { return f.id }
func (f *Free) Model() string { return f.last }
func (f *Free) Private() bool { return f.deny }

type orMsg struct {
	Role       string       `json:"role"`
	Content    string       `json:"content"`
	ToolCalls  []orToolCall `json:"tool_calls,omitempty"`
	ToolCallID string       `json:"tool_call_id,omitempty"`
}

type orToolCall struct {
	ID       string `json:"id"`
	Type     string `json:"type"`
	Function struct {
		Name      string `json:"name"`
		Arguments string `json:"arguments"`
	} `json:"function"`
}

type orResp struct {
	Model   string `json:"model"`
	Choices []struct {
		FinishReason string `json:"finish_reason"`
		Message      orMsg  `json:"message"`
	} `json:"choices"`
	Usage struct {
		Prompt     int64 `json:"prompt_tokens"`
		Completion int64 `json:"completion_tokens"`
	} `json:"usage"`
	Error *struct {
		Code    any    `json:"code"`
		Message string `json:"message"`
	} `json:"error"`
}

var orTools = []map[string]any{
	{"type": "function", "function": map[string]any{
		"name": "read_page", "description": toolReadDesc,
		"parameters": map[string]any{"type": "object", "required": []string{"path"},
			"properties": map[string]any{"path": map[string]any{"type": "string"}}}}},
	{"type": "function", "function": map[string]any{
		"name": "contact_team", "description": toolContactDesc,
		"parameters": map[string]any{"type": "object", "required": []string{"summary"},
			"properties": map[string]any{"summary": map[string]any{"type": "string"}}}}},
}

// errRetryable marks a model-side failure where the next free model is worth trying.
var errRetryable = errors.New("retryable")

func (f *Free) call(model string, msgs []orMsg) (*orResp, error) {
	payload := map[string]any{
		"model": model, "messages": msgs, "tools": orTools, "tool_choice": "auto", "max_tokens": 4000,
	}
	if f.deny {
		// Route only to hosts that do not keep prompts for training.
		payload["provider"] = map[string]any{"data_collection": "deny"}
	}
	body, _ := json.Marshal(payload)
	req, _ := http.NewRequest(http.MethodPost, "https://openrouter.ai/api/v1/chat/completions", bytes.NewReader(body))
	req.Header.Set("Authorization", "Bearer "+f.key)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("HTTP-Referer", "http://localhost")
	req.Header.Set("X-Title", "IT-RAT desk test")
	resp, err := f.http.Do(req)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", errRetryable, err)
	}
	defer resp.Body.Close()
	raw, _ := io.ReadAll(io.LimitReader(resp.Body, 4<<20))
	var r orResp
	if err := json.Unmarshal(raw, &r); err != nil {
		return nil, fmt.Errorf("%w: http %d, unreadable body", errRetryable, resp.StatusCode)
	}
	if r.Error != nil || resp.StatusCode >= 400 || len(r.Choices) == 0 {
		msg := fmt.Sprintf("http %d", resp.StatusCode)
		if r.Error != nil {
			msg += ": " + r.Error.Message
		}
		if resp.StatusCode == 429 || resp.StatusCode >= 500 || resp.StatusCode == 404 || len(r.Choices) == 0 {
			return nil, fmt.Errorf("%w: %s", errRetryable, msg)
		}
		return nil, errors.New(msg)
	}
	return &r, nil
}

// Ask does not stream: the free models answer in one piece, and only the
// status line for a page being read reaches the browser before it.
func (f *Free) Ask(sys string, hist []Turn, t *Tools, _ Emit) (*AskResponse, error) {
	var errs []string
	for _, model := range f.models {
		model = strings.TrimSpace(model)
		out, err := f.ask(model, sys, hist, t)
		if err == nil {
			f.last = model
			return out, nil
		}
		// Name every model's failure: the last one alone hid a data-policy 404
		// behind an unrelated upstream 429.
		errs = append(errs, model+": "+strings.TrimPrefix(err.Error(), "retryable: "))
		if !errors.Is(err, errRetryable) {
			return out, errors.New(strings.Join(errs, "; "))
		}
		// Start the next model clean.
		t.PagesRead, t.OfferContact, t.ContactSummary = nil, false, ""
	}
	return &AskResponse{}, errors.New(strings.Join(errs, "; "))
}

func (f *Free) ask(model, sys string, hist []Turn, t *Tools) (*AskResponse, error) {
	msgs := []orMsg{{Role: "system", Content: sys}}
	for _, turn := range hist {
		msgs = append(msgs, orMsg{Role: turn.Role, Content: turn.Text})
	}
	out := &AskResponse{}
	var kept []string
	for i := 0; i < maxToolLoops; i++ {
		r, err := f.call(model, msgs)
		if err != nil {
			return out, err
		}
		out.Calls++
		out.Usage.In += r.Usage.Prompt
		out.Usage.Out += r.Usage.Completion
		m := r.Choices[0].Message
		text := strings.TrimSpace(m.Content)
		if len(m.ToolCalls) == 0 {
			if text != "" {
				kept = append(kept, text)
			}
			out.Answer = strings.Join(kept, "\n\n")
			if out.Answer == "" {
				return out, fmt.Errorf("%w: empty answer", errRetryable)
			}
			return out, nil
		}
		called := make([]string, 0, len(m.ToolCalls))
		for _, c := range m.ToolCalls {
			called = append(called, c.Function.Name)
		}
		if text != "" && keepTurnText(called) {
			kept = append(kept, text)
		}
		msgs = append(msgs, orMsg{Role: "assistant", Content: m.Content, ToolCalls: m.ToolCalls})
		for _, c := range m.ToolCalls {
			res, _ := t.Run(c.Function.Name, []byte(c.Function.Arguments))
			msgs = append(msgs, orMsg{Role: "tool", ToolCallID: c.ID, Content: res})
		}
	}
	return out, errors.New("no answer within the tool-call limit")
}
