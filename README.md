# apilab — TP guidé en live coding : les API REST en JavaScript

`apilab` est un petit outil en ligne de commande, écrit en Go, qui accompagne l'étudiant **pas à pas**.
Il affiche la notion du cours, une analogie et les consignes, puis **surveille le fichier JavaScript** de l'étudiant.
**À chaque sauvegarde, il relance automatiquement les tests** (de vraies requêtes HTTP envoyées au serveur Express de l'étudiant), affiche ✔ / ✖ avec l'explication, et passe à l'étape suivante quand tout est vert.

Il existe aussi en **interface web** (`apilab quest`) : éditeur, cours, exemples, défis, mini-client HTTP, score et bilan final dans le navigateur.

---

## Installation : trois façons de lancer apilab

Choisissez selon ce qui est installé sur la machine. **Go n'est nécessaire que pour la troisième.**

| Vous avez… | Commande | Ce qu'il faut installer |
|---|---|---|
| **Docker** | `docker compose up` (ou `make docker-run`) | Docker uniquement : ni Go, ni Node.js |
| **Le binaire fourni par l'enseignant** | `./apilab quest` | Node.js ≥ 18 |
| **Les sources** | `make run` | Go ≥ 1.24 et Node.js ≥ 18 |

Dans les trois cas, l'interface s'ouvre sur **http://127.0.0.1:4321/**.

### 1. Avec Docker (aucune autre installation)

```bash
git clone https://github.com/raezon/lab_rest_api.git
cd lab_rest_api
docker compose up          # la première fois, construit l'image (quelques minutes)
```

Ouvrez ensuite http://127.0.0.1:4321/ dans le navigateur. `Ctrl+C` arrête le conteneur.

- **Votre travail est conservé** dans le dossier `travail/` du projet (code dans `travail/quete-api/code/`, score dans `progression.json`, inscription dans `profil.json`). Vous pouvez arrêter et relancer le conteneur sans rien perdre, et ouvrir ces fichiers avec votre éditeur habituel.
- L'image contient déjà Node.js, express et jsonwebtoken : **aucun accès à Internet n'est nécessaire** une fois l'image construite.
- Le port n'est publié que sur l'adresse locale (`127.0.0.1`) : l'outil exécute du code, il ne doit pas être joignable depuis le réseau.

Sans `docker compose`, l'équivalent en deux commandes :

```bash
docker build -t apilab .
docker run --rm -p 127.0.0.1:4321:4321 -v "$PWD/travail:/work" apilab
```

Variantes utiles :

```bash
APILAB_PORT=8080 docker compose up                                # autre port : http://127.0.0.1:8080/
APILAB_UID=$(id -u) APILAB_GID=$(id -g) docker compose up        # Linux, si votre identifiant n'est pas 1000
docker compose run --rm --service-ports apilab quest --no-open --parcours fp   # parcours « paradigme fonctionnel »
```

En cas de message `permission denied` sous Linux : le dossier `travail/` doit exister et vous appartenir. Il est fourni dans le dépôt ; s'il a été supprimé, recréez-le avec `mkdir travail` avant de lancer Docker.

### 2. Avec le binaire fourni

L'enseignant compile un binaire par système (`make dist`, voir plus bas) et le distribue. Il suffit alors d'avoir **Node.js ≥ 18** :

```bash
./apilab quest             # Linux et macOS (Windows : apilab.exe quest)
```

### 3. Depuis les sources, avec le Makefile

```bash
make run                   # compile puis lance l'interface web
```

`make` (ou `make help`) affiche toutes les cibles :

| Cible | Rôle | Nécessite |
|---|---|---|
| `make build` | compile le binaire `./apilab` | Go |
| `make run` | compile et lance l'interface web (API REST guidée) | Go, Node.js |
| `make run-fp` | idem, sur le parcours « paradigme fonctionnel » | Go, Node.js |
| `make cli` | prépare le TP en terminal dans `api-lab/` | Go, Node.js |
| `make test` | vérifie le code Go et les 35 quêtes (corrections, fichiers de départ, exemples) | Go, Node.js |
| `make fmt` | formate le code Go | Go |
| `make dist` | compile les binaires Linux, Windows et macOS dans `dist/` | Go |
| `make clean` | supprime le binaire et les dossiers de test (le travail des élèves est conservé) | — |
| `make docker-build` | construit l'image Docker `apilab` | Docker |
| `make docker-run` | lance l'interface web dans Docker, travail conservé dans `travail/` | Docker |
| `make docker-run-fp` | idem, sur le parcours « paradigme fonctionnel » | Docker |
| `make docker-test` | vérifie les 35 quêtes à l'intérieur de l'image | Docker |
| `make docker-clean` | supprime l'image Docker | Docker |

Le port se change avec `PORT` : `make run PORT=8080` ou `make docker-run PORT=8080`.

Sous Windows, `make` n'est pas installé par défaut : utilisez `docker compose up`, ou les commandes `go build` de la section « Compiler et distribuer ».

### Comment fonctionne l'image Docker

Le `Dockerfile` procède en trois étapes, pour une image finale légère (environ 180 Mo, sans Go) :

1. **Compilation** : une image `golang` compile le binaire `apilab`. Les contenus (quêtes, corrections, interface web) y sont embarqués.
2. **Dépendances** : une image `node` installe express et jsonwebtoken, dont les exercices ont besoin.
3. **Image finale** : Node.js, le binaire et ces dépendances. Go n'y figure pas.

Trois variables d'environnement y sont fixées : `APILAB_HOST=0.0.0.0` (dans un conteneur, le serveur doit écouter sur toutes les interfaces pour que le port publié soit joignable ; hors Docker, il n'écoute que sur `127.0.0.1`), `NODE_PATH` (où trouver express et jsonwebtoken, ce qui évite tout `npm install` au démarrage) et `HOME=/tmp`. Le dossier `/work` du conteneur reçoit le travail de l'élève : c'est lui qu'on relie à `travail/`.

---

## Démarrage en terminal (étudiant)

Prérequis : **Node.js ≥ 18**, un éditeur et le binaire `apilab`.

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

Options : `apilab quest [dossier] [--parcours api|fp] [--port 4321] [--host 127.0.0.1] [--no-open]`. L'adresse d'écoute se règle aussi avec la variable `APILAB_HOST` ; ne la changez que dans un conteneur.

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
make dist      # les quatre binaires d'un coup, dans dist/
```

ou, sans `make` :

```bash
go build -o apilab .                                   # votre machine
GOOS=windows GOARCH=amd64 go build -o apilab.exe .     # Windows
GOOS=darwin  GOARCH=arm64 go build -o apilab-mac .     # Mac Apple Silicon
GOOS=linux   GOARCH=amd64 go build -o apilab-linux .   # Linux
```

Les binaires ne sont pas versionnés : compilez-les (Go ≥ 1.24) et distribuez-les aux étudiants, qui n'ont alors besoin que de Node.js. Pour ceux qui n'ont ni Go ni Node.js, voir la section Docker. Aucune dépendance Go externe : starters, corrections et tests sont embarqués dans le binaire (`embed`).

## Ajouter une étape

1. Ajouter `assets/starters/11-xxx.js` et `assets/solutions/11-xxx.js`.
2. Ajouter une entrée dans `steps` (fichier `steps.go`) avec la notion, l'analogie, les consignes, les indices et une fonction `Check` qui démarre le serveur (`startServer`) et envoie les requêtes (`sv.do`).
3. `go build` puis `apilab selftest`.
