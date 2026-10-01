// Étape 5 — Middlewares : logger, chrono, 404 et gestion d'erreurs
const express = require('express');
const app = express();
app.use(express.json());

// TODO 1 : middleware qui journalise « METHODE URL » et ajoute l'en-tête
//          X-Response-Time (ex : "3ms") à CHAQUE réponse.
//          Astuce : res.setHeader doit être appelé avant l'envoi de la réponse.

app.get('/api/ping', (req, res) => res.json({ pong: true }));
app.get('/api/boom', () => { throw new Error('Le four a pris feu'); });

// TODO 2 : toute route inconnue → 404 { "error": "Route inconnue" }
// TODO 3 : gestionnaire d'erreurs (4 arguments !) → 500 { "error": "Erreur interne" }
//          sans jamais renvoyer le message ni la stack au client

app.listen(process.env.PORT || 3000);
