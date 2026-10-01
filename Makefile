# apilab — commandes courantes. « make » ou « make help » affiche la liste.

BIN     := apilab
IMAGE   := apilab
PORT    := 4321
TRAVAIL := $(CURDIR)/travail

.DEFAULT_GOAL := help
.PHONY: help build run run-fp cli test fmt dist clean docker-build docker-run docker-run-fp docker-test docker-clean

help: ## Affiche cette aide
	@echo "apilab — cibles disponibles :"
	@grep -E '^[a-z-]+:.*## ' $(MAKEFILE_LIST) | awk -F ':.*## ' '{printf "  make %-15s %s\n", $$1, $$2}'
	@echo ""
	@echo "Sans Go : make docker-run   ·   Avec Go et Node.js : make run"

# ---------- avec Go et Node.js installés ----------

build: ## Compile le binaire ./apilab
	go build -o $(BIN) .

run: build ## Lance l'interface web (API REST guidée) sur http://127.0.0.1:4321/
	./$(BIN) quest --port $(PORT)

run-fp: build ## Lance l'interface web sur le parcours « paradigme fonctionnel »
	./$(BIN) quest --parcours fp --port $(PORT)

cli: build ## Prépare le TP en terminal dans ./api-lab (puis : cd api-lab && ../apilab watch)
	@test -d api-lab || ./$(BIN) init api-lab
	cd api-lab && npm install --no-audit --no-fund
	@echo "→ cd api-lab && ../$(BIN) watch"

test: build ## Vérifie le code Go et les quêtes des deux parcours (corrections, départs, exemples)
	@test -z "$$(gofmt -l .)" || (echo "Fichiers à formater : $$(gofmt -l .)" && exit 1)
	go vet ./...
	./$(BIN) quest selftest .selftest/api
	./$(BIN) quest selftest --parcours fp .selftest/fp

fmt: ## Formate le code Go
	gofmt -w .

dist: ## Compile les binaires à distribuer (Linux, Windows, macOS) dans dist/
	@mkdir -p dist
	GOOS=linux   GOARCH=amd64 go build -trimpath -ldflags="-s -w" -o dist/apilab-linux .
	GOOS=windows GOARCH=amd64 go build -trimpath -ldflags="-s -w" -o dist/apilab.exe .
	GOOS=darwin  GOARCH=amd64 go build -trimpath -ldflags="-s -w" -o dist/apilab-mac-intel .
	GOOS=darwin  GOARCH=arm64 go build -trimpath -ldflags="-s -w" -o dist/apilab-mac-arm64 .
	@ls -lh dist

clean: ## Supprime le binaire et les dossiers de test (le travail des élèves est conservé)
	rm -rf $(BIN) $(BIN).exe .selftest

# ---------- avec Docker seulement (ni Go ni Node.js) ----------

docker-build: ## Construit l'image Docker « apilab »
	docker build -t $(IMAGE) .

docker-run: docker-build ## Lance l'interface web dans Docker ; le travail est conservé dans ./travail
	@mkdir -p "$(TRAVAIL)"
	@echo "→ Ouvrez http://127.0.0.1:$(PORT)/   (Ctrl+C pour arrêter)"
	docker run --rm --init -p 127.0.0.1:$(PORT):4321 --user $$(id -u):$$(id -g) -v "$(TRAVAIL):/work" $(IMAGE)

docker-run-fp: docker-build ## Idem, sur le parcours « paradigme fonctionnel »
	@mkdir -p "$(TRAVAIL)"
	@echo "→ Ouvrez http://127.0.0.1:$(PORT)/   (Ctrl+C pour arrêter)"
	docker run --rm --init -p 127.0.0.1:$(PORT):4321 --user $$(id -u):$$(id -g) -v "$(TRAVAIL):/work" $(IMAGE) quest --no-open --parcours fp

docker-test: docker-build ## Vérifie les quêtes des deux parcours à l'intérieur de l'image
	docker run --rm $(IMAGE) quest selftest /tmp/api
	docker run --rm $(IMAGE) quest selftest --parcours fp /tmp/fp

docker-clean: ## Supprime l'image Docker
	-docker rmi $(IMAGE)
