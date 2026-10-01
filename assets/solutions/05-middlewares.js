// Étape 5 — Middlewares : logger, chrono, 404 et gestion d'erreurs
const express = require('express');
const app = express();
app.use(express.json());

// 1. Logger + chronomètre : ajoute X-Response-Time à chaque réponse
app.use((req, res, next) => {
  const debut = Date.now();
  const writeHead = res.writeHead;
  res.writeHead = function (...args) {
    res.setHeader('X-Response-Time', `${Date.now() - debut}ms`);
    return writeHead.apply(this, args);
  };
  console.log(`${req.method} ${req.url}`);
  next();
});

app.get('/api/ping', (req, res) => res.json({ pong: true }));
app.get('/api/boom', () => { throw new Error('Le four a pris feu'); });

// 2. Route inconnue → 404 JSON
app.use((req, res) => res.status(404).json({ error: 'Route inconnue' }));

// 3. Gestionnaire d'erreurs (4 arguments) → 500 JSON
app.use((err, req, res, next) => {
  console.error(err.message);
  res.status(500).json({ error: 'Erreur interne' });
});

app.listen(process.env.PORT || 3000);
