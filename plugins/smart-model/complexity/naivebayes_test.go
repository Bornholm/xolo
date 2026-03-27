package complexity

import (
	"fmt"
	"os"
	"testing"
)

var defaultCategories = []TrainingExample{
	// --- code ---
	{Label: "code", Text: "Write a function in Python that sorts a list"},
	{Label: "code", Text: "Fix the bug in this JavaScript code"},
	{Label: "code", Text: "Écris un programme Go qui lit un fichier CSV"},
	{Label: "code", Text: "Debug this SQL query it returns wrong results"},
	{Label: "code", Text: "Refactor this class to use dependency injection"},
	{Label: "code", Text: "Create a REST API endpoint in Node.js"},
	{Label: "code", Text: "Implémente un algorithme de tri rapide en Rust"},
	{Label: "code", Text: "How do I use goroutines and channels in Go"},
	{Label: "code", Text: "Write unit tests for this function"},
	{Label: "code", Text: "Convert this Python script to TypeScript"},
	{Label: "code", Text: "Explain this regex pattern and optimize it"},
	{Label: "code", Text: "Crée une classe Java avec getters et setters"},

	// --- translation ---
	{Label: "translation", Text: "Translate this text from English to French"},
	{Label: "translation", Text: "Traduis ce paragraphe en anglais"},
	{Label: "translation", Text: "How do you say 'good morning' in Japanese"},
	{Label: "translation", Text: "Translate the following document to Spanish"},
	{Label: "translation", Text: "Convert this French email to English"},
	{Label: "translation", Text: "Traduis en allemand le texte suivant"},
	{Label: "translation", Text: "What is the Italian translation of this phrase"},
	{Label: "translation", Text: "Translate these product descriptions to Portuguese"},

	// --- summarization ---
	{Label: "summarization", Text: "Summarize this article in 3 sentences"},
	{Label: "summarization", Text: "Résume ce document en quelques points clés"},
	{Label: "summarization", Text: "Give me the key takeaways from this report"},
	{Label: "summarization", Text: "TLDR of this long email thread"},
	{Label: "summarization", Text: "Condense this 10 page paper into a paragraph"},
	{Label: "summarization", Text: "Fais un résumé exécutif de ce rapport"},
	{Label: "summarization", Text: "What are the main points of this text"},
	{Label: "summarization", Text: "Summarize the meeting notes briefly"},

	// --- creative ---
	{Label: "creative", Text: "Write a short story about a robot learning to love"},
	{Label: "creative", Text: "Compose a poem about the ocean at sunset"},
	{Label: "creative", Text: "Écris une histoire courte sur un voyage dans le temps"},
	{Label: "creative", Text: "Generate creative names for a coffee shop"},
	{Label: "creative", Text: "Write a dialogue between a cat and a dog"},
	{Label: "creative", Text: "Invente un conte pour enfants avec une princesse"},
	{Label: "creative", Text: "Create a fictional world with magic and dragons"},
	{Label: "creative", Text: "Write song lyrics about heartbreak and hope"},
	{Label: "creative", Text: "Imagine a conversation between Einstein and Newton"},
	{Label: "creative", Text: "Rédige un scénario de court-métrage"},

	// --- factual ---
	{Label: "factual", Text: "What is the capital of Australia"},
	{Label: "factual", Text: "Quelle est la population de la France"},
	{Label: "factual", Text: "When was the Eiffel Tower built"},
	{Label: "factual", Text: "Who invented the telephone"},
	{Label: "factual", Text: "What is the speed of light in vacuum"},
	{Label: "factual", Text: "How many planets are in the solar system"},
	{Label: "factual", Text: "Quel est le plus long fleuve du monde"},
	{Label: "factual", Text: "What year did World War 2 end"},
	{Label: "factual", Text: "Who is the CEO of Tesla"},
	{Label: "factual", Text: "Define photosynthesis"},

	// --- math ---
	{Label: "math", Text: "Solve this equation: 2x + 5 = 17"},
	{Label: "math", Text: "Calculate the integral of x squared"},
	{Label: "math", Text: "Résous ce système d'équations linéaires"},
	{Label: "math", Text: "What is the derivative of sin(x) * cos(x)"},
	{Label: "math", Text: "Prove that the square root of 2 is irrational"},
	{Label: "math", Text: "Calcule la probabilité de tirer deux as"},
	{Label: "math", Text: "Find the eigenvalues of this matrix"},
	{Label: "math", Text: "Simplify this algebraic expression"},
	{Label: "math", Text: "How many permutations of 5 elements exist"},
	{Label: "math", Text: "Explain the Bayes theorem with an example"},

	// --- analysis ---
	{Label: "analysis", Text: "Compare the pros and cons of React vs Vue"},
	{Label: "analysis", Text: "Analyse les avantages et inconvénients du télétravail"},
	{Label: "analysis", Text: "What are the strengths and weaknesses of this strategy"},
	{Label: "analysis", Text: "Evaluate the impact of AI on employment"},
	{Label: "analysis", Text: "Compare microservices vs monolithic architecture"},
	{Label: "analysis", Text: "Fais une analyse SWOT de cette entreprise"},
	{Label: "analysis", Text: "Assess the risks of this investment portfolio"},
	{Label: "analysis", Text: "Review and critique this business plan"},
	{Label: "analysis", Text: "Compare TCP vs UDP for real-time gaming"},
	{Label: "analysis", Text: "Évalue les différentes approches de déploiement"},

	// --- conversation ---
	{Label: "conversation", Text: "Hello how are you today"},
	{Label: "conversation", Text: "Bonjour ça va"},
	{Label: "conversation", Text: "Thanks for your help"},
	{Label: "conversation", Text: "Tell me a joke"},
	{Label: "conversation", Text: "What do you think about the weather"},
	{Label: "conversation", Text: "Merci beaucoup c'est gentil"},
	{Label: "conversation", Text: "Can you help me with something"},
	{Label: "conversation", Text: "Who are you and what can you do"},
	{Label: "conversation", Text: "Salut je m'ennuie raconte moi quelque chose"},
	{Label: "conversation", Text: "Good morning have a nice day"},
	{Label: "conversation", Text: "Hi there how's it going"},
	{Label: "conversation", Text: "Hey how are things"},
	{Label: "conversation", Text: "How's your day been"},
	{Label: "conversation", Text: "Nice to meet you"},
	{Label: "conversation", Text: "What's up"},
	{Label: "conversation", Text: "How's everything going"},

	// --- instruction ---
	{Label: "instruction", Text: "How do I install Docker on Ubuntu"},
	{Label: "instruction", Text: "Comment configurer un serveur Nginx"},
	{Label: "instruction", Text: "Step by step guide to set up a React project"},
	{Label: "instruction", Text: "Explain how to create a virtual environment in Python"},
	{Label: "instruction", Text: "Walk me through setting up SSH keys"},
	{Label: "instruction", Text: "Comment déployer une application sur Kubernetes"},
	{Label: "instruction", Text: "How to configure a GitHub Actions CI pipeline"},
	{Label: "instruction", Text: "Guide me through installing PostgreSQL on macOS"},

	// --- rewriting ---
	{Label: "rewriting", Text: "Rewrite this email to sound more professional"},
	{Label: "rewriting", Text: "Reformule ce texte de manière plus concise"},
	{Label: "rewriting", Text: "Make this paragraph more engaging"},
	{Label: "rewriting", Text: "Paraphrase this sentence in simpler words"},
	{Label: "rewriting", Text: "Rephrase this to be more formal"},
	{Label: "rewriting", Text: "Améliore le style de ce paragraphe"},
	{Label: "rewriting", Text: "Simplify this technical explanation for beginners"},
	{Label: "rewriting", Text: "Rewrite this cover letter with stronger action verbs"},
}

func newTrainedClassifier() *NaiveBayes {
	nb := NewNaiveBayes(1.0, 2)
	nb.Train(defaultCategories)
	return nb
}

func TestNaiveBayesPredict(t *testing.T) {
	nb := newTrainedClassifier()

	cases := []struct {
		text      string
		wantClass string
	}{
		{"Write a Python function to reverse a linked list", "code"},
		{"Traduis ce texte en anglais s'il te plaît", "translation"},
		{"Summarize this document in bullet points", "summarization"},
		{"Write a poem about autumn leaves", "creative"},
		{"What is the capital of Japan", "factual"},
		{"Solve for x: 3x - 7 = 14", "math"},
		{"Compare the advantages of SQL vs NoSQL databases", "analysis"},
		{"Hi there how's it going", "conversation"},
		{"How do I install Node.js on Windows", "instruction"},
		{"Rewrite this paragraph to sound more formal", "rewriting"},
	}

	for _, tc := range cases {
		t.Run(tc.wantClass, func(t *testing.T) {
			pred := nb.Predict(tc.text)
			if pred.Class != tc.wantClass {
				// Show top 3 for debugging
				top3 := nb.PredictTopK(tc.text, 3)
				t.Errorf("text=%q\n  want=%s got=%s (%.1f%%)\n  top3=%v",
					tc.text, tc.wantClass, pred.Class, pred.Prob*100, top3)
			} else {
				t.Logf("%-15s → %s (%.1f%%)", tc.wantClass, pred.Class, pred.Prob*100)
			}
		})
	}
}

func TestNaiveBayesPredictAll(t *testing.T) {
	nb := newTrainedClassifier()
	preds := nb.PredictAll("Écris un algorithme de recherche binaire en Go")

	// Should return all classes
	if len(preds) != len(nb.Classes) {
		t.Errorf("expected %d predictions, got %d", len(nb.Classes), len(preds))
	}

	// Probabilities should sum to ~1
	sum := 0.0
	for _, p := range preds {
		sum += p.Prob
	}
	if sum < 0.99 || sum > 1.01 {
		t.Errorf("probabilities sum to %.4f, expected ~1.0", sum)
	}

	t.Logf("Top prediction: %s (%.1f%%)", preds[0].Class, preds[0].Prob*100)
}

func TestNaiveBayesSaveLoad(t *testing.T) {
	nb := newTrainedClassifier()

	tmpFile := "/tmp/test_nb_model.json"
	defer os.Remove(tmpFile)

	if err := nb.SaveModel(tmpFile); err != nil {
		t.Fatalf("SaveModel: %v", err)
	}

	loaded, err := LoadModelFromFile(tmpFile)
	if err != nil {
		t.Fatalf("LoadModel: %v", err)
	}

	// Both should give same prediction
	text := "Write a sorting algorithm in Rust"
	orig := nb.Predict(text)
	restored := loaded.Predict(text)

	if orig.Class != restored.Class {
		t.Errorf("predictions differ: orig=%s loaded=%s", orig.Class, restored.Class)
	}
	t.Logf("Serialization OK: both predict %s (%.1f%%)", orig.Class, orig.Prob*100)
}

func TestNaiveBayesEvaluate(t *testing.T) {
	nb := newTrainedClassifier()

	// Evaluate on held-out examples (different from training data)
	testSet := []TrainingExample{
		{Label: "code", Text: "Implement a binary tree in C++"},
		{Label: "translation", Text: "Translate this to German please"},
		{Label: "summarization", Text: "Give me a brief summary of the article"},
		{Label: "creative", Text: "Tell me a fantasy story about elves"},
		{Label: "factual", Text: "How tall is Mount Everest"},
		{Label: "math", Text: "What is the factorial of 10"},
		{Label: "analysis", Text: "Evaluate the trade-offs between these two options"},
		{Label: "conversation", Text: "Hey what's up"},
		{Label: "instruction", Text: "How to set up a Python virtual environment"},
		{Label: "rewriting", Text: "Make this email sound friendlier"},
	}

	acc, errors := nb.Evaluate(testSet)
	t.Logf("Accuracy: %.0f%% (%d/%d)", acc*100, int(acc*float64(len(testSet))), len(testSet))
	for _, e := range errors {
		t.Logf("  MISS: %s", e)
	}
	if acc < 0.7 {
		t.Errorf("accuracy %.0f%% below 70%% threshold", acc*100)
	}
}

func TestNaiveBayesSummary(t *testing.T) {
	nb := newTrainedClassifier()
	summary := nb.Summary()
	t.Log(summary)
	if len(summary) == 0 {
		t.Error("empty summary")
	}
}

func TestFullAnalysis(t *testing.T) {
	nb := newTrainedClassifier()
	w := DefaultWeights()

	text := "Compare CQRS and event sourcing patterns. Give pros/cons in a table format, with code examples in Go. Respond in French."

	result := AnalyzeFull(text, w, nb)

	t.Logf("Complexity: %.3f (%s)", result.Complexity.Composite, result.Complexity.Level)
	t.Logf("Category:   %s (%.1f%%)", result.Category.Class, result.Category.Prob*100)
	t.Logf("Top 3:")
	for _, p := range result.TopCategories {
		t.Logf("  %-15s %.1f%%", p.Class, p.Prob*100)
	}
}

// Benchmark to show inference speed
func BenchmarkNaiveBayesPredict(b *testing.B) {
	nb := newTrainedClassifier()
	text := "Write a REST API in Go with authentication and rate limiting"

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		nb.Predict(text)
	}
	pred := nb.Predict(text)
	fmt.Printf("Benchmark: %s (%.1f%%)\n", pred.Class, pred.Prob*100)
}

func BenchmarkFullAnalysis(b *testing.B) {
	nb := newTrainedClassifier()
	w := DefaultWeights()
	text := "Compare microservices vs monolith. Give code examples in Go and a table of trade-offs. Respond in French, max 500 words."

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		AnalyzeFull(text, w, nb)
	}
}
