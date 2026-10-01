// apilab — TP guidé en live coding pour apprendre les API REST en JavaScript.
//
// L'outil crée un espace de travail Node.js, affiche chaque étape (notion du
// cours, analogie, consignes), surveille le fichier de l'étudiant et relance
// automatiquement les tests à chaque sauvegarde, comme un correcteur en direct.
package main

import (
	"embed"
	"encoding/json"
	"fmt"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

//go:embed assets
var assets embed.FS

const stateFile = ".apilab.json"

type State struct {
	Current int          `json:"current"`
	Done    map[int]bool `json:"done"`
}

// ---------- couleurs terminal ----------
const (
	reset  = "\033[0m"
	bold   = "\033[1m"
	dim    = "\033[2m"
	red    = "\033[31m"
	green  = "\033[32m"
	yellow = "\033[33m"
	cyan   = "\033[36m"
	purple = "\033[35m"
)

func c(color, s string) string {
	if os.Getenv("NO_COLOR") != "" {
		return s
	}
	return color + s + reset
}

func main() {
	if len(os.Args) < 2 {
		usage()
		return
	}
	cmd, args := os.Args[1], os.Args[2:]
	var err error
	switch cmd {
	case "init":
		dir := "api-lab"
		if len(args) > 0 {
			dir = args[0]
		}
		err = cmdInit(dir)
	case "list", "ls":
		err = withState(func(st *State) error { cmdList(st); return nil })
	case "show":
		err = withState(func(st *State) error { return cmdShow(st, args) })
	case "check":
		err = withState(func(st *State) error { return cmdCheck(st, args) })
	case "watch", "live":
		err = withState(cmdWatch)
	case "hint":
		err = withState(func(st *State) error { return cmdHint(st, args) })
	case "solution":
		err = withState(func(st *State) error { return cmdSolution(st, args) })
	case "reset":
		err = withState(func(st *State) error { return cmdReset(st, args) })
	case "goto":
		err = withState(func(st *State) error { return cmdGoto(st, args) })
	case "selftest":
		err = withState(cmdSelftest)
	case "quest", "quete", "ui":
		err = cmdQuest(args)
	case "help", "-h", "--help":
		usage()
	default:
		fmt.Println(c(red, "Commande inconnue : "+cmd))
		usage()
	}
	if err != nil {
		fmt.Println(c(red, "✖ "+err.Error()))
		os.Exit(1)
	}
}

func usage() {
	fmt.Print(c(bold+cyan, "\n  apilab") + " — TP guidé : les API REST en JavaScript, en live coding\n\n")
	rows := [][2]string{
		{"init [dossier]", "crée l'espace de travail (par défaut ./api-lab)"},
		{"watch", "MODE LIVE : consignes + tests relancés à chaque sauvegarde"},
		{"list", "liste les étapes et votre progression"},
		{"show [n]", "affiche la notion du cours et les consignes de l'étape n"},
		{"check [n]", "lance les tests de l'étape n (par défaut : l'étape courante)"},
		{"hint [n]", "affiche un indice de plus pour l'étape n"},
		{"solution [n]", "écrit la correction dans solutions/ (après avoir essayé !)"},
		{"goto n", "se placer sur l'étape n"},
		{"reset n", "remet le fichier de l'étape n à son état de départ"},
		{"quest [dossier]", "INTERFACE WEB : API REST guidée (éditeur, requêtes, défis, score)"},
	}
	for _, r := range rows {
		fmt.Printf("  %-18s %s\n", c(purple, r[0]), r[1])
	}
	fmt.Println("\n  Démarrage rapide :\n    apilab init && cd api-lab && npm install && apilab watch")
	fmt.Println()
}

// ---------- état ----------

func withState(fn func(*State) error) error {
	b, err := os.ReadFile(stateFile)
	if err != nil {
		return fmt.Errorf("aucun espace de travail ici. Lancez « apilab init » puis « cd api-lab »")
	}
	var st State
	if err := json.Unmarshal(b, &st); err != nil {
		return err
	}
	if st.Done == nil {
		st.Done = map[int]bool{}
	}
	if st.Current < 1 || st.Current > len(steps) {
		st.Current = 1
	}
	err = fn(&st)
	out, _ := json.MarshalIndent(st, "", "  ")
	_ = os.WriteFile(stateFile, out, 0o644)
	return err
}

func stepArg(st *State, args []string) (*Step, error) {
	n := st.Current
	if len(args) > 0 {
		v, err := strconv.Atoi(args[0])
		if err != nil || v < 1 || v > len(steps) {
			return nil, fmt.Errorf("numéro d'étape invalide (1 à %d)", len(steps))
		}
		n = v
	}
	return &steps[n-1], nil
}

// ---------- commandes ----------

func cmdInit(dir string) error {
	if _, err := os.Stat(filepath.Join(dir, stateFile)); err == nil {
		return fmt.Errorf("%s existe déjà. Entrez dedans : cd %s", dir, dir)
	}
	if err := os.MkdirAll(filepath.Join(dir, "solutions"), 0o755); err != nil {
		return err
	}
	pkg, _ := assets.ReadFile("assets/package.json")
	if err := os.WriteFile(filepath.Join(dir, "package.json"), pkg, 0o644); err != nil {
		return err
	}
	err := fs.WalkDir(assets, "assets/starters", func(p string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return err
		}
		b, _ := assets.ReadFile(p)
		return os.WriteFile(filepath.Join(dir, filepath.Base(p)), b, 0o644)
	})
	if err != nil {
		return err
	}
	st, _ := json.MarshalIndent(State{Current: 1, Done: map[int]bool{}}, "", "  ")
	_ = os.WriteFile(filepath.Join(dir, stateFile), st, 0o644)
	_ = os.WriteFile(filepath.Join(dir, ".gitignore"), []byte("node_modules/\nsolutions/\n"), 0o644)

	fmt.Println(c(green, "✔ Espace de travail créé : "+dir))
	fmt.Println("  10 fichiers d'exercice (01-fetch.js … 10-pubsub.js) + package.json")
	fmt.Println(c(bold, "\nÉtapes suivantes :"))
	fmt.Println("  cd " + dir)
	fmt.Println("  npm install        " + c(dim, "# installe express et jsonwebtoken"))
	fmt.Println("  apilab watch       " + c(dim, "# ouvrez 01-fetch.js dans votre éditeur et codez !"))
	return nil
}

func cmdList(st *State) {
	fmt.Println(c(bold, "\n  Parcours apilab — "+progressBar(st)))
	module := ""
	for i, s := range steps {
		if s.Module != module {
			module = s.Module
			fmt.Println(c(dim, "  ── "+module))
		}
		mark := c(dim, "○")
		if st.Done[i+1] {
			mark = c(green, "✔")
		}
		cur := "  "
		if st.Current == i+1 {
			cur = c(cyan, "▶ ")
		}
		fmt.Printf("  %s%s %02d  %-34s %s\n", cur, mark, i+1, s.Title, c(dim, s.File))
	}
	fmt.Println()
}

func progressBar(st *State) string {
	n := 0
	for _, v := range st.Done {
		if v {
			n++
		}
	}
	bar := strings.Repeat("█", n) + strings.Repeat("░", len(steps)-n)
	return fmt.Sprintf("%s %d/%d", c(green, bar), n, len(steps))
}

func cmdShow(st *State, args []string) error {
	s, err := stepArg(st, args)
	if err != nil {
		return err
	}
	printStep(s)
	return nil
}

func printStep(s *Step) {
	line := c(cyan, strings.Repeat("─", 68))
	fmt.Println(line)
	fmt.Printf("%s  %s\n", c(bold+cyan, fmt.Sprintf("ÉTAPE %02d", s.ID)), c(bold, s.Title))
	fmt.Println(c(dim, "   "+s.Module+"  ·  fichier : "+s.File))
	fmt.Println(line)
	fmt.Println(c(purple, "📘 Notion du cours"))
	for _, l := range s.Theory {
		fmt.Println("   " + l)
	}
	if s.Analogy != "" {
		fmt.Println(c(yellow, "\n🍽  Analogie") + "\n   " + s.Analogy)
	}
	fmt.Println(c(purple, "\n🛠  À coder dans "+s.File))
	for i, l := range s.Tasks {
		fmt.Printf("   %d. %s\n", i+1, l)
	}
	if len(s.Try) > 0 {
		fmt.Println(c(purple, "\n🧪 Essayez vous-même (dans un 2e terminal)"))
		for _, l := range s.Try {
			fmt.Println("   " + c(dim, "$ ") + l)
		}
	}
	fmt.Println(c(dim, "\n   Bloqué ? apilab hint "+strconv.Itoa(s.ID)))
}

func cmdCheck(st *State, args []string) error {
	s, err := stepArg(st, args)
	if err != nil {
		return err
	}
	ok := runAndReport(s)
	if ok {
		st.Done[s.ID] = true
		if s.ID == st.Current && st.Current < len(steps) {
			st.Current++
			fmt.Println(c(cyan, fmt.Sprintf("→ Étape suivante : apilab show %d", st.Current)))
		}
	}
	return nil
}

func runAndReport(s *Step) bool {
	fmt.Println(c(bold, fmt.Sprintf("\n▶ Tests de l'étape %02d — %s", s.ID, s.Title)))
	res := s.Check(s)
	pass := 0
	for _, r := range res {
		if r.OK {
			pass++
			fmt.Println("  " + c(green, "✔ ") + r.Name)
		} else {
			fmt.Println("  " + c(red, "✖ ") + r.Name)
			if r.Detail != "" {
				for _, l := range strings.Split(r.Detail, "\n") {
					fmt.Println("      " + c(dim, l))
				}
			}
		}
	}
	ok := pass == len(res) && len(res) > 0
	if ok {
		fmt.Println(c(green+bold, fmt.Sprintf("\n  🎉 %d/%d tests réussis — étape validée !", pass, len(res))))
		if s.Takeaway != "" {
			fmt.Println(c(yellow, "  💡 À retenir : ") + s.Takeaway)
		}
	} else {
		fmt.Println(c(yellow, fmt.Sprintf("\n  %d/%d tests réussis. Corrigez, sauvegardez, on relance.", pass, len(res))))
	}
	return ok
}

func cmdWatch(st *State) error {
	fmt.Print("\033[2J\033[H")
	fmt.Println(c(bold+cyan, "apilab live") + c(dim, " — Ctrl+C pour quitter. Les tests se relancent à chaque sauvegarde.\n"))
	printStep(&steps[st.Current-1])
	var last time.Time
	hintLevel := map[int]int{}
	_ = hintLevel
	for {
		s := &steps[st.Current-1]
		info, err := os.Stat(s.File)
		if err != nil {
			return fmt.Errorf("fichier %s introuvable (apilab reset %d)", s.File, s.ID)
		}
		if info.ModTime().After(last) {
			if !last.IsZero() {
				fmt.Print("\033[2J\033[H")
				fmt.Println(c(dim, time.Now().Format("15:04:05")+"  modification détectée dans "+s.File))
			}
			last = info.ModTime()
			if runAndReport(s) {
				st.Done[s.ID] = true
				saveState(st)
				if st.Current == len(steps) {
					fmt.Println(c(green+bold, "\n🏆 Parcours terminé ! "+progressBar(st)))
					fmt.Println("   Vous avez vu : fetch, paradigmes, Express, CRUD, middlewares, JWT,")
					fmt.Println("   rate limiting, webhooks, temps réel et Pub/Sub. Bravo !")
					return nil
				}
				st.Current++
				saveState(st)
				fmt.Println(c(dim, "\n   Étape suivante dans 3 secondes…"))
				time.Sleep(3 * time.Second)
				fmt.Print("\033[2J\033[H")
				printStep(&steps[st.Current-1])
				last = time.Time{}
				if fi, err := os.Stat(steps[st.Current-1].File); err == nil {
					last = fi.ModTime() // attendre la première modification
				}
				fmt.Println(c(dim, "\n   ⏳ En attente de votre code dans "+steps[st.Current-1].File+"…"))
			}
		}
		time.Sleep(400 * time.Millisecond)
	}
}

func saveState(st *State) {
	out, _ := json.MarshalIndent(st, "", "  ")
	_ = os.WriteFile(stateFile, out, 0o644)
}

func cmdHint(st *State, args []string) error {
	s, err := stepArg(st, args)
	if err != nil {
		return err
	}
	key := fmt.Sprintf(".hint-%02d", s.ID)
	lvl := 0
	if b, err := os.ReadFile(key); err == nil {
		lvl, _ = strconv.Atoi(strings.TrimSpace(string(b)))
	}
	if lvl >= len(s.Hints) {
		lvl = len(s.Hints) - 1
	}
	fmt.Println(c(yellow, fmt.Sprintf("💡 Indice %d/%d — étape %02d", lvl+1, len(s.Hints), s.ID)))
	fmt.Println("   " + strings.ReplaceAll(s.Hints[lvl], "\n", "\n   "))
	if lvl+1 < len(s.Hints) {
		fmt.Println(c(dim, "   (relancez « apilab hint » pour un indice plus précis)"))
	} else {
		fmt.Println(c(dim, "   Dernier indice. Ensuite : apilab solution "+strconv.Itoa(s.ID)))
	}
	_ = os.WriteFile(key, []byte(strconv.Itoa(lvl+1)), 0o644)
	return nil
}

func cmdSolution(st *State, args []string) error {
	s, err := stepArg(st, args)
	if err != nil {
		return err
	}
	b, err := assets.ReadFile("assets/solutions/" + s.File)
	if err != nil {
		return err
	}
	dst := filepath.Join("solutions", s.File)
	_ = os.MkdirAll("solutions", 0o755)
	if err := os.WriteFile(dst, b, 0o644); err != nil {
		return err
	}
	fmt.Println(c(green, "✔ Correction écrite dans "+dst))
	fmt.Println(c(dim, "  Comparez-la avec votre fichier : votre version n'a pas été modifiée."))
	return nil
}

func cmdReset(st *State, args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("précisez l'étape : apilab reset 3")
	}
	s, err := stepArg(st, args)
	if err != nil {
		return err
	}
	b, _ := assets.ReadFile("assets/starters/" + s.File)
	if err := os.WriteFile(s.File, b, 0o644); err != nil {
		return err
	}
	delete(st.Done, s.ID)
	fmt.Println(c(green, "✔ "+s.File+" remis à zéro"))
	return nil
}

func cmdGoto(st *State, args []string) error {
	s, err := stepArg(st, args)
	if err != nil || len(args) == 0 {
		return fmt.Errorf("usage : apilab goto <n>")
	}
	st.Current = s.ID
	printStep(s)
	return nil
}

// selftest (enseignant) : copie les corrections et vérifie que tout passe.
func cmdSelftest(st *State) error {
	all := true
	for i := range steps {
		s := &steps[i]
		orig, _ := os.ReadFile(s.File)
		sol, _ := assets.ReadFile("assets/solutions/" + s.File)
		_ = os.WriteFile(s.File, sol, 0o644)
		if !runAndReport(s) {
			all = false
		}
		_ = os.WriteFile(s.File, orig, 0o644)
	}
	if !all {
		return fmt.Errorf("au moins une correction échoue")
	}
	fmt.Println(c(green+bold, "\nToutes les corrections passent."))
	return nil
}

// nodeAvailable vérifie la présence de Node.js.
func nodeAvailable() error {
	if _, err := exec.LookPath("node"); err != nil {
		return fmt.Errorf("Node.js est introuvable : installez Node.js ≥ 18")
	}
	return nil
}
