// Étape 3 — Première application Express
const express = require('express');
const app = express();
app.use(express.json());

// TODO 1 : GET /        → texte « Bienvenue au restaurant »
// TODO 2 : GET /health  → JSON { "status": "ok" }

// Le port est fourni par apilab via la variable d'environnement PORT
app.listen(process.env.PORT || 3000, () => console.log('API prête'));
