# Estimation de l'énergie consommée par une requête LLM

## Vue d'ensemble

Xolo estime la consommation énergétique de chaque requête d'inférence LLM en mode **boîte noire** : on ne connaît pas la configuration réelle du datacenter, mais on peut raisonner à partir de paramètres physiques du modèle et d'hypothèses calibrées sur le type d'infrastructure.

Le résultat est toujours une **fourchette** [min, max] accompagnée d'une **valeur médiane** — non pas une valeur unique prétendument exacte. Cette honnêteté sur l'incertitude est un choix délibéré : l'énergie consommée par un LLM en production varie d'un facteur 5 à 10 selon les conditions réelles.

---

## Entrées de l'algorithme

| Paramètre           | Source                       | Description                                                            |
| ------------------- | ---------------------------- | ---------------------------------------------------------------------- |
| `activeParams`      | Configuration du modèle      | Nombre de paramètres actifs par token (ex : 7 milliards pour un 7B)    |
| `inputTokens`       | Requête API                  | Tokens envoyés au modèle (prompt)                                      |
| `outputTokens`      | Réponse API                  | Tokens générés par le modèle (complétion)                              |
| `tokPerSecLow/High` | Config ou heuristique        | Débit de génération en tokens/s (optionnel)                            |
| `cloudTier`         | Configuration du fournisseur | Catégorie d'infrastructure (hyperscaler, cloud majeur, petit provider) |

---

## Étape 1 — Estimation du débit tokens/s

Le débit de génération (tokens par seconde) conditionne la durée de la requête, et donc l'énergie consommée selon la méthode TDP (voir étape 3).

Si l'administrateur n'a pas renseigné de valeur, on utilise une **heuristique empirique** basée sur la taille du modèle :

```
tps_base = 200 / (bParams ^ 0.4)
```

où `bParams` est le nombre de milliards de paramètres actifs.

Quelques valeurs typiques :

| Modèle | Paramètres actifs | tps estimé (base) |
| ------ | ----------------- | ----------------- |
| 7B     | 7 Md              | ~100 tok/s        |
| 13B    | 13 Md             | ~73 tok/s         |
| 70B    | 70 Md             | ~37 tok/s         |
| 400B   | 400 Md            | ~15 tok/s         |

Pour construire la fourchette d'incertitude :

- **Scénario optimiste** : `tps = tps_base` (matériel récent, bon batching)
- **Scénario pessimiste** : `tps = tps_base × 0.5` (matériel saturé, moins optimisé)

> **Pourquoi l'exposant 0.4 ?** La loi de scaling empirique dit que le temps par token croît moins vite que linéairement avec la taille du modèle, car les architectures plus grandes bénéficient proportionnellement mieux du parallélisme GPU.

---

## Étape 2 — Calcul de la durée de la requête

Une requête LLM se décompose en deux phases aux propriétés très différentes :

### Phase prefill (traitement du prompt)

Le prefill traite **tous les tokens d'entrée en parallèle** sur le GPU. C'est une opération massivement parallèle, analogue à une multiplication de matrices dense. Mais le mécanisme d'attention a une complexité quadratique en O(n²) : plus le contexte est long, plus le gain relatif de la parallélisation diminue.

Le facteur de speedup est donc **dynamique** selon la longueur du prompt :

```
speedup_prefill = 20 / (1 + inputTokens / 10000)
durée_prefill   = inputTokens / (tps × speedup_prefill)
```

Valeurs typiques :

| Longueur prompt | speedup prefill | Durée prefill (tps=100) |
| --------------- | --------------- | ----------------------- |
| 100 tokens      | ~19.8×          | ~0.05 s                 |
| 1 000 tokens    | ~18.2×          | ~0.55 s                 |
| 5 000 tokens    | ~13.3×          | ~3.8 s                  |
| 20 000 tokens   | ~6.7×           | ~30 s                   |
| 50 000 tokens   | ~2.9×           | ~172 s                  |

### Phase decode (génération de la réponse)

Le decode génère les tokens **un par un**, séquentiellement. C'est la phase limitante en temps.

```
durée_decode = outputTokens / tps
```

### Durée totale

```
durée_totale = durée_prefill + durée_decode
```

**Exemple** — Modèle 7B, 1000 tokens d'entrée, 200 tokens de sortie, tps = 100 :

- speedup = 20 / (1 + 1000/10000) = **18.2×**
- Prefill : 1000 / (100 × 18.2) = **0.55 s**
- Decode : 200 / 100 = **2.0 s**
- Total : **2.55 s**

---

## Étape 3 — Méthode 1 : estimation par la puissance (TDP-based)

Cette méthode est inspirée de Ji & Jiang (2025) : au lieu de compter les opérations, on modélise directement la **puissance électrique** consommée par les GPU pendant la durée de la requête.

```
puissance_GPU = max(MinGPUWatts, WattsPerBParams × bParams)
E_TDP = puissance_GPU × durée × PUE × 1.20
```

| Terme             | Signification                                                                         |
| ----------------- | ------------------------------------------------------------------------------------- |
| `MinGPUWatts`     | Puissance GPU minimale par requête (W) — plancher lié au batching                     |
| `WattsPerBParams` | Puissance en watts par milliard de paramètres actifs                                  |
| `bParams`         | Nombre de milliards de paramètres actifs                                              |
| `durée`           | Durée totale de la requête (prefill + decode), en secondes                            |
| `PUE`             | _Power Usage Effectiveness_ — overhead du datacenter (refroidissement, alimentation…) |
| `× 1.20`          | +20% pour les overheads serveur (CPU, réseau, KV-cache en mémoire HBM)                |

**Pourquoi un plancher de puissance ?** Sans ce plancher, un modèle 7B chez un hyperscaler donnerait `0.10 × 7 = 0.7 W` — irréaliste. En pratique, chaque requête se voit attribuer une fraction d'un GPU en charge partielle, même avec un batching agressif. Le plancher reflète cette réalité.

**Pourquoi +20% et non +10% ?** Le KV-cache (mémoire HBM des activations d'attention) peut représenter 15–25 % de la puissance serveur totale, une composante non couverte par le PUE.

> **Note sur le double comptage** : `WattsPerBParams` capture implicitement une partie de la consommation mémoire GPU (HBM). Le +20% inclut une approximation du KV-cache non couverte par ce terme. Si `WattsPerBParams` est affiné pour inclure explicitement la mémoire HBM, cet overhead devra être revu à la baisse.

Le `WattsPerBParams` encode indirectement l'efficacité du hardware : un GPU H100 récent consomme moins par paramètre qu'un A100 ou V100.

Les paramètres par tier :

| Tier           | WattsPerBParams  | MinGPUWatts (low/high) | Overhead |
| -------------- | ---------------- | ---------------------- | -------- |
| Hyperscaler    | 0.10 – 0.50 W/Md | 2 / 20 W               | ×1.20    |
| Cloud majeur   | 0.30 – 1.00 W/Md | 8 / 50 W               | ×1.20    |
| Petit provider | 0.50 – 2.00 W/Md | 20 / 100 W             | ×1.20    |

---

## Étape 4 — Méthode 2 : estimation par les opérations flottantes (FLOP-based)

Cette méthode compte le nombre d'opérations mathématiques effectuées, puis convertit en énergie via l'efficacité énergétique du hardware.

### Calcul du nombre de FLOP

Pour un transformer, chaque token mobilise environ `2 × N` opérations flottantes, où `N` est le nombre de paramètres actifs (une multiplication + une addition par paramètre).

```
FLOP_par_token = 2 × activeParams

FLOP_prefill = FLOP_par_token × inputTokens
FLOP_decode  = FLOP_par_token × outputTokens
FLOP_total   = FLOP_prefill + FLOP_decode

attentionFactor = 1 + inputTokens / (inputTokens + 5000)
FLOP_total      = FLOP_total × attentionFactor
```

> **Note importante** : le prefill est compté au **coût plein**, identique au decode. La parallélisation GPU réduit le _temps_ mais pas le nombre d'opérations effectuées.

**Facteur d'attention pour les longs contextes** : le mécanisme d'attention a une complexité O(n²) en prefill — plus le prompt est long, plus les FLOP d'attention représentent une fraction significative du total. L'`attentionFactor` approxime ce surcoût sans nécessiter les paramètres internes des couches :

| Longueur prompt | attentionFactor | Surcoût FLOP |
| --------------- | --------------- | ------------ |
| 100 tokens      | 1.02            | +2%          |
| 1 000 tokens    | 1.17            | +17%         |
| 5 000 tokens    | 1.50            | +50%         |
| 20 000 tokens   | 1.80            | +80%         |
| 50 000 tokens   | 1.91            | +91%         |

_Source : Ji & Jiang (2025) ; approximation simplifiée sans paramètres de couche._

### Conversion en énergie

```
E_FLOP = FLOP_total / (efficacité × 10⁹) × PUE × 1.20
```

où `efficacité` est en GFLOP/J (gigaflops par joule).

Les plages d'efficacité retenues selon le tier :

| Tier                        | Efficacité min | Efficacité max |
| --------------------------- | -------------- | -------------- |
| Hyperscaler (Google, Meta…) | 700 GFLOP/J    | 1000 GFLOP/J   |
| Cloud majeur (AWS, OVH…)    | 350 GFLOP/J    | 700 GFLOP/J    |
| Petit provider              | 150 GFLOP/J    | 350 GFLOP/J    |

> La borne basse hyperscaler est relevée à 700 GFLOP/J (était 600) : les H100/H200 déployés en 2025 atteignent ≥ 700 GFLOP/J effectifs dans les conditions d'inférence typiques. La borne haute reste à 1000 GFLOP/J — 1200+ GFLOP/J supposerait du matériel Blackwell avec quantization FP8 et batching parfait, trop optimiste comme valeur générique.

---

## Étape 5 — Construction de l'enveloppe [min, max]

Les deux méthodes donnent chacune un résultat différent. La construction de l'enveloppe dépend d'un cas particulier : le **plancher TDP actif**.

### Cas 1 — Plancher TDP actif (petits modèles)

Quand `WattsPerBParams_low × bParams < MinGPUWatts_low`, le plancher `MinGPUWatts` est actif dans le scénario optimiste. Cela signifie que la méthode TDP encode l'**overhead d'infrastructure** (batching, matériel minimal), pas la physique du modèle. Dans ce cas, la méthode TDP n'est plus informative et on utilise **FLOP seul** :

```
Si WattsPerBParams_low × bParams < MinGPUWatts_low :
    E_low  = FLOP_low
    E_high = FLOP_high
    E_mid  = FLOP_mid  = √(FLOP_low × FLOP_high)
```

### Cas 2 — Hybride (modèles de taille normale)

Quand le plancher n'est pas actif, on prend l'**enveloppe conservative** des deux méthodes :

```
E_low  = min(TDP_low,  FLOP_low)
E_high = max(TDP_high, FLOP_high)

TDP_mid  = √(TDP_low  × TDP_high)   ← midpoint méthode TDP seule
FLOP_mid = √(FLOP_low × FLOP_high)  ← midpoint méthode FLOP seule

E_mid  = √(TDP_mid × FLOP_mid)      ← moyenne géométrique des midpoints par méthode
```

> **Pourquoi E_mid depuis les midpoints et non l'enveloppe ?** `√(E_low × E_high)` serait dominé par l'enveloppe la plus large (souvent TDP_high), donnant un mid trop pessimiste. Calculer le mid de chaque méthode séparément, puis en prendre la moyenne géométrique, produit une valeur centrale plus représentative du consensus entre les deux approches.

> **Signal de divergence** : si `TDP_mid / FLOP_mid > 5` ou `< 0.2`, les deux méthodes divergent significativement. Cela indique une forte incertitude, souvent due à des paramètres `tokPerSec` ou `activeParams` mal calibrés.

- `E_low` : le scénario le plus favorable retenu
- `E_high` : le scénario le plus défavorable retenu
- `E_mid` : valeur centrale, calculée comme la **moyenne géométrique** (et non arithmétique) car l'incertitude est multiplicative — on est plus sûr d'un facteur d'erreur que d'un écart absolu

> **Pourquoi combiner deux méthodes ?** La méthode TDP est plus précise quand le débit tokens/s est connu avec fiabilité ; la méthode FLOP est plus stable et indépendante du temps de calcul. Leur combinaison couvre mieux l'espace d'incertitude réel. Le fallback FLOP-only évite que le plancher d'infrastructure (constante par tier, indépendante du modèle) dilate artificiellement l'enveloppe pour les petits modèles.

---

## Les presets d'infrastructure (CloudTier)

Les paramètres `WattsPerBParams`, `MinGPUWatts` et `PUE` dépendent fortement du type de datacenter. Xolo propose trois niveaux :

### Hyperscaler (Google, Microsoft, Meta)

- PUE : 1.05 – 1.15 _(refroidissement très optimisé)_
- Utilisation GPU : 50 – 80%
- WattsPerBParams : 0.10 – 0.50 W/Md params
- MinGPUWatts : 2 – 20 W _(batching agressif ~100 req/GPU à léger ~10 req/GPU)_

### Cloud majeur (AWS, OVH, CoreWeave)

- PUE : 1.10 – 1.40
- Utilisation GPU : 30 – 60%
- WattsPerBParams : 0.30 – 1.00 W/Md params
- MinGPUWatts : 8 – 50 W

### Petit provider (startups, régional)

- PUE : 1.20 – 1.60 _(moins d'optimisation)_
- Utilisation GPU : 15 – 40%
- WattsPerBParams : 0.50 – 2.00 W/Md params
- MinGPUWatts : 20 – 100 W _(faible batching, souvent GPU dédié par requête)_

> **PUE (Power Usage Effectiveness)** : ratio entre l'énergie totale du datacenter et l'énergie consommée par les seuls serveurs. Un PUE de 1.0 serait parfait (impossible en pratique). Un PUE de 1.5 signifie que pour 1 W consommé par les GPU, 0.5 W supplémentaire part en refroidissement et distribution électrique.

---

## Étape 6 — Conversion en CO₂

L'énergie électrique n'a pas le même impact carbone selon où elle est produite. On calcule trois valeurs :

```
CO₂ (g) = énergie (Wh) × intensité_carbone (gCO₂/Wh)
```

| Scénario    | Intensité carbone | Contexte                  |
| ----------- | ----------------- | ------------------------- |
| **France**  | 0.027 gCO₂/Wh     | Mix nucléaire             |
| Suède       | 0.045 gCO₂/Wh     | Mix hydraulique/nucléaire |
| Moyenne UE  | 0.276 gCO₂/Wh     | Mix européen              |
| Monde       | 0.475 gCO₂/Wh     | _(valeur par défaut)_     |
| Gaz naturel | 0.490 gCO₂/Wh     | Centrale à gaz            |
| Charbon     | 0.960 gCO₂/Wh     | _(borne haute)_           |

L'affichage présente toujours les trois valeurs : intensité choisie, borne France (meilleur cas), borne charbon (pire cas).

---

## Exemple complet

**Requête** : modèle 7B, 1000 tokens en entrée, 200 tokens en sortie, hébergé chez un hyperscaler.

### Étape 1 — Débit

```
tps_base = 200 / 7^0.4 ≈ 100 tok/s
tps_optimiste = 100 tok/s
tps_pessimiste = 50 tok/s
```

### Étape 2 — Durées

```
speedup = 20 / (1 + 1000/10000) = 18.18×

Optimiste : prefill = 1000/(100×18.18) = 0.55 s  |  decode = 200/100 = 2.0 s  → total = 2.55 s
Pessimiste : prefill = 1000/(50×18.18) = 1.10 s  |  decode = 200/50  = 4.0 s  → total = 5.10 s
```

_(Note : tps pessimiste = 50 tok/s, donc speedup = 20/(1+1000/10000) = 18.18 dans les deux cas — le speedup ne dépend que de la longueur du prompt, pas du tps.)_

### Étape 3 — TDP (preset hyperscaler)

```
puissance_GPU_low  = max(2 W, 0.10×7) = max(2, 0.7) = 2 W   ← plancher actif
puissance_GPU_high = max(20 W, 0.50×7) = max(20, 3.5) = 20 W ← plancher actif

TDP_low  = 2 W × 2.55 s × 1.05 × 1.20 = 6.4 J   ≈ 1.8 mWh
TDP_high = 20 W × 5.10 s × 1.15 × 1.20 = 141 J  ≈ 39.2 mWh
TDP_mid  = √(6.4 × 141) ≈ 30.0 J ≈ 8.3 mWh
```

### Étape 4 — FLOP

```
FLOP_total = 2 × 7×10⁹ × (1000 + 200) = 1.68×10¹³ FLOP

attentionFactor = 1 + 1000/(1000+5000) ≈ 1.167
FLOP_total (corrigé) = 1.68×10¹³ × 1.167 ≈ 1.96×10¹³ FLOP

FLOP_low  = 1.96×10¹³ / (1000×10⁹) × 1.05 × 1.20 ≈ 24.7 J  ≈ 6.9 mWh
FLOP_high = 1.96×10¹³ / (700×10⁹)  × 1.15 × 1.20 ≈ 38.6 J  ≈ 10.7 mWh
FLOP_mid  = √(24.7 × 38.6) ≈ 30.9 J ≈ 8.6 mWh
```

### Étape 5 — Enveloppe et cohérence inter-méthodes

```
Détection du plancher : WattsPerBParams_low × 7 = 0.10 × 7 = 0.7 W < MinGPUWatts_low = 2 W
→ plancher TDP actif → fallback FLOP seul

E_low  = FLOP_low  = 24.7 J  ≈ 6.9 mWh
E_high = FLOP_high = 38.6 J  ≈ 10.7 mWh
E_mid  = FLOP_mid  = 30.9 J  ≈ 8.6 mWh

Spread : 10.7 / 6.9 ≈ ×1.5  (était ×22 avec l'enveloppe hybride brute)

Ratio TDP_mid / FLOP_mid = 30.0 / 30.9 ≈ 0.97  ← cohérence excellente (diagnostic conservé)
```

### Étape 6 — CO₂ (mix monde, 0.475 gCO₂/Wh)

```
CO₂_mid ≈ 8.5 mWh × 0.475 = 4.0 mg CO₂
           (fourchette : 0.049 mg en France → 8.2 mg au charbon)
```

---

## Limites et précautions d'interprétation

1. **Inférence en boîte noire** : on ne connaît pas le hardware réel, le niveau de batching, ni la quantization utilisée. L'incertitude réelle peut dépasser le facteur 10.

2. **Modèles MoE** : pour les architectures _Mixture of Experts_ (comme Mixtral ou GPT-4), `activeParams` doit représenter les paramètres _actifs par token_, pas le total. Un modèle de 56 Md de paramètres totaux avec 8 experts et top-2 routing a ~14 Md de paramètres actifs par token.

3. **Périmètre limité à l'inférence** : l'estimation ne couvre pas l'entraînement, le stockage des données, ni le réseau client.

4. **Débit tokens/s** : si le provider est configuré avec des valeurs `tokPerSecLow/High`, elles remplacent l'heuristique et améliorent significativement la précision de la méthode TDP.

5. **Intensité carbone** : la valeur par défaut (mix mondial, 0.475 gCO₂/Wh) est une approximation grossière. Pour un fournisseur hébergé en France, l'impact réel est 17× inférieur.

---

## Références

Les sources listées ci-dessous ont directement influencé les choix de modélisation. Les passages concernés sont indiqués pour chaque source.

---

**[1] Ji, Z. & Jiang, P. (2025)**
_Energy Consumption of LLM Inference._
Preprint. — **Source principale de la révision du modèle.**

Contributions clés utilisées dans cet algorithme :

- Section 2.2.2 (_online measurement approach_) : fondement de la méthode TDP-based — `E = N_GPU × TDP × heures` est plus fiable que le comptage théorique de FLOP.
- Section 2.2.3 et Table A1 : valeurs d'intensité carbone par pays/source (France 0.027, Chine 0.5366, moyenne monde 0.475 kgCO₂/kWh).
- Analyse prefill/decode : confirmation que le coût FLOP par token est identique entre prefill et decode — seule la _durée_ diffère (parallélisation). Corrige l'ancien facteur 0.3 utilisé dans la version initiale.
- Variabilité de puissance GPU : 1.252 à 2.735 kW selon le niveau de charge.
- PUE observés : 1.05 pour Falcon Computing (Falcon, Mixtral), jusqu'à 1.2+ pour des providers moins optimisés.

> ⚠️ Cette référence est citée dans les sources internes du projet. L'URL exacte de la publication préprint n'a pas été vérifiée indépendamment — à confirmer avant toute citation académique externe.

---

**[2] International Energy Agency (2024)**
_Electricity 2024 — Analysis and forecast to 2026._
IEA, Paris. https://www.iea.org/reports/electricity-2024

Contribution : estimation de l'ordre de grandeur de 0.001 à 0.01 kWh (1 à 10 mWh) par requête ChatGPT, utilisée comme borne de validation de plausibilité des résultats de l'estimateur.

---

**[3] Luccioni, A. S., Viguier, S., & Ligozat, A.-L. (2023)**
_Power Hungry Processing: Watts Driving the Cost of Generative AI Deployment?_
arXiv:2311.16863. https://arxiv.org/abs/2311.16863

Contribution : mesures empiriques de la consommation énergétique de modèles génératifs en production (via wattmètre), utilisées pour calibrer les ordres de grandeur et valider la cohérence des estimations.

---

**[4] SemiAnalysis (2024)**
_The Inference Cost of AI — GPU Economics and Efficiency._
https://www.semianalysis.com

Contribution : données empiriques sur l'efficacité énergétique des GPU en production (A100 : ~300–500 GFLOP/J effectifs ; H100 : ~500–1000 GFLOP/J effectifs), l'impact du batching sur la consommation par token, et les coûts d'inférence par tier d'infrastructure. Ces données ont servi à calibrer les plages `WattsPerBParams` et les efficacités GFLOP/J des presets.

---

**[5] Epoch AI (2024)**
_Estimating the Energy Cost of Frontier AI._
Epoch AI Research. https://epochai.org

Contribution : méthodologie de comptage des FLOP d'inférence (règle des `2 × N` FLOP par forward pass pour un transformer dense) et relation entre taille de modèle et consommation ; utilisée pour la méthode FLOP-based.

---

### Données de référence complémentaires

| Donnée                              | Valeur  | Source                                                 |
| ----------------------------------- | ------- | ------------------------------------------------------ |
| Consommation d'une recherche Google | ~0.3 Wh | Google Environmental Report 2023                       |
| Charge complète d'un smartphone     | ~14 Wh  | Estimations constructeurs (batterie ~4 000 mAh à 3.7V) |
| Consommation d'une ampoule LED 10W  | 10 W    | Physique de base                                       |
| 1 kgCO₂/kWh = 1 gCO₂/Wh             | —       | Équivalence dimensionnelle directe                     |
