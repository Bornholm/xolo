# Getting started

## Présentation

![Accueil](accueil.png)

Plusieurs onglets se présentent :

- Usage : réunit toutes les informations de la plateformes:
  - le quota
  - le nombre de token utilisé
  - le nombre de requete
  - le coùt que celà représente
    TODO RAJOUTER LE RESTE

- Budget : Permet de gérer les quotas(journalier, mensuel, annuel)

- [Membres](./membre/membre.md) : Permet la gestion des rôles utilisateur, envoyer les invitations pour intégrer l'organisation.
- Rôles : Gestion des roles de la plateforme
- Fournisseurs :
  - gestion des fournisseurs de modèle (Openrouter, Anthropic, Mistral, etc)
  - gestion des modèles (pour les rendres accesible via la passerelle/plateform)
- [Invitation](./invitation/invitation.md) : Permet la création de lien d'invitation (permet aux utilisateurs de rejoindre l'organisation)
- Modèle virtuels : Permet la création de modèle "customisable", utiliser un modele existant et brancher des plugins (anonymisation, prompt_system), pour qu'il se comporte à la fin comme n'importe quel modele
- Middlewares : Permet l'ajout de traitements faites par la plateforme (contrôle horaire, filtrage, garde-fous…) appliqués dynamiquement aux modèles de l'organisation.

- Applications : Permet de paraméter des applications pour l'utilisation de la passerelle (exemple: OpenWebUi)
- Paramètres : Permet la gestion de la devise utilisé par l'organisation ainsi que la répartition du budget (définit les quotas)
