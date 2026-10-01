# apilab dans un conteneur : ni Go ni Node.js à installer sur la machine de l'étudiant.
#
#   docker build -t apilab .
#   docker run --rm -p 127.0.0.1:4321:4321 -v "$PWD/travail:/work" apilab
#
# puis ouvrir http://127.0.0.1:4321/

# ---------- étape 1 : compilation du binaire Go ----------
FROM golang:1.24-alpine AS compilation
WORKDIR /src
COPY go.mod ./
COPY *.go ./
COPY assets ./assets
RUN CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o /apilab .

# ---------- étape 2 : dépendances Node des exercices ----------
FROM node:22-alpine AS dependances
WORKDIR /opt/apilab
COPY assets/package.json ./
RUN npm install --omit=dev --no-audit --no-fund

# ---------- étape 3 : image finale (Node.js + binaire) ----------
FROM node:22-alpine
COPY --from=compilation /apilab /usr/local/bin/apilab
COPY --from=dependances /opt/apilab/node_modules /opt/apilab/node_modules

# express et jsonwebtoken sont fournis par l'image : aucun « npm install » au démarrage.
ENV NODE_PATH=/opt/apilab/node_modules \
    APILAB_HOST=0.0.0.0 \
    HOME=/tmp

# /work reçoit le code, le score et le profil de l'élève : montez-y un dossier pour les conserver.
WORKDIR /work
VOLUME /work
EXPOSE 4321

ENTRYPOINT ["apilab"]
CMD ["quest", "--no-open"]
