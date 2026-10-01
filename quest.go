// quest.go — mode « quête » : une interface web (mini-éditeur façon VS Code)
// pour apprendre pas à pas, sur le thème du restaurant.
//
// Deux parcours partagent le même moteur :
//   - api : construire une API REST guidée (les 10 étapes de steps.go) ;
//   - fp  : le paradigme fonctionnel en Node.js (fichiers assets/quest/quetes).
//
// Le serveur Go sert l'interface (HTML/CSS/JS embarqués), exécute le code de
// l'élève avec Node à la demande, lance les tests de chaque quête et tient le score.
package main

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"math"
	"net"
	"net/http"
	"os"
	"os/exec"
	"os/signal"
	"path"
	"path/filepath"
	"runtime"
	"sort"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"time"
)

const questAssets = "assets/quest"

// Secret partagé avec le « prestataire de paiement » simulé (étape webhook).
const secretWebhook = "whsec_demo"

var rangs = []string{"Plongeur", "Commis", "Cuisinier", "Chef de partie", "Sous-chef", "Chef étoilé"}

type parcours struct {
	ID        string `json:"id"`
	Titre     string `json:"titre"`
	SousTitre string `json:"sousTitre"`
	Dossier   string `json:"-"`
}

var parcoursConnus = map[string]parcours{
	"api": {"api", "API Lab", "construire une API REST pas à pas", "quete-api"},
	"fp":  {"fp", "La Brigade Fonctionnelle", "le paradigme fonctionnel en Node.js", "quete-fp"},
}

// Requete : une requête HTTP proposée à l'élève pour essayer son serveur.
type Requete struct {
	Methode string `json:"methode"`
	Chemin  string `json:"chemin"`
	Corps   string `json:"corps"`
	Entetes string `json:"entetes"`
}

// Quete : une étape du parcours. Pour « fp », tout vient d'un fichier .quete
// (en-têtes « @cle valeur » puis sections « === nom »). Pour « api », le
// fichier .quete apporte cours, doc et exemple ; le reste vient de steps.go.
type Quete struct {
	ID        string    `json:"id"`
	Chapitre  string    `json:"chapitre"`
	Titre     string    `json:"titre"`
	Notion    string    `json:"notion"`
	XP        int       `json:"xp"`
	Retenir   string    `json:"retenir"`
	Cours     string    `json:"cours"`
	Doc       string    `json:"doc"`
	Consigne  string    `json:"consigne"`
	NbIndices int       `json:"nbIndices"`
	Serveur   bool      `json:"serveur"` // le défi est un serveur HTTP : onglet « Requête »
	Requetes  []Requete `json:"requetes"`
	Exemple   string    `json:"-"`
	Depart    string    `json:"-"`
	Solution  string    `json:"-"`
	Indices   []string  `json:"-"`
	Step      *Step     `json:"-"` // parcours api : les tests sont ceux de l'étape
}

type Avancement struct {
	Fait        bool `json:"fait"`
	XP          int  `json:"xp"`
	Essais      int  `json:"essais"`
	Indices     int  `json:"indices"`
	SolutionVue bool `json:"solutionVue"`
	Etoiles     int  `json:"etoiles"`
}

// Profil : l'inscription de l'élève, reprise sur le bilan final.
type Profil struct {
	Nom       string `json:"nom"`
	Prenom    string `json:"prenom"`
	Ecole     string `json:"ecole"`
	InscritLe string `json:"inscritLe"`
}

type Resultat struct {
	Nom    string `json:"nom"`
	OK     bool   `json:"ok"`
	Detail string `json:"detail,omitempty"`
}

type Score struct {
	XP         int    `json:"xp"`
	XPMax      int    `json:"xpMax"`
	Faites     int    `json:"faites"`
	Total      int    `json:"total"`
	Etoiles    int    `json:"etoiles"`
	EtoilesMax int    `json:"etoilesMax"`
	Rang       string `json:"rang"`
	Suivant    string `json:"suivant"`
}

// vueAvancement : ce que l'interface reçoit pour une quête.
type vueAvancement struct {
	Avancement
	Code           string   `json:"code"`
	Exemple        string   `json:"exemple"`
	IndicesReveles []string `json:"indicesReveles"`
	Solution       string   `json:"solution,omitempty"`
}

// serveurEssai : le serveur de l'élève, gardé en vie entre deux requêtes de
// l'onglet « Requête » tant que le code ne change pas (l'état en mémoire est conservé).
type serveurEssai struct {
	sv    *nodeServer
	id    string
	code  string
	repos *time.Timer
}

type questServer struct {
	dir      string // espace de travail (chemin absolu)
	parcours parcours
	quetes   []*Quete
	parID    map[string]*Quete
	demoURL  string // API de démonstration servie sous /demo (étape fetch)

	mu     sync.Mutex // protège prog, profil et essai, et n'autorise qu'une exécution à la fois
	prog   map[string]*Avancement
	profil *Profil
	essai  *serveurEssai
}

// ---------- chargement des quêtes ----------

func lireSections(src string) (entetes, sections map[string]string) {
	entetes, sections = map[string]string{}, map[string]string{}
	brut := map[string]*strings.Builder{}
	var cur *strings.Builder
	for _, l := range strings.Split(strings.ReplaceAll(src, "\r\n", "\n"), "\n") {
		if nom, ok := strings.CutPrefix(l, "=== "); ok {
			cur = &strings.Builder{}
			brut[strings.TrimSpace(nom)] = cur
			continue
		}
		if cur != nil {
			cur.WriteString(l + "\n")
			continue
		}
		if reste, ok := strings.CutPrefix(l, "@"); ok {
			cle, val, _ := strings.Cut(reste, " ")
			entetes[cle] = strings.TrimSpace(val)
		}
	}
	for nom, b := range brut {
		if t := strings.Trim(b.String(), "\n"); t != "" {
			sections[nom] = t + "\n"
		}
	}
	return entetes, sections
}

func manque(id string, sections map[string]string, noms ...string) error {
	for _, nom := range noms {
		if sections[nom] == "" {
			return fmt.Errorf("quête %s : section « %s » manquante", id, nom)
		}
	}
	return nil
}

// decouperIndices : les indices d'une section sont séparés par une ligne « --- ».
func decouperIndices(section string) []string {
	var out []string
	for _, ind := range strings.Split(section, "\n---\n") {
		if ind = strings.TrimSpace(ind); ind != "" {
			out = append(out, ind)
		}
	}
	return out
}

func parseQuete(id, src string) (*Quete, error) {
	e, sec := lireSections(src)
	if err := manque(id, sec, "cours", "doc", "consigne", "exemple", "depart", "solution", "indices", "tests"); err != nil {
		return nil, err
	}
	q := &Quete{ID: id, Chapitre: e["chapitre"], Titre: e["titre"], Notion: e["notion"], Retenir: e["retenir"],
		Cours: sec["cours"], Doc: sec["doc"], Consigne: sec["consigne"],
		Exemple: sec["exemple"], Depart: sec["depart"], Solution: sec["solution"], Requetes: []Requete{}}
	q.XP, _ = strconv.Atoi(e["xp"])
	q.Indices = decouperIndices(sec["indices"])
	q.NbIndices = len(q.Indices)
	if q.Titre == "" || q.Chapitre == "" || q.XP <= 0 {
		return nil, fmt.Errorf("quête %s : en-têtes @titre, @chapitre ou @xp manquants", id)
	}
	return q, nil
}

func chargerQuetesFP() ([]*Quete, error) {
	entrees, err := fs.ReadDir(assets, questAssets+"/quetes")
	if err != nil {
		return nil, err
	}
	var out []*Quete
	for _, e := range entrees {
		if !strings.HasSuffix(e.Name(), ".quete") {
			continue
		}
		b, _ := assets.ReadFile(path.Join(questAssets, "quetes", e.Name()))
		q, err := parseQuete(strings.TrimSuffix(e.Name(), ".quete"), string(b))
		if err != nil {
			return nil, err
		}
		out = append(out, q)
	}
	return out, nil
}

// chargerQuetesAPI habille chaque étape de steps.go (consignes, indices,
// fichier de départ, correction, tests) avec le cours, la doc, l'exemple et
// les requêtes d'essai décrits dans assets/quest/api/<étape>.quete.
func chargerQuetesAPI() ([]*Quete, error) {
	var out []*Quete
	for i := range steps {
		st := &steps[i]
		id := strings.TrimSuffix(st.File, ".js")
		b, err := assets.ReadFile(path.Join(questAssets, "api", id+".quete"))
		if err != nil {
			return nil, fmt.Errorf("quête %s : fichier assets/quest/api/%s.quete manquant", id, id)
		}
		e, sec := lireSections(string(b))
		if err := manque(id, sec, "cours", "doc", "exemple"); err != nil {
			return nil, err
		}
		depart, _ := assets.ReadFile("assets/starters/" + st.File)
		solution, _ := assets.ReadFile("assets/solutions/" + st.File)
		q := &Quete{ID: id, Chapitre: st.Module, Titre: st.Title, Notion: e["notion"], Retenir: st.Takeaway,
			Cours: sec["cours"], Doc: sec["doc"], Exemple: sec["exemple"],
			Depart: string(depart), Solution: string(solution),
			Indices: st.Hints, NbIndices: len(st.Hints), Step: st, Requetes: []Requete{}}
		// Le fichier .quete peut fournir des indices plus nombreux et plus progressifs que ceux de l'étape.
		if ind := decouperIndices(sec["indices"]); len(ind) > 0 {
			q.Indices, q.NbIndices = ind, len(ind)
		}
		if q.XP, _ = strconv.Atoi(e["xp"]); q.XP <= 0 {
			return nil, fmt.Errorf("quête %s : en-tête @xp manquant", id)
		}
		// Une requête par ligne : « MÉTHODE chemin | corps JSON | En-tête: valeur »
		for _, l := range strings.Split(sec["requetes"], "\n") {
			if l = strings.TrimSpace(l); l == "" {
				continue
			}
			champs := strings.SplitN(l, "|", 3)
			methode, chemin, _ := strings.Cut(strings.TrimSpace(champs[0]), " ")
			r := Requete{Methode: methode, Chemin: strings.TrimSpace(chemin)}
			if len(champs) > 1 {
				r.Corps = strings.TrimSpace(champs[1])
			}
			if len(champs) > 2 {
				r.Entetes = strings.TrimSpace(champs[2])
			}
			q.Requetes = append(q.Requetes, r)
		}
		q.Serveur = len(q.Requetes) > 0

		var c strings.Builder
		c.WriteString("## Défi : " + st.Title + "\n\nComplète `defi.js` :\n\n")
		for n, t := range st.Tasks {
			fmt.Fprintf(&c, "%d. %s\n", n+1, t)
		}
		if sec["consigne"] != "" {
			c.WriteString("\n" + sec["consigne"])
		}
		if q.Serveur {
			c.WriteString("\nEssaie ton serveur avec l'onglet **Requête** (sous l'éditeur), puis clique sur **Vérifier** pour lancer les tests.\n")
		} else {
			c.WriteString("\nLance ton fichier avec **Exécuter**, puis clique sur **Vérifier** pour lancer les tests.\n")
		}
		q.Consigne = c.String()
		out = append(out, q)
	}
	return out, nil
}

// ---------- espace de travail ----------

func nouveauServeurQuete(dir string, p parcours) (*questServer, error) {
	abs, err := filepath.Abs(dir)
	if err != nil {
		return nil, err
	}
	s := &questServer{dir: abs, parcours: p, parID: map[string]*Quete{}, prog: map[string]*Avancement{}}
	if p.ID == "api" {
		s.quetes, err = chargerQuetesAPI()
	} else {
		s.quetes, err = chargerQuetesFP()
	}
	if err != nil {
		return nil, err
	}
	for _, d := range []string{"code", filepath.Join(".moteur", "quetes")} {
		if err := os.MkdirAll(filepath.Join(abs, d), 0o755); err != nil {
			return nil, err
		}
	}
	for _, q := range s.quetes {
		s.parID[q.ID] = q
	}
	if p.ID == "api" {
		s.installerDependances()
		// Les tests de steps.go lancent « node <fichier> » avec un chemin relatif.
		if err := os.Chdir(filepath.Join(abs, "code")); err != nil {
			return nil, err
		}
	} else {
		// Le moteur de tests est ré-extrait à chaque démarrage : il suit la version du binaire.
		runner, _ := assets.ReadFile(questAssets + "/runner.js")
		if err := os.WriteFile(filepath.Join(abs, ".moteur", "runner.js"), runner, 0o644); err != nil {
			return nil, err
		}
		for _, q := range s.quetes {
			b, _ := assets.ReadFile(path.Join(questAssets, "quetes", q.ID+".quete"))
			if err := os.WriteFile(filepath.Join(abs, ".moteur", "quetes", q.ID+".quete"), b, 0o644); err != nil {
				return nil, err
			}
		}
	}
	if b, err := os.ReadFile(filepath.Join(abs, "progression.json")); err == nil {
		_ = json.Unmarshal(b, &s.prog)
	}
	for _, q := range s.quetes {
		if s.prog[q.ID] == nil {
			s.prog[q.ID] = &Avancement{}
		}
	}
	if b, err := os.ReadFile(filepath.Join(abs, "profil.json")); err == nil {
		var p Profil
		if json.Unmarshal(b, &p) == nil && p.Nom != "" {
			s.profil = &p
		}
	}
	return s, nil
}

// installerDependances : express et jsonwebtoken, une seule fois par espace de travail.
func (s *questServer) installerDependances() {
	pkg := filepath.Join(s.dir, "package.json")
	if _, err := os.Stat(pkg); err != nil {
		b, _ := assets.ReadFile("assets/package.json")
		_ = os.WriteFile(pkg, b, 0o644)
	}
	present := func(module string) bool {
		_, err := os.Stat(filepath.Join(s.dir, "node_modules", module, "package.json"))
		return err == nil
	}
	if present("express") && present("jsonwebtoken") {
		return
	}
	fmt.Println(c(dim, "  Installation d'express et jsonwebtoken (npm install)…"))
	cmd := exec.Command("npm", "install", "--no-audit", "--no-fund")
	cmd.Dir = s.dir
	if out, err := cmd.CombinedOutput(); err != nil {
		fmt.Println(c(yellow, "  npm install a échoué : lancez-le à la main dans "+s.dir))
		fmt.Println(c(dim, tail(string(out), 4)))
	}
}

func (s *questServer) sauverProgression() {
	out, _ := json.MarshalIndent(s.prog, "", "  ")
	_ = os.WriteFile(filepath.Join(s.dir, "progression.json"), out, 0o644)
}

func (s *questServer) chemin(q *Quete, fichier string) string {
	if fichier == "exemple" {
		return filepath.Join(s.dir, "code", q.ID+".exemple.js")
	}
	return filepath.Join(s.dir, "code", q.ID+".js")
}

func (s *questServer) lireCode(q *Quete, fichier string) string {
	if b, err := os.ReadFile(s.chemin(q, fichier)); err == nil {
		return string(b)
	}
	if fichier == "exemple" {
		return q.Exemple
	}
	return q.Depart
}

func (s *questServer) vue(q *Quete) vueAvancement {
	a := s.prog[q.ID]
	v := vueAvancement{Avancement: *a, Code: s.lireCode(q, "defi"), Exemple: s.lireCode(q, "exemple"), IndicesReveles: []string{}}
	n := min(a.Indices, len(q.Indices))
	if a.Fait {
		n = len(q.Indices)
	}
	v.IndicesReveles = append(v.IndicesReveles, q.Indices[:n]...)
	if a.Fait || a.SolutionVue {
		v.Solution = q.Solution
	}
	return v
}

func (s *questServer) score() Score {
	sc := Score{Total: len(s.quetes), EtoilesMax: 3 * len(s.quetes)}
	base := 0
	for _, q := range s.quetes {
		a := s.prog[q.ID]
		base += q.XP
		if a.Fait {
			sc.Faites++
			sc.XP += a.XP
			sc.Etoiles += a.Etoiles
		}
	}
	sc.XPMax = int(math.Round(float64(base) * 1.2)) // 1.2 = bonus « du premier coup »
	niveau := 0
	if sc.Total > 0 {
		niveau = sc.Faites * (len(rangs) - 1) / sc.Total
	}
	sc.Rang = rangs[niveau]
	if niveau+1 < len(rangs) {
		sc.Suivant = rangs[niveau+1]
	}
	return sc
}

// noter fixe les XP et les étoiles à la première réussite d'une quête.
func noter(q *Quete, a *Avancement) {
	mult := max(1.0-0.10*float64(a.Indices), 0.4)
	a.Etoiles = 3
	switch {
	case a.SolutionVue:
		mult, a.Etoiles = 0.25, 1
	case a.Indices > 2 || a.Essais > 6:
		a.Etoiles = 1
	case a.Indices > 0 || a.Essais > 3:
		a.Etoiles = 2
	case a.Essais == 1:
		mult = 1.2
	}
	a.XP = int(math.Round(float64(q.XP) * mult))
	a.Fait = true
}

// ---------- exécution Node ----------

// tampon limite la sortie conservée : un console.log dans une boucle ne sature pas la mémoire.
type tampon struct {
	b   bytes.Buffer
	max int
}

func (t *tampon) Write(p []byte) (int, error) {
	if reste := t.max - t.b.Len(); reste > 0 {
		if len(p) > reste {
			t.b.Write(p[:reste])
			t.b.WriteString("\n… (sortie tronquée)\n")
		} else {
			t.b.Write(p)
		}
	}
	return len(p), nil
}

// sansChemins : les chemins absolus n'apportent rien à l'élève, on ne garde que le nom du fichier.
func (s *questServer) sansChemins(sortie string) string {
	sortie = strings.ReplaceAll(sortie, filepath.Join(s.dir, "code")+string(os.PathSeparator), "")
	return strings.ReplaceAll(sortie, filepath.Join(s.dir, ".moteur")+string(os.PathSeparator), "")
}

func (s *questServer) node(delai time.Duration, args ...string) (sortie string, expire bool, code int) {
	ctx, cancel := context.WithTimeout(context.Background(), delai)
	defer cancel()
	cmd := exec.CommandContext(ctx, "node", append([]string{"--max-old-space-size=64"}, args...)...)
	cmd.Dir = filepath.Join(s.dir, "code")
	cmd.Env = append(os.Environ(), "API_URL="+s.demoURL)
	buf := &tampon{max: 64 << 10}
	cmd.Stdout, cmd.Stderr = buf, buf
	cmd.WaitDelay = time.Second
	err := cmd.Run()
	sortie = s.sansChemins(buf.b.String())
	if ctx.Err() == context.DeadlineExceeded {
		return sortie, true, -1
	}
	var ee *exec.ExitError
	if errors.As(err, &ee) {
		return sortie, false, ee.ExitCode()
	}
	if err != nil {
		return sortie + err.Error(), false, -1
	}
	return sortie, false, 0
}

// lancerTests vérifie le fichier de l'élève (qui doit se trouver dans code/).
func (s *questServer) lancerTests(q *Quete, fichier string) ([]Resultat, string) {
	if q.Step != nil {
		st := *q.Step
		st.File = filepath.Base(fichier) // relatif à code/, le dossier courant du processus
		var res []Resultat
		for _, r := range st.Check(&st) {
			res = append(res, Resultat{r.Name, r.OK, s.sansChemins(r.Detail)})
		}
		return res, ""
	}
	moteur := filepath.Join(s.dir, ".moteur")
	resJSON := filepath.Join(moteur, "resultat.json")
	_ = os.Remove(resJSON)
	sortie, expire, _ := s.node(6*time.Second, filepath.Join(moteur, "runner.js"),
		filepath.Join(moteur, "quetes", q.ID+".quete"), fichier, resJSON)
	var res []Resultat
	if b, err := os.ReadFile(resJSON); err == nil && json.Unmarshal(b, &res) == nil && len(res) > 0 {
		return res, sortie
	}
	if expire {
		return []Resultat{{Nom: "Le programme se termine en moins de 6 secondes",
			Detail: "Temps dépassé : boucle infinie, ou promesse jamais résolue ?"}}, sortie
	}
	return []Resultat{{Nom: "Les tests ont pu démarrer",
		Detail: "Le moteur de tests s'est arrêté avant la fin. Regarde l'onglet Sortie."}}, sortie
}

// ---------- serveur d'essai (onglet « Requête ») ----------

func (s *questServer) arreterEssai() {
	if s.essai == nil {
		return
	}
	s.essai.repos.Stop()
	s.essai.sv.stop()
	s.essai = nil
}

// essaiPour renvoie le serveur de l'élève pour cette quête, en le (re)démarrant
// si le code a changé ou s'il s'est arrêté.
func (s *questServer) essaiPour(q *Quete, code string) (e *serveurEssai, redemarre bool, err error) {
	if e = s.essai; e != nil && e.id == q.ID && e.code == code {
		select {
		case <-e.sv.done: // le processus est mort entre-temps
		default:
			e.repos.Reset(2 * time.Minute)
			return e, false, nil
		}
	}
	s.arreterEssai()
	sv, err := startServer(s.chemin(q, "defi"), "WEBHOOK_SECRET="+secretWebhook)
	if err != nil {
		return nil, true, err
	}
	e = &serveurEssai{sv: sv, id: q.ID, code: code}
	// Un serveur oublié s'arrête seul : pas de processus Node qui traîne.
	e.repos = time.AfterFunc(2*time.Minute, func() {
		s.mu.Lock()
		defer s.mu.Unlock()
		if s.essai == e {
			s.arreterEssai()
		}
	})
	s.essai = e
	return e, true, nil
}

type reponseEssai struct {
	Statut  int         `json:"statut"`
	Libelle string      `json:"libelle"`
	DureeMs int64       `json:"dureeMs"`
	Entetes [][2]string `json:"entetes"`
	Corps   string      `json:"corps"`
	Flux    bool        `json:"flux"` // réponse qui ne se termine pas (SSE) : coupée après 1,5 s
}

func envoyerEssai(base, methode, chemin, corps string, entetes map[string]string) (*reponseEssai, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, methode, base+chemin, strings.NewReader(corps))
	if err != nil {
		return nil, err
	}
	if corps != "" {
		req.Header.Set("Content-Type", "application/json")
	}
	for k, v := range entetes {
		req.Header.Set(k, v)
	}
	debut := time.Now()
	r, err := http.DefaultTransport.RoundTrip(req) // pas de suivi de redirection : on montre la réponse brute
	if err != nil {
		return nil, err
	}
	defer r.Body.Close()
	out := &reponseEssai{Statut: r.StatusCode, Libelle: http.StatusText(r.StatusCode), DureeMs: time.Since(debut).Milliseconds()}
	for k, v := range r.Header {
		out.Entetes = append(out.Entetes, [2]string{k, strings.Join(v, ", ")})
	}
	sort.Slice(out.Entetes, func(i, j int) bool { return out.Entetes[i][0] < out.Entetes[j][0] })
	buf := &tampon{max: 64 << 10}
	fini := make(chan struct{})
	go func() {
		_, _ = io.Copy(buf, r.Body)
		close(fini)
	}()
	select {
	case <-fini:
	case <-time.After(1500 * time.Millisecond):
		out.Flux = true
		cancel()
		<-fini
	}
	out.Corps = buf.b.String()
	return out, nil
}

// ---------- HTTP ----------

type requete struct {
	ID      string `json:"id"`
	Fichier string `json:"fichier"`
	Code    string `json:"code"`
	Methode string `json:"methode"`
	Chemin  string `json:"chemin"`
	Corps   string `json:"corps"`
	Entetes string `json:"entetes"`
}

func repondre(w http.ResponseWriter, v any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	_ = json.NewEncoder(w).Encode(v)
}

func (s *questServer) lire(w http.ResponseWriter, r *http.Request) (*Quete, requete, bool) {
	var rq requete
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 256<<10)).Decode(&rq); err != nil {
		http.Error(w, "requête invalide", http.StatusBadRequest)
		return nil, rq, false
	}
	q := s.parID[rq.ID]
	if q == nil {
		http.Error(w, "quête inconnue", http.StatusNotFound)
		return nil, rq, false
	}
	return q, rq, true
}

// garde : le serveur exécute du code, il ne répond donc qu'à la page qu'il sert
// lui-même (hôte local, même origine, JSON uniquement). L'API de démonstration
// /demo, appelée par le code de l'élève, n'exécute rien et échappe à ces deux contrôles.
func (s *questServer) garde(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hote, _, err := net.SplitHostPort(r.Host)
		if err != nil {
			hote = r.Host
		}
		if hote != "127.0.0.1" && hote != "localhost" {
			http.Error(w, "hôte refusé", http.StatusForbidden)
			return
		}
		if r.Method != http.MethodGet && r.Method != http.MethodHead && !strings.HasPrefix(r.URL.Path, "/demo/") {
			if o := r.Header.Get("Origin"); o != "" && o != "http://"+r.Host {
				http.Error(w, "origine refusée", http.StatusForbidden)
				return
			}
			if !strings.HasPrefix(r.Header.Get("Content-Type"), "application/json") {
				http.Error(w, "JSON attendu", http.StatusUnsupportedMediaType)
				return
			}
		}
		w.Header().Set("Cache-Control", "no-store")
		next.ServeHTTP(w, r)
	})
}

func (s *questServer) routes() http.Handler {
	mux := http.NewServeMux()
	web, _ := fs.Sub(assets, questAssets+"/web")
	mux.Handle("GET /", http.FileServerFS(web))
	mux.HandleFunc("GET /api/etat", s.hEtat)
	mux.HandleFunc("POST /api/sauver", s.hSauver)
	mux.HandleFunc("POST /api/executer", s.hExecuter)
	mux.HandleFunc("POST /api/verifier", s.hVerifier)
	mux.HandleFunc("POST /api/requete", s.hRequete)
	mux.HandleFunc("POST /api/profil", s.hProfil)
	mux.HandleFunc("POST /api/indice", s.hIndice)
	mux.HandleFunc("POST /api/solution", s.hSolution)
	mux.HandleFunc("POST /api/reinit", s.hReinit)
	mux.HandleFunc("POST /api/reinit-tout", s.hReinitTout)
	mux.HandleFunc("GET /demo/users/{id}", demoUser)
	mux.HandleFunc("POST /demo/posts", demoPosts)
	return s.garde(mux)
}

// API de démonstration de l'étape « fetch » : même contrat que celle des tests.
func demoUser(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	if r.PathValue("id") != "1" {
		w.WriteHeader(http.StatusNotFound)
		_, _ = w.Write([]byte(`{"error":"not found"}`))
		return
	}
	_, _ = w.Write([]byte(`{"id":1,"name":"Leanne Graham","email":"Sincere@april.biz","address":{"city":"Gwenborough"}}`))
}

func demoPosts(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	var body map[string]any
	if !strings.HasPrefix(r.Header.Get("Content-Type"), "application/json") ||
		json.NewDecoder(http.MaxBytesReader(w, r.Body, 64<<10)).Decode(&body) != nil || body["title"] == nil {
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte(`{"error":"title requis (Content-Type: application/json)"}`))
		return
	}
	w.WriteHeader(http.StatusCreated)
	_ = json.NewEncoder(w).Encode(map[string]any{"id": 101, "title": body["title"]})
}

func (s *questServer) etat() map[string]any {
	av := map[string]vueAvancement{}
	for _, q := range s.quetes {
		av[q.ID] = s.vue(q)
	}
	version, _, _ := s.node(3*time.Second, "--version")
	return map[string]any{
		"parcours": s.parcours, "profil": s.profil, "quetes": s.quetes, "avancement": av, "score": s.score(),
		"node": strings.TrimSpace(version), "dossier": filepath.Join(s.dir, "code"),
	}
}

func (s *questServer) hEtat(w http.ResponseWriter, r *http.Request) {
	s.mu.Lock()
	defer s.mu.Unlock()
	repondre(w, s.etat())
}

func (s *questServer) hSauver(w http.ResponseWriter, r *http.Request) {
	q, rq, ok := s.lire(w, r)
	if !ok {
		return
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if err := os.WriteFile(s.chemin(q, rq.Fichier), []byte(rq.Code), 0o644); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	repondre(w, map[string]any{"ok": true})
}

func (s *questServer) hExecuter(w http.ResponseWriter, r *http.Request) {
	q, rq, ok := s.lire(w, r)
	if !ok {
		return
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	fichier := s.chemin(q, rq.Fichier)
	_ = os.WriteFile(fichier, []byte(rq.Code), 0o644)
	debut := time.Now()
	sortie, expire, code := s.node(5*time.Second, fichier)
	repondre(w, map[string]any{"sortie": sortie, "expire": expire, "codeSortie": code,
		"dureeMs": time.Since(debut).Milliseconds()})
}

func (s *questServer) hVerifier(w http.ResponseWriter, r *http.Request) {
	q, rq, ok := s.lire(w, r)
	if !ok {
		return
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	fichier := s.chemin(q, "defi")
	_ = os.WriteFile(fichier, []byte(rq.Code), 0o644)
	res, sortie := s.lancerTests(q, fichier)
	reussi := true
	for _, x := range res {
		reussi = reussi && x.OK
	}
	a := s.prog[q.ID]
	xpGagne := 0
	if !a.Fait {
		a.Essais++
		if reussi {
			noter(q, a)
			xpGagne = a.XP
		}
		s.sauverProgression()
	}
	repondre(w, map[string]any{"resultats": res, "sortie": sortie, "reussi": reussi,
		"xpGagne": xpGagne, "avancement": s.vue(q), "score": s.score()})
}

// hRequete : envoie une requête HTTP au serveur de l'élève (démarré au besoin).
func (s *questServer) hRequete(w http.ResponseWriter, r *http.Request) {
	q, rq, ok := s.lire(w, r)
	if !ok {
		return
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	_ = os.WriteFile(s.chemin(q, "defi"), []byte(rq.Code), 0o644)
	e, redemarre, err := s.essaiPour(q, rq.Code)
	if err != nil {
		repondre(w, map[string]any{"erreur": s.sansChemins(err.Error()), "redemarre": redemarre})
		return
	}
	chemin := strings.TrimSpace(rq.Chemin)
	if !strings.HasPrefix(chemin, "/") {
		chemin = "/" + chemin
	}
	entetes := map[string]string{}
	for _, l := range strings.Split(rq.Entetes, "\n") {
		if nom, val, ok := strings.Cut(l, ":"); ok && strings.TrimSpace(nom) != "" {
			// {{signature}} : ce qu'enverrait le prestataire de paiement pour ce corps
			if strings.Contains(val, "{{signature}}") {
				m := hmac.New(sha256.New, []byte(secretWebhook))
				m.Write([]byte(rq.Corps))
				val = strings.ReplaceAll(val, "{{signature}}", hex.EncodeToString(m.Sum(nil)))
			}
			entetes[strings.TrimSpace(nom)] = strings.TrimSpace(val)
		}
	}
	rep, err := envoyerEssai(e.sv.base, strings.ToUpper(strings.TrimSpace(rq.Methode)), chemin, rq.Corps, entetes)
	time.Sleep(60 * time.Millisecond) // laisse au serveur le temps d'écrire son journal
	console := s.sansChemins(tail(e.sv.out.String(), 12))
	if err != nil {
		repondre(w, map[string]any{"erreur": "requête impossible : " + err.Error(), "redemarre": redemarre, "console": console})
		return
	}
	repondre(w, map[string]any{"reponse": rep, "redemarre": redemarre, "console": console})
}

// hProfil enregistre l'inscription (prénom, nom, école) dans profil.json.
func (s *questServer) hProfil(w http.ResponseWriter, r *http.Request) {
	var p Profil
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 4<<10)).Decode(&p); err != nil {
		http.Error(w, "requête invalide", http.StatusBadRequest)
		return
	}
	p.Nom, p.Prenom, p.Ecole = strings.TrimSpace(p.Nom), strings.TrimSpace(p.Prenom), strings.TrimSpace(p.Ecole)
	for _, champ := range []string{p.Prenom, p.Nom, p.Ecole} {
		if champ == "" || len([]rune(champ)) > 80 {
			http.Error(w, "Prénom, nom et école sont obligatoires (80 caractères au plus).", http.StatusBadRequest)
			return
		}
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	p.InscritLe = time.Now().Format("2006-01-02 15:04")
	if s.profil != nil {
		p.InscritLe = s.profil.InscritLe // une correction de nom ne change pas la date d'inscription
	}
	s.profil = &p
	out, _ := json.MarshalIndent(p, "", "  ")
	_ = os.WriteFile(filepath.Join(s.dir, "profil.json"), out, 0o644)
	repondre(w, map[string]any{"profil": s.profil})
}

func (s *questServer) hIndice(w http.ResponseWriter, r *http.Request) {
	q, _, ok := s.lire(w, r)
	if !ok {
		return
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if a := s.prog[q.ID]; !a.Fait && a.Indices < len(q.Indices) {
		a.Indices++
		s.sauverProgression()
	}
	repondre(w, map[string]any{"avancement": s.vue(q), "score": s.score()})
}

func (s *questServer) hSolution(w http.ResponseWriter, r *http.Request) {
	q, _, ok := s.lire(w, r)
	if !ok {
		return
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if a := s.prog[q.ID]; !a.Fait && !a.SolutionVue {
		a.SolutionVue = true
		s.sauverProgression()
	}
	repondre(w, map[string]any{"avancement": s.vue(q), "score": s.score()})
}

func (s *questServer) hReinit(w http.ResponseWriter, r *http.Request) {
	q, rq, ok := s.lire(w, r)
	if !ok {
		return
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	_ = os.Remove(s.chemin(q, rq.Fichier))
	repondre(w, map[string]any{"avancement": s.vue(q), "score": s.score()})
}

// hReinitTout remet le score à zéro ; le code déjà écrit est conservé.
func (s *questServer) hReinitTout(w http.ResponseWriter, r *http.Request) {
	s.mu.Lock()
	defer s.mu.Unlock()
	for _, q := range s.quetes {
		s.prog[q.ID] = &Avancement{}
	}
	s.sauverProgression()
	repondre(w, s.etat())
}

// ---------- commande ----------

func cmdQuest(args []string) error {
	p, dir, port, ouvrir, selftest := parcoursConnus["api"], "", 4321, true, false
	for i := 0; i < len(args); i++ {
		switch a := args[i]; a {
		case "selftest":
			selftest = true
		case "--no-open":
			ouvrir = false
		case "--port", "--parcours":
			if i+1 >= len(args) {
				return fmt.Errorf("%s attend une valeur", a)
			}
			i++
			if a == "--port" {
				n, err := strconv.Atoi(args[i])
				if err != nil {
					return fmt.Errorf("port invalide : %s", args[i])
				}
				port = n
			} else {
				connu, ok := parcoursConnus[args[i]]
				if !ok {
					return fmt.Errorf("parcours inconnu : %s (api ou fp)", args[i])
				}
				p = connu
			}
		default:
			dir = a
		}
	}
	if dir == "" {
		dir = p.Dossier
	}
	if err := nodeAvailable(); err != nil {
		return err
	}
	if selftest {
		port = 0 // un port libre quelconque : seule l'API de démonstration est utile
	}
	var l net.Listener
	var err error
	for essai := 0; essai < 20; essai++ {
		cible := 0
		if port != 0 {
			cible = port + essai
		}
		if l, err = net.Listen("tcp", fmt.Sprintf("127.0.0.1:%d", cible)); err == nil {
			break
		}
	}
	if err != nil {
		return fmt.Errorf("aucun port libre à partir de %d", port)
	}
	url := "http://" + l.Addr().String() + "/"
	if !selftest {
		fmt.Println(c(bold+cyan, "\n  apilab quest") + " — " + p.Titre + " : " + p.SousTitre)
	}
	s, err := nouveauServeurQuete(dir, p)
	if err != nil {
		return err
	}
	s.demoURL = url + "demo"
	if selftest {
		go func() { _ = http.Serve(l, s.routes()) }()
		return s.selftest()
	}
	fmt.Printf("  %d quêtes · interface : %s\n", len(s.quetes), c(green, url))
	fmt.Println(c(dim, "  Votre code est enregistré dans "+filepath.Join(s.dir, "code")))
	fmt.Println(c(dim, "  Ctrl+C pour arrêter.\n"))

	// À l'arrêt, ne pas laisser tourner le serveur d'essai de l'élève.
	sig := make(chan os.Signal, 1)
	signal.Notify(sig, os.Interrupt, syscall.SIGTERM)
	go func() {
		<-sig
		s.mu.Lock()
		s.arreterEssai()
		os.Exit(0)
	}()
	if ouvrir {
		ouvrirNavigateur(url)
	}
	return http.Serve(l, s.routes())
}

func ouvrirNavigateur(url string) {
	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "windows":
		cmd = exec.Command("rundll32", "url.dll,FileProtocolHandler", url)
	case "darwin":
		cmd = exec.Command("open", url)
	default:
		cmd = exec.Command("xdg-open", url)
	}
	_ = cmd.Start()
}

// selftest (enseignant) : chaque correction doit passer, chaque fichier de
// départ doit échouer, et chaque exemple doit s'exécuter sans erreur.
func (s *questServer) selftest() error {
	tmp := filepath.Join(s.dir, "code", ".selftest.js")
	defer os.Remove(tmp)
	passe := func(code string, q *Quete) (bool, []Resultat) {
		_ = os.WriteFile(tmp, []byte(code), 0o644)
		res, _ := s.lancerTests(q, tmp)
		for _, x := range res {
			if !x.OK {
				return false, res
			}
		}
		return true, res
	}
	tout := true
	for _, q := range s.quetes {
		okSol, res := passe(q.Solution, q)
		okDep, _ := passe(q.Depart, q)
		_ = os.WriteFile(tmp, []byte(q.Exemple), 0o644)
		sortie, expire, code := s.node(5*time.Second, tmp)
		okEx := !expire && code == 0
		if okSol && !okDep && okEx {
			fmt.Printf("  %s %-28s %d tests · %d XP\n", c(green, "✔"), q.ID, len(res), q.XP)
			continue
		}
		tout = false
		fmt.Printf("  %s %s\n", c(red, "✖"), q.ID)
		if !okSol {
			for _, x := range res {
				if !x.OK {
					fmt.Println(c(dim, "      correction : "+x.Nom+"\n      "+strings.ReplaceAll(x.Detail, "\n", "\n      ")))
				}
			}
		}
		if okDep {
			fmt.Println(c(dim, "      le fichier de départ passe déjà tous les tests"))
		}
		if !okEx {
			fmt.Println(c(dim, "      l'exemple échoue ou ne se termine pas :\n"+sortie))
		}
	}
	if !tout {
		return fmt.Errorf("au moins une quête est à corriger")
	}
	fmt.Println(c(green+bold, fmt.Sprintf("\nLes %d quêtes du parcours « %s » sont cohérentes.", len(s.quetes), s.parcours.ID)))
	return nil
}
