package main

import (
	"bufio"
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"os/exec"
	"strings"
	"sync"
	"time"
)

type Result struct {
	Name   string
	OK     bool
	Detail string
}

type Step struct {
	ID       int
	Module   string
	Title    string
	File     string
	Theory   []string
	Analogy  string
	Tasks    []string
	Try      []string
	Hints    []string
	Takeaway string
	Check    func(*Step) []Result
}

// ---------- outils de test ----------

func freePort() int {
	l, _ := net.Listen("tcp", "127.0.0.1:0")
	defer l.Close()
	return l.Addr().(*net.TCPAddr).Port
}

type syncBuf struct {
	mu sync.Mutex
	b  bytes.Buffer
}

func (s *syncBuf) Write(p []byte) (int, error) { s.mu.Lock(); defer s.mu.Unlock(); return s.b.Write(p) }
func (s *syncBuf) String() string              { s.mu.Lock(); defer s.mu.Unlock(); return s.b.String() }

type nodeServer struct {
	cmd  *exec.Cmd
	base string
	out  *syncBuf
	done chan struct{}
}

func tail(s string, n int) string {
	lines := strings.Split(strings.TrimRight(s, "\n"), "\n")
	if len(lines) > n {
		lines = lines[len(lines)-n:]
	}
	return strings.Join(lines, "\n")
}

// sansPile retire la pile d'appels interne de Node : il reste la ligne fautive et le message d'erreur.
func sansPile(out string) string {
	var garde []string
	for _, l := range strings.Split(out, "\n") {
		if t := strings.TrimSpace(l); strings.HasPrefix(t, "at ") || strings.HasPrefix(t, "Node.js v") {
			continue
		}
		garde = append(garde, l)
	}
	return strings.TrimSpace(strings.Join(garde, "\n"))
}

// startServer lance « node fichier » avec un PORT libre et attend qu'il écoute.
func startServer(file string, env ...string) (*nodeServer, error) {
	if err := nodeAvailable(); err != nil {
		return nil, err
	}
	port := freePort()
	cmd := exec.Command("node", file)
	cmd.Env = append(os.Environ(), append(env, fmt.Sprintf("PORT=%d", port))...)
	buf := &syncBuf{}
	cmd.Stdout, cmd.Stderr = buf, buf
	if err := cmd.Start(); err != nil {
		return nil, err
	}
	s := &nodeServer{cmd: cmd, base: fmt.Sprintf("http://127.0.0.1:%d", port), out: buf, done: make(chan struct{})}
	go func() { _ = cmd.Wait(); close(s.done) }()
	deadline := time.Now().Add(8 * time.Second)
	for time.Now().Before(deadline) {
		select {
		case <-s.done:
			out := buf.String()
			hint := ""
			if strings.Contains(out, "Cannot find module 'express'") || strings.Contains(out, "Cannot find module 'jsonwebtoken'") {
				hint = "\n→ Lancez « npm install » dans le dossier de travail."
			}
			return nil, fmt.Errorf("le serveur s'est arrêté au démarrage :\n%s%s", tail(sansPile(out), 8), hint)
		default:
		}
		if conn, err := net.DialTimeout("tcp", fmt.Sprintf("127.0.0.1:%d", port), 200*time.Millisecond); err == nil {
			conn.Close()
			return s, nil
		}
		time.Sleep(150 * time.Millisecond)
	}
	s.stop()
	return nil, fmt.Errorf("le serveur n'écoute pas sur process.env.PORT après 8 s.\nVérifiez app.listen(process.env.PORT || 3000)")
}

func (s *nodeServer) stop() {
	if s.cmd.Process != nil {
		_ = s.cmd.Process.Kill()
	}
	<-s.done
}

type resp struct {
	Status int
	Body   string
	Header http.Header
	Err    error
}

func (r resp) JSON(v any) error { return json.Unmarshal([]byte(r.Body), v) }

var client = &http.Client{Timeout: 4 * time.Second}

func (s *nodeServer) do(method, path, body string, headers map[string]string) resp {
	req, _ := http.NewRequest(method, s.base+path, strings.NewReader(body))
	if body != "" {
		req.Header.Set("Content-Type", "application/json")
	}
	for k, v := range headers {
		req.Header.Set(k, v)
	}
	r, err := client.Do(req)
	if err != nil {
		return resp{Err: err}
	}
	defer r.Body.Close()
	b, _ := io.ReadAll(r.Body)
	return resp{Status: r.StatusCode, Body: string(b), Header: r.Header}
}

// expectStatus fabrique un résultat lisible pour l'étudiant.
func expectStatus(name string, r resp, want int) Result {
	if r.Err != nil {
		return Result{name, false, "requête impossible : " + r.Err.Error()}
	}
	if r.Status != want {
		return Result{name, false, fmt.Sprintf("statut attendu %d, reçu %d\ncorps reçu : %s", want, r.Status, short(r.Body))}
	}
	return Result{Name: name, OK: true}
}

func short(s string) string {
	s = strings.TrimSpace(s)
	if len(s) > 160 {
		return s[:160] + "…"
	}
	if s == "" {
		return "(vide)"
	}
	return s
}

func fail(name string, err error) []Result { return []Result{{name, false, err.Error()}} }

func is(name string, ok bool, detail string) Result {
	if ok {
		return Result{Name: name, OK: true}
	}
	return Result{name, false, detail}
}

// ---------- les 10 étapes ----------

var steps = []Step{
	{
		ID: 1, Module: "Module 1 · Concepts", Title: "Consommer une API avec fetch", File: "01-fetch.js",
		Theory: []string{
			"Une API est un CONTRAT : le client envoie une requête (méthode + URL + en-têtes + corps)",
			"et reçoit une réponse (code de statut + en-têtes + corps JSON).",
			"⚠ fetch ne lève PAS d'erreur sur un 404 ou un 500 : il faut tester res.ok vous-même.",
		},
		Analogy: "Vous êtes le client du restaurant : vous ne voyez jamais la cuisine, seulement le menu (l'API).",
		Tasks: []string{
			"getUser(id) : GET /users/:id, lever une erreur si !res.ok, renvoyer le JSON",
			"en cas d'erreur, afficher « Échec : HTTP 404 » et renvoyer null",
			"creerPost(titre) : POST /posts avec Content-Type: application/json et JSON.stringify",
		},
		Try: []string{"node 01-fetch.js   (apilab lance une API de démo pendant les tests)"},
		Hints: []string{
			"const res = await fetch(url); puis if (!res.ok) throw new Error(`HTTP ${res.status}`)",
			"Pour le POST : fetch(url, { method: 'POST', headers: {...}, body: JSON.stringify({...}) })",
			"Affichage attendu : console.log('Créé :', post.id, '(statut', res.status + ')')",
		},
		Takeaway: "méthode + URL + en-têtes + corps d'un côté, statut + corps de l'autre : c'est tout HTTP.",
		Check:    checkFetch,
	},
	{
		ID: 2, Module: "Module 1 · Concepts", Title: "Trois paradigmes, une règle métier", File: "02-paradigmes.js",
		Theory: []string{
			"Procédural : des instructions qui modifient un état (boucle, accumulateur).",
			"Orienté objet : des objets qui encapsulent données ET comportements (classe Commande).",
			"Fonctionnel : des fonctions pures, sans effet de bord, composées (map, reduce).",
			"Dans une API, la logique métier vit dans ces fonctions, pas dans les routes.",
		},
		Analogy: "Même recette, trois cuisiniers : l'un suit la fiche ligne à ligne, l'autre a une brigade organisée, le dernier assemble des préparations toutes prêtes.",
		Tasks: []string{
			"totalProcedural(plats) avec une boucle for",
			"classe Commande : ajouter(plat) chaînable et totalTTC()",
			"totalFonctionnel(plats) avec map + reduce, sans variable modifiée",
		},
		Try:      []string{`node -e "console.log(require('./02-paradigmes').totalFonctionnel([{prix:10,qte:2}]))"`},
		Hints:    []string{"TTC = somme(prix * qte) * 1.2, puis arrondi(...)", "ajouter(plat) { this.plats.push(plat); return this; }", "plats.map(p => p.prix * p.qte).reduce((a, b) => a + b, 0)"},
		Takeaway: "les trois donnent le même résultat ; le fonctionnel est le plus simple à tester.",
		Check:    checkParadigmes,
	},
	{
		ID: 3, Module: "Module 2 · Node.js & Express", Title: "Première application Express", File: "03-hello-express.js",
		Theory: []string{
			"Node.js exécute JavaScript côté serveur avec une boucle d'événements non bloquante.",
			"Express associe une route (méthode + chemin) à une fonction (req, res).",
			"res.send() renvoie du texte, res.json() du JSON avec le bon Content-Type.",
		},
		Analogy:  "Node.js est la cuisine, Express est le chef de rang qui distribue les commandes aux bons postes.",
		Tasks:    []string{"GET / → « Bienvenue au restaurant »", "GET /health → { \"status\": \"ok\" }"},
		Try:      []string{"PORT=3000 node 03-hello-express.js", "curl -i http://localhost:3000/health"},
		Hints:    []string{"app.get('/health', (req, res) => ...)", "res.json({ status: 'ok' })"},
		Takeaway: "/health est la première route de toute API en production (sondes Kubernetes).",
		Check:    checkHello,
	},
	{
		ID: 4, Module: "Module 2 · Node.js & Express", Title: "CRUD REST et codes de statut", File: "04-crud.js",
		Theory: []string{
			"REST : des RESSOURCES (noms au pluriel) manipulées par des VERBES HTTP.",
			"GET lire · POST créer · PUT remplacer · DELETE supprimer.",
			"201 Created (+ Location) · 204 No Content · 400 Bad Request · 404 Not Found.",
		},
		Analogy: "2xx tout va bien · 4xx c'est ta faute (client) · 5xx c'est la mienne (serveur).",
		Tasks: []string{
			"GET /api/plats et GET /api/plats/:id (404 si absent)",
			"POST /api/plats → 201 + Location ; 400 si nom absent ou prix non numérique",
			"PUT /api/plats/:id → 200 ; DELETE /api/plats/:id → 204 ; 404 si absent",
		},
		Try: []string{
			"PORT=3000 node 04-crud.js",
			`curl -i -X POST localhost:3000/api/plats -H 'Content-Type: application/json' -d '{"nom":"Burger","prix":11}'`,
		},
		Hints: []string{
			"req.params.id est une CHAÎNE : comparez avec Number(req.params.id)",
			"res.status(201).location(`/api/plats/${p.id}`).json(p)",
			"Pour 204 : res.status(204).end() — jamais de corps",
		},
		Takeaway: "le verbe porte l'action, l'URL désigne la ressource, le code dit le verdict.",
		Check:    checkCrud,
	},
	{
		ID: 5, Module: "Module 2 · Node.js & Express", Title: "Middlewares et gestion d'erreurs", File: "05-middlewares.js",
		Theory: []string{
			"Un middleware (req, res, next) traite la requête puis passe la main avec next().",
			"L'ordre de déclaration = l'ordre d'exécution (pipeline).",
			"Le gestionnaire d'erreurs a 4 arguments (err, req, res, next) et se déclare en dernier.",
			"Sécurité : ne jamais renvoyer la stack d'erreur au client.",
		},
		Analogy:  "La brigade : chaque poste reçoit l'assiette, fait sa part, puis la passe au suivant.",
		Tasks:    []string{"middleware : log + en-tête X-Response-Time sur chaque réponse", "404 JSON pour les routes inconnues", "500 JSON générique pour /api/boom"},
		Try:      []string{"PORT=3000 node 05-middlewares.js", "curl -i localhost:3000/api/boom"},
		Hints:    []string{"L'en-tête doit être posé AVANT l'envoi : posez-le directement dans le middleware (ex : res.setHeader('X-Response-Time', '0ms')) puis next()", "app.use((req, res) => res.status(404).json(...)) après les routes", "app.use((err, req, res, next) => ...) tout à la fin"},
		Takeaway: "authentification, logs, limites, validation… tout est middleware dans Express.",
		Check:    checkMiddlewares,
	},
	{
		ID: 6, Module: "Module 5 · Sécurité", Title: "JWT : authentification & autorisation", File: "06-auth-jwt.js",
		Theory: []string{
			"Authentification = « qui êtes-vous ? » → 401 Unauthorized.",
			"Autorisation = « avez-vous le droit ? » → 403 Forbidden.",
			"Un JWT = header.payload.signature : ENCODÉ (lisible), pas chiffré. Rien de secret dedans.",
		},
		Analogy: "L'hôtel : la réception vérifie votre identité (401), la carte n'ouvre que votre chambre (403).",
		Tasks: []string{
			"POST /auth/login → { token } (jwt.sign, expiresIn 15m) ; 401 si identifiants faux",
			"middleware auth (Bearer) → 401 ; requireRole('admin') → 403",
			"GET /api/profil protégé ; DELETE /api/plats/:id réservé à l'admin (204)",
		},
		Try: []string{
			"PORT=3000 node 06-auth-jwt.js",
			`curl -s -X POST localhost:3000/auth/login -H 'Content-Type: application/json' -d '{"email":"chef@resto.fr","password":"admin123"}'`,
			"collez le token sur https://jwt.io : le payload est lisible par tous !",
		},
		Hints: []string{
			"const [type, token] = (req.headers.authorization || '').split(' ')",
			"try { req.user = jwt.verify(token, SECRET); next(); } catch { 401 }",
			"const requireRole = role => (req, res, next) => req.user.role === role ? next() : res.status(403)...",
		},
		Takeaway: "401 = je ne sais pas qui vous êtes ; 403 = je sais, mais c'est non.",
		Check:    checkAuth,
	},
	{
		ID: 7, Module: "Module 5 · Sécurité", Title: "Rate limiting fait main", File: "07-rate-limit.js",
		Theory: []string{
			"La limitation de débit protège contre le brute-force et les abus (OWASP API4).",
			"Au-delà du quota : 429 Too Many Requests + en-tête Retry-After.",
			"En production : express-rate-limit + un store partagé (Redis) entre instances.",
		},
		Analogy:  "Le videur de la boîte de nuit : 5 entrées par minute, les suivants attendent dehors.",
		Tasks:    []string{"compter les requêtes par IP sur une fenêtre de 60 s (Map)", "X-RateLimit-Limit / X-RateLimit-Remaining", "6e requête → 429 + Retry-After ; /health jamais limité"},
		Try:      []string{"PORT=3000 node 07-rate-limit.js", "for i in 1 2 3 4 5 6; do curl -s -o /dev/null -w '%{http_code} ' localhost:3000/api/ping; done"},
		Hints:    []string{"const compteurs = new Map() déclarée DANS rateLimit, hors du middleware", "if (now - c.debut > windowMs) { c.n = 0; c.debut = now; }", "res.setHeader('Retry-After', secondes) puis res.status(429).json(...)"},
		Takeaway: "monté sur '/api' uniquement : on choisit précisément quels endpoints protéger.",
		Check:    checkRateLimit,
	},
	{
		ID: 8, Module: "Module 3 · Paradigmes", Title: "Webhook signé (HMAC)", File: "08-webhook.js",
		Theory: []string{
			"Webhook : c'est le SERVEUR tiers qui appelle VOTRE URL quand un événement survient.",
			"Problème : n'importe qui peut appeler cette URL → on vérifie une signature HMAC.",
			"Les webhooks sont souvent renvoyés plusieurs fois : le traitement doit être idempotent.",
		},
		Analogy:  "« Je vous rappelle quand c'est prêt » — et vous vérifiez la voix avant de croire l'appel.",
		Tasks:    []string{"lire le corps BRUT avec express.raw", "HMAC SHA-256 avec WEBHOOK_SECRET, comparaison timingSafeEqual → 401 si faux", "enregistrer chaque paiement une seule fois par id → 200"},
		Try:      []string{"PORT=3000 node 08-webhook.js", "apilab check 8   (apilab joue le rôle du prestataire de paiement)"},
		Hints:    []string{"app.post('/webhooks/paiement', express.raw({ type: 'application/json' }), (req, res) => ...) : req.body est un Buffer", "crypto.createHmac('sha256', WEBHOOK_SECRET).update(req.body).digest('hex')", "Vérifiez les longueurs avant timingSafeEqual (sinon il lève une exception)"},
		Takeaway: "Stripe, GitHub, GitLab… tous signent leurs webhooks exactement ainsi.",
		Check:    checkWebhook,
	},
	{
		ID: 9, Module: "Module 3 · Paradigmes", Title: "Temps réel : polling vs SSE", File: "09-temps-reel.js",
		Theory: []string{
			"Polling : le client redemande toutes les N secondes (simple, mais gaspille des requêtes).",
			"Server-Sent Events : une connexion HTTP reste ouverte, le serveur POUSSE les mises à jour.",
			"WebSocket : comme SSE mais bidirectionnel (chat, jeux).",
		},
		Analogy:  "Polling : l'enfant qui demande « on est arrivés ? ». SSE : le GPS qui annonce lui-même chaque étape.",
		Tasks:    []string{"GET /api/commandes/42/events en text/event-stream", "envoyer l'état courant à la connexion, puis chaque changement", "retirer l'abonné à la déconnexion"},
		Try:      []string{"PORT=3000 node 09-temps-reel.js", "curl -N localhost:3000/api/commandes/42/events", "(2e terminal) curl -X POST localhost:3000/api/commandes/42/avancer"},
		Hints:    []string{"res.set({ 'Content-Type': 'text/event-stream', 'Cache-Control': 'no-cache' }); res.flushHeaders()", "Format d'un message SSE : res.write(`data: ${JSON.stringify(commande)}\\n\\n`)", "for (const r of abonnes) r.write(...) dans avancer()"},
		Takeaway: "1 connexion ouverte au lieu de dizaines de requêtes de polling.",
		Check:    checkSSE,
	},
	{
		ID: 10, Module: "Module 3 & 6 · Événementiel", Title: "Pub/Sub : découpler les services", File: "10-pubsub.js",
		Theory: []string{
			"Le producteur PUBLIE un événement sur un sujet, sans connaître les abonnés.",
			"Ajouter un service = s'abonner, sans toucher au producteur : c'est la base des microservices.",
			"En production : Kafka, RabbitMQ, NATS, Redis à la place de l'EventEmitter.",
		},
		Analogy:  "La cuisine crie « commande 12 prête ! » : le serveur, la caisse et le livreur réagissent chacun.",
		Tasks:    []string{"abonné notifications et abonné facturation sur 'commande.creee'", "POST /api/commandes : valider, créer, publier, répondre 201", "le producteur ne touche jamais aux tableaux notifications/factures"},
		Try:      []string{"PORT=3000 node 10-pubsub.js", `curl -X POST localhost:3000/api/commandes -H 'Content-Type: application/json' -d '{"client":"Alice","montant":24}'`, "curl localhost:3000/api/factures"},
		Hints:    []string{"bus.on('commande.creee', cmd => notifications.push(...))", "bus.emit('commande.creee', cmd) dans la route POST", "La route ne doit contenir ni notifications.push ni factures.push"},
		Takeaway: "le découplage par événements prépare le passage du monolithe aux microservices.",
		Check:    checkPubSub,
	},
}

// ---------- vérifications ----------

func checkFetch(s *Step) []Result {
	if err := nodeAvailable(); err != nil {
		return fail("Node.js disponible", err)
	}
	var mu sync.Mutex
	gotJSONHeader, gotPost := false, false
	mux := http.NewServeMux()
	mux.HandleFunc("/users/", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if strings.TrimPrefix(r.URL.Path, "/users/") != "1" {
			w.WriteHeader(404)
			w.Write([]byte(`{"error":"not found"}`))
			return
		}
		w.Write([]byte(`{"id":1,"name":"Leanne Graham","email":"Sincere@april.biz","address":{"city":"Gwenborough"}}`))
	})
	mux.HandleFunc("/posts", func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		defer mu.Unlock()
		gotPost = r.Method == "POST"
		gotJSONHeader = strings.HasPrefix(r.Header.Get("Content-Type"), "application/json")
		var body map[string]any
		if !gotPost || json.NewDecoder(r.Body).Decode(&body) != nil || body["title"] == nil {
			w.WriteHeader(400)
			w.Write([]byte(`{"error":"title requis (JSON)"}`))
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(201)
		out, _ := json.Marshal(map[string]any{"id": 101, "title": body["title"]})
		w.Write(out)
	})
	port := freePort()
	srv := &http.Server{Addr: fmt.Sprintf("127.0.0.1:%d", port), Handler: mux}
	go srv.ListenAndServe()
	defer srv.Shutdown(context.Background())
	time.Sleep(150 * time.Millisecond)

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, "node", s.File)
	cmd.Env = append(os.Environ(), fmt.Sprintf("API_URL=http://127.0.0.1:%d", port))
	out, _ := cmd.CombinedOutput()
	o := string(out)
	d := "sortie obtenue :\n" + tail(o, 6)
	mu.Lock()
	defer mu.Unlock()
	return []Result{
		is("GET /users/1 affiche « Leanne Graham - Sincere@april.biz »", strings.Contains(o, "Leanne Graham - Sincere@april.biz"), d),
		is("un 404 est détecté avec res.ok → « Échec : HTTP 404 »", strings.Contains(o, "Échec : HTTP 404"), d),
		is("le POST /posts envoie Content-Type: application/json", gotPost && gotJSONHeader, "aucun POST JSON reçu par l'API de démo"),
		is("le POST affiche « Créé : 101 (statut 201) »", strings.Contains(o, "Créé : 101 (statut 201)"), d),
	}
}

func checkParadigmes(s *Step) []Result {
	if err := nodeAvailable(); err != nil {
		return fail("Node.js disponible", err)
	}
	script := `
const m = require('./` + s.File + `');
const plats = [{prix:10,qte:2},{prix:5,qte:1}], vide = [];
const r = {};
const safe = (k, f) => { try { r[k] = f(); } catch (e) { r[k] = 'ERREUR: ' + e.message; } };
safe('proc', () => m.totalProcedural(plats));
safe('procVide', () => m.totalProcedural(vide));
safe('poo', () => new m.Commande([...plats]).totalTTC());
safe('chain', () => new m.Commande([]).ajouter({prix:3.33,qte:3}).ajouter({prix:1,qte:1}).totalTTC());
safe('fonc', () => m.totalFonctionnel(plats));
const copie = JSON.stringify(plats); safe('pur', () => (m.totalFonctionnel(plats), JSON.stringify(plats) === copie));
safe('src', () => m.totalFonctionnel.toString());
console.log(JSON.stringify(r));`
	out, err := exec.Command("node", "-e", script).CombinedOutput()
	var r map[string]any
	if err != nil || json.Unmarshal([]byte(lastLine(string(out))), &r) != nil {
		return fail("le module se charge sans erreur", fmt.Errorf("%s", tail(string(out), 6)))
	}
	eq := func(k string, want float64) Result {
		v, ok := r[k].(float64)
		return is(fmt.Sprintf("%s → %v", map[string]string{"proc": "totalProcedural", "procVide": "totalProcedural([])", "poo": "new Commande(...).totalTTC()", "chain": "Commande.ajouter(...).ajouter(...).totalTTC()", "fonc": "totalFonctionnel"}[k], want),
			ok && v == want, fmt.Sprintf("obtenu : %v", r[k]))
	}
	src, _ := r["src"].(string)
	return []Result{
		eq("proc", 30), eq("procVide", 0), eq("poo", 30), eq("chain", 13.19), eq("fonc", 30),
		is("totalFonctionnel est pur (ne modifie pas son entrée)", r["pur"] == true, "le tableau d'entrée a été modifié"),
		is("totalFonctionnel utilise map/reduce (pas de for)", strings.Contains(src, "reduce") && !strings.Contains(src, "for"), "utilisez map et reduce, sans boucle"),
	}
}

func lastLine(s string) string {
	l := strings.Split(strings.TrimSpace(s), "\n")
	return l[len(l)-1]
}

func checkHello(s *Step) []Result {
	sv, err := startServer(s.File)
	if err != nil {
		return fail("le serveur démarre sur process.env.PORT", err)
	}
	defer sv.stop()
	h := sv.do("GET", "/health", "", nil)
	var j map[string]any
	_ = h.JSON(&j)
	root := sv.do("GET", "/", "", nil)
	return []Result{
		{Name: "le serveur démarre sur process.env.PORT", OK: true},
		expectStatus("GET / → 200", root, 200),
		is("GET / contient « Bienvenue au restaurant »", strings.Contains(root.Body, "Bienvenue au restaurant"), "reçu : "+short(root.Body)),
		expectStatus("GET /health → 200", h, 200),
		is("GET /health renvoie du JSON { status: 'ok' }", j["status"] == "ok" && strings.Contains(h.Header.Get("Content-Type"), "json"), "reçu : "+short(h.Body)+"\nutilisez res.json(...)"),
	}
}

func checkCrud(s *Step) []Result {
	sv, err := startServer(s.File)
	if err != nil {
		return fail("le serveur démarre", err)
	}
	defer sv.stop()
	var list []map[string]any
	g := sv.do("GET", "/api/plats", "", nil)
	_ = g.JSON(&list)
	p := sv.do("POST", "/api/plats", `{"nom":"Burger","prix":11}`, nil)
	var created map[string]any
	_ = p.JSON(&created)
	id := fmt.Sprint(created["id"])
	if f, ok := created["id"].(float64); ok {
		id = fmt.Sprint(int(f))
	}
	get := sv.do("GET", "/api/plats/"+id, "", nil)
	put := sv.do("PUT", "/api/plats/"+id, `{"nom":"Burger XL","prix":14}`, nil)
	var upd map[string]any
	_ = put.JSON(&upd)
	del := sv.do("DELETE", "/api/plats/"+id, "", nil)
	after := sv.do("GET", "/api/plats/"+id, "", nil)
	return []Result{
		is("GET /api/plats → 200 et un tableau avec la Pizza", g.Status == 200 && len(list) >= 1 && list[0]["nom"] == "Pizza", fmt.Sprintf("statut %d, corps %s", g.Status, short(g.Body))),
		expectStatus("POST /api/plats valide → 201 Created", p, 201),
		is("POST renvoie l'en-tête Location /api/plats/<id>", p.Header.Get("Location") == "/api/plats/"+id, "Location reçu : "+p.Header.Get("Location")),
		expectStatus("POST sans prix → 400 Bad Request", sv.do("POST", "/api/plats", `{"nom":"Frites"}`, nil), 400),
		expectStatus("GET /api/plats/<id créé> → 200", get, 200),
		expectStatus("GET /api/plats/999 → 404", sv.do("GET", "/api/plats/999", "", nil), 404),
		is("PUT /api/plats/<id> → 200 et nom mis à jour", put.Status == 200 && upd["nom"] == "Burger XL", fmt.Sprintf("statut %d, corps %s", put.Status, short(put.Body))),
		expectStatus("PUT /api/plats/999 → 404", sv.do("PUT", "/api/plats/999", `{"nom":"X","prix":1}`, nil), 404),
		is("DELETE /api/plats/<id> → 204 sans corps", del.Status == 204 && del.Body == "", fmt.Sprintf("statut %d, corps %s", del.Status, short(del.Body))),
		expectStatus("GET après suppression → 404", after, 404),
		expectStatus("DELETE /api/plats/999 → 404", sv.do("DELETE", "/api/plats/999", "", nil), 404),
	}
}

func checkMiddlewares(s *Step) []Result {
	sv, err := startServer(s.File)
	if err != nil {
		return fail("le serveur démarre", err)
	}
	defer sv.stop()
	ping := sv.do("GET", "/api/ping", "", nil)
	nf := sv.do("GET", "/nimporte/quoi", "", nil)
	boom := sv.do("GET", "/api/boom", "", nil)
	var nfj, bj map[string]any
	_ = nf.JSON(&nfj)
	_ = boom.JSON(&bj)
	time.Sleep(100 * time.Millisecond)
	logs := sv.out.String()
	return []Result{
		expectStatus("GET /api/ping → 200", ping, 200),
		is("en-tête X-Response-Time présent sur /api/ping", ping.Header.Get("X-Response-Time") != "", "en-tête absent"),
		is("en-tête X-Response-Time présent aussi sur un 404", nf.Header.Get("X-Response-Time") != "", "le middleware doit être déclaré AVANT toutes les routes"),
		is("journal « GET /api/ping » affiché dans la console", strings.Contains(logs, "GET /api/ping"), "console du serveur :\n"+tail(logs, 4)),
		is("route inconnue → 404 JSON { error }", nf.Status == 404 && nfj["error"] != nil, fmt.Sprintf("statut %d, corps %s", nf.Status, short(nf.Body))),
		is("erreur levée → 500 JSON { error: 'Erreur interne' }", boom.Status == 500 && bj["error"] == "Erreur interne", fmt.Sprintf("statut %d, corps %s", boom.Status, short(boom.Body))),
		is("la stack et le message interne ne fuient pas au client", !strings.Contains(boom.Body, "four") && !strings.Contains(boom.Body, "at "), "le corps révèle des détails internes : "+short(boom.Body)),
	}
}

func checkAuth(s *Step) []Result {
	sv, err := startServer(s.File)
	if err != nil {
		return fail("le serveur démarre", err)
	}
	defer sv.stop()
	token := func(email, pw string) (string, resp) {
		r := sv.do("POST", "/auth/login", fmt.Sprintf(`{"email":%q,"password":%q}`, email, pw), nil)
		var j map[string]any
		_ = r.JSON(&j)
		t, _ := j["token"].(string)
		return t, r
	}
	admin, ra := token("chef@resto.fr", "admin123")
	user, _ := token("client@resto.fr", "client123")
	_, bad := token("chef@resto.fr", "faux")
	bearer := func(t string) map[string]string { return map[string]string{"Authorization": "Bearer " + t} }
	prof := sv.do("GET", "/api/profil", "", bearer(user))
	var pj map[string]any
	_ = prof.JSON(&pj)
	payloadOK := false
	if parts := strings.Split(admin, "."); len(parts) == 3 {
		if b, err := decodeB64(parts[1]); err == nil {
			var p map[string]any
			payloadOK = json.Unmarshal(b, &p) == nil && p["role"] == "admin" && p["exp"] != nil
		}
	}
	return []Result{
		is("POST /auth/login (admin) → 200 + token JWT", ra.Status == 200 && strings.Count(admin, ".") == 2, fmt.Sprintf("statut %d, corps %s", ra.Status, short(ra.Body))),
		is("le payload contient role et exp (expiresIn)", payloadOK, "signez { sub, role } avec { expiresIn: '15m' }"),
		expectStatus("mauvais mot de passe → 401", bad, 401),
		expectStatus("GET /api/menu (public) → 200", sv.do("GET", "/api/menu", "", nil), 200),
		expectStatus("GET /api/profil sans token → 401", sv.do("GET", "/api/profil", "", nil), 401),
		expectStatus("GET /api/profil avec token falsifié → 401", sv.do("GET", "/api/profil", "", bearer(admin+"x")), 401),
		is("GET /api/profil avec token client → 200 { role: 'user' }", prof.Status == 200 && pj["role"] == "user", fmt.Sprintf("statut %d, corps %s", prof.Status, short(prof.Body))),
		expectStatus("DELETE /api/plats/1 sans token → 401 (authentification)", sv.do("DELETE", "/api/plats/1", "", nil), 401),
		expectStatus("DELETE /api/plats/1 en client → 403 (autorisation)", sv.do("DELETE", "/api/plats/1", "", bearer(user)), 403),
		expectStatus("DELETE /api/plats/1 en admin → 204", sv.do("DELETE", "/api/plats/1", "", bearer(admin)), 204),
	}
}

func checkRateLimit(s *Step) []Result {
	sv, err := startServer(s.File)
	if err != nil {
		return fail("le serveur démarre", err)
	}
	defer sv.stop()
	var codes []int
	var last resp
	for i := 0; i < 6; i++ {
		last = sv.do("GET", "/api/ping", "", nil)
		codes = append(codes, last.Status)
	}
	first5 := true
	for _, c := range codes[:5] {
		first5 = first5 && c == 200
	}
	r1 := sv.do("GET", "/health", "", nil)
	return []Result{
		is("les 5 premières requêtes → 200", first5, fmt.Sprintf("codes obtenus : %v", codes)),
		is("la 6e requête → 429 Too Many Requests", codes[5] == 429, fmt.Sprintf("codes obtenus : %v", codes)),
		is("le 429 contient l'en-tête Retry-After", last.Header.Get("Retry-After") != "", "en-tête Retry-After absent"),
		is("en-têtes X-RateLimit-Limit = 5 et X-RateLimit-Remaining", last.Header.Get("X-RateLimit-Limit") == "5" && last.Header.Get("X-RateLimit-Remaining") != "", "en-têtes reçus : limit="+last.Header.Get("X-RateLimit-Limit")+" remaining="+last.Header.Get("X-RateLimit-Remaining")),
		expectStatus("/health n'est pas limité → 200", r1, 200),
	}
}

func checkWebhook(s *Step) []Result {
	sv, err := startServer(s.File, "WEBHOOK_SECRET=whsec_demo")
	if err != nil {
		return fail("le serveur démarre", err)
	}
	defer sv.stop()
	sign := func(body string) string {
		m := hmac.New(sha256.New, []byte("whsec_demo"))
		m.Write([]byte(body))
		return hex.EncodeToString(m.Sum(nil))
	}
	body := `{"id":"evt_42","type":"paiement.reussi","montant":2400}`
	ok1 := sv.do("POST", "/webhooks/paiement", body, map[string]string{"X-Signature": sign(body)})
	ok2 := sv.do("POST", "/webhooks/paiement", body, map[string]string{"X-Signature": sign(body)})
	forged := `{"id":"evt_99","type":"paiement.reussi","montant":999999}`
	bad := sv.do("POST", "/webhooks/paiement", forged, map[string]string{"X-Signature": sign(body)})
	none := sv.do("POST", "/webhooks/paiement", forged, nil)
	var list []map[string]any
	_ = sv.do("GET", "/api/paiements", "", nil).JSON(&list)
	return []Result{
		expectStatus("webhook correctement signé → 200", ok1, 200),
		expectStatus("signature ne correspondant pas au corps → 401", bad, 401),
		is("sans en-tête X-Signature → 401 (et le serveur ne plante pas)", none.Status == 401, fmt.Sprintf("statut %d", none.Status)),
		is("idempotence : envoyé 2 fois, enregistré 1 seule fois", ok2.Status == 200 && len(list) == 1, fmt.Sprintf("%d paiement(s) enregistré(s)", len(list))),
		is("le paiement falsifié n'est pas enregistré", len(list) == 0 || list[0]["id"] == "evt_42", "un paiement non signé a été accepté !"),
	}
}

func checkSSE(s *Step) []Result {
	sv, err := startServer(s.File)
	if err != nil {
		return fail("le serveur démarre", err)
	}
	defer sv.stop()
	poll := sv.do("GET", "/api/commandes/42", "", nil)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	req, _ := http.NewRequestWithContext(ctx, "GET", sv.base+"/api/commandes/42/events", nil)
	r, err := (&http.Client{}).Do(req)
	if err != nil {
		return []Result{expectStatus("GET /api/commandes/42 (polling) → 200", poll, 200), {"connexion SSE ouverte", false, err.Error()}}
	}
	defer r.Body.Close()
	ct := r.Header.Get("Content-Type")
	events := make(chan string, 10)
	go func() {
		sc := bufio.NewScanner(r.Body)
		for sc.Scan() {
			if l := sc.Text(); strings.HasPrefix(l, "data:") {
				events <- strings.TrimSpace(strings.TrimPrefix(l, "data:"))
			}
		}
		close(events)
	}()
	next := func() string {
		select {
		case e, ok := <-events:
			if ok {
				return e
			}
		case <-time.After(1500 * time.Millisecond):
		}
		return ""
	}
	first := next()
	sv.do("POST", "/api/commandes/42/avancer", "", nil)
	second := next()
	sv.do("POST", "/api/commandes/42/avancer", "", nil)
	third := next()
	return []Result{
		expectStatus("GET /api/commandes/42 (polling) → 200", poll, 200),
		is("Content-Type: text/event-stream", strings.HasPrefix(ct, "text/event-stream"), "Content-Type reçu : "+ct),
		is("état initial poussé à la connexion (statut recue)", strings.Contains(first, `"recue"`), "premier message : "+short(first)),
		is("changement poussé sans redemander (en_cuisine)", strings.Contains(second, `"en_cuisine"`), "message reçu : "+short(second)+"\nappelez res.write dans avancer()"),
		is("deuxième changement poussé (prete)", strings.Contains(third, `"prete"`), "message reçu : "+short(third)),
	}
}

func checkPubSub(s *Step) []Result {
	sv, err := startServer(s.File)
	if err != nil {
		return fail("le serveur démarre", err)
	}
	defer sv.stop()
	c1 := sv.do("POST", "/api/commandes", `{"client":"Alice","montant":24}`, nil)
	sv.do("POST", "/api/commandes", `{"client":"Bob","montant":12.5}`, nil)
	bad := sv.do("POST", "/api/commandes", `{"client":"Eve"}`, nil)
	var notifs []string
	var facts []map[string]any
	_ = sv.do("GET", "/api/notifications", "", nil).JSON(&notifs)
	_ = sv.do("GET", "/api/factures", "", nil).JSON(&facts)
	src, _ := os.ReadFile(s.File)
	route := string(src)
	if i := strings.Index(route, "app.post('/api/commandes'"); i >= 0 {
		route = route[i:]
		if j := strings.Index(route, "\n});"); j >= 0 {
			route = route[:j]
		}
	}
	return []Result{
		expectStatus("POST /api/commandes → 201", c1, 201),
		expectStatus("commande sans montant → 400", bad, 400),
		is("l'abonné notifications a reçu 2 événements", len(notifs) == 2 && strings.Contains(strings.Join(notifs, " "), "Alice"), fmt.Sprintf("notifications : %v", notifs)),
		is("l'abonné facturation a reçu 2 événements", len(facts) == 2, fmt.Sprintf("factures : %v", facts)),
		is("la route publie avec bus.emit('commande.creee', …)", strings.Contains(route, "emit('commande.creee'") || strings.Contains(route, `emit("commande.creee"`), "la route doit publier l'événement"),
		is("découplage : la route n'appelle pas les services directement", !strings.Contains(route, "notifications.push") && !strings.Contains(route, "factures.push"), "déplacez les push dans des bus.on(...)"),
	}
}

func decodeB64(s string) ([]byte, error) { return base64.RawURLEncoding.DecodeString(s) }
