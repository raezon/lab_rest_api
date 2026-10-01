# apilab — TP guidé en live coding : les API REST en JavaScript

`apilab` est un petit outil en ligne de commande, écrit en Go, qui accompagne l'étudiant **pas à pas**.
Il affiche la notion du cours, une analogie et les consignes, puis **surveille le fichier JavaScript** de l'étudiant.
**À chaque sauvegarde, il relance automatiquement les tests** (de vraies requêtes HTTP envoyées au serveur Express de l'étudiant), affiche ✔ / ✖ avec l'explication, et passe à l'étape suivante quand tout est vert.

Prérequis côté étudiant : **Node.js ≥ 18** et un éditeur. Pas besoin d'installer Go : on distribue le binaire.

---

## Démarrage (étudiant)

```bash
apilab init            # crée le dossier api-lab/ avec les 10 fichiers d'exercice
cd api-lab
npm install            # express + jsonwebtoken
apilab watch           # mode live : ouvrez 01-fetch.js dans l'éditeur et codez
```

Disposition conseillée : **éditeur à gauche, terminal `apilab watch` à droite**, un 2e terminal pour les commandes `curl` proposées à chaque étape.

| Commande | Rôle |
|---|---|
| `apilab watch` | mode live : consignes + tests relancés à chaque sauvegarde |
| `apilab list` | progression (barre ████░░) et liste des étapes |
| `apilab show [n]` | notion du cours, analogie, consignes, commandes à essayer |
| `apilab check [n]` | lance les tests d'une étape |
| `apilab hint [n]` | indices progressifs (un de plus à chaque appel) |
| `apilab solution [n]` | écrit la correction dans `solutions/` sans toucher au fichier de l'étudiant |
| `apilab goto n` / `apilab reset n` | changer d'étape / repartir du fichier de départ |

---

## Le parcours : 10 étapes alignées sur le cours

| # | Fichier | Module du cours | Notion | Tests automatiques |
|---|---|---|---|---|
| 1 | `01-fetch.js` | 1 · Concepts | Consommer une API (GET, POST, `res.ok`) | 4 |
| 2 | `02-paradigmes.js` | 1 · Concepts | Procédural, POO, fonctionnel sur une même règle métier | 7 |
| 3 | `03-hello-express.js` | 2 · Express | Première app, `/health`, `res.json` | 5 |
| 4 | `04-crud.js` | 2 · REST | CRUD, verbes HTTP, 200/201/204/400/404, `Location` | 11 |
| 5 | `05-middlewares.js` | 2 · Express | Pipeline, logger, 404 et 500 JSON, pas de fuite de stack | 7 |
| 6 | `06-auth-jwt.js` | 5 · Sécurité | JWT, AuthN (401) vs AuthZ (403), rôles | 10 |
| 7 | `07-rate-limit.js` | 5 · Sécurité | Rate limiting fait main, 429, `Retry-After` | 5 |
| 8 | `08-webhook.js` | 3 · Paradigmes | Webhook signé HMAC, idempotence | 5 |
| 9 | `09-temps-reel.js` | 3 · Paradigmes | Polling vs Server-Sent Events | 5 |
| 10 | `10-pubsub.js` | 3 & 6 · Événementiel | Pub/Sub, découplage → microservices | 6 |

Durée indicative : 2 h 30 à 3 h. Les étapes 1 à 5 tiennent dans une séance de 1 h 30 ; 6 à 10 dans une seconde séance.
L'étape 1 n'a pas besoin d'Internet : `apilab` lance lui-même une API de démonstration pendant le test.

---

## Déroulé « live coding » pour l'enseignant

1. **Projeter** votre terminal `apilab watch` à côté de l'éditeur. Faire l'étape 1 en direct avec la salle : écrire volontairement le `fetch` sans tester `res.ok`, sauvegarder, montrer le test rouge « un 404 est détecté », corriger, voir passer au vert.
2. **Étapes 2 à 5 : en binôme**, l'enseignant circule. Réflexe à installer : *lire le message du test avant de demander de l'aide*. Chaque échec indique le statut attendu et le corps reçu.
3. **Point collectif** après l'étape 6 : faire coller un token sur jwt.io pour montrer que le payload est lisible par tous.
4. **Étapes 8 à 10** : faire comparer, dans le 2e terminal, le nombre de requêtes en polling et en SSE.
5. `apilab list` en fin de séance donne la progression de chacun. Le fichier `.apilab.json` peut être rendu avec le code.

Les indices sont progressifs (3 niveaux) et la correction s'écrit dans `solutions/`, ce qui évite de copier-coller sans comprendre.

Vérifier que toutes les corrections passent sur sa machine avant la séance :

```bash
apilab init demo && cd demo && npm install && apilab selftest
```

---

## Mode quête : l'interface web guidée

```bash
apilab quest                    # API REST guidée : ouvre http://127.0.0.1:4321/, code dans ./quete-api/code
apilab quest --parcours fp      # parcours « paradigme fonctionnel » (25 quêtes), code dans ./quete-fp/code
apilab quest selftest           # enseignant : vérifie les quêtes (correction, départ, exemple) ; accepte --parcours
```

Options : `apilab quest [dossier] [--parcours api|fp] [--port 4321] [--no-open]`.

Une interface web façon VS Code (liste des quêtes, éditeur avec coloration, sortie, tests, thèmes clair et sombre). Chaque quête suit le même déroulé : **cours** → **exemple à exécuter** → **défi** vérifié par des tests → **doc** et **indices**.

### Parcours `api` (par défaut) : construire une API REST

Ce sont les 10 étapes du tableau ci-dessus, avec leurs fichiers de départ, leurs corrections et leurs tests HTTP (`steps.go`). Le mode quête y ajoute, pour chaque étape, un cours, une doc et un exemple exécutable (`assets/quest/api/`).

Pour les étapes 3 à 10, le défi est un serveur : l'onglet **Requête**, sous l'éditeur, le démarre et lui envoie la requête de votre choix (méthode, chemin, corps JSON, en-têtes), puis affiche le statut, les en-têtes, le corps et la console du serveur. Le serveur reste en vie entre deux requêtes tant que le code ne change pas, ce qui permet de créer un plat puis de le relire. Deux variables aident : `{{token}}` (dernier jeton reçu de `/auth/login`) et `{{signature}}` (signature HMAC correcte du corps envoyé, pour l'étape webhook). Une réponse SSE est coupée après 1,5 s.

Au premier lancement, `npm install` est exécuté dans `quete-api/` (express, jsonwebtoken). L'étape 1 utilise une API de démonstration servie par apilab lui-même.

### Parcours `fp` : le paradigme fonctionnel

| Chapitre | Quêtes | Notions |
|---|---|---|
| 1 · La mise en place | 1 à 5 | fonctions, pureté, immutabilité (tableaux, objets), fonctions comme valeurs |
| 2 · À la chaîne | 6 à 10 | `map`, `filter`, `reduce`, chaînage, `find`/`some`/`every`, tri sans mutation |
| 3 · La fabrique à fonctions | 11 à 15 | closures, currying, application partielle, `pipe`/`compose`, `map` et `filter` réécrits avec `reduce` |
| 4 · Le garde-manger | 16 à 20 | regroupement, `flatMap` et `Set`, mise à jour imbriquée, récursion, `Object.entries` |
| 5 · Le coup de feu | 21 à 25 | mémoïsation, erreurs comme valeurs, asynchrone, style point-free, quête finale de synthèse |

### Score, architecture, extension

**Inscription et bilan.** Au premier lancement, l'élève saisit son prénom, son nom et son école (enregistrés dans `profil.json`, dans le dossier de travail). Quand la dernière quête est validée, le **bilan final** s'affiche : identité, date, score, étoiles, rang et détail par quête. Il s'imprime ou s'enregistre en PDF depuis l'onglet Score.

**Indices.** Chaque étape REST propose 6 à 7 indices progressifs, du plus général au plus proche de la solution (section `=== indices` du fichier `.quete`, séparés par `---`).

**Score.** Chaque quête rapporte des XP : +20 % si elle est réussie du premier coup, −10 % par indice, XP divisés par 4 si la correction a été consultée avant. Les étoiles (1 à 3) et le rang (Plongeur → Chef étoilé) sont enregistrés dans `progression.json`, un par parcours. Une quête se débloque quand la précédente est terminée ; les cours restent en lecture libre.

**Architecture.** Partie Go (`quest.go`) : sert l'interface embarquée, exécute le code avec Node à la demande (une exécution à la fois, arrêt après 5 s), gère le serveur d'essai de l'onglet Requête (arrêté après 2 minutes d'inactivité) et tient le score. Partie web (`assets/quest/web/`) : HTML, CSS et JavaScript sans framework ni dépendance. Le serveur n'écoute que sur `127.0.0.1` et refuse les requêtes venant d'une autre origine, puisqu'il exécute du code.

**Ajouter une étape REST.** Suivre « Ajouter une étape » plus bas, puis créer `assets/quest/api/11-xxx.quete` avec les en-têtes `@notion` et `@xp` et les sections `=== cours`, `=== doc`, `=== exemple` (un script qui se termine seul), et pour un serveur `=== requetes` (une par ligne : `MÉTHODE chemin | corps JSON | En-tête: valeur`). Une section `=== consigne` facultative complète les consignes de l'étape.

**Ajouter une quête fonctionnelle.** Créer `assets/quest/quetes/NN-nom.quete` : des en-têtes (`@chapitre`, `@titre`, `@notion`, `@xp`, `@retenir`) puis les sections `=== cours`, `doc`, `exemple`, `consigne`, `depart`, `solution`, `indices` (séparés par `---`) et `tests`. La section `tests` est du JavaScript qui dispose de `test(nom, (m, t) => …)`, `interdit(regex, message)` et `requis(regex, message)` ; `t` fournit `egal`, `proche`, `vrai` et `sansMutation`.

Après toute modification : recompiler, puis `apilab quest selftest`.

---

## Compiler et distribuer

```bash
go build -o apilab .                                   # votre machine
GOOS=windows GOARCH=amd64 go build -o apilab.exe .     # Windows
GOOS=darwin  GOARCH=arm64 go build -o apilab-mac .     # Mac Apple Silicon
GOOS=linux   GOARCH=amd64 go build -o apilab-linux .   # Linux
```

Les binaires ne sont pas versionnés : compilez-les avec les commandes ci-dessus (Go ≥ 1.24) et distribuez-les aux étudiants. Aucune dépendance Go externe : starters, corrections et tests sont embarqués dans le binaire (`embed`).

## Ajouter une étape

1. Ajouter `assets/starters/11-xxx.js` et `assets/solutions/11-xxx.js`.
2. Ajouter une entrée dans `steps` (fichier `steps.go`) avec la notion, l'analogie, les consignes, les indices et une fonction `Check` qui démarre le serveur (`startServer`) et envoie les requêtes (`sv.do`).
3. `go build` puis `apilab selftest`.
