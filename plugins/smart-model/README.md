# smart-model

**smart-model** is a Xolo plugin that automatically selects the most appropriate LLM for each request using fuzzy logic. It routes requests sent to a virtual model (e.g. `my-org/auto`) to a real model chosen from the organisation's provider pool, balancing response quality, energy consumption, and budget pressure.

---

## How it works

The selection pipeline runs in four stages for every incoming request.

### 1. Input scoring

Six variables are computed from the request:

| Variable | Description |
|---|---|
| `complexity` | Composite score [0–1] derived from text length, Shannon entropy, lexical richness, structural depth, readability (Flesch-Kincaid), and constraint density. Short, trivial messages score near 0; long, structured, constraint-heavy prompts score near 1. |
| `category` | Top request category predicted by a Naive Bayes classifier trained on labelled prompts (e.g. `code`, `conversation`, `instruction`, `factual`, `summarization`, `translation`, `rewriting`). Used for preferred-category matching, not as a fuzzy input. |
| `energy_cost` | Estimated energy consumption [0–1] of a reference 70B model for the predicted token count, normalised through a sigmoid centred at 0.001 kWh. |
| `budget_pressure` | Maximum ratio of quota consumed across daily, monthly, and yearly periods [0–1]. Returns 0 when no quota is configured. |
| `energy_sensitivity` | Global frugality weight [0–1] set in the plugin configuration. 0 = ignore energy, 1 = maximise frugality. |
| `has_vision` | Boolean — `true` when the request contains image inputs. Used as a hard feasibility constraint, not a fuzzy input. |
| `has_reasoning` | Boolean — `true` when the request body includes an extended reasoning/thinking parameter (`thinking`, `enable_thinking`, `reasoning_effort`). Used as a hard feasibility constraint. |

#### Complexity sub-metrics

| Metric | Weight | Description |
|---|---|---|
| Length | 10% | Token count; sigmoid centred at 500 tokens. |
| Shannon entropy | 15% | Character-bigram entropy; significant only for texts ≥ 50 characters. |
| Lexical richness | 15% | Type-token ratio dampened for short texts (< 20 tokens). |
| Compression ratio | 10% | gzip ratio as a Kolmogorov-complexity proxy; significant only for texts ≥ 50 bytes. |
| Structural score | 20% | Combination of nesting depth, question count, sentence count, and Markdown elements. |
| Readability | 10% | Flesch-Kincaid grade level, normalised to [0–1] with grade 16 as ceiling. |
| Constraint density | 20% | Count of explicit instructions, format constraints, enumerations, role assignments, and language constraints matched by regex patterns. |

---

### 2. Fuzzy inference

A Mamdani-style fuzzy engine infers a **desired power level** [0–1] from the scored inputs. The default ruleset is shown below.

#### Variable definitions

```
complexity      : very_low  LOW  MEDIUM  HIGH  very_high
budget_pressure : low  medium  high
energy_sensitivity : low  medium  high
energy_cost     : low  medium  high
power_level     : very_low  LOW  MEDIUM  HIGH  very_high
```

All terms use LINEAR (ramp) or TRIANGULAR membership functions on the [0–1] domain.

#### Default rules

**Complexity baseline**
```
IF complexity IS very_low  THEN power_level IS very_low
IF complexity IS low        THEN power_level IS low
IF complexity IS medium     THEN power_level IS medium
IF complexity IS high       THEN power_level IS high
IF complexity IS very_high  THEN power_level IS very_high
```

**Budget pressure overrides** (force lighter model when budget is tight)
```
IF budget_pressure IS high                                THEN power_level IS very_low
IF budget_pressure IS medium AND complexity IS low        THEN power_level IS low
IF budget_pressure IS medium AND complexity IS very_low   THEN power_level IS very_low
```

**Energy sensitivity overrides** (down-scale when frugality is prioritised)
```
IF energy_sensitivity IS high   AND complexity IS very_low  THEN power_level IS very_low
IF energy_sensitivity IS high   AND complexity IS low        THEN power_level IS very_low
IF energy_sensitivity IS high   AND complexity IS medium     THEN power_level IS low
IF energy_sensitivity IS medium AND complexity IS very_low   THEN power_level IS very_low
IF energy_sensitivity IS medium AND complexity IS low        THEN power_level IS low
```

**Energy cost overrides** (down-scale for cheap but high-energy requests)
```
IF energy_cost IS high   AND complexity IS very_low  THEN power_level IS very_low
IF energy_cost IS high   AND complexity IS low        THEN power_level IS very_low
IF energy_cost IS high   AND complexity IS medium     THEN power_level IS low
IF energy_cost IS medium AND complexity IS very_low   THEN power_level IS very_low
```

**Permissive rules** (allow stronger model when budget and energy allow)
```
IF complexity IS high      AND budget_pressure IS low AND energy_sensitivity IS low  THEN power_level IS high
IF complexity IS very_high AND budget_pressure IS low AND energy_sensitivity IS low  THEN power_level IS very_high
IF complexity IS very_high AND energy_sensitivity IS medium                           THEN power_level IS high
```

The output is defuzzified with the **centroid** method (200 integration steps), yielding a single float in [0–1].

---

### 3. Model selection

All enabled real models (non-virtual, context-feasible) are scored as candidates:

```
score = 1 - |model_power_level - desired_power_level|   // proximity to target
      + energy_sensitivity × (1 - model_power_level) × 0.3  // frugality bonus
      + 0.4 if model categories ∩ request category ≠ ∅  // preferred-category bonus
```

**Hard feasibility filters applied before scoring:**

- **Context length**: skipped if `estimated_input_tokens + estimated_output_tokens > model.context_length` (when context length is known).
- **Vision**: skipped if the request contains images and the model does not declare `supports_vision`.
- **Reasoning**: skipped if the request enables extended reasoning and the model does not declare `supports_reasoning`.

The model with the highest score is selected. If no candidate passes the feasibility filters, the plugin passes through without resolving the model.

#### Power level

Each model is assigned a power level in [0–1]:

- **Override** — set explicitly per model in the plugin configuration.
- **Auto** — derived from active parameter count using `log₂(params_B) / log₂(500)`.

| Size | Auto power level |
|---|---|
| 7B | 0.30 |
| 13B | 0.40 |
| 70B | 0.67 |
| 175B | 0.80 |
| 400B | 0.95 |

---

### 4. Trigger guard

The plugin only activates when the requested model is both:
1. Listed as a **virtual model** for the organisation, and
2. Present in the plugin's **trigger model list** (configured in the *Déclenchement* tab).

If the trigger list is empty, the plugin is **inactive** for all requests.

---

## Configuration

| Field | Default | Description |
|---|---|---|
| `energy_sensitivity` | `0.6` | Global frugality weight [0–1]. |
| `rules` | *(default DSL)* | Full fuzzy ruleset; editable in the *Règles* tab. |
| `trigger_models` | `[]` | Virtual model names that activate the plugin. Empty = inactive. |
| `log_enabled` | `false` | Write a JSONL decision log. |
| `log_path` | `smart-model.jsonl` | Path of the decision log file. |

Per-model overrides (set in the *Modèles* tab):

| Field | Description |
|---|---|
| `enabled` | Whether this model is a candidate for automatic selection (default `true`). |
| `power_level_override` | Fixed power level [0–1]; overrides the auto-computed value. |
| `categories` | Preferred request categories for this model (e.g. `code`, `conversation`). |

---

## DSL reference

Rules are written in a simple domain-specific language parsed at runtime.

```
DEFINE variable_name (
    TERM term_name LINEAR(x1, x2),
    TERM term_name TRIANGULAR(left, center, right),
    ...
);

IF var IS term [AND var IS term ...] THEN out_var IS out_term;
```

**Membership functions:**

- `LINEAR(x1, x2)` — ramp from 0 at `x1` to 1 at `x2` (ascending if `x1 < x2`, descending if `x1 > x2`).
- `TRIANGULAR(left, center, right)` — triangle peak of 1 at `center`, 0 at `left` and `right`.

Changes to the ruleset take effect immediately for the next request; no restart is required.
