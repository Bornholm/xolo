## Analyse de la méthode d'estimation

La méthode est globalement **bien conçue et honnête** dans son approche. L'utilisation d'une fourchette plutôt qu'une valeur unique, la combinaison de deux méthodes complémentaires, et la transparence sur les limites sont des choix judicieux. Cependant, il y a plusieurs points qui méritent discussion.

---

### Ce qui est bien fait

1. **Double méthode (TDP + FLOP)** — Cohérent avec l'article qui distingue l'approche "online measurement" et "offline estimation". Couvrir les deux angles réduit le risque de biais systématique.

2. **PUE paramétré par tier** — L'article confirme que le PUE est un facteur critique et varie significativement (1.05 à 1.6+). Les plages choisies sont réalistes.

3. **Intensité carbone contextualisée** — L'article insiste lourdement sur le fait que l'intensité carbone domine souvent l'impact plus que la consommation brute (cf. BLOOM vs GPT-3). Présenter France/Monde/Charbon est pertinent.

4. **Moyenne géométrique** — Bon choix pour une incertitude multiplicative.

---

### Points problématiques

#### 1. L'enveloppe min/max est trop large et potentiellement trompeuse

Dans l'exemple, `E_low = 0.56 mWh` et `E_high = 9.8 mWh` — un facteur **17×**. Quand la méthode TDP donne 0.56 mWh en borne basse et la méthode FLOP donne 4.5 mWh en borne basse, ça signifie que **les deux méthodes ne sont pas cohérentes entre elles**. Prendre le min des mins et le max des max masque ce désaccord plutôt que de le résoudre.

**Suggestion** : Afficher aussi les intervalles de chaque méthode séparément, ou signaler quand les intervalles ne se chevauchent pas (ce qui indiquerait un problème de calibration).

#### 2. La méthode TDP sous-estime pour les petits modèles

Le paramètre `WattsPerBParams` de 0.10 W/Md pour un hyperscaler donne pour un modèle 7B : `0.10 × 7 = 0.7 W` de puissance GPU. C'est **irréaliste** — un GPU H100 au repos consomme déjà ~100W, et en inférence active ~300-700W. Le problème est que `WattsPerBParams` suppose une proportionnalité linéaire avec la taille du modèle, mais il y a un **plancher de puissance** lié au hardware.

L'article mentionne des puissances GPU de 1.252 à 2.735 kW. Un seul GPU H100 a un TDP de ~700W. Pour un modèle 7B qui tient sur un seul GPU :

```
Puissance réelle ≈ 300-500 W (un GPU en charge partielle)
Votre estimation ≈ 0.7 - 3.5 W
```

C'est un écart de **deux ordres de grandeur**. Le `WattsPerBParams` devrait être beaucoup plus élevé pour les petits modèles, ou bien il faut un terme de puissance minimale :

```go
gpuPower := max(minGPUPower, wattsPerBParams * bParams)
```

**Cependant**, si on suppose que le GPU est partagé entre plusieurs requêtes en batch (ce qui est le cas chez les hyperscalers), alors on n'attribue qu'une fraction de la puissance GPU à chaque requête. Si c'est l'intention, **il faut le documenter explicitement** car c'est une hypothèse structurante.

#### 3. Le facteur prefill de 20× est une simplification forte

```
durée_prefill = inputTokens / (tps × 20)
```

Ce facteur 20 est très variable en pratique. Il dépend de :

- La longueur du contexte (l'attention est quadratique en O(n²) pour le prefill)
- Le batching
- L'optimisation (Flash Attention, paged attention, etc.)

Pour un prompt de 100 tokens, le facteur peut être 50×. Pour un prompt de 32K tokens, il peut descendre à 5×. L'article mentionne des architectures comme MLA et sliding window qui modifient radicalement ce ratio.

**Suggestion** : Rendre ce facteur dépendant de la longueur du prompt, par exemple :

```
prefill_speedup = 30 / (1 + inputTokens / 5000)
```

#### 4. La règle `2N` FLOP par token ignore le coût de l'attention

L'article référence la complexité quadratique liée à l'attention. La formule `2 × activeParams` par token est correcte pour les couches feed-forward, mais le mécanisme d'attention ajoute un coût proportionnel à la longueur de séquence :

```
FLOP_attention ≈ 2 × n_layers × d_model × seq_length
```

Pour un modèle 7B avec un prompt de 1000 tokens, ce surcoût est modeste (~5-10%). Mais pour des contextes longs (32K+), il peut devenir **significatif** (30%+). L'article mentionne explicitement les "long context LLMs" comme défi.

#### 5. Les efficacités GFLOP/J semblent élevées

Vous indiquez 600-1200 GFLOP/J pour les hyperscalers. Un H100 a une efficacité théorique d'environ :

- FP16 : ~990 TFLOPS / 700W ≈ **1414 GFLOP/J** en crête
- Utilisation réelle en inférence : ~30-60% → **400-850 GFLOP/J**

Donc 1200 GFLOP/J comme borne haute est **optimiste** sauf pour du matériel de toute dernière génération (Blackwell) avec quantization INT8/FP8. La borne de 600 comme minimum hyperscaler semble correcte.

#### 6. L'overhead serveur de +10% est sous-estimé

L'article détaille que le PUE couvre le refroidissement et la distribution électrique, mais les overheads serveur (CPU, RAM, réseau, stockage) ne sont pas dans le PUE. Le facteur de 1.10 (+10%) est conservateur. Pour l'inférence LLM avec du KV-cache en mémoire HBM, la consommation mémoire peut représenter 15-25% de la puissance totale du serveur.

---

### Validation par rapport aux ordres de grandeur connus

L'article cite :

- ChatGPT : ~564 MWh/jour pour ~10M utilisateurs → **~56 Wh/utilisateur/jour**
- Si un utilisateur fait ~20 requêtes/jour → **~2.8 Wh/requête**

Votre estimation pour un modèle 7B donne **0.56 à 9.8 mWh** par requête. ChatGPT utilise probablement GPT-4 (~1.8T params, ~280B actifs en MoE), soit ~40× plus gros que 7B. En extrapolant linéairement : `2.3 mWh × 40 ≈ 92 mWh`, soit **~0.1 Wh** — c'est environ **28× inférieur** à l'estimation dérivée de l'article.

Cet écart suggère que la méthode TDP sous-estime significativement (cf. point 2 sur la puissance plancher).

---

### Recommandations prioritaires

1. **Ajouter un plancher de puissance GPU** dans la méthode TDP, ou documenter explicitement l'hypothèse de batching/partage
2. **Valider les résultats** contre les données de l'IEA (~1-10 mWh par requête ChatGPT simple) et ajuster les presets si nécessaire
3. **Rendre le facteur prefill dynamique** selon la longueur du prompt
4. **Signaler la cohérence inter-méthodes** — si TDP et FLOP divergent fortement, c'est un signal d'alerte utile pour l'utilisateur
