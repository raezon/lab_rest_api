// Étape 7 — Limitation de débit (rate limiting) sans librairie
const express = require('express');
const app = express();

// Fenêtre fixe : MAX requêtes par IP et par fenêtre
function rateLimit({ windowMs, max }) {
  const compteurs = new Map(); // ip -> { n, debut }
  return (req, res, next) => {
    const now = Date.now();
    const c = compteurs.get(req.ip) || { n: 0, debut: now };
    if (now - c.debut > windowMs) { c.n = 0; c.debut = now; }
    c.n++;
    compteurs.set(req.ip, c);
    res.setHeader('X-RateLimit-Limit', max);
    res.setHeader('X-RateLimit-Remaining', Math.max(0, max - c.n));
    if (c.n > max) {
      res.setHeader('Retry-After', Math.ceil((c.debut + windowMs - now) / 1000));
      return res.status(429).json({ error: 'Trop de requêtes' });
    }
    next();
  };
}

app.use('/api', rateLimit({ windowMs: 60_000, max: 5 }));
app.get('/api/ping', (req, res) => res.json({ pong: true }));
app.get('/health', (req, res) => res.json({ status: 'ok' })); // hors limite

app.listen(process.env.PORT || 3000);
