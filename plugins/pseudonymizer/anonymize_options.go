package main

import (
	"context"
	"log/slog"

	goanon "github.com/bornholm/go-anon"
	proto "github.com/xolo-gateway/xolo/pkg/pluginsdk/proto"
)

const secretKeyHashHMAC = "hash_key"

// hashKeyLoader est l'interface minimale dont buildAnonymizeOptions a besoin.
// pluginsdk.HostClient la satisfait ; la signature d'interface permet de
// stubber facilement le host dans les tests unitaires.
type hashKeyLoader interface {
	GetSecret(ctx context.Context, orgID, pluginName, nodeID, key string) (string, bool, error)
}

// buildAnonymizeOptions assemble les AnonymizeOption à passer à Anonymize()
// en fonction de la configuration et de l'état du store de secrets.
//
// Comportement :
//   - Verification stricte prime sur l'observation (WithStrictVerification
//     inclut la vérification, mais on reste explicite pour la lecture).
//   - La clé HMAC est chargée via le store de secrets de l'hôte ; sans clé
//     valide, aucune option hash n'est ajoutée et l'anonymisation échouera
//     au runtime (warn loggué).
//   - HashScope n'est appliqué que si la clé est valide et que la stratégie
//     est hash.
func buildAnonymizeOptions(
	ctx context.Context,
	cfg Config,
	reqCtx *proto.RequestContext,
	host hashKeyLoader,
) ([]goanon.AnonymizeOption, error) {
	var opts []goanon.AnonymizeOption

	if cfg.VerificationStrict {
		opts = append(opts, goanon.WithStrictVerification())
	} else if cfg.Verification {
		opts = append(opts, goanon.WithVerification())
	}

	if cfg.Strategy == "hash" && host != nil {
		raw, found, err := host.GetSecret(ctx, reqCtx.GetOrgId(), "pseudonymizer", reqCtx.GetNodeId(), secretKeyHashHMAC)
		if err != nil {
			slog.WarnContext(ctx, "pseudonymizer: failed to read HMAC key from secret store", slog.Any("error", err))
		} else if !found || raw == "" {
			slog.WarnContext(ctx, "pseudonymizer: hash strategy requires a HMAC key, none configured")
		} else {
			key, err := goanon.ParseHashKey(raw)
			if err != nil {
				slog.WarnContext(ctx, "pseudonymizer: invalid HMAC key", slog.Any("error", err))
			} else {
				opts = append(opts, goanon.WithHashKey(key))
				if cfg.HashScope != "" {
					opts = append(opts, goanon.WithHashScope(cfg.HashScope))
				}
			}
		}
	}

	return opts, nil
}
