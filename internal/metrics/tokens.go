package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

const (
	NameTotalTokens              = "total_tokens"
	NamePromptTokens             = "prompt_tokens"
	NameCompletionTokens         = "completion_tokens"
	NameChatCompletionRequests   = "chat_completion_requests"
	LabelOrg                     = "org"
)

var TotalTokens = promauto.NewCounterVec(
	prometheus.CounterOpts{
		Name:      NameTotalTokens,
		Help:      "Total tokens",
		Namespace: Namespace,
	},
	[]string{LabelOrg},
)

var CompletionTokens = promauto.NewCounterVec(
	prometheus.CounterOpts{
		Name:      NameCompletionTokens,
		Help:      "Completion tokens",
		Namespace: Namespace,
	},
	[]string{LabelOrg},
)

var PromptTokens = promauto.NewCounterVec(
	prometheus.CounterOpts{
		Name:      NamePromptTokens,
		Help:      "Prompt tokens",
		Namespace: Namespace,
	},
	[]string{LabelOrg},
)

var ChatCompletionRequests = promauto.NewCounterVec(
	prometheus.CounterOpts{
		Name:      NameChatCompletionRequests,
		Help:      "Number of chat completion requests",
		Namespace: Namespace,
	},
	[]string{LabelOrg},
)
