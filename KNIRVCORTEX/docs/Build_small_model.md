**Note:** This process has been updated to incorporate the **R-Zero (Reasoning and Refinement)** method. This enhances the quality of the generated data by adding a self-correction loop, creating more coherent and constraint-aligned stories.

### Step 1 – Define the Limits
**Objective:** Establish clear boundaries for the generated content to simplify the task for the large language model (LLM) and ensure consistency in the small dataset.

**Details:** Pick a hard "ceiling" on vocabulary and world knowledge. For TinyStories, this was limited to words a 3-4-year-old knows. This constraint is crucial for creating a truly "tiny" model that can still generate coherent narratives within its defined scope.

**Example:**
*   **Vocabulary:** Words from a pre-kindergarten reading list.
*   **Concepts:** Simple actions, common animals, basic emotions, everyday objects.
*   **Story Complexity:** Single plot line, few characters, clear beginning, middle, and end.

### Step 2 – Build the Toy-Box
**Objective:** Curate a controlled vocabulary of core linguistic components that adhere to the defined limits.

**Details:** Collect a set of simple nouns, verbs, and adjectives that fit the established vocabulary limits (e.g., ~1,500 words for TinyStories). Group them into categories for easy sampling.

**Example Snippet (Conceptual Python Lists):**
```python
nouns = ["cat", "dog", "ball", "house", "tree", "car", "friend", "toy", "flower", "sky"]
verbs = ["run", "play", "eat", "sleep", "jump", "see", "like", "go", "find", "give"]
adjectives = ["big", "small", "happy", "sad", "red", "blue", "fast", "slow", "new", "old"]
```

### Step 3 – List Story Spices
**Objective:** Define optional narrative elements that can be randomly injected into stories to add variety and complexity.

**Details:** Create a list of optional "features" or "tags" that a story might contain. These act as stylistic or thematic constraints for the LLM.

**Example Features:**
*   `dialogue`: Characters speak to each other.
*   `bad_ending`: The story concludes negatively.
*   `moral`: The story conveys a simple lesson.
*   `plot_twist`: An unexpected turn of events.
*   `foreshadowing`: Hints about future events.
*   `repetition`: Certain phrases or actions repeat.
*   `question`: A character asks a question.

### Step 4 – Spin the Random Prompt
**Objective:** Generate a unique set of constraints for each story by randomly selecting elements from the "toy-box" and "story spices."

**Details:** For every new story, randomly select:
*   One noun.
*   One verb.
*   One adjective.
*   A random subset of the "features" (story spices).

These selected elements form the core "creative constraints" for the LLM's generation task.

**Example Combination:**
*   **Noun:** "bird"
*   **Verb:** "sing"
*   **Adjective:** "loud"
*   **Features:** [`dialogue`, `happy_ending`]

### Step 5 – Multi-Stage Generation with R-Zero (Reasoning & Refinement)
**Objective:** Use a sequence of prompts to generate, critique, and refine a story, ensuring higher quality and better adherence to constraints.

#### Step 5a – Initial Generation Prompt
**Details:** Construct a zero-shot prompt for the LLM (e.g., GPT-4) that includes the initial constraints. This generates the first draft of the story.

**Example Initial Prompt:**
```
"Write a short story (3-5 short paragraphs) which only uses very simple words that a 3-year-old child would understand. The story should use the verb 'decorate', the noun 'thunder' and the adjective 'ancient'. The story should have dialogue and a bad ending."
```

#### Step 5b – Reasoning (Critique) Prompt
**Details:** After generating the initial story, use a second prompt to make the LLM act as a critic. It evaluates its own output against the original constraints and general quality metrics. This generates a "critique" that will guide the refinement step.

**Example Reasoning Prompt:**
```
"You are a story critic. Here is a story:
[Insert generated story from Step 5a here]

Here are the constraints it was supposed to follow:
- Verb: 'decorate'
- Noun: 'thunder'
- Adjective: 'ancient'
- Features: dialogue, bad ending
- Vocabulary: Simple words for a 3-year-old.

Does the story meet all constraints? Is it coherent and engaging? Provide a brief critique and suggest specific improvements to make it better."
```

#### Step 5c – Refinement (Rewrite) Prompt
**Details:** Use a third prompt that provides the original story and the LLM's own critique, asking it to produce a final, improved version. This is the version that will be added to the dataset.

**Example Refinement Prompt:**
```
"Based on the following critique: [Insert critique from Step 5b here], please rewrite the original story to be better. Ensure you still follow all the original constraints.

Original Story: [Insert generated story from Step 5a here]"
```

### Step 6 – Generate, Temperature-Shuffle, and Filter
**Objective:** Produce multiple story candidates from the R-Zero process and select only those that strictly adhere to the prompt's constraints.

**Details:**
*   **Generation:** Use the LLM to generate several completions for each prompt.
*   **Temperature:** Experiment with different temperature settings:
    *   **High temperature (e.g., ≈1.0):** Encourages diversity in generated stories.
    *   **Low temperature (e.g., ≈0.0):** Produces more deterministic and consistent outputs, useful for baseline comparisons.
*   **Filtering:** Implement automated checks to ensure the **final, refined stories** from Step 5c obey all constraints (e.g., presence of required words, adherence to vocabulary, paragraph count, inclusion of features). Discard or re-roll stories that fail these checks.

### Step 7 – De-duplicate on the Fly
**Objective:** Prevent redundancy in the dataset by discarding stories that are too similar to already collected ones.

**Details:** As stories are collected, perform a quick similarity check against the existing dataset. Common metrics include:
*   **N-gram overlap:** Comparing sequences of N words.
*   **ROUGE-L (Recall-Oriented Understudy for Gisting Evaluation - Longest Common Subsequence):** Measures the longest common subsequence between two texts, useful for capturing structural similarity.

If a new story is too close to one already in the corpus (e.g., above a certain similarity threshold), it is discarded, and a new story is generated.

### Step 8 – Auto-annotate for the Instruct Variant (TinyStories-Instruct)
**Objective:** Prepare a separate "instruct-tuning" dataset by automatically extracting key information from each generated story, including the reasoning critique. This teaches the small model not just *what* to write, but *how to think* about writing.

**Details:** For each finalized story, use an LLM (or rule-based system) to automatically extract:
*   A concise one-line summary of the story.
*   One random sentence, typically from the middle of the story, to serve as a context snippet.
*   **A one-paragraph critique (from the R-Zero reasoning step in 5b).** This is a crucial addition that captures the reasoning process.
*   A list of the specific required words (noun, verb, adjective) that were actually used in the story.
*   The list of "features" (story spices) that were successfully incorporated.

These extracted fields become the "instructions" that precede the story in the instruct-tuning dataset, enabling the training of models that can follow specific directives and understand the principles of good storytelling.

### Step 9 – Package & Sanity-Check
**Objective:** Organize the generated dataset into a standard format and perform a final quality assurance review before release.

**Details:**
*   **Packaging:** Format the entire dataset (both the base and instruct variants) into a standard machine learning dataset format, such as Hugging Face's `datasets` library format, typically split into `train` and `validation` sets.
*   **Sanity Check:** Conduct a final manual review of a random sample of stories (e.g., 50-100 stories). This step is crucial to confirm that the automated process has maintained the desired vocabulary, grammar, and creative constraints, and to catch any subtle LLM "hallucinations" or deviations.

This iterative process allows for the systematic generation of large, high-quality, and highly controlled datasets for training small language models.


That’s it—rinse and repeat until you have the desired corpus size.

References: 
R-Zero Method: https://arxiv.org/pdf/2411.15124
Tiny Stories: https://arxiv.org/pdf/2305.07759
