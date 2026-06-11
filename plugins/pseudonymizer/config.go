package main

import (
	"encoding/json"
	"fmt"

	goanon "github.com/bornholm/go-anon"
)

const configSchemaJSON = `{
  "type": "object",
  "properties": {
    "cache_dir": {
      "type": "string",
      "title": "Répertoire de cache",
      "description": "Chemin local pour stocker les modèles téléchargés. Par défaut : répertoire cache système."
    },
    "manifest_url": {
      "type": "string",
      "title": "URL du manifest",
      "description": "URL du manifest des modèles disponibles. Par défaut : dépôt officiel go-anon."
    },
    "offline": {
      "type": "boolean",
      "title": "Mode hors-ligne",
      "description": "Désactive les téléchargements réseau. Nécessite des modèles déjà en cache.",
      "default": false
    },
    "language": {
      "type": "string",
      "title": "Langue",
      "description": "Langue des messages à anonymiser.",
      "default": "fr",
      "enum": ["fr", "en"]
    },
    "strategy": {
      "type": "string",
      "title": "Stratégie d'anonymisation",
      "description": "Mode de remplacement des entités : tag=[PERSON_1], redact=████, hash=[PER_a1b2], consistent=numérotation cohérente.",
      "default": "tag",
      "enum": ["tag", "redact", "hash", "consistent"]
    },
    "min_confidence": {
      "type": "number",
      "title": "Confiance minimale",
      "description": "Seuil de confiance NER (0.0–1.0). Les entités en-dessous du seuil sont ignorées.",
      "default": 0.30,
      "minimum": 0,
      "maximum": 1
    },
    "max_tokens": {
      "type": "integer",
      "title": "Tokens max par entité",
      "description": "Les entités dépassant ce nombre de tokens sont ignorées. 0 = pas de limite.",
      "default": 0,
      "minimum": 0
    },
    "skip_types": {
      "type": "array",
      "title": "Types à ignorer",
      "description": "Types d'entités à ne pas anonymiser.",
      "items": {
        "type": "string",
        "enum": ["PER","LOC","ORG","MISC","EMAIL","IPV4","IPV6","IBAN","SIRET","SIREN","PHONE","API_KEY","JWT","SECRET"]
      }
    },
    "blocklist": {
      "type": "object",
      "title": "Liste de blocage",
      "description": "Mots à ignorer par type d'entité. Ex: {\"PER\": [\"Monsieur\", \"Madame\"]}",
      "additionalProperties": {
        "type": "array",
        "items": {"type": "string"}
      }
    },
    "first_name_reclassify": {
      "type": "boolean",
      "title": "Reclassification des prénoms",
      "description": "Reclasse les entités LOC d'un seul token en PER si le token figure dans le gazetteer de prénoms.",
      "default": false
    },
    "merge": {
      "type": "boolean",
      "title": "Fusion des entités adjacentes",
      "description": "Fusionne les entités adjacentes de même type (ex : prénom + nom de famille).",
      "default": false
    },
    "name_completion": {
      "type": "boolean",
      "title": "Complétion des noms",
      "description": "Complète les entités PER partielles avec le token de nom de famille adjacent.",
      "default": false
    },
    "builtin_regex_patterns": {
      "type": "boolean",
      "title": "Patterns regex intégrés",
      "description": "Détecte automatiquement EMAIL, IPV4/6, IBAN, SIRET, SIREN, PHONE via regex.",
      "default": true
    },
    "builtin_secret_patterns": {
      "type": "boolean",
      "title": "Patterns secrets intégrés",
      "description": "Détecte automatiquement JWT, clés API (OpenAI, AWS, GitHub, Slack…) via regex.",
      "default": true
    },
    "inject_instruction": {
      "type": "boolean",
      "title": "Instruction de préservation des jetons",
      "description": "Ajoute une instruction système demandant au LLM de recopier les jetons de substitution (ex: [PERSON_1]) sans les modifier. Ignoré pour la stratégie 'redact'.",
      "default": true
    }
  }
}`

// Config représente la configuration org-level du plugin.
type Config struct {
	// Modelstore
	CacheDir    string `json:"cache_dir"`
	ManifestURL string `json:"manifest_url"`
	Offline     bool   `json:"offline"`

	// Anonymisation
	Language              string              `json:"language"`
	Strategy              string              `json:"strategy"`
	MinConfidence         float64             `json:"min_confidence"`
	MaxTokens             int                 `json:"max_tokens"`
	SkipTypes             []string            `json:"skip_types"`
	Blocklist             map[string][]string `json:"blocklist"`
	FirstNameReclassify   bool                `json:"first_name_reclassify"`
	Merge                 bool                `json:"merge"`
	NameCompletion        bool                `json:"name_completion"`
	BuiltinRegexPatterns  bool                `json:"builtin_regex_patterns"`
	BuiltinSecretPatterns bool                `json:"builtin_secret_patterns"`
	InjectInstruction     bool                `json:"inject_instruction"`
}

func parseConfig(configJSON string) (Config, error) {
	cfg := defaultConfig()
	if configJSON == "" {
		return cfg, nil
	}
	if err := json.Unmarshal([]byte(configJSON), &cfg); err != nil {
		return Config{}, fmt.Errorf("parse config: %w", err)
	}
	if cfg.Language == "" {
		cfg.Language = "fr"
	}
	if cfg.Strategy == "" {
		cfg.Strategy = "tag"
	}
	return cfg, nil
}

func defaultConfig() Config {
	return Config{
		Language:              "fr",
		Strategy:              "tag",
		MinConfidence:         0.30,
		BuiltinRegexPatterns:  true,
		BuiltinSecretPatterns: true,
		InjectInstruction:     true,
	}
}

// strategyFromString converts the string strategy name to goanon.Strategy.
func strategyFromString(s string) goanon.Strategy {
	switch s {
	case "redact":
		return goanon.Redact
	case "hash":
		return goanon.Hash
	case "consistent":
		return goanon.Consistent
	default:
		return goanon.TagReplace
	}
}

// allEntityTypes is the complete list of entity types the anonymizer knows about.
var allEntityTypes = []goanon.EntityType{
	goanon.TypePER, goanon.TypeLOC, goanon.TypeORG, goanon.TypeMISC,
	goanon.TypeEMAIL, goanon.TypeIPV4, goanon.TypeIPV6, goanon.TypeIBAN,
	goanon.TypeSIRET, goanon.TypeSIREN, goanon.TypePHONE,
	goanon.TypeAPIKey, goanon.TypeJWT, goanon.TypeSecret,
}
