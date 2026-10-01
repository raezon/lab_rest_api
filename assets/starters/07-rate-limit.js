// Étape 7 — Limitation de débit (rate limiting) écrite à la main
const express = require('express');
const app = express();

// TODO : écrire la fabrique de middleware rateLimit({ windowMs, max })
//  - compter les requêtes par IP (req.ip) dans une Map
//  - réinitialiser le compteur quand la fenêtre est écoulée
//  - en-têtes X-RateLimit-Limit et X-RateLimit-Remaining
//  - au-delà de max : 429 { error } + en-tête Retry-After (secondes)
function rateLimit({ windowMs, max }) {
  return (req, res, next) => next();
}

app.use('/api', rateLimit({ windowMs: 60_000, max: 5 }));
app.get('/api/ping', (req, res) => res.json({ pong: true }));
app.get('/health', (req, res) => res.json({ status: 'ok' })); // ne doit PAS être limité

app.listen(process.env.PORT || 3000);
