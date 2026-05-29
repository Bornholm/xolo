package main

// DefaultRulesDSL is the built-in fuzzy logic ruleset, ported from smart-model.
const DefaultRulesDSL = `
DEFINE complexity (
    TERM very_low  LINEAR(0.25, 0.0),
    TERM low       TRIANGULAR(0.0, 0.25, 0.5),
    TERM medium    TRIANGULAR(0.25, 0.5, 0.75),
    TERM high      TRIANGULAR(0.5, 0.75, 1.0),
    TERM very_high LINEAR(0.75, 1.0)
);

DEFINE budget_pressure (
    TERM low    LINEAR(0.4, 0.0),
    TERM medium TRIANGULAR(0.0, 0.5, 1.0),
    TERM high   LINEAR(0.6, 1.0)
);

DEFINE energy_sensitivity (
    TERM low    LINEAR(0.4, 0.0),
    TERM medium TRIANGULAR(0.0, 0.5, 1.0),
    TERM high   LINEAR(0.6, 1.0)
);

DEFINE energy_cost (
    TERM low    LINEAR(0.4, 0.0),
    TERM medium TRIANGULAR(0.0, 0.5, 1.0),
    TERM high   LINEAR(0.6, 1.0)
);

DEFINE power_level (
    TERM very_low  LINEAR(0.25, 0.0),
    TERM low       TRIANGULAR(0.0, 0.25, 0.5),
    TERM medium    TRIANGULAR(0.25, 0.5, 0.75),
    TERM high      TRIANGULAR(0.5, 0.75, 1.0),
    TERM very_high LINEAR(0.75, 1.0)
);

IF complexity IS very_low THEN power_level IS very_low;
IF complexity IS low THEN power_level IS low;
IF complexity IS medium THEN power_level IS medium;
IF complexity IS high THEN power_level IS high;
IF complexity IS very_high THEN power_level IS very_high;

IF budget_pressure IS high THEN power_level IS very_low;
IF budget_pressure IS medium AND complexity IS low THEN power_level IS low;
IF budget_pressure IS medium AND complexity IS very_low THEN power_level IS very_low;

IF energy_sensitivity IS high AND complexity IS very_low THEN power_level IS very_low;
IF energy_sensitivity IS high AND complexity IS low THEN power_level IS very_low;
IF energy_sensitivity IS high AND complexity IS medium THEN power_level IS low;
IF energy_sensitivity IS medium AND complexity IS very_low THEN power_level IS very_low;
IF energy_sensitivity IS medium AND complexity IS low THEN power_level IS low;

IF energy_cost IS high AND complexity IS very_low THEN power_level IS very_low;
IF energy_cost IS high AND complexity IS low THEN power_level IS very_low;
IF energy_cost IS high AND complexity IS medium THEN power_level IS low;
IF energy_cost IS medium AND complexity IS very_low THEN power_level IS very_low;

IF complexity IS high AND budget_pressure IS low AND energy_sensitivity IS low THEN power_level IS high;
IF complexity IS very_high AND budget_pressure IS low AND energy_sensitivity IS low THEN power_level IS very_high;
IF complexity IS very_high AND energy_sensitivity IS medium THEN power_level IS high;
`
