// Étape 4 — CRUD REST complet sur /api/plats
const express = require('express');
const app = express();
app.use(express.json());

let plats = [{ id: 1, nom: 'Pizza', prix: 12.5 }];
let nextId = 2;

// TODO GET    /api/plats      → 200 + liste
// TODO GET    /api/plats/:id  → 200 + plat, ou 404 { error }
// TODO POST   /api/plats      → 201 + plat + en-tête Location ; 400 si nom absent ou prix non numérique
// TODO PUT    /api/plats/:id  → 200 + plat remplacé ; 404 si absent ; 400 si invalide
// TODO DELETE /api/plats/:id  → 204 sans corps ; 404 si absent

app.listen(process.env.PORT || 3000);
